package rules

type ResultFunction[T any] interface {
	Call(payload *T) error
}
