package usersync

import (
	"encoding/base64"
	"encoding/json"
)

type Encoder interface {
	Encode(c *Cookie) string // Encode a cookie into a base 64 string
}

type EncoderV1 struct{}

func (e EncoderV1) Encode(c *Cookie) string {
	j, _ := json.Marshal(c)
	b64 := base64.URLEncoding.EncodeToString(j)

	return b64
}
