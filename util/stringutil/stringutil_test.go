package stringutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStrToInt8Slice(t *testing.T) {
	tests := []struct {
		desc     string
		in       string
		expected []int8
	}{
		{
			desc:     "empty string, expect nil output array",
			in:       "",
			expected: nil,
		},
		{
			desc:     "string doesn't contain digits, expect nil output array",
			in:       "a,b,#$%",
			expected: nil,
		},
		{
			desc:     "string contains int8 digits and non-digits, expect array with a single int8 element",
			in:       "a,2,#$%",
			expected: []int8{int8(2)},
		},
		{
			desc:     "string comes with single digit too big to fit into a signed int8, expect nil output array",
			in:       "128",
			expected: nil,
		},
		{
			desc:     "string comes with single digit that fits into 8 bits, expect array with a single int8 element",
			in:       "127",
			expected: []int8{int8(127)},
		},
		{
			desc:     "string comes with multiple, comma-separated numbers that fit into 8 bits, expect array with int8 elements",
			in:       "127,2,-127",
			expected: []int8{int8(127), int8(2), int8(-127)},
		},
	}

	for _, tt := range tests {
		out := StrToInt8Slice(tt.in)

		assert.Equal(t, tt.expected, out)
	}
}
