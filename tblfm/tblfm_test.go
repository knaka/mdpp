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
