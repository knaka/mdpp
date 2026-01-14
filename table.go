package mdpp

import (
	"bytes"
	"io"
	"os"
	"path"
	"strings"

	myext "github.com/knaka/mdpp/ext"
	"github.com/knaka/mdpp/tblfm"

	gmast "github.com/yuin/goldmark/ast"
	gmextast "github.com/yuin/goldmark/extension/ast"

	//revive:disable-next-line:dot-imports
	. "github.com/knaka/go-utils"
)

// processMillerTable processes a table with Miller script, writes the result to writer, and returns the new writing position.
func processMillerTable(
	sourceMD []byte, // The source markdown content
	writer io.Writer, // The output destination
	writePos int, // The current write position in the source
	directiveNode *gmast.HTMLBlock, // The HTML block node containing the Miller directive
	millerScript string, // The Miller script to process the table
) (
	nextWritePos int, // The next write position after processing
) {
	nextWritePos = writePos
	prevNode := directiveNode.PreviousSibling()
	if prevNode == nil {
		return
	}
	// If table is managed as paragraph.
	if prevNode.Kind() == gmast.KindParagraph {
		tableLines := prevNode.Lines()
		if tableLines.Len() == 0 {
			return
		}
		tableStartPos := tableLines.At(0).Start
		tableEndPos := tableLines.At(tableLines.Len() - 1).Stop
		linePrefixStartPos := getPrefixStart(sourceMD, tableStartPos)
		tableMarkdown := tableLines.Value(sourceMD)
		tempDirPath, cleanupTempDir := mkdirTemp()
		defer cleanupTempDir()
		tempFilePath := path.Join(tempDirPath, "3202c41.md")
		Must(os.WriteFile(tempFilePath, []byte(tableMarkdown), 0600))
		mlrMDInplacePut(tempFilePath, millerScript)
		processedTableMarkdown := Value(os.ReadFile(tempFilePath))
		Must(writer.Write(sourceMD[writePos:linePrefixStartPos]))
		// No prefix
		if tableStartPos == linePrefixStartPos {
			Must(writer.Write(processedTableMarkdown))
		} else
		// Has a prefix
		{
			linePrefixText := string(sourceMD[linePrefixStartPos:tableStartPos])
			for line := range bytes.SplitSeq(processedTableMarkdown, []byte{'\n'}) {
				if len(strings.TrimSpace(string(line))) > 0 {
					Must(writer.Write([]byte(linePrefixText + string(line) + "\n")))
				}
			}
		}
		directiveLines := directiveNode.Lines()
		directiveEndPos := directiveLines.At(directiveLines.Len() - 1).Stop
		Must(writer.Write(sourceMD[tableEndPos+1 : directiveEndPos]))
		nextWritePos = directiveEndPos
	} else
	// If myext.Table is enabled.
	if prevNode.Kind() == gmextast.KindTable {
		segments := myext.SegmentsOf(prevNode)
		if segments == nil {
			return
		}
		tableStartPos := (*segments)[0].Start
		tableEndPos := (*segments)[len(*segments)-1].Stop
		linePrefixStartPos := getPrefixStart(sourceMD, tableStartPos)
		// tableMarkdown := sourceMD[tableStartPos:tableEndPos]
		tableMarkdown := ""
		for _, segment := range *segments {
			tableMarkdown = tableMarkdown + string(sourceMD[segment.Start:segment.Stop])
		}
		tempDirPath, cleanupTempDir := mkdirTemp()
		defer cleanupTempDir()
		tempFilePath := path.Join(tempDirPath, "3202c41.md")
		Must(os.WriteFile(tempFilePath, []byte(tableMarkdown), 0600))
		mlrMDInplacePut(tempFilePath, millerScript)
		processedTableMarkdown := Value(os.ReadFile(tempFilePath))
		Must(writer.Write(sourceMD[writePos:linePrefixStartPos]))
		// No prefix
		if tableStartPos == linePrefixStartPos {
			Must(writer.Write(processedTableMarkdown))
		} else
		// Has a prefix
		{
			linePrefixText := string(sourceMD[linePrefixStartPos:tableStartPos])
			for line := range bytes.SplitSeq(processedTableMarkdown, []byte{'\n'}) {
				if len(strings.TrimSpace(string(line))) > 0 {
					Must(writer.Write([]byte(linePrefixText + string(line) + "\n")))
				}
			}
		}
		directiveLines := directiveNode.Lines()
		directiveEndPos := directiveLines.At(directiveLines.Len() - 1).Stop
		Must(writer.Write(sourceMD[tableEndPos:directiveEndPos]))
		nextWritePos = directiveEndPos
	}
	return
}

// processTBLFMTable processes a table with TBLFM script, writes the result to writer, and returns the new writing position.
func processTBLFMTable(
	sourceMD []byte, // The source markdown content
	writer io.Writer, // The output destination
	writePos int, // The current write position in the source
	directiveNode *gmast.HTMLBlock, // The HTML block node containing the TBLFM directive
	tblfmScripts []string, // The TBLFM scripts to process the table
) (
	nextWritePos int, // The next write position after processing
) {
	nextWritePos = writePos
	tableNode := directiveNode.PreviousSibling()
	if tableNode == nil || tableNode.Kind() != gmextast.KindTable {
		return
	}
	segments := myext.SegmentsOf(tableNode)
	if segments == nil {
		return
	}

	// Extract table data
	table, _ := tableNode.(*gmextast.Table)
	hasHeader := false
	var tableData [][]string
	for rowNode := table.FirstChild(); rowNode != nil; rowNode = rowNode.NextSibling() {
		if _, ok := rowNode.(*gmextast.TableHeader); ok {
			hasHeader = true
		}
		var rowData []string
		for cellNode := rowNode.FirstChild(); cellNode != nil; cellNode = cellNode.NextSibling() {
			if cell, ok := cellNode.(*gmextast.TableCell); ok {
				cellLines := cell.Lines()
				cellText := string(sourceMD[cellLines.At(0).Start:cellLines.At(0).Stop])
				rowData = append(rowData, cellText)
			}
		}
		tableData = append(tableData, rowData)
	}

	// Apply TBLFM formulas
	tblfm.Apply(tableData, tblfmScripts, tblfm.WithHeader(hasHeader))

	// Write processed table
	tableStartPos := (*segments)[0].Start
	tableEndPos := (*segments)[len(*segments)-1].Stop
	linePrefixStartPos := getPrefixStart(sourceMD, tableStartPos)
	Must(writer.Write(sourceMD[writePos:linePrefixStartPos]))

	linePrefix := ""
	if tableStartPos != linePrefixStartPos {
		linePrefix = string(sourceMD[linePrefixStartPos:tableStartPos])
	}

	for rowIndex, rowData := range tableData {
		Must(writer.Write([]byte(linePrefix + "| " + strings.Join(rowData, " | ") + " |\n")))
		if rowIndex == 0 && hasHeader {
			separators := make([]string, len(rowData))
			for i := range separators {
				separators[i] = "---"
			}
			Must(writer.Write([]byte(linePrefix + "| " + strings.Join(separators, " | ") + " |\n")))
		}
	}

	directiveLines := directiveNode.Lines()
	directiveEndPos := directiveLines.At(directiveLines.Len() - 1).Stop
	Must(writer.Write(sourceMD[tableEndPos:directiveEndPos]))
	nextWritePos = directiveEndPos
	return
}
