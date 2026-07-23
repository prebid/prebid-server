package iterutil

import "iter"

// Firsts returns an iterator that yields the first elements of a sequence of pairs.
func Firsts[T, U any](seq2 iter.Seq2[T, U]) iter.Seq[T] {
	return func(yield func(T) bool) {
		for t := range seq2 {
			if !yield(t) {
				return
			}
		}
	}
}

// Seconds returns an iterator that yields the second elements of a sequence of pairs.
func Seconds[T, U any](seq2 iter.Seq2[T, U]) iter.Seq[U] {
	return func(yield func(U) bool) {
		for _, u := range seq2 {
			if !yield(u) {
				return
			}
		}
	}
}
