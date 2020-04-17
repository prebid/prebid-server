package vendorconsent

import (
	"fmt"
)

func parseBitField(data consentMetadata) (*consentBitField, error) {
	vendorBitsRequired := data.MaxVendorID()

	// BitFields start at bit 173. This means the last three bits of byte 21 are part of the bitfield.
	// In this case "others" will never be used, and we don't risk an index-out-of-bounds by using it.
	if vendorBitsRequired <= 3 {
		return &consentBitField{
			consentMetadata: data,
			firstThree:      data[21],
			others:          nil,
		}, nil
	}

	otherBytesRequired := (vendorBitsRequired - 3) / 8
	if (vendorBitsRequired-3)%8 > 0 {
		otherBytesRequired = otherBytesRequired + 1
	}
	dataLengthRequired := 22 + otherBytesRequired
	if uint(len(data)) < uint(dataLengthRequired) {
		return nil, fmt.Errorf("a BitField for %d vendors requires a consent string of %d bytes. This consent string had %d", vendorBitsRequired, dataLengthRequired, len(data))
	}

	return &consentBitField{
		consentMetadata: data,
		firstThree:      data[21],
		others:          data[22:],
	}, nil
}

// A BitField has len(MaxVendorID()) entries, with one bit for every vendor in the range.
type consentBitField struct {
	consentMetadata
	firstThree byte
	others     []byte
}

func (f *consentBitField) VendorConsent(id uint16) bool {
	if id < 1 || id > f.MaxVendorID() {
		return false
	}
	// Careful here... vendor IDs start at index 1...
	if id <= 3 {
		return byteToBool(f.firstThree & (0x08 >> id))
	}
	return isSet(f.others, uint(id-4))
}

// byteToBool returns false if val is 0, and true otherwise
func byteToBool(val byte) bool {
	return val != 0
}
