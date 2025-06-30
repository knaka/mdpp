package mdpp

import (
	"errors"
	"path"
	"strings"

	gmmeta "github.com/yuin/goldmark-meta"
	gmast "github.com/yuin/goldmark/ast"
)

// getMDTitle extracts the title in the following order:
//
//  1. From the metadata of the Markdown file.
//  2. From the only one H1 heading in the Markdown file.
//  3. From the file name of the Markdown file, without the `.md` extension.
func getMDTitle(source []byte, linkPath string) string {
	gmTree, gmContext := gmParse(source)
	m := gmmeta.Get(gmContext)
	for k, v := range m {
		if strings.ToLower(k) == "title" && v != nil {
			if titleStr, ok := v.(string); ok && titleStr != "" {
				return titleStr
			}
		}
	}
	var h1Node *gmast.Heading
	gmast.Walk(gmTree, func(node gmast.Node, entering bool) (gmast.WalkStatus, error) {
		if !entering {
			return gmast.WalkContinue, nil
		}
		if node.Kind() == gmast.KindHeading {
			heading, ok := node.(*gmast.Heading)
			if !ok {
				return gmast.WalkStop, errors.New("failed to downcast heading")
			}
			if heading.Level == 1 {
				if h1Node != nil {
					// If there are multiple H1 headings, the heading is not a title.
					h1Node = nil
					return gmast.WalkStop, nil
				}
				h1Node = heading
			}
		}
		return gmast.WalkContinue, nil
	})
	if h1Node != nil {
		return string(h1Node.Lines().Value(source))
	}
	base := path.Base(linkPath)
	base = strings.TrimSuffix(base, ".md")
	return base
}