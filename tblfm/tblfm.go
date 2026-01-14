// Package tblfm handles Org's TBLFM format.
package tblfm

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
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
	// $4=$2*$3 form formula parser
	formulaRe = regexp.MustCompile(`^\$(\d+)=(.+)$`)
	// Find cell references like $2, $3, $-1, $-2 (supports relative references)
	cellRefRe = regexp.MustCompile(`\$([-+]?\d+)`)
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

		targetCol, _ := strconv.Atoi(matches[1])
		expression := matches[2]

		// Determine data row start position
		startRow := 0
		if cfg.hasHeader {
			startRow = 1
		}

		// Apply formula to each row
		for rowIdx := startRow; rowIdx < len(table); rowIdx++ {
			row := table[rowIdx]

			// Replace cell references with actual values
			expr := cellRefRe.ReplaceAllStringFunc(expression, func(ref string) string {
				colMatch := cellRefRe.FindStringSubmatch(ref)
				if colMatch == nil {
					return ref
				}
				col, _ := strconv.Atoi(colMatch[1])
				var colIdx int
				if col < 0 {
					// Relative reference: $-1 means one column to the left of target column
					colIdx = (targetCol - 1) + col
				} else {
					// Absolute reference: $2 means column 2
					colIdx = col - 1 // 1-based to 0-based
				}
				if colIdx >= 0 && colIdx < len(row) {
					return row[colIdx]
				}
				return "0"
			})

			// Evaluate expression
			result, err := evaluateSimpleExpression(expr)
			if err != nil {
				return resultTable, err
			}

			// Set result to target column
			targetIdx := targetCol - 1 // 1-based to 0-based
			if targetIdx >= 0 && targetIdx < len(row) {
				table[rowIdx][targetIdx] = strconv.Itoa(result)
			}
		}
	}

	return resultTable, nil
}

// evaluateSimpleExpression evaluates a simple arithmetic expression.
// Supports: +, -, *, / (integer division)
func evaluateSimpleExpression(expr string) (int, error) {
	expr = strings.TrimSpace(expr)

	// Try each operator in order (no precedence handling - simple left-to-right)
	operators := []string{"+", "-", "*", "/"}
	for _, op := range operators {
		if strings.Contains(expr, op) {
			parts := strings.Split(expr, op)
			if len(parts) != 2 {
				continue // Try next operator
			}

			a, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				continue // Try next operator
			}

			b, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				continue // Try next operator
			}

			switch op {
			case "+":
				return a + b, nil
			case "-":
				return a - b, nil
			case "*":
				return a * b, nil
			case "/":
				if b == 0 {
					return 0, fmt.Errorf("division by zero")
				}
				return a / b, nil
			}
		}
	}

	// For numeric values only
	return strconv.Atoi(expr)
}
