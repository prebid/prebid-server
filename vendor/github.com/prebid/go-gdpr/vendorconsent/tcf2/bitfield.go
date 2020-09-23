package vendorconsent

import (
	"fmt"
)

func parseBitField(metadata ConsentMetadata, vendorBitsRequired uint16, startbit uint) (*consentBitField, uint, error) {
	data := metadata.data

	bytesRequired := (uint(vendorBitsRequired) + startbit) / 8
	if uint(len(data)) < bytesRequired {
		return nil, 0, fmt.Errorf("a BitField for %d vendors requires a consent string of %d bytes. This consent string had %d", vendorBitsRequired, bytesRequired, len(data))
	}

	return &consentBitField{
		data:        data,
		startbit:    startbit,
		maxVendorID: vendorBitsRequired,
	}, startbit + uint(vendorBitsRequired), nil
}

// A BitField has len(MaxVendorID()) entries, with one bit for every vendor in the range.
type consentBitField struct {
	data        []byte
	startbit    uint
	maxVendorID uint16
}

func (f *consentBitField) MaxVendorID() uint16 {
	if f == nil {
		return 0
	}

	return f.maxVendorID
}

func (f *consentBitField) VendorConsent(id uint16) bool {
	if id < 1 || id > f.maxVendorID {
		return false
	}
	// Careful here... vendor IDs start at index 1...
	return isSet(f.data, f.startbit+uint(id)-1)
}

// byteToBool returns false if val is 0, and true otherwise
func byteToBool(val byte) bool {
	return val != 0
}
