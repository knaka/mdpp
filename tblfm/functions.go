package tblfm

import (
	"math/rand/v2"
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

// vmaxFunction is an Expr function to find the maximum value.
func vmaxFunction(params ...any) (any, error) {
	if len(params) == 0 {
		return 0.0, nil
	}

	var max float64
	var hasValue bool

	// Handle array parameter
	if arr, ok := params[0].([]any); ok {
		for _, v := range arr {
			switch val := v.(type) {
			case int:
				fval := float64(val)
				if !hasValue || fval > max {
					max = fval
					hasValue = true
				}
			case int64:
				fval := float64(val)
				if !hasValue || fval > max {
					max = fval
					hasValue = true
				}
			case float64:
				if !hasValue || val > max {
					max = val
					hasValue = true
				}
			case string:
				if num, err := strconv.ParseFloat(val, 64); err == nil {
					if !hasValue || num > max {
						max = num
						hasValue = true
					}
				}
			}
		}
		if !hasValue {
			return 0.0, nil
		}
		return max, nil
	}

	// Handle variadic parameters
	for _, v := range params {
		switch val := v.(type) {
		case int:
			fval := float64(val)
			if !hasValue || fval > max {
				max = fval
				hasValue = true
			}
		case int64:
			fval := float64(val)
			if !hasValue || fval > max {
				max = fval
				hasValue = true
			}
		case float64:
			if !hasValue || val > max {
				max = val
				hasValue = true
			}
		case string:
			if num, err := strconv.ParseFloat(val, 64); err == nil {
				if !hasValue || num > max {
					max = num
					hasValue = true
				}
			}
		}
	}
	if !hasValue {
		return 0.0, nil
	}
	return max, nil
}

// vmeanFunction is an Expr function to calculate the mean (average) of values.
func vmeanFunction(params ...any) (any, error) {
	if len(params) == 0 {
		return 0.0, nil
	}

	// Handle array parameter
	if arr, ok := params[0].([]any); ok {
		if len(arr) == 0 {
			return 0.0, nil
		}
		var sum float64
		var count int
		for _, v := range arr {
			switch val := v.(type) {
			case int:
				sum += float64(val)
				count++
			case int64:
				sum += float64(val)
				count++
			case float64:
				sum += val
				count++
			case string:
				if num, err := strconv.ParseFloat(val, 64); err == nil {
					sum += num
					count++
				}
			}
		}
		if count == 0 {
			return 0.0, nil
		}
		return sum / float64(count), nil
	}

	// Handle variadic parameters
	var sum float64
	var count int
	for _, v := range params {
		switch val := v.(type) {
		case int:
			sum += float64(val)
			count++
		case int64:
			sum += float64(val)
			count++
		case float64:
			sum += val
			count++
		case string:
			if num, err := strconv.ParseFloat(val, 64); err == nil {
				sum += num
				count++
			}
		}
	}
	if count == 0 {
		return 0.0, nil
	}
	return sum / float64(count), nil
}

// vmedianFunction is an Expr function to calculate the median of values.
func vmedianFunction(params ...any) (any, error) {
	if len(params) == 0 {
		return 0.0, nil
	}

	var values []float64

	// Handle array parameter
	if arr, ok := params[0].([]any); ok {
		for _, v := range arr {
			switch val := v.(type) {
			case int:
				values = append(values, float64(val))
			case int64:
				values = append(values, float64(val))
			case float64:
				values = append(values, val)
			case string:
				if num, err := strconv.ParseFloat(val, 64); err == nil {
					values = append(values, num)
				}
			}
		}
	} else {
		// Handle variadic parameters
		for _, v := range params {
			switch val := v.(type) {
			case int:
				values = append(values, float64(val))
			case int64:
				values = append(values, float64(val))
			case float64:
				values = append(values, val)
			case string:
				if num, err := strconv.ParseFloat(val, 64); err == nil {
					values = append(values, num)
				}
			}
		}
	}

	if len(values) == 0 {
		return 0.0, nil
	}

	// Sort values
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if values[i] > values[j] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}

	// Calculate median
	n := len(values)
	if n%2 == 0 {
		return (values[n/2-1] + values[n/2]) / 2.0, nil
	}
	return values[n/2], nil
}

// randomFunction returns a random integer in the range [start, end].
func randomFunction(params ...any) (any, error) {
	if len(params) < 2 {
		return 0, nil
	}

	var start, end int64

	// Parse start parameter
	switch val := params[0].(type) {
	case int:
		start = int64(val)
	case int64:
		start = val
	case float64:
		start = int64(val)
	case string:
		if num, err := strconv.ParseInt(val, 10, 64); err == nil {
			start = num
		}
	}

	// Parse end parameter
	switch val := params[1].(type) {
	case int:
		end = int64(val)
	case int64:
		end = val
	case float64:
		end = int64(val)
	case string:
		if num, err := strconv.ParseInt(val, 10, 64); err == nil {
			end = num
		}
	}

	if start > end {
		start, end = end, start
	}

	return start + rand.Int64N(end-start+1), nil
}

// randomfFunction returns a random float in the range [0.0, 1.0).
func randomfFunction(params ...any) (any, error) {
	return rand.Float64(), nil
}

// vminFunction is an Expr function to find the minimum value.
func vminFunction(params ...any) (any, error) {
	if len(params) == 0 {
		return 0.0, nil
	}

	var min float64
	var hasValue bool

	// Handle array parameter
	if arr, ok := params[0].([]any); ok {
		for _, v := range arr {
			switch val := v.(type) {
			case int:
				fval := float64(val)
				if !hasValue || fval < min {
					min = fval
					hasValue = true
				}
			case int64:
				fval := float64(val)
				if !hasValue || fval < min {
					min = fval
					hasValue = true
				}
			case float64:
				if !hasValue || val < min {
					min = val
					hasValue = true
				}
			case string:
				if num, err := strconv.ParseFloat(val, 64); err == nil {
					if !hasValue || num < min {
						min = num
						hasValue = true
					}
				}
			}
		}
		if !hasValue {
			return 0.0, nil
		}
		return min, nil
	}

	// Handle variadic parameters
	for _, v := range params {
		switch val := v.(type) {
		case int:
			fval := float64(val)
			if !hasValue || fval < min {
				min = fval
				hasValue = true
			}
		case int64:
			fval := float64(val)
			if !hasValue || fval < min {
				min = fval
				hasValue = true
			}
		case float64:
			if !hasValue || val < min {
				min = val
				hasValue = true
			}
		case string:
			if num, err := strconv.ParseFloat(val, 64); err == nil {
				if !hasValue || num < min {
					min = num
					hasValue = true
				}
			}
		}
	}
	if !hasValue {
		return 0.0, nil
	}
	return min, nil
}

// getBuiltinFunctions returns all built-in functions for expression evaluation.
// Uses sync.OnceValue to ensure expr.Function is only called once.
var getBuiltinFunctions = sync.OnceValue(func() []expr.Option {
	// Sort in dictionary order.
	return []expr.Option{
		// `abs` is a builtin.
		// `ceil` is a builtin.
		// `floor` is a builtin.
		// `int` is a builtin.
		// `mean` is a builtin.
		expr.Function(
			"random",
			randomFunction,
			new(func(int64, int64) int64),
		),
		expr.Function(
			"randomf",
			randomfFunction,
			new(func() float64),
		),
		// `round` is a builtin.
		expr.Function(
			"vmax",
			vmaxFunction,
			new(func([]any) float64),
			new(func(...any) float64),
		),
		expr.Function(
			"vmean",
			vmeanFunction,
			new(func([]any) float64),
			new(func(...any) float64),
		),
		expr.Function(
			"vmedian",
			vmedianFunction,
			new(func([]any) float64),
			new(func(...any) float64),
		),
		expr.Function(
			"vmin",
			vminFunction,
			new(func([]any) float64),
			new(func(...any) float64),
		),
		expr.Function(
			"vsum",
			vsumFunction,
			new(func([]any) float64),
			new(func(...any) float64),
		),
	}
})
