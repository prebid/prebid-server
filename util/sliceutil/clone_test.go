package sliceutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloneSlice(t *testing.T) {
	testCases := []struct {
		name  string
		given []int
	}{
		{
			name:  "nil",
			given: nil,
		},
		{
			name:  "empty",
			given: []int{},
		},
		{
			name:  "one",
			given: []int{1},
		},
		{
			name:  "many",
			given: []int{1, 2},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result := Clone(test.given)
			assert.Equal(t, test.given, result, "equality")
			assert.NotSame(t, test.given, result, "pointer")
		})
	}
}
