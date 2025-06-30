package mdpp

import (
	"bytes"
	"io"
	"os"
	"path"
	"strings"

	gmast "github.com/yuin/goldmark/ast"

	//revive:disable-next-line dot-imports
	. "github.com/knaka/go-utils"
)

// processMillerTable processes a table with Miller script and writes the result to writer
func processMillerTable(sourceMD []byte, writer io.Writer, cursor int, htmlBlockNode *gmast.HTMLBlock, mlrScript string) int {
	prevNode := htmlBlockNode.PreviousSibling()
	if prevNode.Kind() != gmast.KindParagraph {
		return cursor
	}
	tableLines := prevNode.Lines()
	if tableLines.Len() == 0 {
		return cursor
	}
	tableStart := tableLines.At(0).Start
	tableStop := tableLines.At(tableLines.Len() - 1).Stop
	prefixStart := getPrefixStart(sourceMD, tableStart)
	markdownTableText := tableLines.Value(sourceMD)
	tempDirPath, tempDirCleanFn := mkdirTemp()
	defer tempDirCleanFn()
	tempFilePath := path.Join(tempDirPath, "3202c41.md")
	V0(os.WriteFile(tempFilePath, []byte(markdownTableText), 0600))
	mlrMDInplacePut(tempFilePath, mlrScript)
	resultMDTable := V(os.ReadFile(tempFilePath))
	_ = V(writer.Write(sourceMD[cursor:prefixStart]))
	// No prefix
	if tableStart == prefixStart {
		_ = V(writer.Write(resultMDTable))
	} else
	// Has a prefix
	{
		prefixText := string(sourceMD[prefixStart:tableStart])
		for line := range bytes.SplitSeq(resultMDTable, []byte{'\n'}) {
			if len(strings.TrimSpace(string(line))) > 0 {
				_ = V(writer.Write([]byte(prefixText + string(line) + "\n")))
			}
		}
	}
	htmlBlockLines := htmlBlockNode.Lines()
	newCursor := htmlBlockLines.At(htmlBlockLines.Len() - 1).Stop
	_ = V(writer.Write(sourceMD[tableStop+1 : newCursor]))
	return newCursor
}