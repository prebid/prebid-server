package vendorconsent

import (
	"encoding/base64"
	"strings"

	"github.com/prebid/go-gdpr/api"
	tcf1 "github.com/prebid/go-gdpr/vendorconsent/tcf1"
	tcf2 "github.com/prebid/go-gdpr/vendorconsent/tcf2"
)

// ParseString parses a Raw (unpadded) base64 URL encoded string.
func ParseString(consent string) (api.VendorConsents, error) {
	pieces := strings.Split(consent, ".")
	decoded, err := base64.RawURLEncoding.DecodeString(pieces[0])
	if err != nil {
		return nil, err
	}
	version := uint8(decoded[0] >> 2)
	if version == 2 {
		return tcf2.Parse(decoded)
	}
	return tcf1.Parse(decoded)
}

// Backwards compatibility

type VendorConsents interface {
	api.VendorConsents
}

func Parse(data []byte) (api.VendorConsents, error) {
	return tcf1.Parse(data)
}
