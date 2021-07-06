package vendorlist2

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

	parsedList := parsedVendorList{
		version: contract.Version,
		vendors: make(map[uint16]parsedVendor, len(contract.Vendors)),
	}

	for _, v := range contract.Vendors {
		parsedList.vendors[v.ID] = parseVendor(v)
	}

	return parsedList, nil
}

func parseVendor(contract vendorListVendorContract) parsedVendor {
	parsed := parsedVendor{
		purposes:            mapify(contract.Purposes),
		legitimateInterests: mapify(contract.LegitimateInterests),
		flexiblePurposes:    mapify(contract.FlexiblePurposes),
		specialPurposes:     mapify(contract.SpecialPurposes),
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
	purposes            map[consentconstants.Purpose]struct{}
	legitimateInterests map[consentconstants.Purpose]struct{}
	flexiblePurposes    map[consentconstants.Purpose]struct{}
	specialPurposes     map[consentconstants.Purpose]struct{}
}

func (l parsedVendor) Purpose(purposeID consentconstants.Purpose) (hasPurpose bool) {
	_, hasPurpose = l.purposes[purposeID]
	if !hasPurpose {
		_, hasPurpose = l.flexiblePurposes[purposeID]
	}
	return
}

// PurposeStrict checks only for the primary purpose, no considering flex purposes.
func (l parsedVendor) PurposeStrict(purposeID consentconstants.Purpose) (hasPurpose bool) {
	_, hasPurpose = l.purposes[purposeID]
	return
}

// LegitimateInterest returns true if this vendor claims a "Legitimate Interest" to
// use data for the given purpose.
//
// For an explanation of legitimate interest, see https://www.gdpreu.org/the-regulation/key-concepts/legitimate-interest/
func (l parsedVendor) LegitimateInterest(purposeID consentconstants.Purpose) (hasLegitimateInterest bool) {
	_, hasLegitimateInterest = l.legitimateInterests[purposeID]
	if !hasLegitimateInterest {
		_, hasLegitimateInterest = l.flexiblePurposes[purposeID]
	}
	return
}

// LegitimateInterestStrict checks only for the primary legitimate, no considering flex purposes.
func (l parsedVendor) LegitimateInterestStrict(purposeID consentconstants.Purpose) (hasLegitimateInterest bool) {
	_, hasLegitimateInterest = l.legitimateInterests[purposeID]
	return
}

// SpecialPurpose returns true if this vendor claims a need for the given special purpose
func (l parsedVendor) SpecialPurpose(purposeID consentconstants.Purpose) (hasSpecialPurpose bool) {
	_, hasSpecialPurpose = l.specialPurposes[purposeID]
	return
}

type vendorListContract struct {
	Version uint16                              `json:"vendorListVersion"`
	Vendors map[string]vendorListVendorContract `json:"vendors"`
}

type vendorListVendorContract struct {
	ID                  uint16  `json:"id"`
	Purposes            []uint8 `json:"purposes"`
	LegitimateInterests []uint8 `json:"legIntPurposes"`
	FlexiblePurposes    []uint8 `json:"flexiblePurposes"`
	SpecialPurposes     []uint8 `json:"specialPurposes"`
}
