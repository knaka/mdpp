package mdpp

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"

	gm "github.com/yuin/goldmark"
	gmmeta "github.com/yuin/goldmark-meta"
	gmast "github.com/yuin/goldmark/ast"
	gmparser "github.com/yuin/goldmark/parser"
	gmtext "github.com/yuin/goldmark/text"

	//revive:disable-next-line dot-imports
	. "github.com/knaka/go-utils"
)

// gmParser returns a Goldmark parser.
var gmParser = sync.OnceValue(func() gmparser.Parser {
	return gm.New(
		gm.WithExtensions(
			gmmeta.Meta, // Enable the `Meta` extension to parse metadata in the Markdown document
			// Do not add `Table` extension here, as it transforms paragraphs into tables, retains the position of each cell, and discards the position of the table itself.
		),
	).Parser()
})

// gmParse parses the given Markdown source and returns the AST and context.
func gmParse(source []byte) (gmTree gmast.Node, gmContext gmparser.Context) {
	gmContext = gmparser.NewContext()
	// gmparser.WithInlineParsers(),
	gmTree = gmParser().Parse(gmtext.NewReader(source), gmparser.WithContext(gmContext))
	return gmTree, gmContext
}

// regexpMillerDirective returns a compiled regex that matches MLR/MILLER directives in HTML comments.
var regexpMillerDirective = sync.OnceValue(func() *regexp.Regexp {
	// Matches the MLR directive in HTML comments, e.g.:
	//
	//   <!-- +MLR: $Total = $UnitPrice * $Count -->
	//
	// or
	//
	//   <!-- +MLR:
	//     $Total = $UnitPrice * $Count
	//   -->
	//
	// In the second case, the "closure" part is stored in the `.Closure` member of the node.
	return regexp.MustCompile(`(?i)^<!--\s*\+(MLR|MILLER):\s*([^-]+?)\s*(-->\s*)?$`)
})

// millerScriptIndex is the index of the Miller script in the matches of the Miller directive regex.
const millerScriptIndex = 2

var regexpCodeDirective = sync.OnceValue(func() *regexp.Regexp {
	// Matches the code directive in HTML comments, e.g.:
	//
	//   <!-- +CODE: ./path/to/file -->
	return regexp.MustCompile(`(?i)^<!--\s*\+CODE:\s*([^ ]+?)\s*-->\s*$`)
})

const codeSrcIndex = 1

var regexpSyncTitleDirective = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?i)^<!--\s*\+(SYNC_TITLE|TITLE)\s*(-->\s*)?$`)
})

// getPrefixStart returns the BOL of the line at the given start position in the source markdown.
func getPrefixStart(sourceMD []byte, blockStart int) (prefixStart int) {
	for i := blockStart; true; i-- {
		if i == 0 || sourceMD[i-1] == '\n' || sourceMD[i-1] == '\r' {
			return i
		}
	}
	return // Should not be reached
}

// mkdirTemp creates a temporary directory and returns its path and a cleanup function.
func mkdirTemp() (string, func()) {
	tempDirPath := V(os.MkdirTemp("", "mdpp"))
	return tempDirPath, func() {
		os.RemoveAll(tempDirPath)
	}
}



var debug = false

// SetDebug sets the debug mode for the package.
func SetDebug(d bool) {
	debug = d
}

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

// Process parses the source markdown, detects directives in HTML comments, applies modifications, and writes the result to the writer. If dirPathOpt is not nil, it changes the working directory to that path before processing.
//
// Supported directives:
//   - SYNC_TITLE | TITLE : Extract the title from the linked Markdown file and use it as the link title.
//   - MLR | MILLER : Processes the table above the comment using a Miller script.
//   - CODE : Reads the content of the file specified and writes it as a code block.
//
// Planned features:
//   - TBLFM (?)
func Process(sourceMD []byte, writer io.Writer, dirPathOpt *string) error {
	if dirPathOpt != nil && *dirPathOpt != "" {
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		defer os.Chdir(currentDir)
		if err := os.Chdir(*dirPathOpt); err != nil {
			return fmt.Errorf("failed to change directory to %s: %w", *dirPathOpt, err)
		}
	}
	gmTree, _ := gmParse(sourceMD)
	if debug {
		gmTree.Dump(sourceMD, 0)
	}
	cursor := 0
	err := gmast.Walk(gmTree, func(node gmast.Node, entering bool) (gmast.WalkStatus, error) {
		if !entering {
			return gmast.WalkContinue, nil
		}
		switch node.Kind() {
		case gmast.KindHTMLBlock:
			htmlBlockNode := node.(*gmast.HTMLBlock)
			htmlBlockLines := htmlBlockNode.Lines()
			if htmlBlockLines.Len() == 0 {
				break
			}
			text := string(htmlBlockLines.Value(sourceMD))
			if !strings.Contains(text, "<!--") || !strings.Contains(text, "+") {
				break
			}
			// +MILLER | +MLR directive
			if matches := regexpMillerDirective().FindStringSubmatch(text); len(matches) > 0 {
				mlrScript := matches[millerScriptIndex]
				cursor = processMillerTable(sourceMD, writer, cursor, htmlBlockNode, mlrScript)
			} else
			// +CODE directive
			if matches := regexpCodeDirective().FindStringSubmatch(text); len(matches) > 0 {
				codePath := matches[codeSrcIndex]
				newCursor := processFencedCodeBlock(sourceMD, writer, cursor, htmlBlockNode, codePath)
				if newCursor > cursor {
					cursor = newCursor
				} else {
					// Fenced code block processing failed, try indented code block
					cursor = processIndentedCodeBlock(sourceMD, writer, cursor, htmlBlockNode, codePath)
				}
			}
		case gmast.KindRawHTML:
			rawHTMLNode, _ := node.(*gmast.RawHTML)
			segments := rawHTMLNode.Segments
			if segments.Len() == 0 {
				break
			}
			text := string(segments.Value(sourceMD))
			if !strings.Contains(text, "<!--") || !strings.Contains(text, "+") {
				break
			}
			// +TITLE directive gets the link path from the previous link node
			if matches := regexpSyncTitleDirective().FindStringSubmatch(text); len(matches) > 0 {
				cursor = processSyncTitleDirective(sourceMD, writer, cursor, node, segments)
			}
		}
		return gmast.WalkContinue, nil
	})
	if cursor < len(sourceMD) {
		_ = V(writer.Write(sourceMD[cursor:]))
	}
	return err
}
