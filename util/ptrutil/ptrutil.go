package ptrutil

func GetValueOrDefault[T any](v *T, d T) T {
	if v != nil {
		return *v
	}

	return d
}

func ToPtr[T any](v T) *T {
	return &v
}
