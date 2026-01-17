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

func TestApply_RowAndColumnCopy(t *testing.T) {
	tests := []struct {
		name     string
		input    [][]string
		formulas []string
		expected [][]string
	}{
		{
			name: "column copy - $5=$4 (copy entire column)",
			input: [][]string{
				{"a", "b", "c", "d", "e"},
				{"1", "2", "3", "4", ""},
				{"5", "6", "7", "8", ""},
				{"9", "10", "11", "12", ""},
			},
			formulas: []string{"$5=$4"},
			expected: [][]string{
				{"a", "b", "c", "d", "e"},
				{"1", "2", "3", "4", "4"},
				{"5", "6", "7", "8", "8"},
				{"9", "10", "11", "12", "12"},
			},
		},
		{
			name: "row copy - @3=@<< (copy second row to row 3)",
			input: [][]string{
				{"a", "b", "c", "d", "e"},
				{"1", "2", "3", "4", "5"},
				{"", "", "", "", ""},
			},
			formulas: []string{"@3=@<<"},
			expected: [][]string{
				{"a", "b", "c", "d", "e"},
				{"1", "2", "3", "4", "5"},
				{"1", "2", "3", "4", "5"},
			},
		},
		{
			name: "row copy - @4=@>> (copy second to last row to row 4)",
			input: [][]string{
				{"a", "b", "c", "d", "e"},
				{"1", "2", "3", "4", "5"},
				{"6", "7", "8", "9", "10"},
				{"", "", "", "", ""},
			},
			formulas: []string{"@4=@>>"},
			expected: [][]string{
				{"a", "b", "c", "d", "e"},
				{"1", "2", "3", "4", "5"},
				{"6", "7", "8", "9", "10"},
				{"6", "7", "8", "9", "10"},
			},
		},
		{
			name: "row copy with relative reference - @3=@-1",
			input: [][]string{
				{"a", "b", "c", "d", "e"},
				{"1", "2", "3", "4", "5"},
				{"", "", "", "", ""},
			},
			formulas: []string{"@3=@-1"},
			expected: [][]string{
				{"a", "b", "c", "d", "e"},
				{"1", "2", "3", "4", "5"},
				{"1", "2", "3", "4", "5"},
			},
		},
		{
			name: "column copy with relative reference - $2=$-1",
			input: [][]string{
				{"a", "b", "c"},
				{"1", "", ""},
				{"2", "", ""},
				{"3", "", ""},
			},
			formulas: []string{"$2=$-1"},
			expected: [][]string{
				{"a", "b", "c"},
				{"1", "1", ""},
				{"2", "2", ""},
				{"3", "3", ""},
			},
		},
		{
			name: "row copy - @4=@<< (copy second row to row 4)",
			input: [][]string{
				{"a", "b", "c", "d", "e"},
				{"1", "2", "3", "4", "5"},
				{"6", "7", "8", "9", "10"},
				{"", "", "", "", ""},
			},
			formulas: []string{"@4=@<<"},
			expected: [][]string{
				{"a", "b", "c", "d", "e"},
				{"1", "2", "3", "4", "5"},
				{"6", "7", "8", "9", "10"},
				{"1", "2", "3", "4", "5"},
			},
		},
		{
			name: "row copy - @5=@<<< (copy third row to row 5)",
			input: [][]string{
				{"a", "b", "c", "d", "e"},
				{"1", "2", "3", "4", "5"},
				{"6", "7", "8", "9", "10"},
				{"11", "12", "13", "14", "15"},
				{"", "", "", "", ""},
			},
			formulas: []string{"@5=@<<<"},
			expected: [][]string{
				{"a", "b", "c", "d", "e"},
				{"1", "2", "3", "4", "5"},
				{"6", "7", "8", "9", "10"},
				{"11", "12", "13", "14", "15"},
				{"6", "7", "8", "9", "10"},
			},
		},
		{
			name: "row copy - @2=@>> (copy second to last row to row 2)",
			input: [][]string{
				{"a", "b", "c", "d", "e"},
				{"", "", "", "", ""},
				{"6", "7", "8", "9", "10"},
				{"11", "12", "13", "14", "15"},
			},
			formulas: []string{"@2=@>>"},
			expected: [][]string{
				{"a", "b", "c", "d", "e"},
				{"6", "7", "8", "9", "10"},
				{"6", "7", "8", "9", "10"},
				{"11", "12", "13", "14", "15"},
			},
		},
		{
			name: "row copy - @2=@>>> (copy third to last row to row 2)",
			input: [][]string{
				{"a", "b", "c", "d", "e"},
				{"", "", "", "", ""},
				{"6", "7", "8", "9", "10"},
				{"11", "12", "13", "14", "15"},
				{"16", "17", "18", "19", "20"},
			},
			formulas: []string{"@2=@>>>"},
			expected: [][]string{
				{"a", "b", "c", "d", "e"},
				{"6", "7", "8", "9", "10"},
				{"6", "7", "8", "9", "10"},
				{"11", "12", "13", "14", "15"},
				{"16", "17", "18", "19", "20"},
			},
		},
		{
			name: "column copy - $5=$<< (copy second column to column 5)",
			input: [][]string{
				{"a", "b", "c", "d", "e"},
				{"1", "2", "3", "4", ""},
				{"6", "7", "8", "9", ""},
			},
			formulas: []string{"$5=$<<"},
			expected: [][]string{
				{"a", "b", "c", "d", "e"},
				{"1", "2", "3", "4", "2"},
				{"6", "7", "8", "9", "7"},
			},
		},
		{
			name: "column copy - $2=$<<< (copy third column to column 2)",
			input: [][]string{
				{"a", "b", "c", "d", "e"},
				{"1", "", "3", "4", "5"},
				{"6", "", "8", "9", "10"},
			},
			formulas: []string{"$2=$<<<"},
			expected: [][]string{
				{"a", "b", "c", "d", "e"},
				{"1", "3", "3", "4", "5"},
				{"6", "8", "8", "9", "10"},
			},
		},
		{
			name: "column copy - $3=$>> (copy second to last column to column 3)",
			input: [][]string{
				{"a", "b", "c", "d", "e"},
				{"1", "2", "", "4", "5"},
				{"6", "7", "", "9", "10"},
			},
			formulas: []string{"$3=$>>"},
			expected: [][]string{
				{"a", "b", "c", "d", "e"},
				{"1", "2", "4", "4", "5"},
				{"6", "7", "9", "9", "10"},
			},
		},
		{
			name: "column copy - $1=$>>> (copy third to last column to column 1)",
			input: [][]string{
				{"a", "b", "c", "d", "e"},
				{"", "2", "3", "4", "5"},
				{"", "7", "8", "9", "10"},
			},
			formulas: []string{"$1=$>>>"},
			expected: [][]string{
				{"a", "b", "c", "d", "e"},
				{"3", "2", "3", "4", "5"},
				{"8", "7", "8", "9", "10"},
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
				t.Errorf("Apply() returned unexpected result")
				for i := range result {
					t.Errorf("Row %d: Got: %v, Want: %v", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestApply_Range(t *testing.T) {
	tests := []struct {
		name     string
		input    [][]string
		formulas []string
		expected [][]string
		skip     bool
	}{
		{
			name: "copy to range",
			skip: true, // TODO: String literal handling needs to be clarified per TBLFM spec
			input: [][]string{
				{"Item", "Price", "Qty", "Total"},
				{"Apple", "100", "5", ""},
				{"Orange", "150", "3", ""},
				{"Total", "", "", ""},
			},
			formulas: []string{
				"@2$>..@>>$>=@1$>",
			},
			expected: [][]string{
				{"Item", "Price", "Qty", "Total"},
				{"Apple", "100", "5", "Total"},
				{"Orange", "150", "3", "Total"},
				{"Total", "", "", ""},
			},
		},
		{
			name: "multiplication in range",
			input: [][]string{
				{"Item", "Price", "Qty", "Total"},
				{"Apple", "100", "5", ""},
				{"Orange", "150", "3", ""},
				{"Total", "", "", ""},
			},
			formulas: []string{
				"@2$>..@>>$>=$2*$3",
			},
			expected: [][]string{
				{"Item", "Price", "Qty", "Total"},
				{"Apple", "100", "5", "500"},
				{"Orange", "150", "3", "450"},
				{"Total", "", "", ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip("Skipping test - needs clarification per TBLFM spec")
			}
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

func TestApply_RangeFunction(t *testing.T) {
	tests := []struct {
		name     string
		input    [][]string
		formulas []string
		expected [][]string
	}{
		{
			name: "range summary",
			input: [][]string{
				{"Item", "Price", "Qty", "Total"},
				{"Apple", "100", "5", ""},
				{"Orange", "150", "3", ""},
				{"Total", "", "", ""},
			},
			formulas: []string{
				"@2$>..@>>$>=$2*$3",
				"@>$>=vsum(@<..@>>)",
			},
			expected: [][]string{
				{"Item", "Price", "Qty", "Total"},
				{"Apple", "100", "5", "500"},
				{"Orange", "150", "3", "450"},
				{"Total", "", "", "950"},
			},
		},
		{
			name: "range summary with column specification",
			input: [][]string{
				{"Item", "Price", "Qty", "Total"},
				{"Apple", "100", "5", ""},
				{"Orange", "150", "3", ""},
				{"Total", "", "", ""},
			},
			formulas: []string{
				"@2$>..@>>$>=$2*$3",
				"@>$>=vsum(@<$>..@>>$>)",
			},
			expected: [][]string{
				{"Item", "Price", "Qty", "Total"},
				{"Apple", "100", "5", "500"},
				{"Orange", "150", "3", "450"},
				{"Total", "", "", "950"},
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

func TestApply_BuiltinFunctions(t *testing.T) {
	tests := []struct {
		name     string
		input    [][]string
		formulas []string
		expected [][]string
	}{
		{
			name: "vsum with column range",
			input: [][]string{
				{"Item", "Value"},
				{"A", "10"},
				{"B", "20"},
				{"C", "30"},
				{"Total", ""},
			},
			formulas: []string{
				"@>$>=vsum(@<$>..@>>$>)",
			},
			expected: [][]string{
				{"Item", "Value"},
				{"A", "10"},
				{"B", "20"},
				{"C", "30"},
				{"Total", "60"},
			},
		},
		{
			name: "vsum with multiple columns",
			input: [][]string{
				{"Item", "Price", "Qty", "Total"},
				{"Apple", "100", "5", ""},
				{"Orange", "150", "3", ""},
				{"Total", "", "", ""},
			},
			formulas: []string{
				"@2$4..@>>$4=$2*$3",
				"@>$4=vsum(@<$4..@>>$4)",
			},
			expected: [][]string{
				{"Item", "Price", "Qty", "Total"},
				{"Apple", "100", "5", "500"},
				{"Orange", "150", "3", "450"},
				{"Total", "", "", "950"},
			},
		},
		{
			name: "vsum with row-only range (implicit column)",
			input: [][]string{
				{"val"},
				{"10"},
				{"20"},
				{"30"},
				{""},
			},
			formulas: []string{
				"@>$1=vsum(@<..@-1)",
			},
			expected: [][]string{
				{"val"},
				{"10"},
				{"20"},
				{"30"},
				{"60"},
			},
		},
		{
			name: "vsum with row-only range (less than -1)",
			input: [][]string{
				{"val"},
				{"10"},
				{"20"},
				{""},
				{""},
			},
			formulas: []string{
				"@>$1=vsum(@<..@-2)",
			},
			expected: [][]string{
				{"val"},
				{"10"},
				{"20"},
				{""},
				{"30"},
			},
		},
		{
			name: "vsum with row-only range (multiple columns with implicit column)",
			input: [][]string{
				{"A", "B", "C"},
				{"1", "2", "3"},
				{"4", "5", "6"},
				{"7", "8", "9"},
				{"", "", ""},
			},
			formulas: []string{
				"@>$1=vsum(@<..@-1)",
				"@>$2=vsum(@<..@-1)",
				"@>$3=vsum(@<..@-1)",
			},
			expected: [][]string{
				{"A", "B", "C"},
				{"1", "2", "3"},
				{"4", "5", "6"},
				{"7", "8", "9"},
				{"12", "15", "18"},
			},
		},
		{
			name: "vsum with single column reference",
			input: [][]string{
				{"A", "B", "C"},
				{"1", "2", "3"},
				{"4", "5", "6"},
				{"7", "8", "9"},
				{"", "", ""},
			},
			formulas: []string{
				"@>$1=vsum(@<$1..@>>$1)",
				"@>$2=vsum(@<$2..@>>$2)",
				"@>$3=vsum(@<$3..@>>$3)",
			},
			expected: [][]string{
				{"A", "B", "C"},
				{"1", "2", "3"},
				{"4", "5", "6"},
				{"7", "8", "9"},
				{"12", "15", "18"},
			},
		},
		{
			name: "vsum with decimal values",
			input: [][]string{
				{"Item", "Amount"},
				{"A", "10.5"},
				{"B", "20.75"},
				{"C", "30.25"},
				{"Total", ""},
			},
			formulas: []string{
				"@>$>=vsum(@<$>..@>>$>)",
			},
			expected: [][]string{
				{"Item", "Amount"},
				{"A", "10.5"},
				{"B", "20.75"},
				{"C", "30.25"},
				{"Total", "61.5"},
			},
		},
		{
			name: "vsum with empty cells",
			input: [][]string{
				{"Item", "Value"},
				{"A", "10"},
				{"B", ""},
				{"C", "30"},
				{"Total", ""},
			},
			formulas: []string{
				"@>$>=vsum(@<$>..@>>$>)",
			},
			expected: [][]string{
				{"Item", "Value"},
				{"A", "10"},
				{"B", ""},
				{"C", "30"},
				{"Total", "40"},
			},
		},
		{
			name: "vsum in complex formula",
			input: [][]string{
				{"Item", "Q1", "Q2", "Q3", "Q4", "Total", "Average"},
				{"Product A", "100", "150", "200", "250", "", ""},
			},
			formulas: []string{
				"@2$6=vsum(@2$2..@2$5)",
				"@2$7=vsum(@2$2..@2$5)/4",
			},
			expected: [][]string{
				{"Item", "Q1", "Q2", "Q3", "Q4", "Total", "Average"},
				{"Product A", "100", "150", "200", "250", "700", "175"},
			},
		},
		{
			name: "real-world invoice example with compact notation",
			input: [][]string{
				{"Item", "UnitPrice", "Quantity", "Total"},
				{"Apple", "2.5", "12", ""},
				{"Banana", "2.0", "5", ""},
				{"Orange", "1.2", "8", ""},
				{"Total", "", "", ""},
			},
			formulas: []string{
				"@<$>..@>>=$2*$3",
				"@>$>=vsum(@<..@>>)",
			},
			expected: [][]string{
				{"Item", "UnitPrice", "Quantity", "Total"},
				{"Apple", "2.5", "12", "30"},
				{"Banana", "2.0", "5", "10"},
				{"Orange", "1.2", "8", "9.6"},
				{"Total", "", "", "49.6"},
			},
		},
		{
			name: "vmean with column range",
			input: [][]string{
				{"Item", "Value", "Average"},
				{"A", "10", ""},
				{"B", "20", ""},
				{"C", "30", ""},
			},
			formulas: []string{
				"@<$>..@>$>=vmean(@<$2..@>$2)",
			},
			expected: [][]string{
				{"Item", "Value", "Average"},
				{"A", "10", "20"},
				{"B", "20", "20"},
				{"C", "30", "20"},
			},
		},
		{
			name: "vmean with row-only range",
			input: [][]string{
				{"scores"},
				{"85"},
				{"90"},
				{"78"},
				{"92"},
				{""},
			},
			formulas: []string{
				"@>$1=vmean(@<..@-1)",
			},
			expected: [][]string{
				{"scores"},
				{"85"},
				{"90"},
				{"78"},
				{"92"},
				{"86.25"},
			},
		},
		{
			name: "vmean with empty cells",
			input: [][]string{
				{"Value", "Avg"},
				{"10", ""},
				{"", ""},
				{"30", ""},
			},
			formulas: []string{
				"@<$>..@>$>=vmean(@<$1..@>$1)",
			},
			expected: [][]string{
				{"Value", "Avg"},
				{"10", "20"},
				{"", "20"},
				{"30", "20"},
			},
		},
		{
			name: "vmax with multiple values",
			input: [][]string{
				{"Value", "Max"},
				{"15", ""},
				{"42", ""},
				{"8", ""},
				{"33", ""},
			},
			formulas: []string{
				"@<$>..@>$>=vmax(@<$1..@>$1)",
			},
			expected: [][]string{
				{"Value", "Max"},
				{"15", "42"},
				{"42", "42"},
				{"8", "42"},
				{"33", "42"},
			},
		},
		{
			name: "vmin with multiple values",
			input: [][]string{
				{"Value", "Min"},
				{"15", ""},
				{"42", ""},
				{"8", ""},
				{"33", ""},
			},
			formulas: []string{
				"@<$>..@>$>=vmin(@<$1..@>$1)",
			},
			expected: [][]string{
				{"Value", "Min"},
				{"15", "8"},
				{"42", "8"},
				{"8", "8"},
				{"33", "8"},
			},
		},
		{
			name: "vmedian with odd number of values",
			input: [][]string{
				{"Value", "Median"},
				{"10", ""},
				{"20", ""},
				{"30", ""},
			},
			formulas: []string{
				"@<$>..@>$>=vmedian(@<$1..@>$1)",
			},
			expected: [][]string{
				{"Value", "Median"},
				{"10", "20"},
				{"20", "20"},
				{"30", "20"},
			},
		},
		{
			name: "vmedian with even number of values",
			input: [][]string{
				{"Value", "Median"},
				{"10", ""},
				{"20", ""},
				{"30", ""},
				{"40", ""},
			},
			formulas: []string{
				"@<$>..@>$>=vmedian(@<$1..@>$1)",
			},
			expected: [][]string{
				{"Value", "Median"},
				{"10", "25"},
				{"20", "25"},
				{"30", "25"},
				{"40", "25"},
			},
		},
		{
			name: "statistics summary row",
			input: [][]string{
				{"Value", "Sum", "Mean", "Median", "Min", "Max"},
				{"15", "", "", "", "", ""},
				{"42", "", "", "", "", ""},
				{"8", "", "", "", "", ""},
				{"33", "", "", "", "", ""},
				{"", "", "", "", "", ""},
			},
			formulas: []string{
				"@>$2=vsum(@<$1..@>>$1)",
				"@>$3=vmean(@<$1..@>>$1)",
				"@>$4=vmedian(@<$1..@>>$1)",
				"@>$5=vmin(@<$1..@>>$1)",
				"@>$6=vmax(@<$1..@>>$1)",
			},
			expected: [][]string{
				{"Value", "Sum", "Mean", "Median", "Min", "Max"},
				{"15", "", "", "", "", ""},
				{"42", "", "", "", "", ""},
				{"8", "", "", "", "", ""},
				{"33", "", "", "", "", ""},
				{"", "98", "24.5", "24", "8", "42"},
			},
		},
		{
			name: "exp function with integer",
			input: [][]string{
				{"Value", "Result"},
				{"1", ""},
				{"2", ""},
			},
			formulas: []string{
				"@2$2=exp($1)",
				"@3$2=exp($1)",
			},
			expected: [][]string{
				{"Value", "Result"},
				{"1", "2.718281828459045"},
				{"2", "7.38905609893065"},
			},
		},
		{
			name: "pi constant",
			input: [][]string{
				{"Constant", "Value"},
				{"pi", ""},
			},
			formulas: []string{
				"@2$2=pi",
			},
			expected: [][]string{
				{"Constant", "Value"},
				{"pi", "3.141592653589793"},
			},
		},
		{
			name: "exp and pi together",
			input: [][]string{
				{"Expression", "Result"},
				{"exp(pi)", ""},
			},
			formulas: []string{
				"@2$2=exp(pi)",
			},
			expected: [][]string{
				{"Expression", "Result"},
				{"exp(pi)", "23.140692632779267"},
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
				t.Errorf("Apply() returned unexpected result")
				for i := range result {
					if i < len(tt.expected) && !reflect.DeepEqual(result[i], tt.expected[i]) {
						t.Errorf("Row %d: Got: %v, Want: %v", i, result[i], tt.expected[i])
					}
				}
			}
		})
	}
}
