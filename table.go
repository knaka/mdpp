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

// getPrefixStart returns the BOL of the line at the given start position in the source markdown.
func getPrefixStart(sourceMD []byte, blockStart int) (prefixStart int) {
	for i := blockStart; true; i-- {
		if i == 0 || sourceMD[i-1] == '\n' || sourceMD[i-1] == '\r' {
			return i
		}
	}
	return // Should not be reached
}

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
	if prevNode.Kind() != gmextast.KindTable {
		return
	}
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
	return
}

// getTableStartPosition searches backward from cellStart to find the pipe character '|' that marks the start of the table,
// and returns both the table start position and the prefix start position (beginning of line).
func getTableStartPosition(sourceMD []byte, cellStart int) (tableStartPos int, prefixStartPos int) {
	// Search backward from cellStart to find the pipe '|'
	for i := cellStart - 1; i >= 0; i-- {
		if sourceMD[i] == '|' {
			tableStartPos = i
			break
		}
	}
	// Find the beginning of the line (prefix start)
	prefixStartPos = getPrefixStart(sourceMD, tableStartPos)
	return
}

// getTableEndPosition searches forward from cellEnd to find the pipe character '|' that marks the end of the table.
func getTableEndPosition(sourceMD []byte, cellEnd int) (tableEndPos int) {
	// Search forward from cellEnd to find the pipe '|'
	for i := cellEnd; i < len(sourceMD); i++ {
		if sourceMD[i] == '|' {
			tableEndPos = i + 1
			return
		}
	}
	panic("c431e30")
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
	var tableStartPos, linePrefixStartPos int
	firstCell, ok := table.FirstChild().FirstChild().(*gmextast.TableCell)
	if !ok {
		panic("b5ced57")
	}
	segments := firstCell.Lines()
	firstCellStart := segments.At(0).Start
	tableStartPos, linePrefixStartPos = getTableStartPosition(sourceMD, firstCellStart)

	var tableEndPos int
	lastCell, ok := table.LastChild().LastChild().(*gmextast.TableCell)
	if !ok {
		panic("b647e0c")
	}
	segments = lastCell.Lines()
	lastCellEnd := segments.At(segments.Len() - 1).Stop
	tableEndPos = getTableEndPosition(sourceMD, lastCellEnd)

	Must(writer.Write(sourceMD[writePos:linePrefixStartPos]))

	linePrefix := ""
	if tableStartPos != linePrefixStartPos {
		linePrefix = string(sourceMD[linePrefixStartPos:tableStartPos])
	}

	var lines []string
	for rowIndex, rowData := range tableData {
		lines = append(lines, linePrefix+"| "+strings.Join(rowData, " | ")+" |")
		if rowIndex == 0 && hasHeader {
			separators := make([]string, len(rowData))
			for i := range separators {
				switch table.Alignments[i] {
				case gmextast.AlignLeft:
					separators[i] = ":---"
				case gmextast.AlignCenter:
					separators[i] = ":---:"
				case gmextast.AlignRight, gmextast.AlignNone:
					separators[i] = "---"
				}
			}
			lines = append(lines, linePrefix+"| "+strings.Join(separators, " | ")+" |")
		}
	}
	Must(writer.Write([]byte(strings.Join(lines, "\n"))))

	directiveLines := directiveNode.Lines()
	directiveEndPos := directiveLines.At(directiveLines.Len() - 1).Stop
	Must(writer.Write(sourceMD[tableEndPos:directiveEndPos]))
	nextWritePos = directiveEndPos
	return
}
