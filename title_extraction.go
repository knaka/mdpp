package mdpp

import (
	"errors"
	"path"
	"strings"

	gmmeta "github.com/yuin/goldmark-meta"
	gmast "github.com/yuin/goldmark/ast"
)

// getMDTitle extracts the title from a Markdown file in the following order:
//
//  1. From the metadata of the Markdown file.
//  2. From the only one H1 heading in the Markdown file.
//  3. From the file name of the Markdown file, without the `.md` extension.
func getMDTitle(
	markdownContent []byte, // The Markdown file content
	filePath string, // The file path used as fallback for title extraction
) string {
	syntaxTree, parseContext := gmParse(markdownContent)
	metadata := gmmeta.Get(parseContext)
	for key, value := range metadata {
		if strings.ToLower(key) == "title" && value != nil {
			if titleStr, ok := value.(string); ok && titleStr != "" {
				return titleStr
			}
		}
	}
	var h1HeadingNode *gmast.Heading
	gmast.Walk(syntaxTree, func(node gmast.Node, entering bool) (gmast.WalkStatus, error) {
		if !entering {
			return gmast.WalkContinue, nil
		}
		if node.Kind() == gmast.KindHeading {
			headingNode, ok := node.(*gmast.Heading)
			if !ok {
				return gmast.WalkStop, errors.New("failed to cast to heading node")
			}
			if headingNode.Level == 1 {
				if h1HeadingNode != nil {
					// If there are multiple H1 headings, none is considered the title.
					h1HeadingNode = nil
					return gmast.WalkStop, nil
				}
				h1HeadingNode = headingNode
			}
		}
		return gmast.WalkContinue, nil
	})
	if h1HeadingNode != nil {
		return string(h1HeadingNode.Lines().Value(markdownContent))
	}
	baseName := path.Base(filePath)
	baseName = strings.TrimSuffix(baseName, ".md")
	return baseName
}
