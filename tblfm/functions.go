package tblfm

import (
	"math"
	"sort"
	"strconv"

	lua "github.com/yuin/gopher-lua"
)

// registerBuiltinFunctions registers all built-in functions for Lua expression evaluation.
func registerBuiltinFunctions(L *lua.LState) {
	L.SetGlobal("vsum", L.NewFunction(vsumFunction))
	L.SetGlobal("vmean", L.NewFunction(vmeanFunction))
	L.SetGlobal("vmax", L.NewFunction(vmaxFunction))
	L.SetGlobal("vmin", L.NewFunction(vminFunction))
	L.SetGlobal("vmedian", L.NewFunction(vmedianFunction))
	L.SetGlobal("exp", L.NewFunction(expFunction))
}

// vsumFunction is a Lua function to calculate the sum of values.
func vsumFunction(L *lua.LState) int {
	sum := 0.0
	processTable(L, func(v float64) {
		sum += v
	})
	L.Push(lua.LNumber(sum))
	return 1
}

// vmeanFunction is a Lua function to calculate the mean of values.
func vmeanFunction(L *lua.LState) int {
	var sum float64
	var count int
	processTable(L, func(v float64) {
		sum += v
		count++
	})
	if count == 0 {
		L.Push(lua.LNumber(0))
		return 1
	}
	L.Push(lua.LNumber(sum / float64(count)))
	return 1
}

// vmaxFunction is a Lua function to find the maximum value.
func vmaxFunction(L *lua.LState) int {
	var max float64
	var hasValue bool
	processTable(L, func(v float64) {
		if !hasValue || v > max {
			max = v
			hasValue = true
		}
	})
	if !hasValue {
		L.Push(lua.LNumber(0))
		return 1
	}
	L.Push(lua.LNumber(max))
	return 1
}

// vminFunction is a Lua function to find the minimum value.
func vminFunction(L *lua.LState) int {
	var min float64
	var hasValue bool
	processTable(L, func(v float64) {
		if !hasValue || v < min {
			min = v
			hasValue = true
		}
	})
	if !hasValue {
		L.Push(lua.LNumber(0))
		return 1
	}
	L.Push(lua.LNumber(min))
	return 1
}

// vmedianFunction is a Lua function to calculate the median of values.
func vmedianFunction(L *lua.LState) int {
	var values []float64
	processTable(L, func(v float64) {
		values = append(values, v)
	})

	if len(values) == 0 {
		L.Push(lua.LNumber(0))
		return 1
	}

	sort.Float64s(values)

	n := len(values)
	if n%2 == 0 {
		L.Push(lua.LNumber((values[n/2-1] + values[n/2]) / 2.0))
	} else {
		L.Push(lua.LNumber(values[n/2]))
	}
	return 1
}

// expFunction is a Lua function for math.Exp.
func expFunction(L *lua.LState) int {
	val := L.ToNumber(1)
	L.Push(lua.LNumber(math.Exp(float64(val))))
	return 1
}

// processTable is a helper to extract numbers from a Lua table at arg 1.
func processTable(L *lua.LState, processor func(float64)) {
	tbl := L.ToTable(1)
	if tbl == nil {
		return
	}

	tbl.ForEach(func(_, val lua.LValue) {
		var f float64
		var err error
		switch v := val.(type) {
		case lua.LNumber:
			f = float64(v)
		case lua.LString:
			f, err = strconv.ParseFloat(string(v), 64)
			if err != nil {
				return // Skip non-numeric strings
			}
		default:
			return // Skip other types
		}
		processor(f)
	})
}
