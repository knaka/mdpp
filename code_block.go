package mdpp

import (
	"bytes"
	"fmt"
	"io"
	"os"

	gmast "github.com/yuin/goldmark/ast"

	//revive:disable-next-line:dot-imports
	. "github.com/knaka/go-utils"
)

// findLineStart finds the start position of the line containing pos by scanning backward.
func findLineStart(data []byte, pos int) int {
	for pos > 0 && data[pos-1] != '\n' {
		pos--
	}
	return pos
}

// splitLines splits content into lines, removing the trailing empty line if present.
func splitLines(content []byte) [][]byte {
	lines := bytes.Split(content, []byte{'\n'})
	if len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// processFencedCodeBlock processes a fenced code block with a CODE directive, writes the result to writer, and returns the new writing position.
func processFencedCodeBlock(
	sourceMD []byte, // The source markdown content
	writer io.Writer, // The output destination
	writePos int, // The current write position in the source
	directiveNode *gmast.HTMLBlock, // The HTML block node containing the CODE directive
	codeFilePath string, // The path to the code file to include
) (
	nextWritePos int, // The next write position after processing
) {
	nextWritePos = writePos
	prevNode := directiveNode.PreviousSibling()
	if prevNode == nil || prevNode.Kind() != gmast.KindFencedCodeBlock {
		return
	}
	fencedCodeBlock, ok := prevNode.(*gmast.FencedCodeBlock)
	if !ok {
		return
	}
	codeBlockLines := fencedCodeBlock.Lines()
	directiveLines := directiveNode.Lines()
	directiveStartPos := directiveLines.At(0).Start
	directiveEndPos := directiveLines.At(directiveLines.Len() - 1).Stop
	linePrefix := ""
	var codeBlockStartPos int
	var codeBlockEndPos int
	if codeBlockLines.Len() == 0 {
		codeBlockEndPos = directiveStartPos - 1
	outer:
		for ; codeBlockEndPos >= 0; codeBlockEndPos-- {
			if bytes.HasPrefix(sourceMD[codeBlockEndPos:], []byte("```")) {
				for codeBlockEndPos > 0 && sourceMD[codeBlockEndPos-1] == '`' {
					codeBlockEndPos--
				}
				for i := codeBlockEndPos; i >= 0; i-- {
					if i == 0 || sourceMD[i-1] == '\n' {
						linePrefix = string(sourceMD[i:codeBlockEndPos])
						codeBlockEndPos = i
						break outer
					}
				}
			} else if bytes.HasPrefix(sourceMD[codeBlockEndPos:], []byte("~~~")) {
				for codeBlockEndPos > 0 && sourceMD[codeBlockEndPos-1] == '~' {
					codeBlockEndPos--
				}
				for i := codeBlockEndPos; i >= 0; i-- {
					if i == 0 || sourceMD[i-1] == '\n' {
						linePrefix = string(sourceMD[i:codeBlockEndPos])
						codeBlockEndPos = i
						break outer
					}
				}
			}
		}
		codeBlockStartPos = codeBlockEndPos
	} else {
		codeBlockStartPos = findLineStart(sourceMD, codeBlockLines.At(0).Start)
		codeBlockEndPos = codeBlockLines.At(codeBlockLines.Len() - 1).Stop
		for i := codeBlockEndPos; i < len(sourceMD); i++ {
			if sourceMD[i] == '`' {
				linePrefix = string(sourceMD[codeBlockEndPos:i])
				break
			} else if sourceMD[i] == '~' {
				linePrefix = string(sourceMD[codeBlockEndPos:i])
				break
			}
		}
	}
	codeFileContent, err := os.ReadFile(codeFilePath)
	if err != nil {
		return
	}
	Must(writer.Write(sourceMD[writePos:codeBlockStartPos]))
	codeLines := splitLines(codeFileContent)
	for _, line := range codeLines {
		Must(fmt.Fprintf(writer, "%s%s\n", linePrefix, line))
	}
	Must(writer.Write(sourceMD[codeBlockEndPos:directiveEndPos]))
	nextWritePos = directiveEndPos
	return
}

// processIndentedCodeBlock processes an indented code block with a CODE directive, writes the result to writer, and returns the new writing position.
func processIndentedCodeBlock(
	sourceMD []byte, // The source markdown content
	writer io.Writer, // The output destination
	writePos int, // The current write position in the source
	directiveNode *gmast.HTMLBlock, // The HTML block node containing the CODE directive
	codeFilePath string, // The path to the code file to include
) (
	nextWritePos int, // The next write position after processing
) {
	nextWritePos = writePos
	prevNode := directiveNode.PreviousSibling()
	if prevNode == nil || prevNode.Kind() != gmast.KindCodeBlock {
		return
	}
	codeBlock, ok := prevNode.(*gmast.CodeBlock)
	if !ok {
		return
	}
	codeBlockLines := codeBlock.Lines()
	if codeBlockLines.Len() == 0 {
		return
	}
	directiveLines := directiveNode.Lines()
	directiveEndPos := directiveLines.At(directiveLines.Len() - 1).Stop
	codeBlockStartPos := findLineStart(sourceMD, codeBlockLines.At(0).Start)
	codeBlockEndPos := codeBlockLines.At(codeBlockLines.Len() - 1).Stop

	// Get the indentation prefix from the first line
	indentPrefix := ""
	firstLineContentStart := codeBlockLines.At(0).Start
	for i := codeBlockStartPos; i < firstLineContentStart; i++ {
		if sourceMD[i] == ' ' || sourceMD[i] == '\t' {
			indentPrefix += string(sourceMD[i])
		} else {
			break
		}
	}

	codeFileContent, err := os.ReadFile(codeFilePath)
	if err != nil {
		return
	}
	Must(writer.Write(sourceMD[writePos:codeBlockStartPos]))
	codeLines := splitLines(codeFileContent)
	for _, line := range codeLines {
		Must(fmt.Fprintf(writer, "%s%s\n", indentPrefix, line))
	}
	Must(writer.Write(sourceMD[codeBlockEndPos:directiveEndPos]))
	nextWritePos = directiveEndPos
	return
}
