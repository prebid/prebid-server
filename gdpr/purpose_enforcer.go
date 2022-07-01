package gdpr

import (
	"github.com/prebid/go-gdpr/api"
	"github.com/prebid/go-gdpr/consentconstants"
	tcf2 "github.com/prebid/go-gdpr/vendorconsent/tcf2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type PurposeEnforcer interface {
	LegalBasis(vendorInfo VendorInfo, bidderInfo BidderInfo, consent tcf2.ConsentMetadata) bool
	PurposeEnforced() bool
}

type BidderInfo struct {
	bidderCoreName openrtb_ext.BidderName
	bidder         openrtb_ext.BidderName
}
type VendorInfo struct {
	vendorID uint16
	vendor   api.Vendor
}

// type TCF2Enforcement string

const (
	// TCF2BasicEnforcement TCF2Enforcement = "basic"
	// TCF2FullEnforcement  TCF2Enforcement = "full"
	// TCF2NoEnforcement    TCF2Enforcement = "no"
	TCF2BasicEnforcement string = "basic"
	TCF2FullEnforcement  string = "full"
	TCF2NoEnforcement    string = "no"
)

type purposeConfig struct {
	PurposeID                  consentconstants.Purpose
	EnforcePurpose             string //TCF2Enforcement
	EnforceVendors             bool
	VendorExceptionMap         map[openrtb_ext.BidderName]struct{}
	BasicEnforcementVendorsMap map[string]struct{} // currently map[string]struct{} in agg cfg, only needed for BASIC enforcement
}

// called from the TCF2Service and might be injected into the service as a builder to improve testability
func NewPurposeEnforcer(cfg purposeConfig, downgraded bool) PurposeEnforcer {
	if cfg.EnforcePurpose == TCF2FullEnforcement && !downgraded {
		return NewFullEnforcement(cfg)
		// } else if cfg.EnforcePurpose == TCF2BasicEnforcement {
		// 	return NewBasicEnforcement(cfg)
		// } else if downgraded {
		// 	return NewBasicEnforcement(cfg)
	}

	return NewNoEnforcement(cfg)
}
