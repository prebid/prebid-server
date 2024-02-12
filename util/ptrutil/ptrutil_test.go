package ptrutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValueOrDefault(t *testing.T) {
	t.Run("int-nil", func(t *testing.T) {
		var v *int = nil
		r := ValueOrDefault(v)
		assert.Equal(t, 0, r)
	})

	t.Run("int-0", func(t *testing.T) {
		var v *int = ToPtr[int](0)
		r := ValueOrDefault(v)
		assert.Equal(t, 0, r)
	})

	t.Run("int-42", func(t *testing.T) {
		var v *int = ToPtr[int](42)
		r := ValueOrDefault(v)
		assert.Equal(t, 42, r)
	})

	t.Run("string-nil", func(t *testing.T) {
		var v *string = nil
		r := ValueOrDefault(v)
		assert.Equal(t, "", r)
	})

	t.Run("string-empty", func(t *testing.T) {
		var v *string = ToPtr[string]("")
		r := ValueOrDefault(v)
		assert.Equal(t, "", r)
	})

	t.Run("string-something", func(t *testing.T) {
		var v *string = ToPtr[string]("something")
		r := ValueOrDefault(v)
		assert.Equal(t, "something", r)
	})
}
