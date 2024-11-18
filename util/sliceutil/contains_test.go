package sliceutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testDatapoint struct {
	id    int
	value string
}

type testCase[T comparable] struct {
	description string
	givenSlice  []T
	givenValue  T
	expected    bool
}

func TestContains(t *testing.T) {
	stringTestCases := []testCase[string]{
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
			expected:    false,
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
			expected:    false,
		},
		{
			description: "Many - No Match",
			givenSlice:  []string{"a", "b"},
			givenValue:  "z",
			expected:    false,
		},
	}

	intTestCases := []testCase[int]{
		{
			description: "Int - Nil",
			givenSlice:  nil,
			givenValue:  1,
			expected:    false,
		},
		{
			description: "Int - Empty",
			givenSlice:  []int{},
			givenValue:  1,
			expected:    false,
		},
		{
			description: "Int - One - Match",
			givenSlice:  []int{1},
			givenValue:  1,
			expected:    true,
		},
		{
			description: "Int - One - No Match",
			givenSlice:  []int{1},
			givenValue:  2,
			expected:    false,
		},
		{
			description: "Int - Many - Match",
			givenSlice:  []int{1, 2},
			givenValue:  2,
			expected:    true,
		},

		{
			description: "Int - Many - No Match",
			givenSlice:  []int{1, 2},
			givenValue:  3,
			expected:    false,
		},
	}

	testDatapointTestCases := []testCase[testDatapoint]{
		{
			description: "Struct - Nil",
			givenSlice:  nil,
			givenValue:  testDatapoint{id: 1, value: "a"},
			expected:    false,
		},
		{
			description: "Struct - Empty",
			givenSlice:  []testDatapoint{},
			givenValue:  testDatapoint{id: 1, value: "a"},
			expected:    false,
		},
		{
			description: "Struct - One - Match",
			givenSlice:  []testDatapoint{{id: 1, value: "a"}},
			givenValue:  testDatapoint{id: 1, value: "a"},
			expected:    true,
		},
		{
			description: "Struct - One - No Match",
			givenSlice:  []testDatapoint{{id: 1, value: "a"}},
			givenValue:  testDatapoint{id: 2, value: "a"},
			expected:    false,
		},
		{
			description: "Struct - Many - Match",
			givenSlice:  []testDatapoint{{id: 1, value: "a"}, {id: 2, value: "b"}},
			givenValue:  testDatapoint{id: 2, value: "b"},
			expected:    true,
		},
		{
			description: "Struct - Many - No Match - Different ID",
			givenSlice:  []testDatapoint{{id: 1, value: "a"}, {id: 2, value: "b"}},
			givenValue:  testDatapoint{id: 3, value: "b"},
			expected:    false,
		},
		{
			description: "Struct - Many - No Match - Different Value",
			givenSlice:  []testDatapoint{{id: 1, value: "a"}, {id: 2, value: "b"}},
			givenValue:  testDatapoint{id: 2, value: "c"},
			expected:    false,
		},
	}

	for _, test := range stringTestCases {
		result := Contains(test.givenSlice, test.givenValue)
		assert.Equal(t, test.expected, result, test.description)
	}

	for _, test := range intTestCases {
		result := Contains(test.givenSlice, test.givenValue)
		assert.Equal(t, test.expected, result, test.description)
	}

	for _, test := range testDatapointTestCases {
		result := Contains(test.givenSlice, test.givenValue)
		assert.Equal(t, test.expected, result, test.description)
	}
}
