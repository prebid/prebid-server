package gdpr

import (
	"github.com/prebid/go-gdpr/api"
	"github.com/prebid/go-gdpr/consentconstants"
	tcf2 "github.com/prebid/go-gdpr/vendorconsent/tcf2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type PurposeEnforcer interface {
	LegalBasis(vendorInfo VendorInfo, bidder openrtb_ext.BidderName, consent tcf2.ConsentMetadata) bool
}

type PurposeEnforcerBuilder func(cfg purposeConfig, downgraded bool) (PurposeEnforcer)

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
	TCF2BasicEnforcement string = "basic"
	TCF2FullEnforcement  string = "full"
)

type purposeConfig struct {
	PurposeID                  consentconstants.Purpose
	EnforceAlgo                string //TCF2Enforcement
	EnforcePurpose             bool
	EnforceVendors             bool
	VendorExceptionMap         map[openrtb_ext.BidderName]struct{}
	BasicEnforcementVendorsMap map[string]struct{} // currently map[string]struct{} in agg cfg, only needed for BASIC enforcement
}

func (pc *purposeConfig) BasicEnforcementVendor(bidder openrtb_ext.BidderName) bool {
	if pc.BasicEnforcementVendorsMap == nil {
		return false
	}
	if _, found := pc.BasicEnforcementVendorsMap[string(bidder)]; found {
		return true
	}
	return false
}

func (pc *purposeConfig) vendorException(bidder openrtb_ext.BidderName) bool {
	if pc.VendorExceptionMap == nil {
		return false
	}
	if _, found := pc.VendorExceptionMap[bidder]; found {
		return true
	}
	return false
}

// called from the TCF2Service and might be injected into the service as a builder to improve testability
func NewPurposeEnforcer(cfg purposeConfig, downgraded bool) PurposeEnforcer {
	if cfg.EnforceAlgo == TCF2BasicEnforcement {
		return NewBasicEnforcement(cfg)
	} else if downgraded {
		return NewBasicEnforcement(cfg)
	}
	return NewFullEnforcement(cfg)
}
