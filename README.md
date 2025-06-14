---
title: Document for mdpp(1)
---

mdpp(1)

[![https://pkg.go.dev/github.com/knaka/mdpp](https://pkg.go.dev/badge/github.com/knaka/mdpp.svg)](https://pkg.go.dev/github.com/knaka/mdpp)
[![Actions: Result](https://github.com/knaka/mdpp/actions/workflows/test.yml/badge.svg)](https://github.com/knaka/mdpp/actions/workflows/test.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![https://goreportcard.com/report/github.com/knaka/mdpp](https://goreportcard.com/badge/github.com/knaka/mdpp)](https://goreportcard.com/report/github.com/knaka/mdpp)

# NAME

mdpp - Markdown preprocessor for cross-file code, table, and title synchronization

# INSTALLATION

    go install github.com/knaka/mdpp/cmd/mdpp@latest

# SYNOPSIS

Concatenate the rewritten results and output to standard output:

    mdpp input1.md input2.md >output.md

In-place rewriting:

    mdpp -i rewritten1.md rewritten2.md

# DESCRIPTION

mdpp(1) is a Markdown preprocessor that synchronizes code blocks, tables, and link titles across files using special HTML comment directives. It is designed for use in documentation build pipelines or as an editor integration to keep Markdown content up-to-date with source files and other Markdown documents.

## Supported Directives

### +SYNC_TITLE / +TITLE
Replaces the link text with the title from the target Markdown file.

> The title is determined in the following order of priority:
> 1. The `title` property in YAML Front Matter
> 2. The first H1 (`#`) heading in the document
> 3. The file name (without extension)

**Input:**

````markdown
[link text](docs/hello.md)<!-- +SYNC_TITLE -->
````

**Output:**

````markdown
[Hello document](docs/hello.md)<!-- +SYNC_TITLE -->
````

### +MILLER / +MLR
Processes the table above the directive using a [Miller](https://miller.readthedocs.io/en/latest/) script. This feature is inspired by the `#+TBLFM: ...` line comment of Emacs Org-mode.

**Input:**

````markdown
| Item | Unit Price | Quantity | Total |
| --- | --- | --- | --- |
| Apple | 2.5 | 12 | 0 |
| Banana | 2.0 | 5 | 0 |
| Orange | 1.2 | 8 | 0 |

<!-- +MLR:
  $Total = ${Unit Price} * $Quantity;
-->
````

**Output:**

````markdown
| Item | Unit Price | Quantity | Total |
| --- | --- | --- | --- |
| Apple | 2.5 | 12 | 30 |
| Banana | 2.0 | 5 | 10 |
| Orange | 1.2 | 8 | 9.6 |

<!-- +MLR:
  $Total = ${Unit Price} * $Quantity;
-->
````

### +CODE
Inserts the contents of an external file into a fenced code block.

**Input:**

````markdown
```
foo
bar
```

<!-- +CODE: path/to/file.c -->
````

**Output (after running mdpp):**

````markdown
```
#include <stdio.h>

int main(int argc, char** argv) {
    printf("Hello, World!\n");
    return 0;
}
```

<!-- +CODE: path/to/file.c -->
````

# USAGE EXAMPLES

- Write to standard output:

      mdpp README.md >README.out.md

- In-place update (for editor integration):

      mdpp -i README.md

> For in-place usage, VSCode's plugin “[Run on Save](https://github.com/emeraldwalk/vscode-runonsave)” can automatically run mdpp when saving a Markdown file. Example settings:
>
> ```json
> "emeraldwalk.runonsave": {
>   "commands": [
>     {
>       "match": "\\.md$",
>       "cmd": "mdpp -i ${file}"
>     }
>   ]
> },
> ```

# NOTES

- Directives must be written as HTML comments immediately after the relevant code block, table block, or link inline-element.
- Directive names are case-insensitive.
- The output preserves the directive comments, so repeated runs are idempotent.
- Title extraction uses the following priority:
  1. The `title` property in YAML Front Matter
  2. The first H1 (`#`) heading in the document
  3. The file name (without extension)

# LICENSE

MIT License
