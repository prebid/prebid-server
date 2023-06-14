package usersync

import (
	"encoding/base64"
	"encoding/json"
)

type Base64Encoder interface {
	Encode(c *Cookie) string // Encode a cookie into a base 64 string
}

type Base64EncoderV1 struct{}

func (e Base64EncoderV1) Encode(c *Cookie) string {
	j, _ := json.Marshal(c)
	b64 := base64.URLEncoding.EncodeToString(j)

	return b64
}
