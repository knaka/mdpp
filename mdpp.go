package mdpp

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	mtext "github.com/yuin/goldmark/text"
)

// Write TOC to io.Writer with indent
func writeToc(writer io.Writer, wildcard string, indent string, includerPath string) error {
	paths, err := filepath.Glob(wildcard)
	if err != nil {
		return err
	}
	sort.Strings(paths)
	for _, path := range paths {
		if filepath.Separator != '/' {
			path = filepath.ToSlash(path)
		}
		title := GetMarkdownTitle(path)
		s := title
		if a, err := filepath.Abs(path); err != nil {
			return err
		} else if a != includerPath {
			s = "[" + title + "](" + path + ")"
		}
		if _, err := fmt.Fprintln(writer, indent+"* "+s); err != nil {
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
	position int, segments *mtext.Segments, fix int) (int, error) {
	firstSegment := segments.At(0)
	buf := source[position : firstSegment.Start+fix]
	if _, err := writer.Write(buf); err != nil {
		return position, err
	}
	lastSegment := segments.At(segments.Len() - 1)
	return lastSegment.Stop, nil
}

func writeStrBeforeSegmentsStop(writer io.Writer, source []byte,
	position int, segments *mtext.Segments) (int, error) {
	lastSegment := segments.At(segments.Len() - 1)
	buf := source[position:lastSegment.Stop]
	if _, err := writer.Write(buf); err != nil {
		return position, err
	}
	return lastSegment.Stop, nil
}

const strReBegin = `<!-- *(mdpp[_a-zA-Z0-9]*)( ([_a-zA-Z][_a-zA-Z0-9]*)=([^ ]*))? *-->`
const strReEnd = `<!-- /(mdpp[_a-zA-Z0-9]*) -->`

func PreprocessWithoutDir(writer io.Writer, reader io.Reader) error {
	_, _, err := Preprocess(writer, reader, "", "")
	return err
}

func Preprocess(writerOut io.Writer, reader io.Reader,
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
		switch node.Kind() {
		case ast.KindRawHTML:
			rawHtml, ok := node.(*ast.RawHTML)
			if !ok {
				return ast.WalkStop, errors.New("failed to downcast")
			}
			segments := rawHtml.Segments
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
					if err := writeToc(writer, elem.pattern, indent, inPath); err != nil {
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
			meta.Meta,
		),
	)
	context := parser.NewContext()
	parserOption := parser.WithContext(context)
	doc := markdown.Parser().Parse(mtext.NewReader(source), parserOption)
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

func getIndentBeforeSegment(segment mtext.Segment, source []byte) string {
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
