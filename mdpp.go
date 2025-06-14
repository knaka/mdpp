package mdpp

import (
	"bytes"
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

const linkIndex = 1

// getPrefixStart returns the prefix of the line at the given start position in the source markdown.
func getPrefixStart(sourceMD []byte, blockStart int) (prefixStart int) {
	for i := blockStart; true; i-- {
		if i == -1 || sourceMD[i] == '\n' || sourceMD[i] == '\r' {
			return i + 1
		}
	}
	return
}

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

func getMDTitle(source []byte, linkPath string) string {
	gmTree, gmContext := gmParse(source)
	m := gmmeta.Get(gmContext)
	if m != nil {
		for k, v := range m {
			if strings.ToLower(k) == "title" && v != nil {
				if titleStr, ok := v.(string); ok && titleStr != "" {
					return titleStr
				}
			}
		}
	}
	title := ""
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
				// Return the first level 1 heading as the title
				if heading.Lines().Len() > 0 {
					title = string(heading.Lines().Value(source))
					return gmast.WalkStop, nil
				}
			}
		}
		return gmast.WalkContinue, nil
	})
	if title != "" {
		return title
	}
	base := path.Base(linkPath)
	if strings.HasSuffix(base, ".md") {
		base = base[:len(base)-3]
	}
	return base
}

// Process parses the source markdown, detects directives in HTML comments, applies modifications, and writes the result to the writer. If dirPathOpt is not nil, it changes the working directory to that path before processing.
//
// Supported directives:
//   - SYNC_TITLE | TITLE : Extract the title from the linked Markdown file and use it as the link title.
//   - MLR | MILLER : Processes the table above the comment using a Miller script.
//
// Planned features:
//   - MLR_SRC | MILLER_SRC : Reads the Miller script from the specified file and applies it to the table above the comment.
//   - CODE : Reads the content of the file specified and writes it as a code block.
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
	pos := 0
	err := gmast.Walk(gmTree, func(node gmast.Node, entering bool) (gmast.WalkStatus, error) {
		if !entering {
			return gmast.WalkContinue, nil
		}
		switch node.Kind() {
		case gmast.KindHTMLBlock:
			htmlBlockLines := node.Lines()
			if htmlBlockLines.Len() == 0 {
				break
			}
			text := string(htmlBlockLines.Value(sourceMD))
			// MLR directive
			if matches := regexpMillerDirective().FindStringSubmatch(text); len(matches) > 0 {
				mlrScript := matches[millerScriptIndex]
				prevNode := node.PreviousSibling()
				if prevNode.Kind() != gmast.KindParagraph {
					break
				}
				tableLines := prevNode.Lines()
				if tableLines.Len() == 0 {
					break
				}
				tableStart := tableLines.At(0).Start
				tableEnd := tableLines.At(tableLines.Len() - 1).Stop
				prefixStart := getPrefixStart(sourceMD, tableStart)
				markdownTableText := tableLines.Value(sourceMD)
				func() {
					tempDirPath, tempDirCleanFn := mkdirTemp()
					defer tempDirCleanFn()
					tempFilePath := path.Join(tempDirPath, "3202c41.md")
					V0(os.WriteFile(tempFilePath, []byte(markdownTableText), 0600))
					mlrMDInplacePut(tempFilePath, mlrScript)
					result := V(os.ReadFile(tempFilePath))
					_ = V(writer.Write(sourceMD[pos:prefixStart]))
					// No prefix
					if tableStart == prefixStart {
						_ = V(writer.Write(result))
					} else
					// Has a prefix
					{
						prefixText := string(sourceMD[prefixStart:tableStart])
						for _, line := range bytes.Split(result, []byte{'\n'}) {
							if len(strings.TrimSpace(string(line))) > 0 {
								_ = V(writer.Write([]byte(prefixText + string(line) + "\n")))
							}
						}
					}
					pos = htmlBlockLines.At(htmlBlockLines.Len() - 1).Stop
					_ = V(writer.Write(sourceMD[tableEnd+1 : pos]))
				}()
			} else
			//  aaa
			if matches := regexpCodeDirective().FindStringSubmatch(text); len(matches) > 0 {
				codeSrc := matches[codeSrcIndex]
				prevNode := node.PreviousSibling()
				println(codeSrc, prevNode.Kind())
				// Fenced code block is OK
				if prevNode.Kind() == gmast.KindFencedCodeBlock {
					fencedCodeBlock, _ := prevNode.(*gmast.FencedCodeBlock)
					segments := fencedCodeBlock.Lines()
					// Empty fenced code block does not have segments. Search the start and end positions
					cmtStart := htmlBlockLines.At(0).Start
					cmtStop := htmlBlockLines.At(htmlBlockLines.Len() - 1).Stop
					prefix := ""
					var blockStart int
					var blockStop int
					if segments.Len() == 0 {
						blockStop = cmtStart - 1
					outer:
						for ; blockStop >= 0; blockStop-- {
							if bytes.HasPrefix(sourceMD[blockStop:], []byte("```")) {
								for i := blockStop; i >= 0; i-- {
									if i == 0 || sourceMD[i-1] == '\n' {
										prefix = string(sourceMD[i:blockStop])
										blockStop = i
										break outer
									}
								}
							}
						}
						blockStart = blockStop
					} else {
						blockStart = segments.At(0).Start
						for blockStart > 0 && sourceMD[blockStart-1] != '\n' {
							blockStart--
						}
						blockStop = segments.At(segments.Len() - 1).Stop
						for i := blockStop; i < len(sourceMD); i++ {
							if sourceMD[i] == '`' {
								prefix = string(sourceMD[blockStop:i])
								break
							}
						}
					}
					codeContent, err := os.ReadFile(codeSrc)
					if err != nil {
						break
					}
					x := sourceMD[pos:blockStart]
					_ = V(writer.Write(x))
					pos = blockStop
					codeLines := bytes.Split(codeContent, []byte{'\n'})
					if len(codeLines) > 0 && len(codeLines[len(codeLines)-1]) == 0 {
						codeLines = codeLines[:len(codeLines)-1]
					}
					for _, line := range codeLines {
						_ = V(fmt.Fprintf(writer, "%s%s\n", prefix, line))
					}
					y := sourceMD[pos:cmtStop]
					_ = V(writer.Write(y))
					pos = cmtStop

					t := segments.Value(sourceMD)
					println(t)
				} else
				// Code block is OK
				if prevNode.Kind() == gmast.KindCodeBlock {

				} else
				// Otherwise, break
				{
					break
				}
			}
		case gmast.KindRawHTML:
			rawHTMLNode, _ := node.(*gmast.RawHTML)
			segments := rawHTMLNode.Segments
			if segments.Len() == 0 {
				break
			}
			text := string(segments.Value(sourceMD))
			// +TITLE directive gets the link path from the previous link node
			if matches := regexpSyncTitleDirective().FindStringSubmatch(text); len(matches) > 0 {
				prevNode := node.PreviousSibling()
				if prevNode == nil || prevNode.Kind() != gmast.KindLink {
					break
				}
				linkNode, _ := prevNode.(*gmast.Link)
				linkPath := string(linkNode.Destination)
				targetMD, err := os.ReadFile(linkPath)
				if err != nil {
					break
				}
				title := getMDTitle(targetMD, linkPath)
				cmtStart := segments.At(0).Start
				linkStart := cmtStart - 1
				for ; linkStart >= 0; linkStart-- {
					if sourceMD[linkStart] == '[' && (linkStart == 0 || sourceMD[linkStart-1] != '\\') {
						break
					}
				}
				_ = V(writer.Write(sourceMD[pos:linkStart]))
				_ = V(fmt.Fprintf(writer, "[%s](%s)", title, linkPath))
				pos = segments.At(segments.Len() - 1).Stop
				_ = V(writer.Write(sourceMD[cmtStart:pos]))
			}
		}
		return gmast.WalkContinue, nil
	})
	if pos < len(sourceMD) {
		_ = V(writer.Write(sourceMD[pos:]))
	}
	return err
}
