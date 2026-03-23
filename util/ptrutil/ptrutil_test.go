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

func TestEqualInt(t *testing.T) {
	tests := []struct {
		name     string
		value1   *int
		value2   *int
		expected bool
	}{
		{
			name:     "nil",
			value1:   nil,
			value2:   nil,
			expected: true,
		},
		{
			name:     "nil-value1",
			value1:   nil,
			value2:   ToPtr(42),
			expected: false,
		},
		{
			name:     "nil-value2",
			value1:   ToPtr(42),
			value2:   nil,
			expected: false,
		},
		{
			name:     "same",
			value1:   ToPtr(42),
			value2:   ToPtr(42),
			expected: true,
		},
		{
			name:     "different",
			value1:   ToPtr(42),
			value2:   ToPtr(24),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Equal(tt.value1, tt.value2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEqualString(t *testing.T) {
	tests := []struct {
		name     string
		value1   *string
		value2   *string
		expected bool
	}{
		{
			name:     "nil",
			value1:   nil,
			value2:   nil,
			expected: true,
		},
		{
			name:     "nil-value1",
			value1:   nil,
			value2:   ToPtr("hello"),
			expected: false,
		},
		{
			name:     "nil-value2",
			value1:   ToPtr("hello"),
			value2:   nil,
			expected: false,
		},
		{
			name:     "same",
			value1:   ToPtr("hello"),
			value2:   ToPtr("hello"),
			expected: true,
		},
		{
			name:     "different",
			value1:   ToPtr("hello"),
			value2:   ToPtr("world"),
			expected: false,
		},
		{
			name:     "empty",
			value1:   ToPtr(""),
			value2:   ToPtr(""),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Equal(tt.value1, tt.value2)
			assert.Equal(t, tt.expected, result)
		})
	}
}
