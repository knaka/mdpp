# embed.py - Embeds minified file contents into shell-script files
#
# This script processes input files line by line, looking for #EMBED directives.
# When a line matches the pattern:
#   'original content' #EMBED: path/to/file
# It replaces the content between quotes with the minified contents of the
# referenced file, preserving the #EMBED comment for future updates.
#
# Usage:
#   python embed.py input_file >output_file
#
# Supported file types for minification:
#   .awk  - Removes comments, leading whitespace, and appends semicolons
#   .jq   - Removes comments and collapses leading whitespace to single space
#   other - Just joins lines without any preprocessing

import re
import sys
from collections.abc import Iterator
from pathlib import Path


def _minify_awk(lines: Iterator[str]) -> Iterator[str]:
    """Apply AWK-specific minification rules."""
    for line in lines:
        # 1. Remove comment lines
        line = re.sub(r'^\s*#.*', '', line)
        # 2. Remove all leading whitespace
        line = re.sub(r'^\s*', '', line)
        # 3. Append semicolon to lines not ending with {, }, or ;
        line = re.sub(r'([^{};])$', r'\1;', line)
        yield line


def _minify_jq(lines: Iterator[str]) -> Iterator[str]:
    """Apply jq-specific minification rules."""
    for line in lines:
        # 1. Remove comment lines
        line = re.sub(r'^\s*#.*', '', line)
        # 2. Collapse leading whitespace to single space
        line = re.sub(r'^\s+', ' ', line)
        yield line

def _minify_py(lines: Iterator[str]) -> Iterator[str]:
    """Apply py-specific minification rules.

    Note: This only works for Python code written in a one-liner-friendly style:
    - No multi-line control structures (use ternary, list comprehensions, etc.)
    - Single statements per line
    """
    for line in lines:
        # 1. Remove comment lines (but not inline comments)
        if re.match(r'^\s*#', line):
            continue
        # 2. Remove all leading whitespace
        line = re.sub(r'^\s*', '', line)
        # 3. Skip empty lines
        if not line:
            continue
        # 4. Append semicolon if line doesn't end with : or ; (statement separator)
        if not line.endswith(':') and not line.endswith(';'):
            line = line + ';'
        yield line

def minify(path: Path) -> str:
    """
    Reads a file and returns its contents as a single line.
    Applies minification rules based on file extension.

    Args:
        path: Path to the file to read

    Returns:
        String with file contents joined into a single line
    """

    with open(path, 'r') as f:
        lines = (line.rstrip('\n') for line in f)

        match path.suffix:
            case '.awk':
                lines = _minify_awk(lines)
            case '.jq':
                lines = _minify_jq(lines)
            case '.py':
                lines = _minify_py(lines)

        # Join all lines without separator
        return ''.join(lines)


# Matches lines with pattern: 'content' #EMBED: path or "content" #EMBED: path
# Captures:
#   pre  - Everything up to and including the opening quote
#   post - The closing quote through #EMBED: (content between quotes is replaced)
#   path - The file path to embed
EMBED_PATTERN_SINGLE = re.compile(r"^(?P<pre>[^']*')[^']*(?P<post>'.*#EMBED:\s*)(?P<path>.+)$")
EMBED_PATTERN_DOUBLE = re.compile(r'^(?P<pre>[^"]*")[^"]*(?P<post>".*#EMBED:\s*)(?P<path>.+)$')


def process_line(line: str, input_path: Path) -> None:
    """Process a single line, printing the result."""

    match = EMBED_PATTERN_SINGLE.match(line) or EMBED_PATTERN_DOUBLE.match(line)
    if not match:
        print(line, end='')
        return

    embed_path = Path(match.group('path'))
    # Resolve relative paths based on the directory of the input file
    if not embed_path.is_absolute():
        embed_path = input_path.parent / embed_path

    minified = minify(embed_path)
    # Output: prefix + minified content + suffix (including original path)
    print(f"{match.group('pre')}{minified}{match.group('post')}{match.group('path')}")


def process_file(filepath: str) -> None:
    """Process a single file, replacing #EMBED directives with minified content."""

    input_path = Path(filepath)
    with open(input_path, 'r') as f:
        for line in f:
            process_line(line, input_path)


if __name__ == '__main__':
    sys.stdout.reconfigure(newline='\n')
    for filepath in sys.argv[1:]:
        process_file(filepath)
