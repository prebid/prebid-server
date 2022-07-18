package gdpr

import (
	"fmt"
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

	fmt.Println(be.cfg)
	if !be.PurposeEnforced() && !be.cfg.EnforceVendors {
		return true
	}
	if be.cfg.vendorException(bidder) {
		return true
	}
	if be.cfg.BasicEnforcementVendor(bidder) {
		return true
	}
	if be.PurposeEnforced() && !consent.PurposeAllowed(be.cfg.PurposeID) {
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

// PurposeEnforced...
func (be *BasicEnforcement) PurposeEnforced() bool {
	if be.cfg.EnforcePurpose == TCF2FullEnforcement || be.cfg.EnforcePurpose == TCF2BasicEnforcement {
		return true
	}
	return false
}

// vendorException...
// func (be *BasicEnforcement) vendorException(bidder openrtb_ext.BidderName) bool {
// 	if be.cfg.VendorExceptionMap != nil {
// 		if _, found := be.cfg.VendorExceptionMap[bidder]; found {
// 			return true
// 		}
// 	}
// 	return false
// }
