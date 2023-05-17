package sliceutil

func Clone[T any](s []T) []T {
	if s == nil {
		return nil
	}

	c := make([]T, len(s))
	copy(c, s)

	return c
}
