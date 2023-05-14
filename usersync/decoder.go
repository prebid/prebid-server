package usersync

import (
	"encoding/base64"
	"encoding/json"
)

type Decoder interface {
	Decode(v string) *Cookie // Takes an encoded string, and decodes it into a cookie
}

type DecodeV1 struct{}

func (d DecodeV1) Decode(encodedValue string) *Cookie {
	jsonValue, err := base64.URLEncoding.DecodeString(encodedValue)
	if err != nil {
		// corrupted cookie; we should reset
		return NewCookie()
	}

	var cookie Cookie
	if err = json.Unmarshal(jsonValue, &cookie); err != nil {
		// corrupted cookie; we should reset
		return NewCookie()
	}

	return &cookie
}
