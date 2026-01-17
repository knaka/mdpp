// Package tblfm handles Org's TBLFM format.
package tblfm

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/expr-lang/expr"
)

// Option is a functional option for Apply.
type Option func(*config)

// config holds the configuration for Apply.
type config struct {
	hasHeader bool
}

// WithHeader specifies whether the first row is a header row.
// Default is true (has header).
func WithHeader(hasHeader bool) Option {
	return func(c *config) {
		c.hasHeader = hasHeader
	}
}

// regexps holds all compiled regular expressions used for TBLFM parsing.
type regexps struct {
	formulaRe  *regexp.Regexp
	cellRefRe  *regexp.Regexp
	rowRefRe   *regexp.Regexp
	cellPosRe  *regexp.Regexp
	rangeRefRe *regexp.Regexp
}

// getRegexps returns all compiled regular expressions.
// Uses sync.OnceValue to ensure regexp.MustCompile is only called once.
var getRegexps = sync.OnceValue(func() *regexps {
	return &regexps{
		// Formula parser: supports $4=$2*$3 (column), @3=@2 (row), @3$4=@2$2 (cell)
		// Also supports range syntax: @2$>..@>>$>=@1$>
		formulaRe: regexp.MustCompile(`^((?:@[-+]?\d+|@<{1,3}|@>{1,3})?(?:\$[-+]?\d+|\$<{1,3}|\$>{1,3})?)(?:\.\.((?:@[-+]?\d+|@<{1,3}|@>{1,3})?(?:\$[-+]?\d+|\$<{1,3}|\$>{1,3})?))?=(.+)$`),
		// Find cell references like @2$3, $2, $3, $-1, $-2 (with optional row)
		// Supports <, <<, <<< (up to 3 levels) and >, >>, >>> (up to 3 levels)
		cellRefRe: regexp.MustCompile(`(@([-+]?\d+|<{1,3}|>{1,3}))?(\$([-+]?\d+|<{1,3}|>{1,3}))`),
		// Find standalone row references like @2, @<, @<<, @<<< (this will also match @2$ but we process cellRefRe first)
		rowRefRe: regexp.MustCompile(`@([-+]?\d+|<{1,3}|>{1,3})`),
		// Parse cell position like @2$3, $4, @3
		cellPosRe: regexp.MustCompile(`^(?:@([-+]?\d+|<{1,3}|>{1,3}))?(?:\$([-+]?\d+|<{1,3}|>{1,3}))?$`),
		// Find range references like @<..@>> or @2$1..@5$3
		rangeRefRe: regexp.MustCompile(`((?:@[-+]?\d+|@<{1,3}|@>{1,3})?(?:\$[-+]?\d+|\$<{1,3}|\$>{1,3})?)\.\.((?:@[-+]?\d+|@<{1,3}|@>{1,3})?(?:\$[-+]?\d+|\$<{1,3}|\$>{1,3})?)`),
	}
})

// expandRange expands a range reference like "@<..@>>" into an array of values
func expandRange(startPos, endPos string, table [][]string, currentRow, currentCol, dataStartRow int) []any {
	maxRowLen := 0
	for _, r := range table {
		if len(r) > maxRowLen {
			maxRowLen = len(r)
		}
	}

	startRow, startCol := parseCellPosition(startPos, dataStartRow, len(table), maxRowLen, currentRow, currentCol)
	endRow, endCol := parseCellPosition(endPos, dataStartRow, len(table), maxRowLen, currentRow, currentCol)

	var values []any

	// If only row is specified (no column), assume current column
	if startCol == -1 && endCol == -1 {
		startCol = currentCol - 1
		endCol = currentCol - 1
	}

	// If only column is specified (no row), iterate through all data rows
	if startRow == -1 && endRow == -1 {
		startRow = dataStartRow
		endRow = len(table) - 1
	}

	// Expand the range
	for r := startRow; r >= 0 && r <= endRow && r < len(table); r++ {
		for c := startCol; c >= 0 && c <= endCol && c < len(table[r]); c++ {
			val := table[r][c]
			// Try to parse as number
			if num, err := strconv.ParseFloat(val, 64); err == nil {
				values = append(values, num)
			} else {
				values = append(values, val)
			}
		}
	}

	return values
}

// parseCellPosition parses a cell position specification like "@2$3", "$4", "@3"
// Returns (row, col) where -1 means "any" (not specified)
// currentRow and currentCol are 1-based positions used for relative references
func parseCellPosition(pos string, startRow int, tableLen int, rowLen int, currentRow int, currentCol int) (row int, col int) {
	row = -1
	col = -1

	if pos == "" {
		return
	}

	matches := getRegexps().cellPosRe.FindStringSubmatch(pos)
	if matches == nil {
		return
	}

	rowSpec := matches[1]
	colSpec := matches[2]

	// Parse row
	if rowSpec != "" {
		switch {
		case rowSpec == "<":
			row = 0 // First row (header if exists)
		case rowSpec == "<<":
			row = 1 // Second row
		case rowSpec == "<<<":
			row = 2 // Third row
		case rowSpec == ">":
			row = tableLen - 1
		case rowSpec == ">>":
			row = tableLen - 2
		case rowSpec == ">>>":
			row = tableLen - 3
		default:
			rowNum, _ := strconv.Atoi(rowSpec)
			if rowNum > 0 {
				row = rowNum - 1 // 1-based to 0-based
			} else if rowNum < 0 && currentRow > 0 {
				// Relative reference: @-1 means one row above current
				row = currentRow - 1 + rowNum
			}
		}
	}

	// Parse column
	if colSpec != "" {
		switch {
		case colSpec == "<":
			col = 0
		case colSpec == "<<":
			col = 1
		case colSpec == "<<<":
			col = 2
		case colSpec == ">":
			col = rowLen - 1
		case colSpec == ">>":
			col = rowLen - 2
		case colSpec == ">>>":
			col = rowLen - 3
		default:
			colNum, _ := strconv.Atoi(colSpec)
			if colNum > 0 {
				col = colNum - 1 // 1-based to 0-based
			} else if colNum < 0 && currentCol > 0 {
				// Relative reference: $-1 means one column to the left
				col = currentCol - 1 + colNum
			}
		}
	}

	return
}

// Apply performs table calculations using TBLFM formulas on the input 2D array and returns the modified table.
func Apply(
	table [][]string, // Input table (modified in place)
	formulas []string, // TBLFM formula strings
	opts ...Option, // Functional options
) (
	resultTable [][]string, // Updated table (or the same pointer)
	err error,
) {
	cfg := &config{
		hasHeader: true, // Default: has header
	}
	for _, opt := range opts {
		opt(cfg)
	}

	resultTable = table

	// If formulas are empty, do nothing
	if len(formulas) == 0 {
		return
	}

	// Determine data row start position
	dataStartRow := 0
	if cfg.hasHeader {
		dataStartRow = 1
	}

	// Apply each formula in order
	for _, formula := range formulas {
		formula = strings.TrimSpace(formula)
		if formula == "" {
			continue
		}

		// Parse formula
		matches := getRegexps().formulaRe.FindStringSubmatch(formula)
		if matches == nil {
			return resultTable, fmt.Errorf("invalid formula format: %s", formula)
		}

		startPosSpec := matches[1] // e.g., "@2$>" or "$4" or empty
		endPosSpec := matches[2]   // e.g., "@>>$>" or empty (if no range)
		expression := matches[3]

		// Determine maximum row length for column parsing
		maxRowLen := 0
		for _, r := range table {
			if len(r) > maxRowLen {
				maxRowLen = len(r)
			}
		}

		// Parse start position (no current position for target specification)
		targetStartRow, targetStartCol := parseCellPosition(startPosSpec, dataStartRow, len(table), maxRowLen, 0, 0)

		// Parse end position (if range specified)
		var targetEndRow, targetEndCol int = -1, -1
		if endPosSpec != "" {
			targetEndRow, targetEndCol = parseCellPosition(endPosSpec, dataStartRow, len(table), maxRowLen, 0, 0)
		}

		// Determine target range
		var targetRowStart, targetRowEnd int
		var targetColStart, targetColEnd int

		if endPosSpec == "" {
			// Single cell or column/row specification
			targetRowStart = targetStartRow
			targetRowEnd = targetStartRow
			targetColStart = targetStartCol
			targetColEnd = targetStartCol
		} else {
			// Range specification
			targetRowStart = targetStartRow
			targetRowEnd = targetEndRow
			targetColStart = targetStartCol
			targetColEnd = targetEndCol
		}

		// Double loop: iterate over all rows and columns
		for rowIdx := dataStartRow; rowIdx < len(table); rowIdx++ {
			row := table[rowIdx]

			// Check if this row matches the target range
			if targetRowStart != -1 && rowIdx < targetRowStart {
				continue // Skip rows before start
			}
			if targetRowEnd != -1 && rowIdx > targetRowEnd {
				continue // Skip rows after end
			}

			for colIdx := 0; colIdx < len(row); colIdx++ {
				// Check if this column matches the target range
				if targetColStart != -1 && colIdx < targetColStart {
					continue // Skip columns before start
				}
				if targetColEnd != -1 && colIdx > targetColEnd {
					continue // Skip columns after end
				}

				// This cell is a target, evaluate the expression
				currentRow := rowIdx + 1 // 1-based
				currentCol := colIdx + 1 // 1-based

				// Replace row and column references with actual values
				evaluableExpr := expression

				// First, replace range references with array literals (before cell/row references)
				evaluableExpr = getRegexps().rangeRefRe.ReplaceAllStringFunc(evaluableExpr, func(rangeRef string) string {
					matches := getRegexps().rangeRefRe.FindStringSubmatch(rangeRef)
					if matches == nil {
						return rangeRef
					}

					startPos := matches[1]
					endPos := matches[2]

					// Expand the range into an array
					values := expandRange(startPos, endPos, table, currentRow, currentCol, dataStartRow)

					// Convert to array literal string
					var parts []string
					for _, v := range values {
						switch val := v.(type) {
						case float64:
							parts = append(parts, strconv.FormatFloat(val, 'f', -1, 64))
						case string:
							// Skip empty strings in numeric contexts
							if val != "" {
								// Quote strings
								parts = append(parts, fmt.Sprintf("\"%s\"", val))
							}
						default:
							parts = append(parts, fmt.Sprintf("%v", val))
						}
					}

					return "[" + strings.Join(parts, ",") + "]"
				})

				// Then, replace cell references (with optional row) like @2$3, $2
				evaluableExpr = getRegexps().cellRefRe.ReplaceAllStringFunc(evaluableExpr, func(ref string) string {
					matches := getRegexps().cellRefRe.FindStringSubmatch(ref)
					if matches == nil {
						return ref
					}

					// matches[1] = full row part (e.g., "@2")
					// matches[2] = row spec (e.g., "2", "<", ">")
					// matches[3] = full col part (e.g., "$3")
					// matches[4] = col spec (e.g., "3", "-1")

					rowSpec := matches[2]
					colSpec := matches[4]

					// Determine source row
					var sourceRow int
					if rowSpec == "" {
						// No row specified, use current row
						sourceRow = rowIdx
					} else {
						switch {
						case rowSpec == "<":
							// First data row
							sourceRow = 0
						case rowSpec == "<<":
							// Second data row
							sourceRow = 0 + 1
						case rowSpec == "<<<":
							// Third data row
							sourceRow = 0 + 2
						case rowSpec == ">":
							// Last row
							sourceRow = len(table) - 1
						case rowSpec == ">>":
							// Second to last row
							sourceRow = len(table) - 2
						case rowSpec == ">>>":
							// Third to last row
							sourceRow = len(table) - 3
						default:
							// Numeric row reference
							rowNum, _ := strconv.Atoi(rowSpec)
							if rowNum < 0 {
								// Relative reference
								sourceRow = currentRow - 1 + rowNum
							} else {
								// Absolute reference (1-based)
								sourceRow = rowNum - 1
							}
						}
					}

					// Determine source column
					var sourceCol int
					switch {
					case colSpec == "<":
						// First column
						sourceCol = 0
					case colSpec == "<<":
						// Second column
						sourceCol = 1
					case colSpec == "<<<":
						// Third column
						sourceCol = 2
					case colSpec == ">":
						// Last column
						sourceCol = len(table[sourceRow]) - 1
					case colSpec == ">>":
						// Second to last column
						sourceCol = len(table[sourceRow]) - 2
					case colSpec == ">>>":
						// Third to last column
						sourceCol = len(table[sourceRow]) - 3
					default:
						// Numeric column reference
						colNum, _ := strconv.Atoi(colSpec)
						if colNum < 0 {
							// Relative reference: $-1 means one column to the left of current column
							sourceCol = currentCol - 1 + colNum
						} else {
							// Absolute reference: $2 means column 2
							sourceCol = colNum - 1 // 1-based to 0-based
						}
					}

					// Get the cell value
					if sourceRow >= 0 && sourceRow < len(table) &&
						sourceCol >= 0 && sourceCol < len(table[sourceRow]) {
						return table[sourceRow][sourceCol]
					}
					return "0"
				})

				// Then, replace standalone row references like @<, @<<, @> (for row copy operations)
				evaluableExpr = getRegexps().rowRefRe.ReplaceAllStringFunc(evaluableExpr, func(ref string) string {
					matches := getRegexps().rowRefRe.FindStringSubmatch(ref)
					if matches == nil {
						return ref
					}

					rowSpec := matches[1]
					var sourceRow int

					switch {
					case rowSpec == "<":
						// First row (header if exists)
						sourceRow = 0
					case rowSpec == "<<":
						// Second row
						sourceRow = 1
					case rowSpec == "<<<":
						// Third row
						sourceRow = 2
					case rowSpec == ">":
						// Last row
						sourceRow = len(table) - 1
					case rowSpec == ">>":
						// Second to last row
						sourceRow = len(table) - 2
					case rowSpec == ">>>":
						// Third to last row
						sourceRow = len(table) - 3
					default:
						// Numeric row reference
						rowNum, _ := strconv.Atoi(rowSpec)
						if rowNum < 0 {
							// Relative reference
							sourceRow = currentRow - 1 + rowNum
						} else {
							// Absolute reference (1-based)
							sourceRow = rowNum - 1
						}
					}

					// For row copy operations, return the value from the same column in the source row
					if sourceRow >= 0 && sourceRow < len(table) &&
						colIdx >= 0 && colIdx < len(table[sourceRow]) {
						return table[sourceRow][colIdx]
					}
					return "0"
				})

				// Evaluate expression using expr library
				var resultStr string
				if evaluableExpr == "" {
					// If expression is empty (e.g., copying an empty cell), use empty string
					resultStr = ""
				} else {
					// Try to parse as a number first
					if num, err := strconv.ParseFloat(evaluableExpr, 64); err == nil {
						// It's a number, use it directly
						if num == float64(int64(num)) {
							resultStr = strconv.FormatInt(int64(num), 10)
						} else {
							resultStr = strconv.FormatFloat(num, 'f', -1, 64)
						}
					} else {
						// Compile and evaluate the expression with built-in functions
						program, err := expr.Compile(evaluableExpr, getBuiltinFunctions()...)
						if err != nil {
							// If compilation fails, it might be a plain string value
							resultStr = evaluableExpr
						} else {
							output, err := expr.Run(program, map[string]any{})
							if err != nil {
								// If evaluation fails, it might be a plain string value from a cell reference
								// In this case, just use the value as-is
								resultStr = evaluableExpr
							} else {
								// Convert result to string
								switch v := output.(type) {
								case int:
									resultStr = strconv.Itoa(v)
								case int64:
									resultStr = strconv.FormatInt(v, 10)
								case float64:
									// Check if it's a whole number
									if v == float64(int64(v)) {
										resultStr = strconv.FormatInt(int64(v), 10)
									} else {
										resultStr = strconv.FormatFloat(v, 'f', -1, 64)
									}
								case string:
									resultStr = v
								default:
									resultStr = fmt.Sprintf("%v", output)
								}
							}
						}
					}
				}

				// Set result to target cell
				table[rowIdx][colIdx] = resultStr
			}
		}
	}

	return resultTable, nil
}
