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

// FullEnforcement determines if legal basis is satisfied for a given purpose and bidder using
// the TCF2 full enforcement algorithm. The algorithm is a detailed confirmation that reads the
// GVL, interprets the consent string and performs legal basis analysis necessary to perform a
// privacy-protected activity.
// FullEnforcement implements the PurposeEnforcer interface
type FullEnforcement struct {
	cfg purposeConfig
}

// LegalBasis determines if legal basis is satisfied for a given purpose and bidder based on the
// vendor claims in the GVL, publisher restrictions and user consent.
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

	purposeAllowed := fe.consentEstablished(consent, vendorInfo.vendor, vendorInfo.vendorID)
	legitInterest := fe.legitInterestEstablished(consent, vendorInfo.vendor, vendorInfo.vendorID)

	if consent.CheckPubRestriction(uint8(fe.cfg.PurposeID), pubRestrictRequireConsent, vendorInfo.vendorID) {
		return purposeAllowed
	}
	if consent.CheckPubRestriction(uint8(fe.cfg.PurposeID), pubRestrictRequireLegitInterest, vendorInfo.vendorID) {
		return legitInterest
	}

	return purposeAllowed || legitInterest
}

// consentEstablished determines if consent has been established for a given purpose and bidder
// based on the purpose config, user consent and the GVL. For consent to be established, the vendor
// must declare the purpose as either consent or flex and the user must consent in accordance with
// the purpose configs.
func (fe *FullEnforcement) consentEstablished(consent tcf2.ConsentMetadata, vendor api.Vendor, vendorID uint16) bool {
	if !vendor.Purpose(fe.cfg.PurposeID) {
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
// purpose and bidder based on the purpose config, user consent and the GVL. For consent to be
// established, the vendor must declare the purpose as either legit interest or flex and the user
// must have been provided notice for the legit interest basis in accordance with the purpose configs.
func (fe *FullEnforcement) legitInterestEstablished(consent tcf2.ConsentMetadata, vendor api.Vendor, vendorID uint16) bool {
	if !vendor.LegitimateInterest(fe.cfg.PurposeID) {
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
