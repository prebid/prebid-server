package gdpr

import (
	tcf2 "github.com/prebid/go-gdpr/vendorconsent/tcf2"
)

// BasicEnforcement determines if legal basis is satisfied for a given purpose and bidder/analytics adapter using
// the TCF2 basic enforcement algorithm. The algorithm is a high-level mode of consent confirmation
// that looks for a good-faith indication that the user has provided consent or legal basis signals
// necessary to perform a privacy-protected activity. The algorithm does not involve the GVL.
// BasicEnforcement implements the PurposeEnforcer interface
type BasicEnforcement struct {
	cfg purposeConfig
}

// LegalBasis determines if legal basis is satisfied for a given purpose and bidder/analytics adapter based on user consent
// and legal basis signals.
func (be *BasicEnforcement) LegalBasis(vendorInfo VendorInfo, name string, consent tcf2.ConsentMetadata, overrides Overrides) bool {
	enforcePurpose, enforceVendors := be.applyEnforceOverrides(overrides)

	if !enforcePurpose && !enforceVendors {
		return true
	}
	if be.cfg.vendorException(name) && !overrides.blockVendorExceptions {
		return true
	}
	if !enforcePurpose && be.cfg.basicEnforcementVendor(name) {
		return true
	}
	if enforcePurpose && consent.PurposeAllowed(be.cfg.PurposeID) && be.cfg.basicEnforcementVendor(name) {
		return true
	}
	if enforcePurpose && consent.PurposeLITransparency(be.cfg.PurposeID) && overrides.allowLITransparency {
		return true
	}
	if enforcePurpose && !consent.PurposeAllowed(be.cfg.PurposeID) {
		return false
	}
	if !enforceVendors {
		return true
	}
	return consent.VendorConsent(vendorInfo.vendorID)
}

// applyEnforceOverrides returns the enforce purpose and enforce vendor configuration values unless
// those values have been overridden, in which case they return true
func (be *BasicEnforcement) applyEnforceOverrides(overrides Overrides) (enforcePurpose, enforceVendors bool) {
	enforcePurpose = be.cfg.EnforcePurpose
	if overrides.enforcePurpose {
		enforcePurpose = true
	}
	enforceVendors = be.cfg.EnforceVendors
	if overrides.enforceVendors {
		enforceVendors = true
	}
	return
}
