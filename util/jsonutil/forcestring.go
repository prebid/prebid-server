package jsonutil

import (
	"errors"

	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/tidwall/gjson"
)

// ParseIntoString Parse json bytes into a string pointer
func ParseIntoString(b []byte, ppString **string) error {
	if ppString == nil {
		return errors.New("ppString is nil")
	}
	result := gjson.ParseBytes(b)
	if result.Exists() && result.Raw != `null` {
		*ppString = ptrutil.ToPtr(result.String())
	}
	return nil
}
