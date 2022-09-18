package mdpp

import (
	"bufio"
	"io"
	"os"
	"strings"
	"unicode"
)

type MarkdownStyle int8

const (
	UnknownStyle MarkdownStyle = iota
	MultiMarkdownStyle
	PandocTitleBlockStyle
	YamlMetadataBlockStyle
)

func GetMarkdownTitle(path string) string {
	input, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() {
		_ = input.Close()
	}()
	return GetMarkdownTitleSub(input, path)
}

func GetMarkdownTitleSub(input io.Reader, defaultTitle string) string {
	title := defaultTitle
	style := UnknownStyle
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if style == UnknownStyle {
			if line == "---" {
				style = YamlMetadataBlockStyle
				continue
			} else if line[0] == '%' {
				style = PandocTitleBlockStyle
			} else if strings.ContainsRune(line, ':') {
				style = MultiMarkdownStyle
			} else {
				continue
			}
		}
		if style == YamlMetadataBlockStyle {
			if line == "---" {
				break
			}
			fields := strings.SplitN(line, ":", 2)
			if len(fields) >= 2 {
				key := strings.TrimSpace(fields[0])
				if strings.ToLower(key) == "title" {
					title = strings.TrimFunc(fields[1], func(r rune) bool {
						return r == '"' || r == '\'' || unicode.IsSpace(r)
					})
					break
				}
			}
		} else if style == PandocTitleBlockStyle {
			title = strings.TrimLeftFunc(line, func(r rune) bool {
				return r == '%' || unicode.IsSpace(r)
			})
			for scanner.Scan() {
				line = scanner.Text()
				if strings.HasPrefix(line, " ") {
					title += line
				} else {
					break
				}
			}
			break
		} else if style == MultiMarkdownStyle {
			if !strings.ContainsRune(line, ':') {
				break
			}
			fields := strings.Split(line, ":")
			if len(fields) >= 2 {
				key := strings.TrimSpace(fields[0])
				if strings.ToLower(key) == "title" {
					title = strings.TrimFunc(fields[1], func(r rune) bool {
						return r == '"' || r == '\'' || unicode.IsSpace(r)
					})
					break
				}
			} else {
				break
			}
		} else {
			return ""
		}
	}
	return title
}
