package rules

type ResultFunction[T1 any, T2 any] interface {
	Call(payloadIn *T1, payloadOut *T2) error
}
