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

// Parse the metadata from the consent string.
// This returns an error if the input is too short to answer questions about that data.
func parseMetadata(data []byte) (consentMetadata, error) {
	if len(data) < 22 {
		return nil, fmt.Errorf("vendor consent strings are at least 22 bytes long. This one was %d", len(data))
	}
	metadata := consentMetadata(data)
	if metadata.MaxVendorID() < 1 {
		return nil, fmt.Errorf("the consent string encoded a MaxVendorID of %d, but this value must be greater than or equal to 1", metadata.MaxVendorID())
	}
	if metadata.Version() < 1 {
		return nil, fmt.Errorf("the consent string encoded a Version of %d, but this value must be greater than or equal to 1", metadata.Version())
	}
	if metadata.VendorListVersion() == 0 {
		return nil, errInvalidVendorListVersion

	}
	return consentMetadata(data), nil
}

// consemtMetadata implements the parts of the VendorConsents interface which are common
// to BitFields and RangeSections. This relies on Parse to have done some validation already,
// to make sure that functions on it don't overflow the bounds of the byte array.
type consentMetadata []byte

func (c consentMetadata) Version() uint8 {
	// Stored in bits 0-5
	return uint8(c[0] >> 2)
}

const (
	nanosPerDeci = 100000000
	decisPerOne  = 10
)

func (c consentMetadata) Created() time.Time {
	_ = c[5]
	// Stored in bits 6-41.. which is [000000xx xxxxxxxx xxxxxxxx xxxxxxxx xxxxxxxx xx000000] starting at the 1st byte
	deciseconds := int64(binary.BigEndian.Uint64([]byte{
		0x0,
		0x0,
		0x0,
		(c[0]&0x3)<<2 | c[1]>>6,
		c[1]<<2 | c[2]>>6,
		c[2]<<2 | c[3]>>6,
		c[3]<<2 | c[4]>>6,
		c[4]<<2 | c[5]>>6,
	}))
	return time.Unix(deciseconds/decisPerOne, (deciseconds%decisPerOne)*nanosPerDeci)
}

func (c consentMetadata) LastUpdated() time.Time {
	// Stored in bits 42-77... which is [00xxxxxx xxxxxxxx xxxxxxxx xxxxxxxx xxxxxx00 ] starting at the 6th byte
	deciseconds := int64(binary.BigEndian.Uint64([]byte{
		0x0,
		0x0,
		0x0,
		(c[5] >> 2) & 0x0f,
		c[5]<<6 | c[6]>>2,
		c[6]<<6 | c[7]>>2,
		c[7]<<6 | c[8]>>2,
		c[8]<<6 | c[9]>>2,
	}))
	return time.Unix(deciseconds/decisPerOne, (deciseconds%decisPerOne)*nanosPerDeci)
}

func (c consentMetadata) CmpID() uint16 {
	// Stored in bits 78-89... which is [000000xx xxxxxxxx xx000000] starting at the 10th byte
	leftByte := ((c[9] & 0x03) << 2) | c[10]>>6
	rightByte := (c[10] << 2) | c[11]>>6
	return binary.BigEndian.Uint16([]byte{leftByte, rightByte})
}

func (c consentMetadata) CmpVersion() uint16 {
	// Stored in bits 90-101.. which is [00xxxxxx xxxxxx00] starting at the 12th byte
	leftByte := (c[11] >> 2) & 0x0f
	rightByte := (c[11] << 6) | c[12]>>2
	return binary.BigEndian.Uint16([]byte{leftByte, rightByte})
}

func (c consentMetadata) ConsentScreen() uint8 {
	// Stored in bits 102-107.. which is [000000xx xxxx0000] starting at the 13th byte
	return uint8(((c[12] & 0x03) << 4) | c[13]>>4)
}

func (c consentMetadata) ConsentLanguage() string {
	// Stored in bits 108-119... which is [0000xxxx xxxxxxxx] starting at the 14th byte.
	// Each letter is stored as 6 bits, with A=0 and Z=25
	leftChar := ((c[13] & 0x0f) << 2) | c[14]>>6
	rightChar := c[14] & 0x3f
	return string([]byte{leftChar + 65, rightChar + 65}) // Unicode A-Z is 65-90
}

func (c consentMetadata) VendorListVersion() uint16 {
	// The vendor list version is stored in bits 120 - 131
	rightByte := ((c[16] & 0xf0) >> 4) | ((c[15] & 0x0f) << 4)
	leftByte := c[15] >> 4
	return binary.BigEndian.Uint16([]byte{leftByte, rightByte})
}

func (c consentMetadata) MaxVendorID() uint16 {
	// The max vendor ID is stored in bits 156 - 171
	leftByte := byte((c[19]&0x0f)<<4 + (c[20]&0xf0)>>4)
	rightByte := byte((c[20]&0x0f)<<4 + (c[21]&0xf0)>>4)
	return binary.BigEndian.Uint16([]byte{leftByte, rightByte})
}

func (c consentMetadata) PurposeAllowed(id consentconstants.Purpose) bool {
	// Purposes are stored in bits 132 - 155. The interface contract only defines behavior for ints in the range [1, 24]...
	// so in the valid range, this won't even overflow a uint8.
	return isSet(c, uint(id)+131)
}

// Returns true if the bitIndex'th bit in data is a 1, and false if it's a 0.
func isSet(data []byte, bitIndex uint) bool {
	byteIndex := bitIndex / 8
	bitOffset := bitIndex % 8
	return byteToBool(data[byteIndex] & (0x80 >> bitOffset))
}
