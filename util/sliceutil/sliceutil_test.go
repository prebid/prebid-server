package sliceutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainsStringIgnoreCase(t *testing.T) {
	testCases := []struct {
		description string
		givenSlice  []string
		givenValue  string
		expected    bool
	}{
		{
			description: "Nil",
			givenSlice:  nil,
			givenValue:  "a",
			expected:    false,
		},
		{
			description: "Empty",
			givenSlice:  []string{},
			givenValue:  "a",
			expected:    false,
		},
		{
			description: "One - Match - Same Case",
			givenSlice:  []string{"a"},
			givenValue:  "a",
			expected:    true,
		},
		{
			description: "One - Match - Different Case",
			givenSlice:  []string{"a"},
			givenValue:  "A",
			expected:    true,
		},
		{
			description: "One - No Match",
			givenSlice:  []string{"a"},
			givenValue:  "z",
			expected:    false,
		},
		{
			description: "Many - Match - Same Case",
			givenSlice:  []string{"a", "b"},
			givenValue:  "b",
			expected:    true,
		},
		{
			description: "Many - Match - Different Case",
			givenSlice:  []string{"a", "b"},
			givenValue:  "B",
			expected:    true,
		},
		{
			description: "Many - No Match",
			givenSlice:  []string{"a", "b"},
			givenValue:  "z",
			expected:    false,
		},
	}

	for _, test := range testCases {
		result := ContainsStringIgnoreCase(test.givenSlice, test.givenValue)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestCloneSlice(t *testing.T) {
	testCases := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "NilSlice", // Test we handle nils properly
			test: func(t *testing.T) {
				var testSlice, copySlice []int = nil, nil // copySlice is a manual copy of testSLice
				clone := CloneSlice(testSlice)
				testSlice = []int{5, 7}
				assert.Equal(t, copySlice, clone)
			},
		},
		{
			name: "String", // Test a simple string map
			test: func(t *testing.T) {
				var testSlice, copySlice []string = []string{"foo", "bar", "first", "one"}, []string{"foo", "bar", "first", "one"}
				clone := CloneSlice(testSlice)
				testSlice[1] = "baz"
				testSlice = append(testSlice, "the clown")
				assert.Equal(t, copySlice, clone)
			},
		},
		{
			name: "Int", // Test a simple map[string]int
			test: func(t *testing.T) {
				var testSlice, copySlice []int = []int{2, 4, 5, 7}, []int{2, 4, 5, 7}
				clone := CloneSlice(testSlice)
				testSlice[2] = 7
				testSlice = append(testSlice, 13)
				assert.Equal(t, copySlice, clone)
			},
		},
	}
	for _, test := range testCases {
		t.Run(test.name, test.test)
	}
}
