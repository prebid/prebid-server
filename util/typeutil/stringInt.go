package typeutil

import (
	"encoding/json"
	"strconv"
)

type StringInt int

func (st *StringInt) UnmarshalJSON(b []byte) error {
	//convert the bytes into an interface as this will help us check the type of our value
	var item interface{}
	if err := json.Unmarshal(b, &item); err != nil {
		return err
	}

	switch v := item.(type) {
	case int:
		*st = StringInt(v)
	case float64:
		*st = StringInt(int(v))
	case string:
		///here convert the string into an int
		i, err := strconv.Atoi(v)
		if err != nil {
			return err

		}
		*st = StringInt(i)
	}
	return nil
}
