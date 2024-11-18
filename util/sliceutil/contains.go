package sliceutil

func Contains[T comparable](array []T, value T) bool {
	for _, item := range array {
		if item == value {
			return true
		}
	}

	return false
}
