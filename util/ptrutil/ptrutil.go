package ptrutil

func ToPtr[T any](v T) *T {
	return &v
}

func Clone[T any](v *T) *T {
	if v == nil {
		return nil
	}

	clone := *v
	return &clone
}

func ValueOrDefault[T any](v *T) T {
	if v != nil {
		return *v
	}

	var def T
	return def
}

func Equal[T comparable](v1, v2 *T) bool {
	if v1 == nil && v2 == nil {
		return true
	}

	if v1 == nil || v2 == nil {
		return false
	}

	return *v1 == *v2
}
