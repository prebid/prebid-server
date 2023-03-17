package jsonutil

import (
	"github.com/buger/jsonparser"
)

type StringInt int

func (st *StringInt) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}

	if b[0] == '"' {
		b = b[1 : len(b)-1]
	}

	if len(b) == 0 {
		return nil
	}

	i, err := jsonparser.ParseInt(b)
	if err != nil {
		return err
	}

	*st = StringInt(i)
	return nil
}
