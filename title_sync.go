package mdpp

import (
	"fmt"
	"io"
	"os"

	gmast "github.com/yuin/goldmark/ast"
	gmtext "github.com/yuin/goldmark/text"

	//revive:disable-next-line dot-imports
	. "github.com/knaka/go-utils"
)

// processSyncTitleDirective processes a SYNC_TITLE directive and writes the result to writer
func processSyncTitleDirective(sourceMD []byte, writer io.Writer, cursor int, node gmast.Node, segments *gmtext.Segments) int {
	prevNode := node.PreviousSibling()
	if prevNode == nil || prevNode.Kind() != gmast.KindLink {
		return cursor
	}
	linkNode, _ := prevNode.(*gmast.Link)
	linkPath := string(linkNode.Destination)
	targetMD, err := os.ReadFile(linkPath)
	if err != nil {
		return cursor
	}
	title := getMDTitle(targetMD, linkPath)
	cmtStart := segments.At(0).Start
	linkStart := cmtStart - 1
	for ; linkStart >= 0; linkStart-- {
		if sourceMD[linkStart] == '(' {
			linkStart--
			break
		}
	}
	nest := 0
	for ; ; linkStart-- {
		if sourceMD[linkStart] == ']' && (linkStart == 0 || sourceMD[linkStart-1] != '\\') {
			nest++
		} else if sourceMD[linkStart] == '[' && (linkStart == 0 || sourceMD[linkStart-1] != '\\') {
			nest--
			if nest == 0 {
				break
			}
		}
	}
	_ = V(writer.Write(sourceMD[cursor:linkStart]))
	_ = V(fmt.Fprintf(writer, "[%s](%s)", title, linkPath))
	newCursor := segments.At(segments.Len() - 1).Stop
	_ = V(writer.Write(sourceMD[cmtStart:newCursor]))
	return newCursor
}