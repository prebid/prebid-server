package iterators

import "iter"

// SlicePointers returns an iterator that yields indices and pointers to the elements of a slice.
func SlicePointers[Slice ~[]T, T any](s Slice) iter.Seq2[int, *T] {
	return func(yield func(int, *T) bool) {
		for i := range s {
			if !yield(i, &s[i]) {
				return
			}
		}
	}
}

// SlicePointerValues returns an iterator that yields pointers to the elements of a slice.
func SlicePointerValues[Slice ~[]T, T any](s Slice) iter.Seq[*T] {
	return func(yield func(*T) bool) {
		for i := range s {
			if !yield(&s[i]) {
				return
			}
		}
	}
}
