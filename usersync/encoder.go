package usersync

import (
	"encoding/base64"
	"encoding/json"
)

type Base64Encoder interface {
	// Encode a cookie into a base 64 string
	Encode(c *Cookie) (string, error)
}

type Base64EncoderV1 struct{}

func (e Base64EncoderV1) Encode(c *Cookie) (string, error) {
	j, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	b64 := base64.URLEncoding.EncodeToString(j)

	return b64, nil
}
