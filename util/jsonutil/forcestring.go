package jsonutil

import "github.com/tidwall/gjson"

// ParseIntoString Parse json bytes into a string pointer
func ParseIntoString(b []byte, ppString **string) {
	if ppString == nil {
		panic("ppString is nil")
	}
	result := gjson.ParseBytes(b)
	if result.Exists() && result.Raw != `null` {
		*ppString = new(string)
		**ppString = result.String()
	}
}
