// Package tblfm handles Org's TBLFM format.
package tblfm

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

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

var (
	// Formula parser: supports $4=$2*$3 (column), @3=@2 (row), @3$4=@2$2 (cell)
	formulaRe = regexp.MustCompile(`^(@\d+)?(\$\d+)?=(.+)$`)
	// Find cell references like @2$3, $2, $3, $-1, $-2 (with optional row)
	// Supports <, <<, <<< (up to 3 levels) and >, >>, >>> (up to 3 levels)
	cellRefRe = regexp.MustCompile(`(@([-+]?\d+|<{1,3}|>{1,3}))?(\$([-+]?\d+|<{1,3}|>{1,3}))`)
	// Find standalone row references like @2, @<, @<<, @<<< (this will also match @2$ but we process cellRefRe first)
	rowRefRe = regexp.MustCompile(`@([-+]?\d+|<{1,3}|>{1,3})`)
)

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
	startRow := 0
	if cfg.hasHeader {
		startRow = 1
	}

	// Apply each formula in order
	for _, formula := range formulas {
		formula = strings.TrimSpace(formula)
		if formula == "" {
			continue
		}

		// Parse formula
		matches := formulaRe.FindStringSubmatch(formula)
		if matches == nil {
			return resultTable, fmt.Errorf("invalid formula format: %s", formula)
		}

		targetRowSpec := matches[1] // e.g., "@3" or empty
		targetColSpec := matches[2] // e.g., "$4" or empty
		expression := matches[3]

		// Parse target row and column specifications
		var targetRow int = -1 // -1 means all rows
		if targetRowSpec != "" {
			targetRow, _ = strconv.Atoi(targetRowSpec[1:]) // Remove @ and convert
		}

		var targetCol int = -1 // -1 means all columns
		if targetColSpec != "" {
			targetCol, _ = strconv.Atoi(targetColSpec[1:]) // Remove $ and convert
		}

		// Double loop: iterate over all rows and columns
		for rowIdx := startRow; rowIdx < len(table); rowIdx++ {
			row := table[rowIdx]

			// Check if this row matches the target
			if targetRow != -1 && (rowIdx+1) != targetRow {
				continue // Skip this row if it doesn't match the target
			}

			for colIdx := 0; colIdx < len(row); colIdx++ {
				// Check if this column matches the target
				if targetCol != -1 && (colIdx+1) != targetCol {
					continue // Skip this column if it doesn't match the target
				}

				// This cell is a target, evaluate the expression
				currentRow := rowIdx + 1 // 1-based
				currentCol := colIdx + 1 // 1-based

				// Replace row and column references with actual values
				evaluableExpr := expression

				// First, replace cell references (with optional row) like @2$3, $2
				evaluableExpr = cellRefRe.ReplaceAllStringFunc(evaluableExpr, func(ref string) string {
					matches := cellRefRe.FindStringSubmatch(ref)
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
							sourceRow = startRow
						case rowSpec == "<<":
							// Second data row
							sourceRow = startRow + 1
						case rowSpec == "<<<":
							// Third data row
							sourceRow = startRow + 2
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
				evaluableExpr = rowRefRe.ReplaceAllStringFunc(evaluableExpr, func(ref string) string {
					matches := rowRefRe.FindStringSubmatch(ref)
					if matches == nil {
						return ref
					}

					rowSpec := matches[1]
					var sourceRow int

					switch {
					case rowSpec == "<":
						// First data row
						sourceRow = startRow
					case rowSpec == "<<":
						// Second data row
						sourceRow = startRow + 1
					case rowSpec == "<<<":
						// Third data row
						sourceRow = startRow + 2
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
					output, err := expr.Eval(evaluableExpr, nil)
					if err != nil {
						return resultTable, fmt.Errorf("failed to evaluate expression '%s': %w", evaluableExpr, err)
					}

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

				// Set result to target cell
				table[rowIdx][colIdx] = resultStr
			}
		}
	}

	return resultTable, nil
}
