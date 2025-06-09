package mdpp

import (
	"testing"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

func TestAST(t *testing.T) {
	source := []byte(`これは[]()<!-- +LINK ./foo.md -->です。
	
# A

* foo

  | AAA | BBB |
  | --- | --- |
  | x | y |

	<!-- for item -->

> | AAA | B | C | D |
> | --- | --- | --- | --- |
> | 1 | 10000000 | 1 | 0 |
> | 2 | 20 | 2 | 0 |
> | 3 | 30 | 3 | 255 |
> 
> <!-- foo -->

<!-- bar -->

bar

<!-- baz -->
`)
	reader := text.NewReader(source)
	md := goldmark.New(
		// goldmark.WithParserOptions(
		// 	parser.WithBlockParsers()
		// ),
		goldmark.WithExtensions(
			meta.Meta, // Enable meta extension to parse metadata of the Markdown document
			// extension.Table,
		),
	)
	doc := md.Parser().Parse(reader)
	ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		kind := node.Kind().String()
		t.Logf("Kind: %s", kind)
		if textNode, ok := node.(*ast.Text); ok {
			t.Logf("  Text: %d, %d", textNode.Segment.Start, textNode.Segment.Stop)
			return ast.WalkContinue, nil
		}
		if node.Type() != ast.TypeBlock {
			t.Log("  Not a block node")
			return ast.WalkContinue, nil
		}
		lines := node.Lines()
		if lines.Len() == 0 {
			t.Log("  No lines")
			return ast.WalkContinue, nil
		}
		for i := range lines.Len() {
			line := lines.At(i)
			t.Logf("  Line %d-%d: %s", line.Start, line.Stop, source[line.Start:line.Stop])
		}
		return ast.WalkContinue, nil
	})
	doc.Dump(source, 0)
	t.Logf("Text: %s", doc.Text(source))
}
