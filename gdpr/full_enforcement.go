package gdpr

import (
	"github.com/prebid/go-gdpr/api"
	"github.com/prebid/go-gdpr/consentconstants"
	tcf2 "github.com/prebid/go-gdpr/vendorconsent/tcf2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const (
	pubRestrictNotAllowed           = 0
	pubRestrictRequireConsent       = 1
	pubRestrictRequireLegitInterest = 2
)

type FullEnforcementBuilder func()

// FullEnforcement determines legal basis for a given purpose using the TCF2 full enforcement algorithm
// FullEnforcement implements the PurposeEnforcer interface
type FullEnforcement struct {
	purposeCfg purposeConfig
}

func NewFullEnforcement(cfg purposeConfig) *FullEnforcement {
	return &FullEnforcement{
		purposeCfg: cfg,
	}
}

func (fe *FullEnforcement) LegalBasis(vendorInfo VendorInfo, bidder openrtb_ext.BidderName, consent tcf2.ConsentMetadata) bool {
	if consent.CheckPubRestriction(uint8(fe.purposeCfg.PurposeID), pubRestrictNotAllowed, vendorInfo.vendorID) {
		return false
	}

	//TODO: is this new if statement correct?
	//If so, add comment as to why this is here in the full enforcement module
	if !fe.PurposeEnforced() && !fe.purposeCfg.EnforceVendors {
		return true
	}

	if fe.purposeCfg.VendorExceptionMap != nil {
		if _, found := fe.purposeCfg.VendorExceptionMap[bidder]; found {
			return true
		}
	}

	purposeAllowed := fe.consentEstablished(consent, vendorInfo.vendor, vendorInfo.vendorID, fe.purposeCfg.PurposeID, fe.purposeCfg.EnforceVendors /*, weakVendorEnforcement*/)
	legitInterest := fe.legitInterestEstablished(consent, vendorInfo.vendor, vendorInfo.vendorID, fe.purposeCfg.PurposeID, fe.purposeCfg.EnforceVendors /*, weakVendorEnforcement*/)

	if consent.CheckPubRestriction(uint8(fe.purposeCfg.PurposeID), pubRestrictRequireConsent, vendorInfo.vendorID) {
		return purposeAllowed
	}
	if consent.CheckPubRestriction(uint8(fe.purposeCfg.PurposeID), pubRestrictRequireLegitInterest, vendorInfo.vendorID) {
		// Need LITransparency here
		return legitInterest
	}

	return purposeAllowed || legitInterest
}

func (fe *FullEnforcement) PurposeEnforced() bool {
	if fe.purposeCfg.EnforcePurpose == TCF2FullEnforcement {
		return true
	}
	return false
}

func (fe *FullEnforcement) consentEstablished(consent tcf2.ConsentMetadata, vendor api.Vendor, vendorID uint16, purpose consentconstants.Purpose, enforceVendors /*, weakVendorEnforcement*/ bool) bool {
	if fe.PurposeEnforced() && !consent.PurposeAllowed(purpose) {
		return false
	}
	if !enforceVendors {
		return true
	}
	if vendor.Purpose(purpose) && consent.VendorConsent(vendorID) {
		return true
	}
	return false
}

func (fe *FullEnforcement) legitInterestEstablished(consent tcf2.ConsentMetadata, vendor api.Vendor, vendorID uint16, purpose consentconstants.Purpose, enforceVendors /*, weakVendorEnforcement*/ bool) bool {
	if fe.PurposeEnforced() && !consent.PurposeLITransparency(purpose) {
		return false
	}
	if !enforceVendors {
		return true
	}
	if vendor.LegitimateInterest(purpose) && consent.VendorLegitInterest(vendorID) {
		return true
	}
	return false
}
