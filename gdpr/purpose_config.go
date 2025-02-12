package gdpr

import (
	"github.com/prebid/go-gdpr/consentconstants"
	"github.com/prebid/prebid-server/v3/config"
)

// purposeConfig represents all of the config info selected from the host and account configs for
// a particular purpose needed to determine legal basis using one of the GDPR enforcement algorithms
type purposeConfig struct {
	PurposeID                  consentconstants.Purpose
	EnforceAlgo                config.TCF2EnforcementAlgo
	EnforcePurpose             bool
	EnforceVendors             bool
	VendorExceptionMap         map[string]struct{}
	BasicEnforcementVendorsMap map[string]struct{}
}

// basicEnforcementVendor returns true if a given bidder/analytics adapter is configured as a basic enforcement vendor
// for the purpose
func (pc *purposeConfig) basicEnforcementVendor(name string) bool {
	if pc.BasicEnforcementVendorsMap == nil {
		return false
	}
	_, found := pc.BasicEnforcementVendorsMap[name]
	return found
}

// vendorException returns true if a given bidder/analytics adapter is configured as a vendor exception
// for the purpose
func (pc *purposeConfig) vendorException(name string) bool {
	if pc.VendorExceptionMap == nil {
		return false
	}
	_, found := pc.VendorExceptionMap[name]
	return found
}
