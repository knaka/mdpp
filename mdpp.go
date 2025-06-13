package mdpp

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	fileex "github.com/spiegel-im-spiegel/file"
	be "github.com/thomasheller/braceexpansion"
	"github.com/yuin/goldmark"
	gm "github.com/yuin/goldmark"
	gmmeta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/ast"
	gmast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	gmextast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	gmparser "github.com/yuin/goldmark/parser"
	gmtext "github.com/yuin/goldmark/text"

	. "github.com/knaka/go-utils"
)

// Write index to io.Writer with indent
func writeIndex(writer io.Writer, wildcard string, indent string, includerPath string) error {
	includerPath, err := filepath.EvalSymlinks(includerPath)
	if err != nil {
		return err
	}
	// paths, err := fileex.Glob(wildcard)
	tree, err := be.New().Parse(wildcard)
	if err != nil {
		return err
	}
	var paths []string
	for _, pattern := range tree.Expand() {
		pathsNew, err := fileex.Glob(pattern)
		if err != nil {
			return err
		}
		paths = append(paths, pathsNew...)
	}
	if err != nil {
		return err
	}
	for i, path := range paths {
		if !strings.ContainsRune(path, '/') {
			paths[i] = "./" + path
		}
	}
	sort.Strings(paths)
	dirnamePrev := ""
	dirIndent := ""
	for _, path := range paths {
		if filepath.Separator != '/' {
			path = filepath.ToSlash(path)
		}
		title := GetMarkdownTitle(path)
		dirname := filepath.Dir(path)
		if filepath.Separator != '/' {
			dirname = filepath.ToSlash(dirname)
		}
		if dirname != "." && dirname != dirnamePrev {
			if _, err := fmt.Fprintln(writer, indent+"* "+dirname); err != nil {
				return err
			}
			dirIndent = "  "
		}
		if path == title {
			title = filepath.Base(path)
		}
		dirnamePrev = dirname
		s := title
		if a, err := filepath.Abs(path); err != nil {
			return err
		} else if a != includerPath {
			s = "[" + title + "](" + path + ")"
		}
		if _, err := fmt.Fprintln(writer, indent+dirIndent+"* "+s); err != nil {
			return err
		}
	}
	return nil
}

func writeFileWithIndent(writer io.Writer, pathForCodeBlock string, indent string) (errReturn error) {
	blockInput, err := os.Open(pathForCodeBlock)
	if err != nil {
		return err
	}
	defer func() {
		if err := blockInput.Close(); err != nil {
			errReturn = err
		}
	}()
	scannerBlockInput := bufio.NewScanner(blockInput)
	for scannerBlockInput.Scan() {
		s := scannerBlockInput.Text()
		if _, err := fmt.Fprintln(writer, indent+s); err != nil {
			return err
		}
	}
	return nil
}

func writeStrBeforeSegmentsStart(writer io.Writer, source []byte,
	position int, segments *gmtext.Segments, fix int) (int, error) {
	firstSegment := segments.At(0)
	buf := source[position : firstSegment.Start+fix]
	if _, err := writer.Write(buf); err != nil {
		return position, err
	}
	lastSegment := segments.At(segments.Len() - 1)
	return lastSegment.Stop, nil
}

func writeStrBeforeSegmentsStop(
	writer io.Writer,
	source []byte,
	position int,
	segments *gmtext.Segments,
) (int, error) {
	lastSegment := segments.At(segments.Len() - 1)
	buf := source[position:lastSegment.Stop]
	if _, err := writer.Write(buf); err != nil {
		return position, err
	}
	return lastSegment.Stop, nil
}

const strReBegin = `<!-- *(mdpp[_a-zA-Z0-9]*)( ([_a-zA-Z][_a-zA-Z0-9]*)=([^ ]*))? *-->`
const strReEnd = `<!-- /(mdpp[_a-zA-Z0-9]*) -->`

// PreprocessWithoutDir processes the input reader and writes the result to writerOut
func PreprocessWithoutDir(writer io.Writer, reader io.Reader) error {
	_, _, err := PreprocessOld(writer, reader, "", "")
	return err
}

// PreprocessOld processes the input reader and writes the result to writerOut.
func PreprocessOld(writerOut io.Writer, reader io.Reader,
	workDir string, inPath string) (foundMdppDirective bool, changed bool, errReturn error) {
	foundMdppDirective = false
	changed = false
	dirSaved, err := os.Getwd()
	if err != nil {
		return foundMdppDirective, changed, err
	}
	defer func() {
		if err := os.Chdir(dirSaved); err != nil {
			errReturn = err
		}
	}()
	if workDir != "" {
		if err := os.Chdir(workDir); err != nil {
			return foundMdppDirective, changed, err
		}
	}
	var absPath string
	absPath, err = filepath.Abs(filepath.Join(workDir, inPath))
	readBuffer := bytes.NewBuffer(nil)
	if _, err := io.Copy(readBuffer, reader); err != nil {
		return foundMdppDirective, changed, err
	}
	source := readBuffer.Bytes()
	writer := bytes.NewBuffer(nil)
	// Position on source
	position := 0
	// Current location on AST
	var location []*ast.Node
	// Current stack of MDPP commands
	var mdppStack []mdppElemMethods
	// RE objects are allocated locally to avoid lock among threads
	var reBegin *regexp.Regexp = nil
	var reEnd *regexp.Regexp = nil
	walker := func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			location = location[:len(location)-1]
			return ast.WalkContinue, nil
		}
		location = append(location, &node)
		nodeKind := node.Kind()
		switch nodeKind {
		case ast.KindRawHTML:
			rawHTML, ok := node.(*ast.RawHTML)
			if !ok {
				return ast.WalkStop, errors.New("failed to downcast")
			}
			segments := rawHTML.Segments
			segment := segments.At(0)
			text := string(source[segment.Start:segment.Stop])
			if strings.HasPrefix(text, "<!-- mdpp") {
				foundMdppDirective = true
				if reBegin == nil {
					reBegin = regexp.MustCompile(strReBegin)
				}
				match := reBegin.FindStringSubmatch(text)
				if match == nil {
					return ast.WalkStop, NewError("could not match regexp", absPath, source, segment.Start)
				}
				command := match[1]
				baseElem := mdppElem{len(location)}
				if command == "mdpplink" {
					key := match[3]
					if key == "href" {
						value := match[4]
						mdppStack = append(mdppStack, &mdppLinkElem{baseElem, value})
					}
				}
			} else if strings.HasPrefix(text, "<!-- /mdpp") {
				if reEnd == nil {
					reEnd = regexp.MustCompile(strReEnd)
				}
				match := reEnd.FindStringSubmatch(text)
				command := match[1]
				if len(mdppStack) == 0 {
					return ast.WalkStop, NewError("unexpected inline closing command", absPath, source, segment.Start)
				}
				if mdppStack[len(mdppStack)-1].Name() != command {
					return ast.WalkStop, NewError("unbalanced closing command", absPath, source, segment.Start)
				}
				if command == "mdpplink" {
					elem, ok := mdppStack[len(mdppStack)-1].(*mdppLinkElem)
					if !ok {
						return ast.WalkStop, NewError("failed to downcast mdpplink", absPath, source, segment.Start)
					}
					mdppStack = mdppStack[:len(mdppStack)-1]
					title := GetMarkdownTitle(elem.href)
					modified := "[" + title + "](" + elem.href + ")"
					if _, err := fmt.Fprint(writer, modified); err != nil {
						return ast.WalkStop, err
					}
				}
				position = segment.Start
			}
			position, err = writeStrBeforeSegmentsStop(writer, source, position, segments)
			if err != nil {
				return ast.WalkStop, err
			}
		case ast.KindHTMLBlock:
			htmlBlock, ok := node.(*ast.HTMLBlock)
			if !ok {
				return ast.WalkStop, errors.New("failed to downcast htmlblock")
			}
			if htmlBlock.HTMLBlockType != ast.HTMLBlockType2 {
				break
			}
			segments := node.Lines()
			firstLine := segments.At(0)
			txt := string(source[firstLine.Start:firstLine.Stop])
			if strings.HasPrefix(txt, "<!-- mdpp") {
				if reBegin == nil {
					reBegin = regexp.MustCompile(strReBegin)
				}
				match := reBegin.FindStringSubmatch(txt)
				command := match[1]
				mdppElem := mdppElem{len(location)}
				switch command {
				case "mdppcode":
					key := match[3]
					if key == "src" {
						mdppStack = append(mdppStack, &mdppCodeElem{mdppElem, match[4]})
					} else {
						return ast.WalkStop, NewError("attribute \"src\" required", absPath, source, firstLine.Start)
					}
				case "mdppindex":
					key := match[3]
					if key == "pattern" {
						mdppStack = append(mdppStack, &mdppIndexElem{mdppElem, match[4]})
					} else {
						return ast.WalkStop, NewError("attribute \"pattern\" required", absPath, source, firstLine.Start)
					}
				default:
					return ast.WalkStop, NewError("unknown MDPP command", absPath, source, firstLine.Start)
				}
			} else if strings.HasPrefix(txt, "<!-- /mdpp") {
				if reEnd == nil {
					reEnd = regexp.MustCompile(strReEnd)
				}
				match := reEnd.FindStringSubmatch(txt)
				command := match[1]
				if len(mdppStack) == 0 && command != "mdppcode" {
					return ast.WalkStop, NewError("unexpected block closing command", absPath, source, firstLine.Start)
				}
				switch command {
				case "mdppcode":
					if len(mdppStack) > 0 && mdppStack[len(mdppStack)-1].Name() == command && mdppStack[len(mdppStack)-1].Depth() == len(location) {
						mdppStack = mdppStack[:len(mdppStack)-1]
					}
				case "mdppindex":
					firstSegment := segments.At(0)
					indent := getIndentBeforeSegment(firstSegment, source)
					elem, ok := mdppStack[len(mdppStack)-1].(*mdppIndexElem)
					if !ok {
						return ast.WalkStop, NewError("downcast failed", absPath, source, firstLine.Start)

					}
					mdppStack = mdppStack[:len(mdppStack)-1]
					if err := writeIndex(writer, elem.pattern, indent, inPath); err != nil {
						return ast.WalkStop, err
					}
					if elem.Name() != command || elem.Depth() != len(location) {
						return ast.WalkStop, NewError("commands do not match", absPath, source, firstLine.Start)
					}
					position = firstSegment.Start - len(indent)
				default:
					return ast.WalkStop, NewError("unknown closing command", absPath, source, firstLine.Start)
				}
			}
			position, err = writeStrBeforeSegmentsStop(writer, source, position, segments)
			if err != nil {
				return ast.WalkStop, err
			}
		case ast.KindParagraph:
			break
		case gmextast.KindTable:
			table, ok := node.(*gmextast.Table)
			if !ok {
				return ast.WalkStop, errors.New("failed to downcast table")
			}
			lines := table.FirstChild().Lines()
			// lines := table.Lines()
			println("lines", lines.Len())
		case gmextast.KindTableHeader:
			header, ok := node.(*gmextast.TableHeader)
			if !ok {
				return ast.WalkStop, errors.New("failed to downcast table header")
			}
			lines := header.Lines()
			println("header lines", lines.Len())
		case ast.KindCodeBlock:
			fallthrough
		case ast.KindFencedCodeBlock:
			if len(mdppStack) == 0 || mdppStack[len(mdppStack)-1].Name() != "mdppcode" {
				break
			}
			segments := node.Lines()
			if segments.Len() == 0 {
				return ast.WalkStop, NewError("empty fenced code block", absPath, source, position)
			}
			firstSegment := segments.At(0)
			indent := getIndentBeforeSegment(firstSegment, source)
			position, err = writeStrBeforeSegmentsStart(writer, source, position, segments, -len(indent))
			if err != nil {
				return ast.WalkStop, err
			}
			mdppCodeElem1, ok := mdppStack[len(mdppStack)-1].(*mdppCodeElem)
			if !ok {
				return ast.WalkStop, NewError("downcast failed", absPath, source, firstSegment.Start)
			}
			mdppStack = mdppStack[:len(mdppStack)-1]
			err := writeFileWithIndent(writer, mdppCodeElem1.filepath, indent)
			if err != nil {
				return ast.WalkStop, err
			}
		}
		return ast.WalkContinue, nil
	}
	markdown := goldmark.New(
		goldmark.WithExtensions(
			gmmeta.Meta, // Enable meta extension to parse metadata of the Markdown document
			extension.Table,
		),
	)
	context := parser.NewContext()
	parserOption := parser.WithContext(context)
	doc := markdown.Parser().Parse(gmtext.NewReader(source), parserOption)
	// doc.Dump(source, 0)
	if err := ast.Walk(doc, walker); err != nil {
		return foundMdppDirective, changed, err
	}
	if len(mdppStack) != 0 {
		return foundMdppDirective, changed, errors.New("stack not empty")
	}
	_, err = writer.Write(source[position:])
	if err != nil {
		return foundMdppDirective, changed, err
	}
	dest := writer.Bytes()
	if bytes.Compare(source, dest) != 0 {
		changed = true
	}
	if _, err := io.Copy(writerOut, writer); err != nil {
		return foundMdppDirective, changed, err
	}
	return foundMdppDirective, changed, nil
}

func getIndentBeforeSegment(segment gmtext.Segment, source []byte) string {
	indent := ""
	for i := segment.Start - 1; i >= 0; i-- {
		r := rune(source[i])
		if r == ' ' || r == '\t' {
			indent = string(r) + indent
		} else {
			break
		}
	}
	return indent
}

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

var regexpLinkDirective = sync.OnceValue(func() *regexp.Regexp {
	// Matches the LINK directive in HTML comments, e.g.:
	//
	//   <!-- +LINK: ./foo.md -->
	return regexp.MustCompile(`^<!--\s*\+LINK:\s*([^-]+?)\s*(-->\s*)?$`)
})

var regexpTitleDirective = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?i)^<!--\s*\+(TITLE|EXTRACT_TITLE|REPLACE_TITLE)\s*(-->\s*)?$`)
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

// Process parses the source markdown, detects directives in HTML comments, applies modifications, and writes the result to the writer.
//
// Supported directives:
//   - MLR | MILLER : Processes the table above the comment using a Miller script.
//   - LINK : Replaces the link with the title from the target Markdown file.
//
// Planned features:
//   - TITLE | EXTRACT_TITLE : Extract the title from the linked Markdown file and use it as the link title.
//   - MLR_SRC | MILLER_SRC : Reads the Miller script from the specified file and applies it to the table above the comment.
//   - CODE : Reads the content of the file specified and writes it as a code block.
//   - TBLFM (?)
func Process(sourceMD []byte, writer io.Writer, dirPath string) error {
	if len(dirPath) > 0 {
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		defer func() {
			os.Chdir(currentDir)
		}()
		if err := os.Chdir(dirPath); err != nil {
			return fmt.Errorf("failed to change directory to %s: %w", dirPath, err)
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
					if tableStart == prefixStart { // No prefix
						_ = V(writer.Write(result))
					} else { // Has a prefix
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
			}
		case gmast.KindRawHTML:
			rawHTML, _ := node.(*gmast.RawHTML)
			segments := rawHTML.Segments
			if segments.Len() == 0 {
				break
			}
			text := string(segments.Value(sourceMD))
			if matches := regexpTitleDirective().FindStringSubmatch(text); len(matches) > 0 {
				prevNode := node.PreviousSibling()
				if prevNode.Kind() != gmast.KindLink {
					break
				}
				nodeLink, _ := prevNode.(*gmast.Link)
				linkPath := string(nodeLink.Destination)
				targetMDContent, err := os.ReadFile(linkPath)
				if err != nil {
					break
				}
				title := getMDTitle(targetMDContent, linkPath)
				cmtStart := segments.At(0).Start
				linkStart := cmtStart - 1
				for ; linkStart > 0; linkStart-- {
					if sourceMD[linkStart] == '[' && (linkStart == 0 || sourceMD[linkStart-1] != '\\') {
						break
					}
				}
				_ = V(writer.Write(sourceMD[pos:linkStart]))
				pos = segments.At(segments.Len() - 1).Stop
				_ = V(fmt.Fprintf(writer, "[%s](%s)", title, linkPath))
				_ = V(writer.Write(sourceMD[cmtStart:pos]))
			} else if matches := regexpLinkDirective().FindStringSubmatch(text); len(matches) > 0 {
				linkPath := matches[linkIndex]
				prevNode := node.PreviousSibling()
				if prevNode.Kind() != gmast.KindLink {
					break
				}
				targetMDContent, err := os.ReadFile(linkPath)
				if err != nil {
					break
				}
				title := getMDTitle(targetMDContent, linkPath)
				cmtStart := segments.At(0).Start
				linkStart := cmtStart - 1
				for ; linkStart > 0; linkStart-- {
					if sourceMD[linkStart] == '[' && (linkStart == 0 || sourceMD[linkStart-1] != '\\') {
						break
					}
				}
				_ = V(writer.Write(sourceMD[pos:linkStart]))
				pos = segments.At(segments.Len() - 1).Stop
				_ = V(fmt.Fprintf(writer, "[%s](%s)", title, linkPath))
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
