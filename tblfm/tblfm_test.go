package tblfm

import (
	"reflect"
	"testing"
)

func TestApply_EmptyFormula(t *testing.T) {
	input := [][]string{
		{"Item", "Price", "Qty", "Total"},
		{"Apple", "100", "5", ""},
		{"Orange", "150", "3", ""},
	}

	expected := [][]string{
		{"Item", "Price", "Qty", "Total"},
		{"Apple", "100", "5", ""},
		{"Orange", "150", "3", ""},
	}

	result, err := Apply(input, []string{})
	if err != nil {
		t.Fatalf("Apply() returned error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Apply() returned unexpected result\nGot:  %v\nWant: %v", result, expected)
	}
}

func TestApply_Arithmetic(t *testing.T) {
	tests := []struct {
		name     string
		input    [][]string
		formulas []string
		expected [][]string
	}{
		{
			name: "multiplication",
			input: [][]string{
				{"Item", "Price", "Qty", "Total"},
				{"Apple", "100", "5", ""},
				{"Orange", "150", "3", ""},
			},
			formulas: []string{"$4=$2*$3"},
			expected: [][]string{
				{"Item", "Price", "Qty", "Total"},
				{"Apple", "100", "5", "500"},
				{"Orange", "150", "3", "450"},
			},
		},
		{
			name: "addition",
			input: [][]string{
				{"Item", "A", "B", "Sum"},
				{"Row1", "10", "20", ""},
				{"Row2", "15", "25", ""},
			},
			formulas: []string{"$4=$2+$3"},
			expected: [][]string{
				{"Item", "A", "B", "Sum"},
				{"Row1", "10", "20", "30"},
				{"Row2", "15", "25", "40"},
			},
		},
		{
			name: "subtraction",
			input: [][]string{
				{"Item", "A", "B", "Diff"},
				{"Row1", "100", "30", ""},
				{"Row2", "50", "15", ""},
			},
			formulas: []string{"$4=$2-$3"},
			expected: [][]string{
				{"Item", "A", "B", "Diff"},
				{"Row1", "100", "30", "70"},
				{"Row2", "50", "15", "35"},
			},
		},
		{
			name: "division",
			input: [][]string{
				{"Item", "Total", "Count", "Average"},
				{"Row1", "100", "5", ""},
				{"Row2", "150", "3", ""},
			},
			formulas: []string{"$4=$2/$3"},
			expected: [][]string{
				{"Item", "Total", "Count", "Average"},
				{"Row1", "100", "5", "20"},
				{"Row2", "150", "3", "50"},
			},
		},
		{
			name: "multiple formulas",
			input: [][]string{
				{"Item", "Price", "Qty", "Total", "Tax", "Grand Total"},
				{"Apple", "100", "5", "", "", ""},
				{"Orange", "150", "3", "", "", ""},
			},
			formulas: []string{
				"$4=$2*$3", // Total = Price * Qty
				"$5=$4/10", // Tax = Total / 10 (10% tax)
				"$6=$4+$5", // Grand Total = Total + Tax
			},
			expected: [][]string{
				{"Item", "Price", "Qty", "Total", "Tax", "Grand Total"},
				{"Apple", "100", "5", "500", "50", "550"},
				{"Orange", "150", "3", "450", "45", "495"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Apply(tt.input, tt.formulas)
			if err != nil {
				t.Fatalf("Apply() returned error: %v", err)
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Apply() returned unexpected result\nGot:  %v\nWant: %v", result, tt.expected)
			}
		})
	}
}

func TestApply_SimpleMultiplication_NoHeader(t *testing.T) {
	input := [][]string{
		{"Apple", "100", "5", ""},
		{"Orange", "150", "3", ""},
	}

	expected := [][]string{
		{"Apple", "100", "5", "500"},
		{"Orange", "150", "3", "450"},
	}

	result, err := Apply(input, []string{"$4=$2*$3"}, WithHeader(false))
	if err != nil {
		t.Fatalf("Apply() returned error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Apply() returned unexpected result\nGot:  %v\nWant: %v", result, expected)
	}
}

func TestApply_RelativeColumnReferences(t *testing.T) {
	tests := []struct {
		name     string
		input    [][]string
		formulas []string
		expected [][]string
	}{
		{
			name: "straight column copy test - copy from column A to B",
			input: [][]string{
				{"a", "b", "c"},
				{"10", "", ""},
				{"20", "", ""},
				{"30", "", ""},
			},
			formulas: []string{"$2=$1"},
			expected: [][]string{
				{"a", "b", "c"},
				{"10", "10", ""},
				{"20", "20", ""},
				{"30", "30", ""},
			},
		},
		{
			name: "relative column copy test - copy from -1 position (one column left)",
			input: [][]string{
				{"a", "b", "c"},
				{"10", "", ""},
				{"20", "", ""},
				{"30", "", ""},
			},
			formulas: []string{"$2=$-1"},
			expected: [][]string{
				{"a", "b", "c"},
				{"10", "10", ""},
				{"20", "20", ""},
				{"30", "30", ""},
			},
		},
		{
			name: "relative reference with arithmetic - add 5 to previous column",
			input: [][]string{
				{"a", "b"},
				{"10", ""},
				{"20", ""},
			},
			formulas: []string{"$2=$-1+5"},
			expected: [][]string{
				{"a", "b"},
				{"10", "15"},
				{"20", "25"},
			},
		},
		{
			name: "relative reference in middle column - copy from -2 to current",
			input: [][]string{
				{"a", "b", "c", "d"},
				{"10", "20", "", ""},
				{"30", "40", "", ""},
			},
			formulas: []string{"$3=$-2"},
			expected: [][]string{
				{"a", "b", "c", "d"},
				{"10", "20", "10", ""},
				{"30", "40", "30", ""},
			},
		},
		{
			name: "multiply previous two columns - relative reference arithmetic",
			input: [][]string{
				{"a", "b", "result"},
				{"5", "3", ""},
				{"10", "2", ""},
			},
			formulas: []string{"$3=$-2*$-1"},
			expected: [][]string{
				{"a", "b", "result"},
				{"5", "3", "15"},
				{"10", "2", "20"},
			},
		},
		{
			name: "addition - from org test: multiple rows",
			input: [][]string{
				{"s1", "s2", "desc", "result"},
				{"1", "2", "1+2", ""},
				{"2", "1", "2+1", ""},
			},
			formulas: []string{
				"$4=$1+$2",
			},
			expected: [][]string{
				{"s1", "s2", "desc", "result"},
				{"1", "2", "1+2", "3"},
				{"2", "1", "2+1", "3"},
			},
		},
		{
			name: "subtraction - from org test: a-b",
			input: [][]string{
				{"s1", "s2", "desc", "result"},
				{"2", "1", "a-b", ""},
				{"1", "2", "a-b", ""},
			},
			formulas: []string{
				"$4=$1-$2",
			},
			expected: [][]string{
				{"s1", "s2", "desc", "result"},
				{"2", "1", "a-b", "1"},
				{"1", "2", "a-b", "-1"},
			},
		},
		{
			name: "mixed absolute and relative references",
			input: [][]string{
				{"base", "multiplier", "result"},
				{"10", "2", ""},
				{"20", "3", ""},
			},
			formulas: []string{"$3=$1*$-1"},
			expected: [][]string{
				{"base", "multiplier", "result"},
				{"10", "2", "20"},
				{"20", "3", "60"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Apply(tt.input, tt.formulas)
			if err != nil {
				t.Fatalf("Apply() returned error: %v", err)
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Apply() returned unexpected result\nGot:  %v\nWant: %v", result, tt.expected)
			}
		})
	}
}
