package rules

import ()

type ResultFunction[T any] interface {
	Call(payload *T) error
}