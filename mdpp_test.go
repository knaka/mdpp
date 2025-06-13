package mdpp

import (
	"bytes"
	"strings"
	"testing"

	"github.com/andreyvit/diff"
	"github.com/stretchr/testify/assert"

	. "github.com/knaka/go-utils"
)

func TestCodeBlockOld(t *testing.T) {
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

func TestTitleOld(t *testing.T) {
	input1 := bytes.NewBufferString(`---
title: My Title
---
`)
	title := GetMarkdownTitleSub(input1, "default")
	if title != "My Title" {
		t.Fatal("Could not find title")
	}
}

func TestNotTitle(t *testing.T) {
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

// Tests above are for the old version of the library.

func TestMillerTable(t *testing.T) {
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

func TestPrefixedMillerTable(t *testing.T) {
	tests := []struct {
		run        bool
		name       string
		sourceMD   []byte
		expectedMD []byte
	}{
		{
			run:  true,
			name: "Basic case",
			sourceMD: []byte(`foo

| Item | UnitPrice | Quantity | Total |
| --- | --- | --- | --- |
| Apple | 2.5 | 12 | 0 |
| Banana | 2.0 | 5 | 0 |
| Orange | 1.2 | 8 | 0 |
<!-- +mlr:
  $Total = $UnitPrice * $Quantity
-->

bar
`),
			expectedMD: []byte(`foo

| Item | UnitPrice | Quantity | Total |
| --- | --- | --- | --- |
| Apple | 2.5 | 12 | 30 |
| Banana | 2.0 | 5 | 10 |
| Orange | 1.2 | 8 | 9.6 |
<!-- +mlr:
  $Total = $UnitPrice * $Quantity
-->

bar
`),
		},
		{
			run:  true,
			name: "Table in blockquote",
			sourceMD: []byte(`foo

> | Item | UnitPrice | Quantity | Total |
> | --- | --- | --- | --- |
> | Apple | 1.5 | 12 | 0 |
> | Banana | 2.0 | 5 | 0 |
> | Orange | 1.2 | 8 | 0 |
>
> <!-- +Miller: $Total = $UnitPrice * $Quantity -->

bar
`),
			expectedMD: []byte(`foo

> | Item | UnitPrice | Quantity | Total |
> | --- | --- | --- | --- |
> | Apple | 1.5 | 12 | 18 |
> | Banana | 2.0 | 5 | 10 |
> | Orange | 1.2 | 8 | 9.6 |
>
> <!-- +Miller: $Total = $UnitPrice * $Quantity -->

bar
`),
		},
	}

	for _, tt := range tests {
		if !tt.run {
			t.Logf("Skipping test %s", tt.name)
			continue
		}
		t.Run(tt.name, func(t *testing.T) {
			writer := bytes.NewBuffer(nil)
			V0(Process(tt.sourceMD, writer, ""))
			if bytes.Compare(tt.expectedMD, writer.Bytes()) != 0 {
				t.Fatalf(`Unmatched for %s:

%s`, tt.name, diff.LineDiff(string(tt.expectedMD), writer.String()))
			}
		})
	}
}

func TestTitleExtraction(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name: "Lowercase Title",
			input: []byte(`---
title: My Title
---

foo

bar
`),
			expected: "My Title",
		},
		{
			name: "Uppercase Title",
			input: []byte(`---
Title: My Title
---

foo

bar
`),
			expected: "My Title",
		},
		{
			name: "Header Title",
			input: []byte(`# The Title

hoge fuga
`),
			expected: "The Title",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title := getMDTitle(tt.input, "default")
			assert.Equal(t, tt.expected, title, "Could not find title")
		})
	}
}

func TestSyncTitle(t *testing.T) {
	input := bytes.NewBufferString(`Links:

Inline-links [](misc/foo.md)<!-- +TITLE -->
and [](./misc/bar.md)<!-- +SYNC_TITLE --> works.
`)
	expected := []byte(`Links:

Inline-links [foo](misc/foo.md)<!-- +TITLE -->
and [Bar ドキュメント](./misc/bar.md)<!-- +SYNC_TITLE --> works.
`)
	writer := bytes.NewBuffer(nil)
	V0(Process(input.Bytes(), writer, "."))
	if bytes.Compare(expected, writer.Bytes()) != 0 {
		t.Fatalf(`Unmatched:
%s`, diff.LineDiff(string(expected), writer.String()))
	}
}

// func TestCodeBlock(t *testing.T) {
// 	input := bytes.NewBufferString(`Code block:

// 			hello

// 			world

// <!-- +CODE misc/hello.c -->

// * foo

//       foo

//       bar

// 	<!-- +CODE misc/hello.c -->

// Done.
// `)
// 	expected := []byte(`Code block:

// <!-- mdppcode src=misc/hello.c -->

// 			#include <stdio.h>

// 			int main (int argc, char** argv) {
// 				printf("Hello!\n");
// 			}

// * foo

//   <!-- mdppcode src=misc/hello.c -->

//       #include <stdio.h>

//       int main (int argc, char** argv) {
//       	printf("Hello!\n");
//       }

// Done.
// `)
// 	output := bytes.NewBuffer(nil)
// 	if err := PreprocessWithoutDir(output, input); err != nil {
// 		t.Fatal("error")
// 	}
// 	if bytes.Compare(expected, output.Bytes()) != 0 {
// 		t.Fatalf(`Unmatched:

// %s`, diff.LineDiff(string(expected), output.String()))
// 	}
// }
