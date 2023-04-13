package sliceutil

import (
	"strings"
)

func ContainsStringIgnoreCase(s []string, v string) bool {
	for _, i := range s {
		if strings.EqualFold(i, v) {
			return true
		}
	}
	return false
}

// CloneSlice creates an indepent copy of a slice,
func CloneSlice[T any](s []T) []T {
	if s == nil {
		return nil
	}
	clone := make([]T, len(s))
	copy(clone, s)
	return clone
}
