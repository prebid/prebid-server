package sliceutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEqualIgnoreOrderInt(t *testing.T) {
	tests := []struct {
		name     string
		slice1   []int
		slice2   []int
		expected bool
	}{
		{
			name:     "nil-both",
			slice1:   nil,
			slice2:   nil,
			expected: true,
		},
		{
			name:     "nil-s1",
			slice1:   nil,
			slice2:   []int{1},
			expected: false,
		},
		{
			name:     "nil-s2",
			slice1:   []int{1},
			slice2:   nil,
			expected: false,
		},
		{
			name:     "empty-both",
			slice1:   []int{},
			slice2:   []int{},
			expected: true,
		},
		{
			name:     "empty-s1",
			slice1:   []int{},
			slice2:   []int{1},
			expected: false,
		},
		{
			name:     "empty-s2",
			slice1:   []int{1},
			slice2:   []int{},
			expected: false,
		},
		{
			name:     "nil-empty-mix-1",
			slice1:   nil,
			slice2:   []int{},
			expected: true,
		},
		{
			name:     "nil-empty-mix-2",
			slice1:   []int{},
			slice2:   nil,
			expected: true,
		},
		{
			name:     "one-same",
			slice1:   []int{1},
			slice2:   []int{1},
			expected: true,
		},
		{
			name:     "one-different",
			slice1:   []int{1},
			slice2:   []int{2},
			expected: false,
		},
		{
			name:     "many-same-ordered",
			slice1:   []int{1, 2},
			slice2:   []int{1, 2},
			expected: true,
		},
		{
			name:     "many-same-unordered",
			slice1:   []int{1, 2},
			slice2:   []int{2, 1},
			expected: true,
		},
		{
			name:     "many-different-lengths",
			slice1:   []int{1},
			slice2:   []int{2, 3},
			expected: false,
		},
		{
			name:     "many-different",
			slice1:   []int{1, 2},
			slice2:   []int{3, 4},
			expected: false,
		},
		{
			name:     "many-different-duplicates-unordered",
			slice1:   []int{1, 2, 2, 3},
			slice2:   []int{2, 1, 2, 4},
			expected: false,
		},
		{
			name:     "many-same-duplicates-unordered",
			slice1:   []int{1, 2, 2},
			slice2:   []int{2, 1, 2},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := EqualIgnoreOrder(test.slice1, test.slice2)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestEqualIgnoreOrderString(t *testing.T) {
	tests := []struct {
		name     string
		slice1   []string
		slice2   []string
		expected bool
	}{
		{
			name:     "nil-both",
			slice1:   nil,
			slice2:   nil,
			expected: true,
		},
		{
			name:     "nil-s1",
			slice1:   nil,
			slice2:   []string{"a"},
			expected: false,
		},
		{
			name:     "nil-s2",
			slice1:   []string{"a"},
			slice2:   nil,
			expected: false,
		},
		{
			name:     "empty-both",
			slice1:   []string{},
			slice2:   []string{},
			expected: true,
		},
		{
			name:     "empty-s1",
			slice1:   []string{},
			slice2:   []string{"a"},
			expected: false,
		},
		{
			name:     "empty-s2",
			slice1:   []string{"a"},
			slice2:   []string{},
			expected: false,
		},
		{
			name:     "nil-empty-mix-1",
			slice1:   nil,
			slice2:   []string{},
			expected: true,
		},
		{
			name:     "nil-empty-mix-2",
			slice1:   []string{},
			slice2:   nil,
			expected: true,
		},
		{
			name:     "one-same",
			slice1:   []string{"a"},
			slice2:   []string{"a"},
			expected: true,
		},
		{
			name:     "one-different",
			slice1:   []string{"a"},
			slice2:   []string{"b"},
			expected: false,
		},
		{
			name:     "many-same-ordered",
			slice1:   []string{"a", "b"},
			slice2:   []string{"a", "b"},
			expected: true,
		},
		{
			name:     "many-same-unordered",
			slice1:   []string{"a", "b"},
			slice2:   []string{"b", "a"},
			expected: true,
		},
		{
			name:     "many-different-lengths",
			slice1:   []string{"a"},
			slice2:   []string{"b", "c"},
			expected: false,
		},
		{
			name:     "many-different",
			slice1:   []string{"a", "b"},
			slice2:   []string{"c", "d"},
			expected: false,
		},
		{
			name:     "many-different-duplicates-unordered",
			slice1:   []string{"a", "b", "b", "c"},
			slice2:   []string{"b", "a", "b", "d"},
			expected: false,
		},
		{
			name:     "many-same-duplicates-unordered",
			slice1:   []string{"a", "b", "b"},
			slice2:   []string{"b", "a", "b"},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := EqualIgnoreOrder(test.slice1, test.slice2)
			assert.Equal(t, test.expected, result)
		})
	}
}
