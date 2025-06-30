package mdpp

import (
	"bytes"
	"fmt"
	"io"
	"os"

	gmast "github.com/yuin/goldmark/ast"

	//revive:disable-next-line dot-imports
	. "github.com/knaka/go-utils"
)

// processFencedCodeBlock processes a fenced code block with a CODE directive and writes the result to writer
func processFencedCodeBlock(sourceMD []byte, writer io.Writer, cursor int, htmlBlockNode *gmast.HTMLBlock, codePath string) int {
	prevNode := htmlBlockNode.PreviousSibling()
	if prevNode.Kind() != gmast.KindFencedCodeBlock {
		return cursor
	}
	fencedCodeBlock, _ := prevNode.(*gmast.FencedCodeBlock)
	segments := fencedCodeBlock.Lines()
	htmlBlockLines := htmlBlockNode.Lines()
	cmtStart := htmlBlockLines.At(0).Start
	cmtStop := htmlBlockLines.At(htmlBlockLines.Len() - 1).Stop
	prefix := ""
	var blockStart int
	var blockStop int
	if segments.Len() == 0 {
		blockStop = cmtStart - 1
	outer:
		for ; blockStop >= 0; blockStop-- {
			if bytes.HasPrefix(sourceMD[blockStop:], []byte("```")) {
				for blockStop > 0 && sourceMD[blockStop-1] == '`' {
					blockStop--
				}
				for i := blockStop; i >= 0; i-- {
					if i == 0 || sourceMD[i-1] == '\n' {
						prefix = string(sourceMD[i:blockStop])
						blockStop = i
						break outer
					}
				}
			} else if bytes.HasPrefix(sourceMD[blockStop:], []byte("~~~")) {
				for blockStop > 0 && sourceMD[blockStop-1] == '~' {
					blockStop--
				}
				for i := blockStop; i >= 0; i-- {
					if i == 0 || sourceMD[i-1] == '\n' {
						prefix = string(sourceMD[i:blockStop])
						blockStop = i
						break outer
					}
				}
			}
		}
		blockStart = blockStop
	} else {
		blockStart = segments.At(0).Start
		for blockStart > 0 && sourceMD[blockStart-1] != '\n' {
			blockStart--
		}
		blockStop = segments.At(segments.Len() - 1).Stop
		for i := blockStop; i < len(sourceMD); i++ {
			if sourceMD[i] == '`' {
				prefix = string(sourceMD[blockStop:i])
				break
			} else if sourceMD[i] == '~' {
				prefix = string(sourceMD[blockStop:i])
				break
			}
		}
	}
	codeContent, err := os.ReadFile(codePath)
	if err != nil {
		return cursor
	}
	_ = V(writer.Write(sourceMD[cursor:blockStart]))
	cursor = blockStop
	codeLines := bytes.Split(codeContent, []byte{'\n'})
	if len(codeLines) > 0 && len(codeLines[len(codeLines)-1]) == 0 {
		codeLines = codeLines[:len(codeLines)-1]
	}
	for _, line := range codeLines {
		_ = V(fmt.Fprintf(writer, "%s%s\n", prefix, line))
	}
	_ = V(writer.Write(sourceMD[cursor:cmtStop]))
	return cmtStop
}

// processIndentedCodeBlock processes an indented code block with a CODE directive and writes the result to writer
func processIndentedCodeBlock(sourceMD []byte, writer io.Writer, cursor int, htmlBlockNode *gmast.HTMLBlock, codePath string) int {
	prevNode := htmlBlockNode.PreviousSibling()
	if prevNode.Kind() != gmast.KindCodeBlock {
		return cursor
	}
	codeBlock, _ := prevNode.(*gmast.CodeBlock)
	segments := codeBlock.Lines()
	if segments.Len() == 0 {
		return cursor
	}
	htmlBlockLines := htmlBlockNode.Lines()
	cmtStop := htmlBlockLines.At(htmlBlockLines.Len() - 1).Stop
	blockStart := segments.At(0).Start
	blockStop := segments.At(segments.Len() - 1).Stop

	// Find the start of the line containing the first segment
	for blockStart > 0 && sourceMD[blockStart-1] != '\n' {
		blockStart--
	}

	// Get the indentation prefix from the first line
	prefix := ""
	firstLineStart := segments.At(0).Start
	for i := blockStart; i < firstLineStart; i++ {
		if sourceMD[i] == ' ' || sourceMD[i] == '\t' {
			prefix += string(sourceMD[i])
		} else {
			break
		}
	}

	codeContent, err := os.ReadFile(codePath)
	if err != nil {
		return cursor
	}
	_ = V(writer.Write(sourceMD[cursor:blockStart]))
	cursor = blockStop
	codeLines := bytes.Split(codeContent, []byte{'\n'})
	if len(codeLines) > 0 && len(codeLines[len(codeLines)-1]) == 0 {
		codeLines = codeLines[:len(codeLines)-1]
	}
	for _, line := range codeLines {
		_ = V(fmt.Fprintf(writer, "%s%s\n", prefix, line))
	}
	_ = V(writer.Write(sourceMD[cursor:cmtStop]))
	return cmtStop
}