package gdpr

import (
	"github.com/prebid/go-gdpr/consentconstants"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

// purposeConfig represents all of the config info selected from the host and account configs for
// a particular purpose needed to determine legal basis using one of the GDPR enforcement algorithms
type purposeConfig struct {
	PurposeID                  consentconstants.Purpose
	EnforceAlgo                config.TCF2EnforcementAlgo
	EnforcePurpose             bool
	EnforceVendors             bool
	VendorExceptionMap         map[openrtb_ext.BidderName]struct{}
	BasicEnforcementVendorsMap map[string]struct{}
}

// basicEnforcementVendor returns true if a given bidder is configured as a basic enforcement vendor
// for the purpose
func (pc *purposeConfig) basicEnforcementVendor(bidder openrtb_ext.BidderName) bool {
	if pc.BasicEnforcementVendorsMap == nil {
		return false
	}
	_, found := pc.BasicEnforcementVendorsMap[string(bidder)]
	return found
}

// vendorException returns true if a given bidder is configured as a vendor exception
// for the purpose
func (pc *purposeConfig) vendorException(bidder openrtb_ext.BidderName) bool {
	if pc.VendorExceptionMap == nil {
		return false
	}
	_, found := pc.VendorExceptionMap[bidder]
	return found
}
