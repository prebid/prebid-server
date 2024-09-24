package gdpr

import (
	tcf2 "github.com/prebid/go-gdpr/vendorconsent/tcf2"
)

const (
	pubRestrictNotAllowed           = 0
	pubRestrictRequireConsent       = 1
	pubRestrictRequireLegitInterest = 2
)

// FullEnforcement determines if legal basis is satisfied for a given purpose and bidde/analytics adapterr using
// the TCF2 full enforcement algorithm. The algorithm is a detailed confirmation that reads the
// GVL, interprets the consent string and performs legal basis analysis necessary to perform a
// privacy-protected activity.
// FullEnforcement implements the PurposeEnforcer interface
type FullEnforcement struct {
	cfg purposeConfig
}

// LegalBasis determines if legal basis is satisfied for a given purpose and bidder/analytics adapter based on the
// vendor claims in the GVL, publisher restrictions and user consent.
func (fe *FullEnforcement) LegalBasis(vendorInfo VendorInfo, name string, consent tcf2.ConsentMetadata, overrides Overrides) bool {
	enforcePurpose, enforceVendors := fe.applyEnforceOverrides(overrides)

	if consent.CheckPubRestriction(uint8(fe.cfg.PurposeID), pubRestrictNotAllowed, vendorInfo.vendorID) {
		return false
	}
	if !enforcePurpose && !enforceVendors {
		return true
	}
	if fe.cfg.vendorException(name) && !overrides.blockVendorExceptions {
		return true
	}

	purposeAllowed := fe.consentEstablished(consent, vendorInfo, enforcePurpose, enforceVendors)
	legitInterest := fe.legitInterestEstablished(consent, vendorInfo, enforcePurpose, enforceVendors)

	if consent.CheckPubRestriction(uint8(fe.cfg.PurposeID), pubRestrictRequireConsent, vendorInfo.vendorID) {
		return purposeAllowed
	}
	if consent.CheckPubRestriction(uint8(fe.cfg.PurposeID), pubRestrictRequireLegitInterest, vendorInfo.vendorID) {
		return legitInterest
	}

	return purposeAllowed || legitInterest
}

// applyEnforceOverrides returns the enforce purpose and enforce vendor configuration values unless
// those values have been overridden, in which case they return true
func (fe *FullEnforcement) applyEnforceOverrides(overrides Overrides) (enforcePurpose, enforceVendors bool) {
	enforcePurpose = fe.cfg.EnforcePurpose
	if overrides.enforcePurpose {
		enforcePurpose = true
	}
	enforceVendors = fe.cfg.EnforceVendors
	if overrides.enforceVendors {
		enforceVendors = true
	}
	return
}

// consentEstablished determines if consent has been established for a given purpose and bidder
// based on the purpose config, user consent and the GVL. For consent to be established, the vendor
// must declare the purpose as either consent or flex and the user must consent in accordance with
// the purpose configs.
func (fe *FullEnforcement) consentEstablished(consent tcf2.ConsentMetadata, vi VendorInfo, enforcePurpose bool, enforceVendors bool) bool {
	if vi.vendor == nil {
		return false
	}
	if !vi.vendor.Purpose(fe.cfg.PurposeID) {
		return false
	}
	if enforcePurpose && !consent.PurposeAllowed(fe.cfg.PurposeID) {
		return false
	}
	if enforceVendors && !consent.VendorConsent(vi.vendorID) {
		return false
	}
	return true
}

// legitInterestEstablished determines if legitimate interest has been established for a given
// purpose and bidder based on the purpose config, user consent and the GVL. For consent to be
// established, the vendor must declare the purpose as either legit interest or flex and the user
// must have been provided notice for the legit interest basis in accordance with the purpose configs.
func (fe *FullEnforcement) legitInterestEstablished(consent tcf2.ConsentMetadata, vi VendorInfo, enforcePurpose bool, enforceVendors bool) bool {
	if vi.vendor == nil {
		return false
	}
	if !vi.vendor.LegitimateInterest(fe.cfg.PurposeID) {
		return false
	}
	if enforcePurpose && !consent.PurposeLITransparency(fe.cfg.PurposeID) {
		return false
	}
	if enforceVendors && !consent.VendorLegitInterest(vi.vendorID) {
		return false
	}
	return true
}
