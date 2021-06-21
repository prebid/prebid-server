package vendorconsent

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/prebid/go-gdpr/consentconstants"
)

var (
	errInvalidVendorListVersion = errors.New("the consent string encoded a VendorListVersion of 0, but this value must be greater than or equal to 1")
)

// parseMetadata parses the metadata from the consent string.
// This returns an error if the input is too short to answer questions about that data.
func parseMetadata(data []byte) (ConsentMetadata, error) {
	if len(data) < 29 {
		return ConsentMetadata{}, fmt.Errorf("vendor consent strings are at least 29 bytes long. This one was %d", len(data))
	}
	metadata := ConsentMetadata{
		data: data,
	}
	if metadata.Version() < 2 {
		version := metadata.Version()
		metadata.data = nil
		return metadata, fmt.Errorf("the consent string encoded a Version of %d, but this value must be greater than or equal to 2", version)
	}
	if metadata.VendorListVersion() == 0 {
		metadata.data = nil
		return metadata, errInvalidVendorListVersion

	}
	return metadata, nil
}

// ConsentMetadata implements the parts of the VendorConsents interface which are common
// to BitFields and RangeSections. This relies on Parse to have done some validation already,
// to make sure that functions on it don't overflow the bounds of the byte array.
type ConsentMetadata struct {
	data                          []byte
	vendorLegitimateInterestStart uint
	pubRestrictionsStart          uint
	vendorConsents                vendorConsentsResolver
	vendorLegitimateInterests     vendorConsentsResolver
	publisherRestrictions         pubRestrictResolver
}

type vendorConsentsResolver interface {
	MaxVendorID() uint16
	VendorConsent(id uint16) bool
}

type pubRestrictResolver interface {
	CheckPubRestriction(purposeID uint8, restrictType uint8, vendor uint16) bool
}

// Version returns the version stored in the first 6 bits
func (c ConsentMetadata) Version() uint8 {
	// Stored in bits 0-5
	return uint8(c.data[0] >> 2)
}

const (
	nanosPerDeci = 100000000
	decisPerOne  = 10
)

// Created returns the created date stored in bits 7 to 42
func (c ConsentMetadata) Created() time.Time {
	// Stored in bits 6-41.. which is [000000xx xxxxxxxx xxxxxxxx xxxxxxxx xxxxxxxx xx000000] starting at the 1st byte
	deciseconds := int64(binary.BigEndian.Uint64([]byte{
		0x0,
		0x0,
		0x0,
		(c.data[0]&0x3)<<2 | c.data[1]>>6,
		c.data[1]<<2 | c.data[2]>>6,
		c.data[2]<<2 | c.data[3]>>6,
		c.data[3]<<2 | c.data[4]>>6,
		c.data[4]<<2 | c.data[5]>>6,
	}))
	return time.Unix(deciseconds/decisPerOne, (deciseconds%decisPerOne)*nanosPerDeci)
}

// LastUpdated returns the last updated date stored in bits 43 to 78
func (c ConsentMetadata) LastUpdated() time.Time {
	// Stored in bits 42-77... which is [00xxxxxx xxxxxxxx xxxxxxxx xxxxxxxx xxxxxx00 ] starting at the 6th byte
	deciseconds := int64(binary.BigEndian.Uint64([]byte{
		0x0,
		0x0,
		0x0,
		(c.data[5] >> 2) & 0x0f,
		c.data[5]<<6 | c.data[6]>>2,
		c.data[6]<<6 | c.data[7]>>2,
		c.data[7]<<6 | c.data[8]>>2,
		c.data[8]<<6 | c.data[9]>>2,
	}))
	return time.Unix(deciseconds/decisPerOne, (deciseconds%decisPerOne)*nanosPerDeci)
}

// CmpID returns the Consent Management Platform identifier stored in bits 79 to 90
func (c ConsentMetadata) CmpID() uint16 {
	// Stored in bits 78-89... which is [000000xx xxxxxxxx xx000000] starting at the 10th byte
	leftByte := ((c.data[9] & 0x03) << 2) | c.data[10]>>6
	rightByte := (c.data[10] << 2) | c.data[11]>>6
	return binary.BigEndian.Uint16([]byte{leftByte, rightByte})
}

// CmpVersion returns the Consent Management Platform version stored in bits 91 to 102
func (c ConsentMetadata) CmpVersion() uint16 {
	// Stored in bits 90-101.. which is [00xxxxxx xxxxxx00] starting at the 12th byte
	leftByte := (c.data[11] >> 2) & 0x0f
	rightByte := (c.data[11] << 6) | c.data[12]>>2
	return binary.BigEndian.Uint16([]byte{leftByte, rightByte})
}

// ConsentScreen returns the consent screen info stored in bits 103 to 108
func (c ConsentMetadata) ConsentScreen() uint8 {
	// Stored in bits 102-107.. which is [000000xx xxxx0000] starting at the 13th byte
	return uint8(((c.data[12] & 0x03) << 4) | c.data[13]>>4)
}

// ConsentLanguage returns the two letter code for consent language stored in bits 109 to 120
func (c ConsentMetadata) ConsentLanguage() string {
	// Stored in bits 108-119... which is [0000xxxx xxxxxxxx] starting at the 14th byte.
	// Each letter is stored as 6 bits, with A=0 and Z=25
	leftChar := ((c.data[13] & 0x0f) << 2) | c.data[14]>>6
	rightChar := c.data[14] & 0x3f
	return string([]byte{leftChar + 65, rightChar + 65}) // Unicode A-Z is 65-90
}

// VendorLegitInterestMaxID returns the vendor legitimate interest max id
func (c ConsentMetadata) VendorLegitInterestMaxID() uint16 {
	return c.vendorLegitimateInterests.MaxVendorID()
}

// VendorListVersion returns the vendor list version stored in bits 121 to 132
func (c ConsentMetadata) VendorListVersion() uint16 {
	// The vendor list version is stored in bits 121 - 132
	rightByte := ((c.data[16] & 0xf0) >> 4) | ((c.data[15] & 0x0f) << 4)
	leftByte := c.data[15] >> 4
	return binary.BigEndian.Uint16([]byte{leftByte, rightByte})
}

// MaxVendorID returns the maximum value for vendor identifier in bits 214 to 229
func (c ConsentMetadata) MaxVendorID() uint16 {
	// The max vendor ID is stored in bits 214 - 229
	leftByte := ((c.data[26] & 0x07) << 5) | ((c.data[27] & 0xf8) >> 3)
	rightByte := ((c.data[27] & 0x07) << 5) | ((c.data[28] & 0xf8) >> 3)
	return binary.BigEndian.Uint16([]byte{leftByte, rightByte})
}

// PurposeAllowed returns if the given purpose (1 to 24 max) is enabled, info stored in bits 153 to 176
func (c ConsentMetadata) PurposeAllowed(id consentconstants.Purpose) bool {
	// Purposes are stored in bits 152 - 175. The interface contract only defines behavior for ints in the range [1, 24]...
	// so in the valid range, this won't even overflow a uint8.
	if id > 24 {
		return false
	}
	return isSet(c.data, uint(id)+151)
}

// PurposeLITransparency returns if the given purpose transparency (1 to 24 max) is enabled, info stored in bits 177 to 200
func (c ConsentMetadata) PurposeLITransparency(id consentconstants.Purpose) bool {
	// Purposes are stored in bits 176 - 199. The interface contract only defines behavior for ints in the range [1, 24]...
	// so in the valid range, this won't even overflow a uint8.
	if id > 24 {
		return false
	}
	return isSet(c.data, uint(id)+175)
}

// PurposeOneTreatment returns if Purpose 1 is enable, info stored in bit 201
func (c ConsentMetadata) PurposeOneTreatment() bool {
	return isSet(c.data, 200)
}

// SpecialFeatureOptIn returns if the given special feature is enable, stored in bits 140 to 152
func (c ConsentMetadata) SpecialFeatureOptIn(id uint16) bool {
	if id > 12 {
		return false
	}
	return isSet(c.data, 140+uint(id)-1)
}

// VendorConsent returns true if there is consent for the given vendor id
func (c ConsentMetadata) VendorConsent(id uint16) bool {
	return c.vendorConsents.VendorConsent(id)
}

// VendorLegitInterest returns true if there is legitimate interest established for the given vendor id
func (c ConsentMetadata) VendorLegitInterest(id uint16) bool {
	return c.vendorLegitimateInterests.VendorConsent(id)
}

// CheckPubRestriction returns the publisher restriction for a given purpose id, restriction type and vendor id
func (c ConsentMetadata) CheckPubRestriction(purposeID uint8, restrictType uint8, vendor uint16) bool {
	return c.publisherRestrictions.CheckPubRestriction(purposeID, restrictType, vendor)
}

// Returns true if the bitIndex'th bit in data is a 1, and false if it's a 0.
func isSet(data []byte, bitIndex uint) bool {
	byteIndex := bitIndex / 8
	bitOffset := bitIndex % 8
	return byteToBool(data[byteIndex] & (0x80 >> bitOffset))
}
