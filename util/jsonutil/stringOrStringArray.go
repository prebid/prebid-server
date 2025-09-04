package jsonutil

import (
	"errors"
)

type StringOrStringArray []string

// parse string or string array input into string array
func (t *StringOrStringArray) UnmarshalJSON(b []byte) error {
	// try to unmarshal as a single string first
	var stringResult string
	if err := UnmarshalValid(b, &stringResult); err == nil {
		*t = []string{stringResult}
		return nil
	}

	// then try to unmarshal as an array of strings
	var arrayResult []string
	if err := UnmarshalValid(b, &arrayResult); err == nil {
		*t = arrayResult
		return nil
	}

	// if both attempts fail, return an error
	return errors.New("value should be of type string or []string")
}
