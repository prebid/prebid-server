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
