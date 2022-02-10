package sliceutil

import (
	"strings"
)

func ContainsStringIgnoreCase(s []string, v string) bool {
	for _, i := range s {
		if strings.EqualFold(i, v) {
			return true
		}
	}
	return false
}
