package sliceutil

// IndexPointerFunc returns the index of the first element in the slice for which the function f returns true.
func IndexPointerFunc[Slice ~[]T, T any](s Slice, f func(*T) bool) int {
	for i := range s {
		if f(&s[i]) {
			return i
		}
	}
	return -1
}

// DeletePointerFunc deletes all elements from the slice for which the function f returns true.
func DeletePointerFunc[Slice ~[]T, T any](s Slice, f func(*T) bool) Slice {
	i := IndexPointerFunc(s, f)
	if i == -1 {
		return s
	}
	for j := i + 1; j < len(s); j++ {
		if v := &s[j]; !f(v) {
			s[i] = *v
			i++
		}
	}
	clear(s[i:]) // zero/nil out the obsolete elements, for GC
	return s[:i]
}
