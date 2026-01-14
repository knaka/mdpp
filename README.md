# mdpp(1)

[![https://pkg.go.dev/github.com/knaka/mdpp](https://pkg.go.dev/badge/github.com/knaka/mdpp.svg)](https://pkg.go.dev/github.com/knaka/mdpp)
[![Actions: Result](https://github.com/knaka/mdpp/actions/workflows/test.yml/badge.svg)](https://github.com/knaka/mdpp/actions/workflows/test.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![https://goreportcard.com/report/github.com/knaka/mdpp](https://goreportcard.com/badge/github.com/knaka/mdpp)](https://goreportcard.com/report/github.com/knaka/mdpp)

Markdown preprocessor for cross-file code, table, title synchronization, and file inclusion. Processing is idempotent.

## INSTALLATION

Pre-built binaries are available at [Releases](https://github.com/knaka/mdpp/releases).

Build from source:

    go install github.com/knaka/mdpp/cmd/mdpp@latest

## SYNOPSIS

Concatenate the rewritten results and output to standard output:

    mdpp input1.md input2.md >output.md

In-place rewriting:

    mdpp -i rewritten1.md rewritten2.md

## DESCRIPTION

mdpp(1) is a Markdown preprocessor that synchronizes code blocks, tables, and link titles across files, and includes external Markdown files using special HTML comment directives. It is designed for use in documentation build pipelines or as an editor integration to keep Markdown content up-to-date with source files and other Markdown documents.

### Supported Directives

#### +SYNC_TITLE / +TITLE
Replaces the link text with the title from the target Markdown file.

> The title is determined in the following order of priority:
> 1. The `title` property in YAML Front Matter
> 2. The only H1 (`#`) heading in the document (if there is exactly one)
> 3. The file name (without extension)

**Input:**

````markdown
[link text](docs/hello.md)<!-- +SYNC_TITLE -->
````

**Output:**

````markdown
[Hello document](docs/hello.md)<!-- +SYNC_TITLE -->
````

#### +MILLER / +MLR
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

#### +TBLFM
Processes the table above the directive using table formulas similar to Emacs Org-mode's `#+TBLFM:` feature. This directive supports Org-mode-style cell references and formulas.

**Input:**

````markdown
| Item | UnitPrice | Quantity | Total |
| --- | --- | --- | --- |
| Apple | 2.5 | 12 | 0 |
| Banana | 2.0 | 5 | 0 |
| Orange | 1.2 | 8 | 0 |
|  |  |  |  |

<!-- +TBLFM:
  @2$>..@>>$>=$2*$3
  @>$>=vsum(@<..@>>)
-->
````

**Output:**

````markdown
| Item | UnitPrice | Quantity | Total |
| --- | --- | --- | --- |
| Apple | 2.5 | 12 | 30 |
| Banana | 2.0 | 5 | 10 |
| Orange | 1.2 | 8 | 9.6 |
|  |  |  | 49.6 |

<!-- +TBLFM:
  @2$>..@>>$>=$2*$3
  @>$>=vsum(@<..@>>)
-->
````

The formula syntax uses Org-mode-style cell references:
- `@2` refers to row 2 (first data row after header)
- `$>` refers to the last column
- `@>` refers to the last row
- `@>>` refers to the second-to-last row
- `@<` refers to the first data row
- Ranges are specified with `..` (e.g., `@2$>..@>>$>` means "from row 2 last column to second-to-last row last column")
- Multiple formulas can be specified, separated by newlines or `::`

#### +INCLUDE ... +END

Includes the content of an external Markdown file or remote URL.

**Input:**

````markdown
<!-- +INCLUDE: path/to/another.md -->
<!-- +END -->
````

**Output (after running mdpp):**

```markdown
<!-- +INCLUDE: path/to/another.md -->
## Content from another.md

This is the content of `another.md`.
<!-- +END -->
```

**Remote URL support:**

By default, only local files can be included. To enable fetching content from remote URLs, use the `--allow-remote` flag:

```bash
mdpp --allow-remote document.md
```

**Example with remote URL:**

````markdown
<!-- +INCLUDE: https://example.com/content.md -->
<!-- +END -->
````

**Features:**

- **Nested inclusion**: Files included with `+INCLUDE` can contain their own `+INCLUDE` directives, supporting multiple levels of nesting.
- **Cycle detection**: The processor automatically detects and prevents infinite loops when files include each other in a cycle (works for both local files and URLs).
- **Security**: Remote URL fetching is disabled by default and must be explicitly enabled with `--allow-remote` flag.

**Limitations:**

- **Indented directives**: The `+INCLUDE` and `+END` directives must be at the beginning of their lines (ignoring leading/trailing whitespace). Indented directives within code blocks or blockquotes are not supported.
- **Relative path resolution**: When including a file from another directory, relative paths within the included content (such as image paths) are not automatically resolved relative to the included file's location. They remain relative to the main document's directory.
- **URL schemes**: Only `http://` and `https://` URLs are supported for remote content.

#### +CODE
Inserts the contents of an external file into a fenced or indented code block.

**Input (fenced code block):**

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

**Input (indented code block):**

````markdown
    int x = 0;
    printf("%d", x);

<!-- +CODE: path/to/file.c -->
````

**Output (indented code block):**

````markdown
    #include <stdio.h>
    
    int main(int argc, char** argv) {
        printf("Hello, World!\n");
        return 0;
    }

<!-- +CODE: path/to/file.c -->
````

## USAGE EXAMPLES

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

## NOTES

- Directives must be written as HTML comments immediately after the relevant code block, table block, or link inline-element.
- For `+INCLUDE` directives, both `+INCLUDE` and `+END` comments must be at the beginning of their lines (ignoring leading/trailing whitespace).
- Directive names are case-insensitive.
- The output preserves the directive comments, so repeated runs are idempotent.
- Title extraction uses the following priority:
  1. The `title` property in YAML Front Matter
  2. The only H1 (`#`) heading in the document (if there is exactly one)
  3. The file name (without extension)

## LICENSE

MIT License
