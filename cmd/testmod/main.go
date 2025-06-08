// Main of test
package main

import (
	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

func main() {
	source := []byte(`# Hello World

foo


| AAA | B | C | D |
| --- | --- | --- | --- |
| 1 | 10000000 | 1 | 0 |
| 2 | 20 | 2 | 0 |
| 3 | 30 | 3 | 0 |

<!-- +MLR:
  $D = $B * $C
-->

bar
`)
	// renderer := markdown.NewRenderer(markdown.WithHeadingStyle(markdown.HeadingStyleATX))
	md := goldmark.New(
		goldmark.WithExtensions(
			meta.Meta, // Enable meta extension to parse metadata of the Markdown document
			extension.Table,
		),
		// goldmark.WithRenderer(
		// 	markdown.NewRenderer(
		// 		markdown.WithHeadingStyle(markdown.HeadingStyleATX),
		// 	),
		// ),
	)
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)
	doc.Dump(source, 0)
	// doc.Text(source)

	// var buf bytes.Buffer
	// err := md.Renderer().Render(&buf, source, doc) // sourceは元のソース、docは修正されたAST
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Println("--- Modified Markdown ---")
	// fmt.Println(buf.String())
}
