package iterutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type Simple[T any] struct {
	Value T
}

func TestSlicePointers(t *testing.T) {
	s := []Simple[int]{{1}, {2}, {3}}
	for i, v := range SlicePointers(s) {
		v.Value = 99999
		assert.EqualValues(t, 99999, s[i].Value)
	}
	for _, v := range s {
		assert.EqualValues(t, 99999, v.Value)
	}
}

func TestSlicePointerValues(t *testing.T) {
	s := []Simple[int]{{1}, {2}, {3}}
	for v := range SlicePointerValues(s) {
		v.Value = 99999
	}
	for _, v := range s {
		assert.EqualValues(t, 99999, v.Value)
	}
}
