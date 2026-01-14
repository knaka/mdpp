// Package ext is ext
package ext

import (
	mylink "github.com/knaka/mdpp/ext/link"
	mytable "github.com/knaka/mdpp/ext/table"
	gm "github.com/yuin/goldmark"
	gmast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	gmparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	gmtext "github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	gmutil "github.com/yuin/goldmark/util"
)

type linkWithSegments struct{}

// Link is a link parsing extension with segment information
var Link = &linkWithSegments{}

func newLinkParser() gmparser.InlineParser {
	return mylink.DefaultLinkParser
}

func newTableParagraphTransformer() parser.ParagraphTransformer {
	return mytable.DefaultTableParagraphTransformer
}

func (e *linkWithSegments) Extend(m gm.Markdown) {
	m.Parser().AddOptions(
		gmparser.WithInlineParsers(
			gmutil.Prioritized(newLinkParser(), 0),
		),
	)
}

type tableWithSegments struct {
	options []mytable.TableOption
}

// Table is a table trasforming extension with segment information
var Table = &tableWithSegments{}

func (e *tableWithSegments) Extend(m gm.Markdown) {
	m.Parser().AddOptions(
		parser.WithParagraphTransformers(
			gmutil.Prioritized(newTableParagraphTransformer(), 200),
		),
		gmparser.WithASTTransformers(
			gmutil.Prioritized(mytable.NewTableASTTransformer(), 0),
		),
	)
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(mytable.NewTableHTMLRenderer(e.options...), 500),
	))
}

// SegmentsOf returns the segment of the given node if it's a link or an image. Returns nil otherwise.
func SegmentsOf(node gmast.Node) *[]gmtext.Segment {
	if segments, ok := mylink.SegmentsMap[node]; ok {
		return &segments
	}
	if segments, ok := mytable.SegmentsMap[node]; ok {
		return &segments
	}
	return nil
}
