package gdpr

import (
	tcf2 "github.com/prebid/go-gdpr/vendorconsent/tcf2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// BasicEnforcement determines legal basis for a given purpose using the TCF2 basic enforcement algorithm
// BasicEnforcement implements the PurposeEnforcer interface
type BasicEnforcement struct {
	cfg purposeConfig
}

func NewBasicEnforcement(cfg purposeConfig) *BasicEnforcement {
	return &BasicEnforcement{
		cfg: cfg,
	}
}

// LegalBasis...
func (be *BasicEnforcement) LegalBasis(vendorInfo VendorInfo, bidder openrtb_ext.BidderName, consent tcf2.ConsentMetadata) bool {
	if !be.cfg.EnforcePurpose && !be.cfg.EnforceVendors {
		return true
	}
	if be.cfg.vendorException(bidder) {
		return true
	}
	if be.cfg.BasicEnforcementVendor(bidder) {
		return true
	}
	if be.cfg.EnforcePurpose && !consent.PurposeAllowed(be.cfg.PurposeID) {
		return false
	}
	if !be.cfg.EnforceVendors {
		return true
	}
	if vendorInfo.vendor.Purpose(be.cfg.PurposeID) && consent.VendorConsent(vendorInfo.vendorID) {
		return true
	}
	return false
}
