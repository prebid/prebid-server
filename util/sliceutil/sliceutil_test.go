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
