package vendorlist

import (
	"encoding/json"
	"errors"

	"github.com/prebid/go-gdpr/api"
	"github.com/prebid/go-gdpr/consentconstants"
)

// ParseEagerly interprets and validates the Vendor List data up front, before returning it.
// The returned object can be shared safely between goroutines.
//
// This is ideal if:
//   1. You plan to call functions on the returned VendorList many times before discarding it.
//   2. You need strong input validation and good error messages.
//
// Otherwise, you may get better performance with ParseLazily.
func ParseEagerly(data []byte) (api.VendorList, error) {
	var contract vendorListContract
	if err := json.Unmarshal(data, &contract); err != nil {
		return nil, err
	}

	if contract.Version == 0 {
		return nil, errors.New("data.vendorListVersion was 0 or undefined. Versions should start at 1")
	}
	if len(contract.Vendors) == 0 {
		return nil, errors.New("data.vendors was undefined or had no elements")
	}

	parsedList := parsedVendorList{
		version: contract.Version,
		vendors: make(map[uint16]parsedVendor, len(contract.Vendors)),
	}

	for i := 0; i < len(contract.Vendors); i++ {
		thisVendor := contract.Vendors[i]
		parsedList.vendors[thisVendor.ID] = parseVendor(thisVendor)
	}

	return parsedList, nil
}

func parseVendor(contract vendorListVendorContract) parsedVendor {
	parsed := parsedVendor{
		purposeIDs:            mapify(contract.PurposeIDs),
		legitimateInterestIDs: mapify(contract.LegitimateInterestIDs),
	}

	return parsed
}

func mapify(input []uint8) map[consentconstants.Purpose]struct{} {
	m := make(map[consentconstants.Purpose]struct{}, len(input))
	var s struct{}
	for _, value := range input {
		m[consentconstants.Purpose(value)] = s
	}
	return m
}

type parsedVendorList struct {
	version uint16
	vendors map[uint16]parsedVendor
}

func (l parsedVendorList) Version() uint16 {
	return l.version
}

func (l parsedVendorList) Vendor(vendorID uint16) api.Vendor {
	vendor, ok := l.vendors[vendorID]
	if ok {
		return vendor
	}
	return nil
}

type parsedVendor struct {
	purposeIDs            map[consentconstants.Purpose]struct{}
	legitimateInterestIDs map[consentconstants.Purpose]struct{}
}

func (l parsedVendor) Purpose(purposeID consentconstants.Purpose) (hasPurpose bool) {
	_, hasPurpose = l.purposeIDs[purposeID]
	return
}

func (l parsedVendor) PurposeStrict(purposeID consentconstants.Purpose) (hasPurpose bool) {
	_, hasPurpose = l.purposeIDs[purposeID]
	return
}

// LegitimateInterest retursn true if this vendor claims a "Legitimate Interest" to
// use data for the given purpose.
//
// For an explanation of legitimate interest, see https://www.gdpreu.org/the-regulation/key-concepts/legitimate-interest/
func (l parsedVendor) LegitimateInterest(purposeID consentconstants.Purpose) (hasLegitimateInterest bool) {
	_, hasLegitimateInterest = l.legitimateInterestIDs[purposeID]
	return
}

func (l parsedVendor) LegitimateInterestStrict(purposeID consentconstants.Purpose) (hasLegitimateInterest bool) {
	_, hasLegitimateInterest = l.legitimateInterestIDs[purposeID]
	return
}

// V1 vedndor list does not support special purposes.
func (l parsedVendor) SpecialPurpose(purposeID consentconstants.Purpose) bool {
	return false
}

type vendorListContract struct {
	Version uint16                     `json:"vendorListVersion"`
	Vendors []vendorListVendorContract `json:"vendors"`
}

type vendorListVendorContract struct {
	ID                    uint16  `json:"id"`
	PurposeIDs            []uint8 `json:"purposeIds"`
	LegitimateInterestIDs []uint8 `json:"legIntPurposeIds"`
}
