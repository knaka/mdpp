package mdpp

import (
	"fmt"
	"io"
	"os"

	gmast "github.com/yuin/goldmark/ast"
	gmtext "github.com/yuin/goldmark/text"

	//revive:disable-next-line:dot-imports
	. "github.com/knaka/go-utils"
)

// isEscaped checks if the character at pos is escaped by a backslash.
func isEscaped(data []byte, pos int) bool {
	return pos > 0 && data[pos-1] == '\\'
}

// findLinkTextStart finds the start position of the markdown link text by scanning backward.
// Returns -1 if the link start cannot be found.
func findLinkTextStart(sourceMD []byte, startPos int) int {
	// Find the closing parenthesis of the link destination
	pos := startPos
	for pos >= 0 {
		if sourceMD[pos] == '(' {
			pos--
			break
		}
		pos--
	}
	if pos < 0 {
		return -1
	}

	// Find the opening bracket of the link text, considering nested brackets
	bracketNestLevel := 0
	for pos >= 0 {
		if sourceMD[pos] == ']' && !isEscaped(sourceMD, pos) {
			bracketNestLevel++
		} else if sourceMD[pos] == '[' && !isEscaped(sourceMD, pos) {
			bracketNestLevel--
			if bracketNestLevel == 0 {
				return pos
			}
		}
		pos--
	}
	return -1
}

// processSyncTitleDirective processes a SYNC_TITLE directive, writes the result to writer, and returns the new writing position.
func processSyncTitleDirective(
	sourceMD []byte, // The source markdown content
	writer io.Writer, // The output destination
	writePos int, // The current write position in the source
	directiveNode gmast.Node, // The AST node containing the directive
	directiveSegments *gmtext.Segments, // The text segments of the directive
) (
	nextWritePos int, // The next write position after processing.
) {
	nextWritePos = writePos
	if directiveSegments == nil || directiveSegments.Len() != 1 {
		return
	}
	prevNode := directiveNode.PreviousSibling()
	if prevNode == nil || prevNode.Kind() != gmast.KindLink {
		return
	}
	linkNode, ok := prevNode.(*gmast.Link)
	if !ok {
		return
	}
	linkedFilePath := string(linkNode.Destination)
	// If reading the target file fails, return without doing anything.
	linkedFileMD, err := os.ReadFile(linkedFilePath)
	if err != nil {
		return
	}
	linkedFileTitle := getMDTitle(linkedFileMD, linkedFilePath)
	directiveStartPos := directiveSegments.At(0).Start
	// Find the start of the link text by scanning backward from the directive
	linkTextStartPos := findLinkTextStart(sourceMD, directiveStartPos-1)
	if linkTextStartPos < 0 {
		// If we cannot find the link start, skip processing
		return
	}
	Must(writer.Write(sourceMD[writePos:linkTextStartPos]))
	Must(fmt.Fprintf(writer, "[%s](%s)", linkedFileTitle, linkedFilePath))
	nextWritePos = directiveSegments.At(directiveSegments.Len() - 1).Stop
	Must(writer.Write(sourceMD[directiveStartPos:nextWritePos]))
	return
}
