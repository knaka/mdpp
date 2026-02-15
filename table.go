package mdpp

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/knaka/tblcalc/mlr"
	"github.com/knaka/tblcalc/tblfm"

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

// processTable processes a table with a custom processing function, writes the result to writer, and returns the new writing position.
func processTable(
	sourceMD []byte, // The source markdown content
	writer io.Writer, // The output destination
	writePos int, // The current write position in the source
	directiveNode *gmast.HTMLBlock, // The HTML block node containing the directive
	processFunc func(tableData [][]string, hasHeader bool) [][]string, // The function to process table data
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

	// Process table data with the provided function
	processedTableData := processFunc(tableData, hasHeader)

	// Get table boundaries
	var tableStartPos, linePrefixStartPos int
	firstCell, ok := table.FirstChild().FirstChild().(*gmextast.TableCell)
	if !ok {
		panic("ace5d42")
	}
	segments := firstCell.Lines()
	firstCellStart := segments.At(0).Start
	tableStartPos, linePrefixStartPos = getTableStartPosition(sourceMD, firstCellStart)

	var tableEndPos int
	lastCell, ok := table.LastChild().LastChild().(*gmextast.TableCell)
	if !ok {
		panic("ad0e1f3")
	}
	segments = lastCell.Lines()
	lastCellEnd := segments.At(segments.Len() - 1).Stop
	tableEndPos = getTableEndPosition(sourceMD, lastCellEnd)

	Must(writer.Write(sourceMD[writePos:linePrefixStartPos]))

	linePrefix := ""
	if tableStartPos != linePrefixStartPos {
		linePrefix = string(sourceMD[linePrefixStartPos:tableStartPos])
	}

	// Write processed table
	var lines []string
	for rowIndex, rowData := range processedTableData {
		lines = append(lines, linePrefix+"| "+strings.Join(rowData, " | ")+" |")
		if rowIndex == 0 && hasHeader {
			separators := make([]string, len(rowData))
			for i := range separators {
				if i >= len(table.Alignments) {
					separators[i] = "---"
					continue
				}
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
	return processTable(sourceMD, writer, writePos, directiveNode, func(tableData [][]string, hasHeader bool) [][]string {
		tempIn := Value(os.CreateTemp("", "data-*.tsv"))
		tempInPath := tempIn.Name()
		defer (func() {
			Ignore(tempIn.Close())
			Must(os.Remove(tempInPath))
		})()
		for _, rowData := range tableData {
			Must(fmt.Fprintln(tempIn, strings.Join(rowData, "\t")))
		}
		Must(tempIn.Close())
		tempOut := Value(os.CreateTemp("", "data-*.tsv"))
		tempOutPath := tempOut.Name()
		defer (func() {
			Ignore(tempOut.Close())
			Must(os.Remove(tempOutPath))
		})()
		err := mlr.Put(
			[]string{tempInPath},
			[]string{millerScript},
			true,
			"tsv",
			"tsv",
			tempOut,
		)
		if err != nil {
			return tableData
		}
		Must(tempOut.Close())
		tempOut2 := Value(os.Open(tempOutPath))
		defer (func() { Must(tempOut2.Close()) })()
		return Value(loadTableFromReader(tempOut2, "tsv"))
	})
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
	return processTable(sourceMD, writer, writePos, directiveNode, func(tableData [][]string, hasHeader bool) [][]string {
		// Apply TBLFM formulas
		Must(tblfm.Apply(tableData, tblfmScripts, tblfm.WithHeader(hasHeader)))
		return tableData
	})
}

// loadTableFromReader loads table data from a reader in the specified format.
// format should be "csv" or "tsv".
func loadTableFromReader(reader io.Reader, format string) ([][]string, error) {
	if format == "csv" {
		csvReader := csv.NewReader(reader)
		return csvReader.ReadAll()
	}

	// For TSV files, use bufio.Scanner to read line by line
	var tableData [][]string
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 {
			tableData = append(tableData, strings.Split(line, "\t"))
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return tableData, nil
}

// loadTableFromFile loads table data from a CSV or TSV file based on file extension.
func loadTableFromFile(filePath string) ([][]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer (func() { Must(file.Close()) })()

	ext := strings.ToLower(path.Ext(filePath))

	// Use encoding/csv for CSV files to properly handle quoted fields, commas, and newlines
	format := "tsv"
	if ext == ".csv" || ext == "" {
		format = "csv"
	}

	return loadTableFromReader(file, format)
}

// processTableInclude processes a table include directive, loads data from file, and writes the result to writer.
func processTableInclude(
	sourceMD []byte, // The source markdown content
	writer io.Writer, // The output destination
	writePos int, // The current write position in the source
	directiveNode *gmast.HTMLBlock, // The HTML block node containing the directive
	filePath string, // The path to the file to include
) (
	nextWritePos int, // The next write position after processing
) {
	return processTable(sourceMD, writer, writePos, directiveNode, func(tableData [][]string, hasHeader bool) [][]string {
		// Load table data from file and replace the existing table
		loadedData, err := loadTableFromFile(filePath)
		if err != nil {
			// If file cannot be read, return original table data
			return tableData
		}
		if len(loadedData) == 0 {
			// If file is empty, return original table data
			return tableData
		}
		// Return loaded data (assuming first row is header)
		return loadedData
	})
}
