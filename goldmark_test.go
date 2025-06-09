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
	mdTree := gmParse(sourceMD)
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
	t.Logf("Text: %s", mdTree.Text(sourceMD))
}
