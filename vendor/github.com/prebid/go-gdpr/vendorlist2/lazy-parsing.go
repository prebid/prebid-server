package vendorlist2

import (
	"strconv"

	"github.com/buger/jsonparser"
	"github.com/prebid/go-gdpr/api"
	"github.com/prebid/go-gdpr/consentconstants"
)

// ParseLazily returns a view of the data which re-calculates things on each function call.
// The returned object can be shared safely between goroutines.
//
// This is ideal if:
//   1. You only need to look up a few vendors or purpose IDs
//   2. You don't need good errors on malformed input
//
// Otherwise, you may get better performance with ParseEagerly.
func ParseLazily(data []byte) api.VendorList {
	return lazyVendorList(data)
}

type lazyVendorList []byte

func (l lazyVendorList) Version() uint16 {
	if val, ok := lazyParseInt(l, "vendorListVersion"); ok {
		return uint16(val)
	}
	return 0
}

func (l lazyVendorList) Vendor(vendorID uint16) api.Vendor {
	vendorBytes, _, _, err := jsonparser.Get(l, "vendors", strconv.Itoa(int(vendorID)))
	if err == nil && len(vendorBytes) > 0 {
		return lazyVendor(vendorBytes)
	}
	return nil
}

type lazyVendor []byte

func (l lazyVendor) Purpose(purposeID consentconstants.Purpose) bool {
	exists := idExists(l, int(purposeID), "purposes")
	if exists {
		return true
	}
	return idExists(l, int(purposeID), "flexiblePurposes")
}

// PurposeStrict checks only for the primary purpose, no considering flex purposes.
func (l lazyVendor) PurposeStrict(purposeID consentconstants.Purpose) bool {
	return idExists(l, int(purposeID), "purposes")
}

func (l lazyVendor) LegitimateInterest(purposeID consentconstants.Purpose) bool {
	exists := idExists(l, int(purposeID), "legIntPurposes")
	if exists {
		return true
	}
	return idExists(l, int(purposeID), "flexiblePurposes")
}

// LegitimateInterestStrict checks only for the primary legitimate, no considering flex purposes.
func (l lazyVendor) LegitimateInterestStrict(purposeID consentconstants.Purpose) (hasLegitimateInterest bool) {
	return idExists(l, int(purposeID), "legIntPurposes")
}

// SpecialPurpose returns true if this vendor claims a need for the given special purpose
func (l lazyVendor) SpecialPurpose(purposeID consentconstants.Purpose) (hasSpecialPurpose bool) {
	return idExists(l, int(purposeID), "specialPurposes")
}

// Returns false unless "id" exists in an array located at "data.key".
func idExists(data []byte, id int, key string) bool {
	hasID := false

	jsonparser.ArrayEach(data, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		if err == nil && dataType == jsonparser.Number {
			if intVal, err := strconv.ParseInt(string(value), 10, 0); err == nil {
				if int(intVal) == id {
					hasID = true
				}
			}
		}
	}, key)

	return hasID
}

func lazyParseInt(data []byte, key string) (int, bool) {
	if value, dataType, _, err := jsonparser.Get(data, key); err == nil && dataType == jsonparser.Number {
		intVal, err := strconv.Atoi(string(value))
		if err != nil {
			return 0, false
		}
		return intVal, true
	}
	return 0, false
}
