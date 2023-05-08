package stringutil

import (
	"strconv"
	"strings"
)

func StrToInt8Slice(str string) []int8 {
	var r []int8

	if len(str) > 0 {
		strSlice := strings.Split(str, ",")
		for _, s := range strSlice {
			if v, err := strconv.ParseInt(s, 10, 8); err == nil {
				r = append(r, int8(v))
			}
		}
	}

	return r
}
