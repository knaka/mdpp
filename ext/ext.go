// Package ext is ext
package ext

import (
	myparser "github.com/knaka/mdpp/ext/parser"
	gm "github.com/yuin/goldmark"
	gmast "github.com/yuin/goldmark/ast"
	gmparser "github.com/yuin/goldmark/parser"
	gmtext "github.com/yuin/goldmark/text"
	gmutil "github.com/yuin/goldmark/util"
)

type linkWithSegments struct{}

// Link is a link parsing extension with segment information
var Link = &linkWithSegments{}

func newLinkParser() gmparser.InlineParser {
	return myparser.DefaultLinkParser
}

func (e *linkWithSegments) Extend(m gm.Markdown) {
	m.Parser().AddOptions(
		gmparser.WithInlineParsers(
			gmutil.Prioritized(newLinkParser(), 0),
		),
	)
}

// SegmentOf returns the segment of the given node if it's a link or an image. Returns nil otherwise.
func SegmentOf(node gmast.Node) *gmtext.Segment {
	if segment, ok := myparser.LinkSegments[node]; ok {
		return &segment
	}
	return nil
}
