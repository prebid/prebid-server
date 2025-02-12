package stringutil

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStrToInt8Slice(t *testing.T) {
	type testOutput struct {
		arr []int8
		err error
	}
	tests := []struct {
		desc     string
		in       string
		expected testOutput
	}{
		{
			desc: "empty string, expect nil output array",
			in:   "",
			expected: testOutput{
				arr: nil,
				err: nil,
			},
		},
		{
			desc: "string doesn't contain digits, expect nil output array",
			in:   "malformed",
			expected: testOutput{
				arr: nil,
				err: &strconv.NumError{Func: "ParseInt", Num: "malformed", Err: strconv.ErrSyntax},
			},
		},
		{
			desc: "string contains int8 digits and non-digits, expect array with a single int8 element",
			in:   "malformed,2,malformed",
			expected: testOutput{
				arr: nil,
				err: &strconv.NumError{Func: "ParseInt", Num: "malformed", Err: strconv.ErrSyntax},
			},
		},
		{
			desc: "string comes with single digit too big to fit into a signed int8, expect nil output array",
			in:   "128",
			expected: testOutput{
				arr: nil,
				err: &strconv.NumError{Func: "ParseInt", Num: "128", Err: strconv.ErrRange},
			},
		},
		{
			desc: "string comes with single digit that fits into 8 bits, expect array with a single int8 element",
			in:   "127",
			expected: testOutput{
				arr: []int8{int8(127)},
				err: nil,
			},
		},
		{
			desc: "string comes with multiple, comma-separated numbers that fit into 8 bits, expect array with int8 elements",
			in:   "127,2,-127",
			expected: testOutput{
				arr: []int8{int8(127), int8(2), int8(-127)},
				err: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			outArr, outErr := StrToInt8Slice(tt.in)
			assert.Equal(t, tt.expected.arr, outArr, tt.desc)
			assert.Equal(t, tt.expected.err, outErr, tt.desc)
		})
	}
}
