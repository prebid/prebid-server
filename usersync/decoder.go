package usersync

import (
	"encoding/base64"

	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type Decoder interface {
	Decode(encodedValue string) *Cookie
}

type Base64Decoder struct{}

func (d Base64Decoder) Decode(encodedValue string) *Cookie {
	jsonValue, err := base64.URLEncoding.DecodeString(encodedValue)
	if err != nil {
		return NewCookie()
	}

	var cookie Cookie
	if err = jsonutil.UnmarshalValid(jsonValue, &cookie); err != nil {
		return NewCookie()
	}

	return &cookie
}
