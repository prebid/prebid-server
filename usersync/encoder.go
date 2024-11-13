package usersync

import (
	"encoding/base64"

	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type Encoder interface {
	// Encode a cookie into a base 64 string
	Encode(c *Cookie) (string, error)
}

type Base64Encoder struct{}

func (e Base64Encoder) Encode(c *Cookie) (string, error) {
	j, err := jsonutil.Marshal(c)
	if err != nil {
		return "", err
	}
	b64 := base64.URLEncoding.EncodeToString(j)

	return b64, nil
}
