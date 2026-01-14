package mdpp

import (
	"bytes"
	"io"
	"os"
	"path"
	"strings"

	gmast "github.com/yuin/goldmark/ast"

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
	if prevNode == nil || prevNode.Kind() != gmast.KindParagraph {
		return
	}
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
	return
}
