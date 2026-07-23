package sliceutil

import (
	"maps"
)

// EqualIgnoreOrder checks if two slices contain the same elements, regardless of their order. This
// is not optimized for memory usage and is intended for use at startup only. Empty and nil slices
// are considered equal, matching slices.Equal behavior.
func EqualIgnoreOrder[T comparable](s1, s2 []T) bool {
	if len(s1) != len(s2) {
		return false
	}

	counts1 := make(map[T]int, len(s1))
	for _, item := range s1 {
		counts1[item]++
	}

	counts2 := make(map[T]int, len(s2))
	for _, item := range s2 {
		counts2[item]++
	}

	return maps.Equal(counts1, counts2)
}
