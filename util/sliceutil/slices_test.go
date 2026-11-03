package sliceutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type Simple[T any] struct {
	Value T
}

func TestIndexPointerFunc(t *testing.T) {
	s := []Simple[int]{{1}, {2}, {3}}
	assert.Equal(t, 1, IndexPointerFunc(s, func(v *Simple[int]) bool { return v.Value == 2 }))
	assert.Equal(t, -1, IndexPointerFunc(s, func(v *Simple[int]) bool { return v.Value == 4 }))
}

func TestDeletePointerFunc(t *testing.T) {
	s := []Simple[int]{{1}, {2}, {3}, {4}, {5}}
	s = DeletePointerFunc(s, func(v *Simple[int]) bool { return v.Value%2 == 0 })
	assert.Equal(t, []Simple[int]{{1}, {3}, {5}}, s)

	s = DeletePointerFunc(s, func(v *Simple[int]) bool { return v.Value > 10 })
	assert.Equal(t, []Simple[int]{{1}, {3}, {5}}, s)

	s = DeletePointerFunc(s, func(v *Simple[int]) bool { return true })
	assert.Empty(t, s)
}
