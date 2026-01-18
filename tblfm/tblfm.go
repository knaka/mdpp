// Package tblfm handles table formulas using Lua for expression evaluation.
package tblfm

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	lua "github.com/yuin/gopher-lua"
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

// regexps returns all compiled regular expressions.
// Uses sync.OnceValue to ensure regexp.MustCompile is only called once.
var regexps = sync.OnceValue(func() (r struct {
	formula  *regexp.Regexp
	cellRef  *regexp.Regexp
	rowRef   *regexp.Regexp
	cellPos  *regexp.Regexp
	rangeRef *regexp.Regexp
}) {
	// Formula parser: supports $4=$2*$3 (column), @3=@2 (row), @3$4=@2$2 (cell)
	// Also supports range syntax: @2$>..@>>$>=@1$>
	r.formula = regexp.MustCompile(`^((?:@[-+]?\d+|@<{1,3}|@>{1,3})?(?:\$[-+]?\d+|\$<{1,3}|\$>{1,3})?)(?:\.\.((?:@[-+]?\d+|@<{1,3}|@>{1,3})?(?:\$[-+]?\d+|\$<{1,3}|\$>{1,3})?))?\s*=\s*(.+)$`)

	// Find cell references like @2$3, $2, $3, $-1, $-2 (with optional row)
	// Supports <, <<, <<< (up to 3 levels) and >, >>, >>> (up to 3 levels)
	r.cellRef = regexp.MustCompile(`(@([-+]?\d+|<{1,3}|>{1,3}))?(\$([-+]?\d+|<{1,3}|>{1,3}))`)

	// Find standalone row references like @2, @<, @<<, @<<< (this will also match @2$ but we process cellRefRe first)
	r.rowRef = regexp.MustCompile(`@([-+]?\d+|<{1,3}|>{1,3})`)

	// Parse cell position like @2$3, $4, @3
	r.cellPos = regexp.MustCompile(`^(?:@([-+]?\d+|<{1,3}|>{1,3}))?(?:\$([-+]?\d+|<{1,3}|>{1,3}))?$`)

	// Find range references like @<..@>> or @2$1..@5$3
	r.rangeRef = regexp.MustCompile(`((?:@[-+]?\d+|@<{1,3}|@>{1,3})?(?:\$[-+]?\d+|\$<{1,3}|\$>{1,3})?)\.\.((?:@[-+]?\d+|@<{1,3}|@>{1,3})?(?:\$[-+]?\d+|\$<{1,3}|\$>{1,3})?)`)

	return
})

// parseCellPosition parses a cell position specification like "@2$3", "$4", "@3"
// Returns (row, col) where -1 means "any" (not specified)
// currentRow and currentCol are 1-based positions used for relative references
func parseCellPosition(pos string, tableLen int, rowLen int, currentRow int, currentCol int) (row int, col int) {
	row = -1
	col = -1

	if pos == "" {
		return
	}

	matches := regexps().cellPos.FindStringSubmatch(pos)
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

	// Create Lua state
	L := lua.NewState()
	defer L.Close()

	// Register built-in functions
	registerBuiltinFunctions(L)

	// Apply each formula in order
	for _, formula := range formulas {
		formula = strings.TrimSpace(formula)
		if formula == "" {
			continue
		}

		// Parse formula
		matches := regexps().formula.FindStringSubmatch(formula)
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
		targetStartRow, targetStartCol := parseCellPosition(startPosSpec, len(table), maxRowLen, 0, 0)

		// Parse end position (if range specified)
		var targetEndRow, targetEndCol int = -1, -1
		if endPosSpec != "" {
			targetEndRow, targetEndCol = parseCellPosition(endPosSpec, len(table), maxRowLen, 0, 0)
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

				// Evaluate the expression using Lua
				resultStr, err := evaluateExpression(L, expression, table, currentRow, currentCol, dataStartRow)
				if err != nil {
					return resultTable, fmt.Errorf("error evaluating formula %s at @%d$%d: %w", formula, currentRow, currentCol, err)
				}

				// Set result to target cell
				table[rowIdx][colIdx] = resultStr
			}
		}
	}

	return resultTable, nil
}

// evaluateExpression evaluates a Lua expression with cell references replaced by actual values
func evaluateExpression(L *lua.LState, expression string, table [][]string, currentRow int, currentCol int, dataStartRow int) (string, error) {
	// Replace cell and row references with Lua code
	evaluableExpr := expression

	// First, replace range references with Lua table literals
	evaluableExpr = regexps().rangeRef.ReplaceAllStringFunc(evaluableExpr, func(rangeRef string) string {
		matches := regexps().rangeRef.FindStringSubmatch(rangeRef)
		if matches == nil {
			return rangeRef
		}

		startPos := matches[1]
		endPos := matches[2]

		if startPos == "" || endPos == "" {
			return rangeRef
		}

		// Expand the range into a Lua table
		values := expandRange(startPos, endPos, table, currentRow, currentCol, dataStartRow)

		// Convert to Lua table literal string
		var parts []string
		for _, v := range values {
			switch val := v.(type) {
			case float64:
				parts = append(parts, strconv.FormatFloat(val, 'f', -1, 64))
			case string:
				// Skip empty strings in numeric contexts
				if val != "" {
					// Quote strings
					parts = append(parts, fmt.Sprintf("%q", val))
				}
			default:
				parts = append(parts, fmt.Sprintf("%v", val))
			}
		}

		return "{" + strings.Join(parts, ",") + "}"
	})

	// Then, replace cell references (with optional row) like @2$3, $2
	evaluableExpr = regexps().cellRef.ReplaceAllStringFunc(evaluableExpr, func(ref string) string {
		matches := regexps().cellRef.FindStringSubmatch(ref)
		if matches == nil {
			return ref
		}

		rowSpec := matches[2]
		colSpec := matches[4]

		// Determine source row
		var sourceRow int
		if rowSpec == "" {
			sourceRow = currentRow - 1 // 1-based to 0-based
		} else {
			switch {
			case rowSpec == "<":
				sourceRow = 0
			case rowSpec == "<<":
				sourceRow = 1
			case rowSpec == "<<<":
				sourceRow = 2
			case rowSpec == ">":
				sourceRow = len(table) - 1
			case rowSpec == ">>":
				sourceRow = len(table) - 2
			case rowSpec == ">>>":
				sourceRow = len(table) - 3
			default:
				rowNum, _ := strconv.Atoi(rowSpec)
				if rowNum < 0 {
					sourceRow = currentRow - 1 + rowNum
				} else {
					sourceRow = rowNum - 1
				}
			}
		}

		// Determine source column
		var sourceCol int
		switch {
		case colSpec == "<":
			sourceCol = 0
		case colSpec == "<<":
			sourceCol = 1
		case colSpec == "<<<":
			sourceCol = 2
		case colSpec == ">":
			sourceCol = len(table[sourceRow]) - 1
		case colSpec == ">>":
			sourceCol = len(table[sourceRow]) - 2
		case colSpec == ">>>":
			sourceCol = len(table[sourceRow]) - 3
		default:
			colNum, _ := strconv.Atoi(colSpec)
			if colNum < 0 {
				sourceCol = currentCol - 1 + colNum
			} else {
				sourceCol = colNum - 1
			}
		}

		// Get the cell value
		if sourceRow >= 0 && sourceRow < len(table) &&
			sourceCol >= 0 && sourceCol < len(table[sourceRow]) {
			cellValue := table[sourceRow][sourceCol]

			// Try to parse as number
			if num, err := strconv.ParseFloat(cellValue, 64); err == nil {
				return strconv.FormatFloat(num, 'f', -1, 64)
			}

			// Return as quoted string
			return fmt.Sprintf("%q", cellValue)
		}
		return "0"
	})

	// Then, replace standalone row references like @<, @<<, @> (for row copy operations)
	evaluableExpr = regexps().rowRef.ReplaceAllStringFunc(evaluableExpr, func(ref string) string {
		matches := regexps().rowRef.FindStringSubmatch(ref)
		if matches == nil {
			return ref
		}

		rowSpec := matches[1]
		var sourceRow int

		switch {
		case rowSpec == "<":
			sourceRow = 0
		case rowSpec == "<<":
			sourceRow = 1
		case rowSpec == "<<<":
			sourceRow = 2
		case rowSpec == ">":
			sourceRow = len(table) - 1
		case rowSpec == ">>":
			sourceRow = len(table) - 2
		case rowSpec == ">>>":
			sourceRow = len(table) - 3
		default:
			rowNum, _ := strconv.Atoi(rowSpec)
			if rowNum < 0 {
				sourceRow = currentRow - 1 + rowNum
			} else {
				sourceRow = rowNum - 1
			}
		}

		// For row copy operations, return the value from the same column in the source row
		if sourceRow >= 0 && sourceRow < len(table) &&
			currentCol-1 >= 0 && currentCol-1 < len(table[sourceRow]) {
			cellValue := table[sourceRow][currentCol-1]

			// Try to parse as number
			if num, err := strconv.ParseFloat(cellValue, 64); err == nil {
				return strconv.FormatFloat(num, 'f', -1, 64)
			}

			// Return as quoted string
			return fmt.Sprintf("%q", cellValue)
		}
		return "0"
	})

	// Execute Lua code to get result
	if err := L.DoString("return " + evaluableExpr); err != nil {
		return "", err
	}

	// Get the result from Lua stack
	ret := L.Get(-1)
	L.Pop(1)

	// Convert result to string
	switch v := ret.(type) {
	case *lua.LNumber:
		num := float64(*v)
		if num == float64(int64(num)) {
			return strconv.FormatInt(int64(num), 10), nil
		}
		return strconv.FormatFloat(num, 'f', -1, 64), nil
	case lua.LString:
		return string(v), nil
	case lua.LBool:
		if v {
			return "true", nil
		}
		return "false", nil
	default:
		return fmt.Sprintf("%v", ret), nil
	}
}

// expandRange expands a range reference like "@<..@>>" into an array of values
func expandRange(startPos, endPos string, table [][]string, currentRow, currentCol, dataStartRow int) []any {
	maxRowLen := 0
	for _, r := range table {
		if len(r) > maxRowLen {
			maxRowLen = len(r)
		}
	}

	startRow, startCol := parseCellPosition(startPos, len(table), maxRowLen, currentRow, currentCol)
	endRow, endCol := parseCellPosition(endPos, len(table), maxRowLen, currentRow, currentCol)

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
