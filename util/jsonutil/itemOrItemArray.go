package jsonutil

import (
	"fmt"
)

// ItemOrItemArray is a custom type that can hold either a single value of type T or an array of values of type T.
type ItemOrItemArray[T any] []T

func (t *ItemOrItemArray[T]) UnmarshalJSON(b []byte) error {
	// try to unmarshal as a single value of type T first
	var itemResult T
	if err := UnmarshalValid(b, &itemResult); err == nil {
		*t = []T{itemResult}
		return nil
	}

	// then try to unmarshal as an array of type T
	var arrayResult []T
	if err := UnmarshalValid(b, &arrayResult); err == nil {
		*t = arrayResult
		return nil
	}

	// if both attempts fail, return an error
	return fmt.Errorf("value should be of type %T or %T", itemResult, arrayResult)
}
