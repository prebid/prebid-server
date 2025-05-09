package rules

type ResultFunction[T any] interface {
	Call(payload *T, schemaFunctionsResults map[string]string) error
}
