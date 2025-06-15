package mdpp

import (
	"bytes"
	"testing"

	"github.com/andreyvit/diff"
	"github.com/stretchr/testify/assert"

	. "github.com/knaka/go-utils"
)

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
			V0(Process(tt.sourceMD, writer, nil))
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
		{
			name: "Multiple Headers",
			input: []byte(`# The First

hoge fuga

# The Second

foo bar
`),
			expected: "default",
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
	input := []byte(`Links:

Inline-links [link [contains] brackets](misc/foo.md)<!-- +TITLE -->
and [escaped \[ bracket](./misc/bar.md)<!-- +SYNC_TITLE --> works.
`)
	expected := []byte(`Links:

Inline-links [foo](misc/foo.md)<!-- +TITLE -->
and [Bar ドキュメント](./misc/bar.md)<!-- +SYNC_TITLE --> works.
`)
	writer := bytes.NewBuffer(nil)
	V0(Process(input, writer, nil))
	if bytes.Compare(expected, writer.Bytes()) != 0 {
		t.Fatalf(`Unmatched:
%s`, diff.LineDiff(string(expected), writer.String()))
	}
}

func TestCodeBlock(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name: "basic code block",
			input: []byte(`Code block:

* foo

  ` + "```" + `
  foo
  bar
  ` + "```" + `

  <!-- +CODE: misc/hello.c -->
`),
			expected: []byte(`Code block:

* foo

  ` + "```" + `
  #include <stdio.h>
  
  int main (int argc, char** argv) {
    printf("Hello!\n");
  }
  ` + "```" + `

  <!-- +CODE: misc/hello.c -->
`),
		},
		{
			name: "empty code block",
			input: []byte(`Code block:

* foo

  ` + "````" + `
  ` + "````" + `

  <!-- +CODE: misc/hello.c -->
`),
			expected: []byte(`Code block:

* foo

  ` + "````" + `
  #include <stdio.h>
  
  int main (int argc, char** argv) {
    printf("Hello!\n");
  }
  ` + "````" + `

  <!-- +CODE: misc/hello.c -->
`),
		},
		{
			name: "indented code block",
			input: []byte(`Code block:

* foo

      int x = 10;
      printf("%d", x);

  <!-- +CODE: misc/hello.c -->
`),
			expected: []byte(`Code block:

* foo

      #include <stdio.h>
      
      int main (int argc, char** argv) {
        printf("Hello!\n");
      }

  <!-- +CODE: misc/hello.c -->
`),
		},
		{
			name: "tilde fenced code block",
			input: []byte(`Code block:

* foo

  ~~~
  foo
  bar
  ~~~

  <!-- +CODE: misc/hello.c -->
`),
			expected: []byte(`Code block:

* foo

  ~~~
  #include <stdio.h>
  
  int main (int argc, char** argv) {
    printf("Hello!\n");
  }
  ~~~

  <!-- +CODE: misc/hello.c -->
`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := bytes.NewBuffer(nil)
			if err := Process(tt.input, output, nil); err != nil {
				t.Fatal("error")
			}
			if bytes.Compare(tt.expected, output.Bytes()) != 0 {
				t.Fatalf(`Unmatched:\n\n%s`, diff.LineDiff(string(tt.expected), output.String()))
			}
		})
	}
}
