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
			if !bytes.Equal(tt.expectedMD, writer.Bytes()) {
				t.Fatalf(`Unmatched for %s:

%s`, tt.name, diff.LineDiff(string(tt.expectedMD), writer.String()))
			}
		})
	}
}

func TestPrefixedTBLFMTable(t *testing.T) {
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
| :---: | --- | --- | :--- |
| Apple | 2.5 | 12 | 0 |
| Banana | 2.0 | 5 | 0 |
| Orange | 1.2 | 8 | 0 |
|  |  |  |  |
<!-- +TBLFM: @2$>..@>>$>=$2*$3::@>$>=vsum(@<..@>>) -->

bar
`),
			expectedMD: []byte(`foo

| Item | UnitPrice | Quantity | Total |
| :---: | --- | --- | :--- |
| Apple | 2.5 | 12 | 30 |
| Banana | 2.0 | 5 | 10 |
| Orange | 1.2 | 8 | 9.6 |
|  |  |  | 49.6 |
<!-- +TBLFM: @2$>..@>>$>=$2*$3::@>$>=vsum(@<..@>>) -->

bar
`),
		},
		{
			run:  true,
			name: "Basic case",
			sourceMD: []byte(`foo

> | Item | UnitPrice | Quantity | Total |
> | --- | --- | --- | --- |
> | Apple | 2.5 | 12 | 0 |
> | Banana | 2.0 | 5 | 0 |
> | Orange | 1.2 | 8 | 0 |
> |  |  |  |  |
> 
> <!-- +TBLFM:
>   @2$>..@>>$>=$2*$3
>   @>$>=vsum(@<..@>>)
> -->

bar
`),
			expectedMD: []byte(`foo

> | Item | UnitPrice | Quantity | Total |
> | --- | --- | --- | --- |
> | Apple | 2.5 | 12 | 30 |
> | Banana | 2.0 | 5 | 10 |
> | Orange | 1.2 | 8 | 9.6 |
> |  |  |  | 49.6 |
> 
> <!-- +TBLFM:
>   @2$>..@>>$>=$2*$3
>   @>$>=vsum(@<..@>>)
> -->

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
			if !bytes.Equal(tt.expectedMD, writer.Bytes()) {
				t.Fatalf(`Unmatched for %s:

%s`, tt.name, diff.LineDiff(string(tt.expectedMD), writer.String()))
			}
		})
	}
}

func TestTableInclude(t *testing.T) {
	tests := []struct {
		run        bool
		name       string
		sourceMD   []byte
		expectedMD []byte
	}{
		{
			run:  true,
			name: "CSV include",
			sourceMD: []byte(`Test table from CSV:

| Old | Data | Here |
| --- | --- | --- |
| x | y | z |
<!-- +TABLE_INCLUDE: misc/test_table.csv -->

Done.
`),
			expectedMD: []byte(`Test table from CSV:

| Name | Age | City |
| --- | --- | --- |
| Alice | 30 | Tokyo |
| Bob | 25 | Osaka |
| Charlie | 35 | Kyoto |
<!-- +TABLE_INCLUDE: misc/test_table.csv -->

Done.
`),
		},
		{
			run:  true,
			name: "TSV include with TINCLUDE alias",
			sourceMD: []byte(`Test table from TSV:

| Old | Data |
| :---: | :--- |
| a | b |
<!-- +TINCLUDE: misc/test_table.tsv -->

Done.
`),
			expectedMD: []byte(`Test table from TSV:

| Product | Price | Quantity |
| :---: | :--- | --- |
| Apple | 100 | 5 |
| Banana | 80 | 10 |
| Orange | 120 | 3 |
<!-- +TINCLUDE: misc/test_table.tsv -->

Done.
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
			if !bytes.Equal(tt.expectedMD, writer.Bytes()) {
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
	if !bytes.Equal(expected, writer.Bytes()) {
		t.Fatalf(`Unmatched:
%s`, diff.LineDiff(string(expected), writer.String()))
	}
}

func TestIncludeDirective(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name: "basic include",
			input: []byte(`# Main Document

Some content before include.

<!-- +INCLUDE: misc/include_test.md -->
<!-- +END -->

Some content after include.
`),
			expected: []byte(`# Main Document

Some content before include.

<!-- +INCLUDE: misc/include_test.md -->
# Included Content

This is content from an included file.

## Features

- Feature 1
- Feature 2
- Feature 3

End of included content.
<!-- +END -->

Some content after include.
`),
		},
		{
			name: "include with missing file",
			input: []byte(`# Test

<!-- +INCLUDE: nonexistent.md -->
<!-- +END -->

Done.
`),
			expected: []byte(`# Test

<!-- +INCLUDE: nonexistent.md -->
<!-- +END -->

Done.
`),
		},
		{
			name: "include without matching end",
			input: []byte(`# Test

<!-- +INCLUDE: misc/include_test.md -->

Some text without END directive.
`),
			expected: []byte(`# Test

<!-- +INCLUDE: misc/include_test.md -->

Some text without END directive.
`),
		},
		{
			name: "nested include",
			input: []byte(`# Main Document

Including nested content:

<!-- +INCLUDE: misc/nested_level1.md -->
<!-- +END -->

Done.
`),
			expected: []byte(`# Main Document

Including nested content:

<!-- +INCLUDE: misc/nested_level1.md -->
# Level 1 Content

This includes content from level 2:

<!-- +INCLUDE: misc/nested_level2.md -->
## Level 2 Content

This is the deepest level of nesting.
<!-- +END -->

End of level 1.
<!-- +END -->

Done.
`),
		},
		{
			name: "cyclic include detection",
			input: []byte(`# Test Cycles

<!-- +INCLUDE: misc/cycle_a.md -->
<!-- +END -->

End of test.
`),
			expected: []byte(`# Test Cycles

<!-- +INCLUDE: misc/cycle_a.md -->
# File A

This file includes File B:

<!-- +INCLUDE: misc/cycle_b.md -->
# File B

This file includes File A (creating a cycle):

<!-- +INCLUDE: misc/cycle_a.md -->
<!-- +END -->
<!-- +END -->
<!-- +END -->

End of test.
`),
		},
		{
			name: "canonical path cycle detection",
			input: []byte(`# Test Canonical Path Cycles

<!-- +INCLUDE: misc/canonical_test_a.md -->
<!-- +END -->

End of canonical test.
`),
			expected: []byte(`# Test Canonical Path Cycles

<!-- +INCLUDE: misc/canonical_test_a.md -->
# File A (canonical test)

This file includes File B using different path:

<!-- +INCLUDE: misc/canonical_test_b.md -->
# File B (canonical test)

This file includes File A using different path representation:

<!-- +INCLUDE: ./canonical_test_a.md -->
<!-- +END -->
<!-- +END -->
<!-- +END -->

End of canonical test.
`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First run
			output1 := bytes.NewBuffer(nil)
			if err := Process(tt.input, output1, nil); err != nil {
				t.Fatal("error on first run")
			}
			if !bytes.Equal(tt.expected, output1.Bytes()) {
				t.Fatalf(`Unmatched on first run:\n\n%s`, diff.LineDiff(string(tt.expected), output1.String()))
			}

			// Second run for idempotency test
			output2 := bytes.NewBuffer(nil)
			if err := Process(output1.Bytes(), output2, nil); err != nil {
				t.Fatal("error on second run")
			}
			if !bytes.Equal(output1.Bytes(), output2.Bytes()) {
				t.Fatalf(`Process is not idempotent for %s:\n\n%s`, tt.name, diff.LineDiff(output1.String(), output2.String()))
			}
		})
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
			if !bytes.Equal(tt.expected, output.Bytes()) {
				t.Fatalf(`Unmatched:\n\n%s`, diff.LineDiff(string(tt.expected), output.String()))
			}
		})
	}
}
