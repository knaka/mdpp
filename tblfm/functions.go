package tblfm

import (
	"strconv"
	"sync"

	"github.com/expr-lang/expr"
)

// vsumFunction is an Expr function to calculate the sum of values.
func vsumFunction(params ...any) (any, error) {
	if len(params) == 0 {
		return 0.0, nil
	}

	// Handle array parameter
	if arr, ok := params[0].([]any); ok {
		var sum float64
		for _, v := range arr {
			switch val := v.(type) {
			case int:
				sum += float64(val)
			case int64:
				sum += float64(val)
			case float64:
				sum += val
			case string:
				if num, err := strconv.ParseFloat(val, 64); err == nil {
					sum += num
				}
			}
		}
		return sum, nil
	}

	// Handle variadic parameters
	var sum float64
	for _, v := range params {
		switch val := v.(type) {
		case int:
			sum += float64(val)
		case int64:
			sum += float64(val)
		case float64:
			sum += val
		case string:
			if num, err := strconv.ParseFloat(val, 64); err == nil {
				sum += num
			}
		}
	}
	return sum, nil
}

// getBuiltinFunctions returns all built-in functions for expression evaluation.
// Uses sync.OnceValue to ensure expr.Function is only called once.
var getBuiltinFunctions = sync.OnceValue(func() []expr.Option {
	// Sort in dictionary order.
	return []expr.Option{
		expr.Function(
			"vsum",
			vsumFunction,
			new(func([]any) float64),
			new(func(...any) float64),
		),
	}
})
