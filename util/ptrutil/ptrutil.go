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
