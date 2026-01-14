package mdpp

import (
	"testing"

	gmast "github.com/yuin/goldmark/ast"
)

func TestAST(t *testing.T) {
	sourceMD := []byte(`これは[]()<!-- +LINK ./foo.md -->です。
	
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
	mdTree, _ := gmParse(sourceMD)
	gmast.Walk(mdTree, func(node gmast.Node, entering bool) (gmast.WalkStatus, error) {
		if !entering {
			return gmast.WalkContinue, nil
		}
		if textNode, ok := node.(*gmast.Text); ok {
			t.Logf("  Text: %d, %d", textNode.Segment.Start, textNode.Segment.Stop)
			return gmast.WalkContinue, nil
		}
		if node.Type() != gmast.TypeBlock {
			t.Log("  Not a block node")
			return gmast.WalkContinue, nil
		}
		lines := node.Lines()
		if lines.Len() == 0 {
			t.Log("  No lines")
			return gmast.WalkContinue, nil
		}
		for i := range lines.Len() {
			line := lines.At(i)
			t.Logf("  Line %d-%d: %s", line.Start, line.Stop, sourceMD[line.Start:line.Stop])
		}
		return gmast.WalkContinue, nil
	})
	mdTree.Dump(sourceMD, 0)
}

func TestLinkWithSegments(t *testing.T) {
	sourceMD := []byte(`This is a [link name](./foo.md) !

And this is a ![image name](./bar.png) !
`)
	mdTree, _ := gmParse(sourceMD)
	mdTree.Dump(sourceMD, 0)
	gmast.Walk(mdTree, func(node gmast.Node, entering bool) (gmast.WalkStatus, error) {
		if !entering {
			return gmast.WalkContinue, nil
		}
		if link, ok := node.(*gmast.Link); ok {
			t.Logf("Link found: Destination=%s, Title=%s", link.Destination, link.Title)
			for c := link.FirstChild(); c != nil; c = c.NextSibling() {
				if textNode, ok := c.(*gmast.Text); ok {
					t.Logf("  Text: %s", textNode.Value(sourceMD))
				} else {
					t.Log("  Unknown")
				}
			}
		}
		if link, ok := node.(*gmast.Image); ok {
			t.Logf("Image found: Destination=%s, Title=%s", link.Destination, link.Title)
			for c := link.FirstChild(); c != nil; c = c.NextSibling() {
				if textNode, ok := c.(*gmast.Text); ok {
					t.Logf("  Text: %s", textNode.Value(sourceMD))
				} else {
					t.Log("  Unknown")
				}
			}
		}
		return gmast.WalkContinue, nil
	})
}
