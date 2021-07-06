package vendorconsent

import (
	"encoding/base64"
	"strings"

	"github.com/prebid/go-gdpr/api"
	"github.com/prebid/go-gdpr/bitutils"
	"github.com/prebid/go-gdpr/consentconstants"
)

const (
	consentStringTCF2Separator = '.'
	consentStringTCF2Prefix    = 'C'
)

// ParseString parses the TCF 2.0 vendor string base64 encoded
func ParseString(consent string) (api.VendorConsents, error) {
	if consent == "" {
		return nil, consentconstants.ErrEmptyDecodedConsent
	}
	// split TCF 2.0 segments
	if index := strings.IndexByte(consent, consentStringTCF2Separator); index != -1 {
		consent = consent[:index]
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

// Parse parses the TCF 2.0 vendor consent data from the string. This string should *not* be encoded (by base64 or any other encoding).
// If the data is malformed and cannot be interpreted as a vendor consent string, this will return an error.
func Parse(data []byte) (api.VendorConsents, error) {
	metadata, err := parseMetadata(data)
	if err != nil {
		return nil, err
	}

	var vendorConsents vendorConsentsResolver
	var vendorLegitInts vendorConsentsResolver

	var legitIntStart uint
	var pubRestrictsStart uint
	// Bit 229 determines whether or not the consent string encodes Vendor data in a RangeSection or BitField.
	if isSet(data, 229) {
		vendorConsents, legitIntStart, err = parseRangeSection(metadata, metadata.MaxVendorID(), 230)
	} else {
		vendorConsents, legitIntStart, err = parseBitField(metadata, metadata.MaxVendorID(), 230)
	}
	if err != nil {
		return nil, err
	}

	metadata.vendorConsents = vendorConsents
	metadata.vendorLegitimateInterestStart = legitIntStart + 17
	legIntMaxVend, err := bitutils.ParseUInt16(data, legitIntStart)
	if err != nil {
		return nil, err
	}

	if isSet(data, legitIntStart+16) {
		vendorLegitInts, pubRestrictsStart, err = parseRangeSection(metadata, legIntMaxVend, metadata.vendorLegitimateInterestStart)
	} else {
		vendorLegitInts, pubRestrictsStart, err = parseBitField(metadata, legIntMaxVend, metadata.vendorLegitimateInterestStart)
	}
	if err != nil {
		return nil, err
	}

	metadata.vendorLegitimateInterests = vendorLegitInts
	metadata.pubRestrictionsStart = pubRestrictsStart

	pubRestrictions, _, err := parsePubRestriction(metadata, pubRestrictsStart)
	if err != nil {
		return nil, err
	}

	metadata.publisherRestrictions = pubRestrictions

	return metadata, err

}

// IsConsentV2 return true if the consent strings looks like a tcf v2 consent string
func IsConsentV2(consent string) bool {
	return len(consent) > 0 && consent[0] == consentStringTCF2Prefix
}
