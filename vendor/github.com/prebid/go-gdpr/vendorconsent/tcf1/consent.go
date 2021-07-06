package vendorconsent

import (
	"encoding/base64"

	"github.com/prebid/go-gdpr/api"
	"github.com/prebid/go-gdpr/consentconstants"
)

// ParseString parses the TCF 1.x vendor string base64 encoded
func ParseString(consent string) (api.VendorConsents, error) {
	if consent == "" {
		return nil, consentconstants.ErrEmptyDecodedConsent
	}

	buff := []byte(consent)
	decoded := buff
	n, err := base64.RawURLEncoding.Decode(decoded, buff)
	if err != nil {
		return nil, err
	}
	decoded = decoded[:n:n]

	return Parse(decoded)
}

// Parse the vendor consent data from the string. This string should *not* be encoded (by base64 or any other encoding).
// If the data is malformed and cannot be interpreted as a vendor consent string, this will return an error.
func Parse(data []byte) (api.VendorConsents, error) {
	metadata, err := parseMetadata(data)
	if err != nil {
		return nil, err
	}

	// Bit 172 determines whether or not the consent string encodes Vendor data in a RangeSection or BitField.
	if isSet(data, 172) {
		return parseRangeSection(metadata)
	}

	return parseBitField(metadata)
}
