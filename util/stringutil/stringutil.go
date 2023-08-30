package stringutil

import (
	"strconv"
	"strings"
)

// StrToInt8Slice breaks a string into a series of tokens using a comma as a delimiter but only
// appends the tokens into the return array if tokens can be interpreted as an 'int8'
func StrToInt8Slice(str string) ([]int8, error) {
	var r []int8

	if len(str) > 0 {
		strSlice := strings.Split(str, ",")
		for _, s := range strSlice {
			v, err := strconv.ParseInt(s, 10, 8)
			if err != nil {
				return nil, err
			}
			r = append(r, int8(v))
		}
	}

	return r, nil
}
