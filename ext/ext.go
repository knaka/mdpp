// Package ext is ext
package ext

import (
	myparser "github.com/knaka/mdpp/ext/parser"
	gm "github.com/yuin/goldmark"
	gmparser "github.com/yuin/goldmark/parser"
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
