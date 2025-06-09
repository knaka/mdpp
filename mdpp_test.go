package mdpp

import (
	"bytes"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/andreyvit/diff"
	"github.com/stretchr/testify/assert"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"

	. "github.com/knaka/go-utils"
)

func TestCodeBlock(t *testing.T) {
	input := bytes.NewBufferString(`Code block:

<!-- mdppcode src=misc/hello.c -->


			hello
	
			world

* foo

  <!-- mdppcode src=misc/hello.c -->

      foo

      bar

Done.
`)
	expected := []byte(`Code block:

<!-- mdppcode src=misc/hello.c -->


			#include <stdio.h>
			
			int main (int argc, char** argv) {
				printf("Hello!\n");
			}

* foo

  <!-- mdppcode src=misc/hello.c -->

      #include <stdio.h>
      
      int main (int argc, char** argv) {
      	printf("Hello!\n");
      }

Done.
`)
	output := bytes.NewBuffer(nil)
	if err := PreprocessWithoutDir(output, input); err != nil {
		t.Fatal("error")
	}
	if bytes.Compare(expected, output.Bytes()) != 0 {
		t.Fatalf(`Unmatched:

%s`, diff.LineDiff(string(expected), output.String()))
	}
}

func TestFencedCodeBlock(t *testing.T) {
	input := bytes.NewBufferString(`Code block:

<!-- mdppcode src=misc/hello.c -->

~~~
hello

world
~~~

* foo

  <!-- mdppcode src=misc/hello.c -->

  ~~~
  hello
  
  world
  ~~~

Done.
`)
	expected := []byte(`Code block:

<!-- mdppcode src=misc/hello.c -->

~~~
#include <stdio.h>

int main (int argc, char** argv) {
	printf("Hello!\n");
}
~~~

* foo

  <!-- mdppcode src=misc/hello.c -->

  ~~~
  #include <stdio.h>
  
  int main (int argc, char** argv) {
  	printf("Hello!\n");
  }
  ~~~

Done.
`)
	output := bytes.NewBuffer(nil)
	if err := PreprocessWithoutDir(output, input); err != nil {
		t.Fatal("error")
	}
	if bytes.Compare(expected, output.Bytes()) != 0 {
		t.Fatalf(`Unmatched:

%s`, diff.LineDiff(string(expected), output.String()))
	}
}

func TestFencedCodeBlockNotClosing(t *testing.T) {
	input := bytes.NewBufferString(`Code block:

<!-- mdppcode src=misc/hello.c -->

Done
`)
	output := bytes.NewBuffer(nil)
	if err := PreprocessWithoutDir(output, input); err == nil || err.Error() != "stack not empty" {
		t.Fatal("error")
	}
}

func TestToc(t *testing.T) {
	input := bytes.NewBufferString(`TOC:

<!-- mdppindex pattern=misc/*.md -->
<!-- /mdppindex -->

* foo

  <!-- mdppindex pattern=misc/*.md -->
  foo  
  <!-- /mdppindex -->

`)
	expected := []byte(`TOC:

<!-- mdppindex pattern=misc/*.md -->
* misc
  * [Bar ドキュメント](misc/bar.md)
  * [foo.md](misc/foo.md)
<!-- /mdppindex -->

* foo

  <!-- mdppindex pattern=misc/*.md -->
  * misc
    * [Bar ドキュメント](misc/bar.md)
    * [foo.md](misc/foo.md)
  <!-- /mdppindex -->

`)
	output := bytes.NewBuffer(nil)
	if err := PreprocessWithoutDir(output, input); err != nil {
		t.Fatal(err.Error())
	}
	if bytes.Compare(expected, output.Bytes()) != 0 {
		t.Fatalf(`Unmatched:

%s`, diff.LineDiff(string(expected), output.String()))
	}
}

func TestTocDifferentDepth(t *testing.T) {
	input := bytes.NewBufferString(`TOC:

<!-- mdppindex pattern=misc/*.md -->
* foo
* bar

other document

* foo

  <!-- /mdppindex -->
`)
	output := bytes.NewBuffer(nil)
	if err := PreprocessWithoutDir(output, input); err == nil {
		t.Fatal("Do not succeed")
	} else {
		if !strings.HasPrefix(err.Error(), "commands do not match") {
			t.Fatal("not expected error")
		}
	}
}

func TestLinks(t *testing.T) {
	input := bytes.NewBufferString(`Links:

Inline-links <!-- mdpplink href=misc/foo.md -->...<!-- /mdpplink -->
and <!-- mdpplink href=misc/bar.md -->...<!-- /mdpplink --> works.
`)
	expected := []byte(`Links:

Inline-links <!-- mdpplink href=misc/foo.md -->[misc/foo.md](misc/foo.md)<!-- /mdpplink -->
and <!-- mdpplink href=misc/bar.md -->[Bar ドキュメント](misc/bar.md)<!-- /mdpplink --> works.
`)
	output := bytes.NewBuffer(nil)
	if err := PreprocessWithoutDir(output, input); err != nil {
		t.Fatal(err.Error())
	}
	if bytes.Compare(expected, output.Bytes()) != 0 {
		t.Fatalf(`Unmatched:

%s`, diff.LineDiff(string(expected), output.String()))
	}
}

func _TestIncludes(t *testing.T) {
	input := bytes.NewBufferString(`Includes:

<!-- mdppinclude src=misc/foo.md -->
<!-- /mdppinclude -->
`)
	expected := []byte(`Includes:

<!-- mdppinclude src=misc/foo.md -->
<!-- /mdppinclude -->
`)
	output := bytes.NewBuffer(nil)
	if err := PreprocessWithoutDir(output, input); err != nil {
		t.Fatal(err.Error())
	}
	if bytes.Compare(expected, output.Bytes()) != 0 {
		t.Fatalf(`Unmatched:

%s`, diff.LineDiff(string(expected), output.String()))
	}
}

func TestTitle(t *testing.T) {
	input1 := bytes.NewBufferString(`---
title: My Title
---
`)
	title := GetMarkdownTitleSub(input1, "default")
	if title != "My Title" {
		t.Fatal("Could not find title")
	}
}

func TestTitle2(t *testing.T) {
	input1 := bytes.NewBufferString(`---

---
title: Foo Bar
`)
	title := GetMarkdownTitleSub(input1, "default")
	if title != "default" {
		t.Fatal("How did you get it?")
	}
}

func TestTitle3(t *testing.T) {
	input1 := bytes.NewBufferString(`% My Document

Document.
`)
	title := GetMarkdownTitleSub(input1, "default")
	if title != "My Document" {
		t.Fatal("Could not get title")
	}
}

func TestTitle4(t *testing.T) {
	input1 := bytes.NewBufferString(`% My document title 
 is long
Document.
`)
	title := GetMarkdownTitleSub(input1, "default")
	if title != "My document title is long" {
		t.Fatal("Could not get title")
	}
}

func TestTitle5(t *testing.T) {
	input1 := bytes.NewBufferString(`Title:   My title
Author:  Foo Bar

Main document.
`)
	title := GetMarkdownTitleSub(input1, "default")
	if title != "My title" {
		t.Fatal("Could not get title")
	}
}

// Unknown commands are ignored
func TestUnknown(t *testing.T) {
	input := bytes.NewBufferString(`Includes:

<!-- mdppunknown src=misc/foo.md -->
<!-- /mdppunknown -->

`)
	output := bytes.NewBuffer(nil)
	if err := PreprocessWithoutDir(output, input); err == nil {
		t.Fatal(err.Error())
	}
}

func TestTocFail(t *testing.T) {
	input := bytes.NewBufferString(`TOC:

<!-- mdppindex pattern=misc/*.md -->
<!-- /mdppindex -->
<!-- /mdppindex -->

`)
	output := bytes.NewBuffer(nil)
	if err := PreprocessWithoutDir(output, input); err == nil {
		t.Fatal("Error")
	}
}

func TestCodeBlockWithBlankLines(t *testing.T) {
	// fails to figure out correct indent
	// library does not have meta-info of the block
	t.Skip()
	input := bytes.NewBufferString(`Code block:

* foo

  <!-- mdppcode src=misc/code_with_blank_lines.c -->

  ~~~
    
  
  #include <stdio.h>
  
  int main (int argc, char** argv) {
  printf("Hello!\n");
  }
  ~~~

<!-- /mdppcode -->
`)
	expected := []byte(`Code block:

* foo

  <!-- mdppcode src=misc/code_with_blank_lines.c -->

  ~~~
  
  
  #include <stdio.h>
  
  int main (int argc, char** argv) {
  	printf("Hello!\n");
  }
  ~~~

<!-- /mdppcode -->
`)
	output := bytes.NewBuffer(nil)
	if err := PreprocessWithoutDir(output, input); err != nil {
		t.Fatal("error")
	}
	if bytes.Compare(expected, output.Bytes()) != 0 {
		t.Fatalf(`Unmatched:

%s`, diff.LineDiff(string(expected), output.String()))
	}
}

func TestIndexRec(t *testing.T) {
	input := bytes.NewBufferString(`<!-- mdppindex pattern=misc/**/*.txt -->
<!-- /mdppindex -->
`)
	expected := []byte(`<!-- mdppindex pattern=misc/**/*.txt -->
* misc/dir1/dir1-1
  * [foo.txt](misc/dir1/dir1-1/foo.txt)
* misc/dir2
  * [bar.txt](misc/dir2/bar.txt)
<!-- /mdppindex -->
`)
	output := bytes.NewBuffer(nil)
	if err := PreprocessWithoutDir(output, input); err != nil {
		t.Fatal(err.Error())
	}
	if bytes.Compare(expected, output.Bytes()) != 0 {
		t.Fatalf(`Unmatched:

%s`, diff.LineDiff(string(expected), output.String()))
	}
}

func TestTable(t *testing.T) {
	input := bytes.NewBufferString(`Table:

| Item | UnitPrice | Count | Total |
| --- | --- | --- | --- |
| Apple | 10 | 1 | 0 |
| Orange | 20 | 2 | 0 |
| Lemon | 30 | 3 | 0 |

<!-- +MLR:
  $Total = $UnitPrice * $Count
-->
`)
	expected := []byte(`Table:

| Item | UnitPrice | Count | Total |
| --- | --- | --- | --- |
| Apple | 10 | 1 | 10 |
| Orange | 20 | 2 | 40 |
| Lemon | 30 | 3 | 90 |

<!-- +MLR:
  $Total = $UnitPrice * $Count
-->
`)
	output := bytes.NewBuffer(nil)
	if err := PreprocessWithoutDir(output, input); err != nil {
		t.Fatal("error")
	}
	if bytes.Compare(expected, output.Bytes()) != 0 {
		t.Fatalf(`Unmatched:

%s`, diff.LineDiff(string(expected), output.String()))
	}
}

func parseMarkdown(sourceMS []byte) ast.Node {
	reader := text.NewReader(sourceMS)
	md := goldmark.New()
	doc := md.Parser().Parse(reader)
	if doc == nil {
		panic("Failed to parse markdown")
	}
	return doc
}

var regexpMLRComment = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`<!--\s*\+MLR:\s*([^-]+?)\s*-->`)
})

func TestTableMlr(t *testing.T) {
	sourceMD := []byte(`foo
	
> | Item | UnitPrice | Quantity | Total |
> | --- | --- | --- | --- |
> | Apple | 1.5 | 12 | 0 |
> | Banana | 2.0 | 5 | 0 |
> | Orange | 1.2 | 8 | 0 |
>
> <!-- +MLR: $Total = $UnitPrice * $Quantity -->

bar
`)
	doc := parseMarkdown(sourceMD)
	doc.Dump(sourceMD, 0)
	assert.NotNil(t, doc, "Document should not be nil")
	V0(ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node.Kind() {
		case ast.KindHTMLBlock:
			lines := node.Lines()
			if lines.Len() == 0 {
				break
			}
			segment := lines.At(0)
			text := string(sourceMD[segment.Start:segment.Stop])
			if matches := regexpMLRComment().FindStringSubmatch(text); len(matches) >= 2 {
				mlrScript := matches[1]
				// t.Log("MLR script:", mlrScript)
				prevNode := node.PreviousSibling()
				if prevNode.Kind() != ast.KindParagraph {
					break
				}
				lines := prevNode.Lines()

				j := lines.At(0).Start
				prefix := ""
				for i := j; true; i-- {
					if i == -1 || sourceMD[i] == '\n' || sourceMD[i] == '\r' {
						prefix = string(sourceMD[i+1:j]) + prefix
						break
					}
				}
				t.Log("Prefix:", prefix)
				markdownTable := lines.Value(sourceMD)
				// t.Log("Markdown table:", string(markdownTable))
				func() {
					tempDirPath, tempDirCleanFn := mkdirTemp()
					defer tempDirCleanFn()
					tempFilePath := path.Join(tempDirPath, "data.md")
					V0(os.WriteFile(tempFilePath, []byte(markdownTable), 0600))
					mlrMDInplacePut(tempFilePath, mlrScript)
					result := V(os.ReadFile(tempFilePath))
					t.Log("Result:", string(result))
					// Print each line of the result with prefix
					for _, line := range bytes.Split(result, []byte{'\n'}) {
						if len(line) > 0 {
							t.Log(prefix + string(line))
						}
					}
				}()
			}
		}
		return ast.WalkContinue, nil
	}))
}
