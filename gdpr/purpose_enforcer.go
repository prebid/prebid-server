package gdpr

import (
	"github.com/prebid/go-gdpr/api"
	"github.com/prebid/go-gdpr/consentconstants"
	tcf2 "github.com/prebid/go-gdpr/vendorconsent/tcf2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// PurposeEnforcer represents the enforcement strategy for determining if legal basis is achieved for a purpose
type PurposeEnforcer interface {
	LegalBasis(vendorInfo VendorInfo, bidder openrtb_ext.BidderName, consent tcf2.ConsentMetadata, overrides Overrides) bool
}

// PurposeEnforcerBuilder generates an instance of PurposeEnforcer for a given purpose and bidder
type PurposeEnforcerBuilder func(p consentconstants.Purpose, bidder openrtb_ext.BidderName) PurposeEnforcer

// Overrides specifies enforcement algorithm rule adjustments
type Overrides struct {
	allowLITransparency   bool
	blockVendorExceptions bool
}

type BidderInfo struct {
	bidderCoreName openrtb_ext.BidderName
	bidder         openrtb_ext.BidderName
}
type VendorInfo struct {
	vendorID uint16
	vendor   api.Vendor
}

const (
	TCF2BasicEnforcement string = "basic"
	TCF2FullEnforcement  string = "full"
)

// PurposeEnforcers holds the full and basic enforcers for a purpose
type PurposeEnforcers struct {
	Full  PurposeEnforcer
	Basic PurposeEnforcer
}

// NewPurposeEnforcerBuilder creates a new instance of PurposeEnforcerBuilder. This function uses
// closures so that any enforcer generated by the returned builder may use the config and also be
// cached and reused within a request context
func NewPurposeEnforcerBuilder(cfg TCF2ConfigReader) PurposeEnforcerBuilder {
	cachedEnforcers := make([]PurposeEnforcers, 10)

	return func(purpose consentconstants.Purpose, bidder openrtb_ext.BidderName) PurposeEnforcer {
		index := purpose - 1

		basicEnforcementVendor := cfg.BasicEnforcementVendor(bidder)
		if purpose == consentconstants.Purpose(1) {
			basicEnforcementVendor = false
		}

		enforceAlgo := cfg.PurposeEnforcementAlgo(purpose)
		downgraded := isDowngraded(enforceAlgo, basicEnforcementVendor)

		if enforceAlgo == TCF2BasicEnforcement || downgraded {
			if cachedEnforcers[index].Basic != nil {
				return cachedEnforcers[index].Basic
			}

			purposeCfg := purposeConfig{
				PurposeID:                  purpose,
				EnforceAlgo:                enforceAlgo,
				EnforcePurpose:             cfg.PurposeEnforced(purpose),
				EnforceVendors:             cfg.PurposeEnforcingVendors(purpose),
				VendorExceptionMap:         cfg.PurposeVendorExceptions(purpose),
				BasicEnforcementVendorsMap: cfg.BasicEnforcementVendors(),
			}

			enforcer := &BasicEnforcement{
				cfg: purposeCfg,
			}
			cachedEnforcers[index].Basic = enforcer
			return enforcer
		} else {
			if cachedEnforcers[index].Full != nil {
				return cachedEnforcers[index].Full
			}

			purposeCfg := purposeConfig{
				PurposeID:                  purpose,
				EnforceAlgo:                enforceAlgo,
				EnforcePurpose:             cfg.PurposeEnforced(purpose),
				EnforceVendors:             cfg.PurposeEnforcingVendors(purpose),
				VendorExceptionMap:         cfg.PurposeVendorExceptions(purpose),
				BasicEnforcementVendorsMap: cfg.BasicEnforcementVendors(),
			}

			enforcer := &FullEnforcement{
				cfg: purposeCfg,
			}
			cachedEnforcers[index].Full = enforcer
			return enforcer
		}
	}
}

// isDowngraded determines if the enforcement algorithm used to determine legal basis for a
// purpose should be downgraded from full enforcement to basic
func isDowngraded(enforceAlgo string, basicEnforcementVendor bool) bool {
	if enforceAlgo == TCF2FullEnforcement && basicEnforcementVendor {
		return true
	}
	return false
}
