package jsonutil

import (
	"errors"

	"github.com/tidwall/gjson"
)

type IntString string

func (st *IntString) UnmarshalJSON(b []byte) error {
	res := gjson.ParseBytes(b)
	if res.Type != gjson.Number && res.Type != gjson.String {
		return errors.New("invalid type")
	}

	*st = IntString(res.String())
	return nil
}
