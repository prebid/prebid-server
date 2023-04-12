package ptrutil

func ClonePtr[T any](v *T) *T {
	if v == nil {
		return nil
	}

	clone := *v
	return &clone
}

func ToPtr[T any](v T) *T {
	return &v
}
