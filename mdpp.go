package mdpp

import (
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	myext "github.com/knaka/mdpp/ext"
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
			myext.Link,
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

var regexpIncludeDirective = sync.OnceValue(func() *regexp.Regexp {
	// Matches the INCLUDE directive in HTML comments, e.g.:
	//
	//   <!-- +INCLUDE: ./path/to/file.md -->
	return regexp.MustCompile(`(?i)^<!--\s*\+INCLUDE:\s*([^ ]+?)\s*-->\s*$`)
})

const includePathIndex = 1

var regexpEndDirective = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?i)^<!--\s*\+END\s*-->\s*$`)
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

// processIncludeDirectives processes +INCLUDE ... +END directives and returns the modified source
func processIncludeDirectives(sourceMD []byte) []byte {
	return processIncludeDirectivesWithLoopDetection(sourceMD, make(map[string]bool))
}

// processIncludeDirectivesWithLoopDetection processes +INCLUDE ... +END directives with cycle detection
func processIncludeDirectivesWithLoopDetection(sourceMD []byte, visited map[string]bool) []byte {
	lines := strings.Split(string(sourceMD), "\n")
	var result []string
	includeDepth := 0 // Track nesting depth to avoid processing nested directives
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		// Check for +INCLUDE directive only at top level (depth 0)
		if includeDepth == 0 && regexpIncludeDirective().MatchString(strings.TrimSpace(line)) {
			matches := regexpIncludeDirective().FindStringSubmatch(strings.TrimSpace(line))
			if len(matches) > 0 {
				includePath := matches[includePathIndex]
				// Get canonical path for cycle detection
				canonicalPath, err := filepath.Abs(includePath)
				if err != nil {
					// If we can't get absolute path, fall back to original path
					canonicalPath = includePath
				} else {
					// Clean the path to resolve . and .. components
					canonicalPath = filepath.Clean(canonicalPath)
				}
				canonicalPath, err = filepath.EvalSymlinks(canonicalPath)
				if err != nil {
					// If we can't evaluate symlinks, fall back to original path
					canonicalPath = includePath
				}
				// Find the corresponding +END directive
				endIndex := -1
				tempDepth := 1
				for j := i + 1; j < len(lines); j++ {
					if regexpIncludeDirective().MatchString(strings.TrimSpace(lines[j])) {
						tempDepth++
					} else if regexpEndDirective().MatchString(strings.TrimSpace(lines[j])) {
						tempDepth--
						if tempDepth == 0 {
							endIndex = j
							break
						}
					}
				}
				if endIndex == -1 {
					// No matching +END found, just add the line as-is
					result = append(result, line)
					continue
				}
				// Add the +INCLUDE directive line
				result = append(result, line)
				// Check for cycles using canonical path
				if visited[canonicalPath] {
					// Cycle detected, skip inclusion but preserve directives
					// Add content between directives as-is
					for k := i + 1; k < endIndex; k++ {
						result = append(result, lines[k])
					}
					result = append(result, lines[endIndex])
					i = endIndex
					continue
				}
				// Read and include the external file content
				if includeContent, err := os.ReadFile(includePath); err == nil {
					// Mark this canonical path as visited to prevent cycles
					newVisited := make(map[string]bool)
					maps.Copy(newVisited, visited)
					newVisited[canonicalPath] = true
					// Recursively process the included content for nested includes
					processedContent := processIncludeDirectivesWithLoopDetection(includeContent, newVisited)
					// Add the processed content (without trailing newline to avoid extra blank lines)
					content := strings.TrimRight(string(processedContent), "\n")
					if content != "" {
						result = append(result, content)
					}
				} else {
					// File not found, preserve existing content between directives
					for k := i + 1; k < endIndex; k++ {
						result = append(result, lines[k])
					}
				}
				// Add the +END directive line
				result = append(result, lines[endIndex])
				// Skip to after the +END directive
				i = endIndex
				continue
			}
		}

		// Track include depth for nested directives
		if regexpIncludeDirective().MatchString(strings.TrimSpace(line)) {
			includeDepth++
		} else if regexpEndDirective().MatchString(strings.TrimSpace(line)) {
			includeDepth--
		}

		result = append(result, line)
	}
	return []byte(strings.Join(result, "\n"))
}

// Process parses the source markdown, detects directives in HTML comments, applies modifications, and writes the result to the writer. If dirPathOpt is not nil, it changes the working directory to that path before processing.
//
// Supported directives:
//   - INCLUDE ... END : Include the content of an external Markdown file.
//   - SYNC_TITLE | TITLE : Extract the title from the linked Markdown file and use it as the link title.
//   - MLR | MILLER : Processes the table above the comment using a Miller script.
//   - CODE : Reads the content of the file specified and writes it as a code block.
//
// Planned features:
//   - H1INCLUDE, H2INCLUDE, ...
//   - TBLFM (?)
func Process(sourceMD []byte, writer io.Writer, dirPathOpt *string) error {
	// Change working directory if dirPathOpt is provided.
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
	// First, parse and process +INCLUDE ... +END directive
	sourceMD = processIncludeDirectives(sourceMD)

	// Then, parse the other directives
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
