package api

import "github.com/prebid/go-gdpr/consentconstants"

// VendorList is an interface used to fetch information about an IAB Global Vendor list.
// For the latest version, see: https://vendorlist.consensu.org/vendorlist.json
type VendorList interface {
	// Version returns the version of the vendor list which this is.
	//
	// If the input was malformed, this will return 0.
	Version() uint16

	// Vendor returns info about the vendor with the given ID.
	// This returns nil if that vendor isn't in this list, or the input was malformed somehow.
	//
	// If callers need to query multiple Purpose or LegitimateInterest statuses from the same vendor,
	// they should call this function once and then reuse the object it returns for future queries.
	Vendor(vendorID uint16) Vendor
}

// Vendor describes which purposes a given vendor claims to use data for, in this vendor list.
type Vendor interface {
	// Purpose returns true if this vendor claims to use data for the given purpose, or false otherwise
	Purpose(purposeID consentconstants.Purpose) bool
	// PurposeStrict checks only for the primary purpose, not considering flex purposes.
	PurposeStrict(purposeID consentconstants.Purpose) bool

	// LegitimateInterest returns true if this vendor claims a "Legitimate Interest" to
	// use data for the given purpose.
	//
	// For an explanation of legitimate interest, see https://www.gdpreu.org/the-regulation/key-concepts/legitimate-interest/
	LegitimateInterest(purposeID consentconstants.Purpose) bool
	// LegitimateInterestStrict checks only for the primary legitimate, not considering flex purposes.
	LegitimateInterestStrict(purposeID consentconstants.Purpose) (hasLegitimateInterest bool)
	// SpecialPurpose returns true if this vendor claims a need for the given special purpose
	SpecialPurpose(purposeID consentconstants.Purpose) (hasSpecialPurpose bool)
}
