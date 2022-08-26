package gdpr

import (
	tcf2 "github.com/prebid/go-gdpr/vendorconsent/tcf2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// BasicEnforcement determines if legal basis is satisfied for a given purpose and bidder using
// the TCF2 basic enforcement algorithm. The algorithm is a high-level mode of consent confirmation
// that looks for a good-faith indication that the user has provided consent or legal basis signals
// necessary to perform a privacy-protected activity. The algorithm does not involve the GVL.
// BasicEnforcement implements the PurposeEnforcer interface
type BasicEnforcement struct {
	cfg purposeConfig
}

// NewBasicEnforcement creates a BasicEnforcement object
func NewBasicEnforcement(cfg purposeConfig) *BasicEnforcement {
	return &BasicEnforcement{
		cfg: cfg,
	}
}

// LegalBasis determines if legal basis is satisfied for a given purpose and bidder based on user consent
// and legal basis signals.
func (be *BasicEnforcement) LegalBasis(vendorInfo VendorInfo, bidder openrtb_ext.BidderName, consent tcf2.ConsentMetadata) bool {
	if !be.cfg.EnforcePurpose && !be.cfg.EnforceVendors {
		return true
	}
	if be.cfg.vendorException(bidder) {
		return true
	}
	if be.cfg.basicEnforcementVendor(bidder) {
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
