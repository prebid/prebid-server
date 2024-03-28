package reflectutil

import (
	"testing"

	"github.com/modern-go/reflect2"
	"github.com/stretchr/testify/assert"
)

func TestUnsafeSliceClone(t *testing.T) {
	testCases := []struct {
		name  string
		given []int
	}{
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
			original := test.given
			clonePtr := UnsafeSliceClone(reflect2.PtrOf(test.given), reflect2.TypeOf([]int{}).(*reflect2.UnsafeSliceType))
			clone := *(*[]int)(clonePtr)

			assert.NotSame(t, original, clone, "reference")
			assert.Equal(t, original, clone, "equality")
			assert.Equal(t, len(original), len(clone), "len")
			assert.Equal(t, cap(original), cap(clone), "cap")
		})
	}
}
