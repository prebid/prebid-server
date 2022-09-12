package gdpr

import (
	"github.com/prebid/go-gdpr/api"
	tcf2 "github.com/prebid/go-gdpr/vendorconsent/tcf2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const (
	pubRestrictNotAllowed           = 0
	pubRestrictRequireConsent       = 1
	pubRestrictRequireLegitInterest = 2
)

type flexType uint8

const (
	flexible   flexType = 0
	inflexible flexType = 1
)

// FullEnforcement determines if legal basis is satisfied for a given purpose and bidder using
// the TCF2 full enforcement algorithm. The algorithm is a detailed confirmation that reads the
// GVL, interprets the consent string and performs legal basis analysis necessary to perform a
// privacy-protected activity.
// FullEnforcement implements the PurposeEnforcer interface
type FullEnforcement struct {
	cfg purposeConfig
}

// NewFullEnforcement creates a FullEnforcement object
func NewFullEnforcement(cfg purposeConfig) *FullEnforcement {
	return &FullEnforcement{
		cfg: cfg,
	}
}

// LegalBasis determines if legal basis is satisfied for a given purpose and bidder based on the
// GVL and user consent.
func (fe *FullEnforcement) LegalBasis(vendorInfo VendorInfo, bidder openrtb_ext.BidderName, consent tcf2.ConsentMetadata, overrides Overrides) bool {
	if consent.CheckPubRestriction(uint8(fe.cfg.PurposeID), pubRestrictNotAllowed, vendorInfo.vendorID) {
		return false
	}
	if !fe.cfg.EnforcePurpose && !fe.cfg.EnforceVendors {
		return true
	}
	if fe.cfg.vendorException(bidder) && !overrides.blockVendorExceptions {
		return true
	}

	if consent.CheckPubRestriction(uint8(fe.cfg.PurposeID), pubRestrictRequireConsent, vendorInfo.vendorID) {
		return fe.consentEstablished(inflexible, consent, vendorInfo.vendor, vendorInfo.vendorID)
	}
	if consent.CheckPubRestriction(uint8(fe.cfg.PurposeID), pubRestrictRequireLegitInterest, vendorInfo.vendorID) {
		return fe.legitInterestEstablished(inflexible, consent, vendorInfo.vendor, vendorInfo.vendorID)
	}

	purposeAllowed := fe.consentEstablished(flexible, consent, vendorInfo.vendor, vendorInfo.vendorID)
	legitInterest := fe.legitInterestEstablished(flexible, consent, vendorInfo.vendor, vendorInfo.vendorID)

	return purposeAllowed || legitInterest
}

// consentEstablished determines if consent has been established for a given purpose and bidder
// based on the purpose config, user consent and the GVL.
func (fe *FullEnforcement) consentEstablished(vendorClaim flexType, consent tcf2.ConsentMetadata, vendor api.Vendor, vendorID uint16) bool {
	if vendorClaim == flexible && !vendor.Purpose(fe.cfg.PurposeID) {
		return false
	}
	if vendorClaim == inflexible && !vendor.PurposeStrict(fe.cfg.PurposeID) {
		return false
	}
	if fe.cfg.EnforcePurpose && !consent.PurposeAllowed(fe.cfg.PurposeID) {
		return false
	}
	if fe.cfg.EnforceVendors && !consent.VendorConsent(vendorID) {
		return false
	}
	return true
}

// legitInterestEstablished determines if legitimate interest has been established for a given
// purpose and bidder based on the purpose config, user consent and the GVL.
func (fe *FullEnforcement) legitInterestEstablished(vendorClaim flexType, consent tcf2.ConsentMetadata, vendor api.Vendor, vendorID uint16) bool {
	if vendorClaim == flexible && !vendor.LegitimateInterest(fe.cfg.PurposeID) {
		return false
	}
	if vendorClaim == inflexible && !vendor.LegitimateInterestStrict(fe.cfg.PurposeID) {
		return false
	}
	if fe.cfg.EnforcePurpose && !consent.PurposeLITransparency(fe.cfg.PurposeID) {
		return false
	}
	if fe.cfg.EnforceVendors && !consent.VendorLegitInterest(vendorID) {
		return false
	}
	return true
}
