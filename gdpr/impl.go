package gdpr

import (
	"context"
	"fmt"

	"github.com/prebid/go-gdpr/api"
	tcf1constants "github.com/prebid/go-gdpr/consentconstants"
	consentconstants "github.com/prebid/go-gdpr/consentconstants/tcf2"
	"github.com/prebid/go-gdpr/vendorconsent"
	tcf2 "github.com/prebid/go-gdpr/vendorconsent/tcf2"
	"github.com/prebid/go-gdpr/vendorlist"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// This file implements GDPR permissions for the app.
// For more info, see https://github.com/prebid/prebid-server/issues/501
//
// Nothing in this file is exported. Public APIs can be found in gdpr.go

type Signal int

const (
	SignalAmbiguous Signal = -1
	SignalNo        Signal = 0
	SignalYes       Signal = 1
)

type permissionsImpl struct {
	cfg             config.GDPR
	vendorIDs       map[openrtb_ext.BidderName]uint16
	fetchVendorList map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error)
}

func (p *permissionsImpl) HostCookiesAllowed(ctx context.Context, gdprSignal Signal, consent string) (bool, error) {
	gdprSignal = p.normalizeGDPR(gdprSignal)

	if gdprSignal == SignalNo {
		return true, nil
	}

	return p.allowSync(ctx, uint16(p.cfg.HostVendorID), consent)
}

func (p *permissionsImpl) BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName, gdprSignal Signal, consent string) (bool, error) {
	gdprSignal = p.normalizeGDPR(gdprSignal)

	if gdprSignal == SignalNo {
		return true, nil
	}

	id, ok := p.vendorIDs[bidder]
	if ok {
		return p.allowSync(ctx, id, consent)
	}

	return false, nil
}

func (p *permissionsImpl) PersonalInfoAllowed(ctx context.Context, bidder openrtb_ext.BidderName, PublisherID string, gdprSignal Signal, consent string) (allowPI bool, allowGeo bool, allowID bool, err error) {
	if _, ok := p.cfg.NonStandardPublisherMap[PublisherID]; ok {
		return true, true, true, nil
	}

	gdprSignal = p.normalizeGDPR(gdprSignal)

	if gdprSignal == SignalNo {
		return true, true, true, nil
	}

	if consent == "" && gdprSignal == SignalYes {
		return false, false, false, nil
	}

	if id, ok := p.vendorIDs[bidder]; ok {
		return p.allowPI(ctx, id, consent)
	}

	return p.defaultVendorPermissions()
}

func (p *permissionsImpl) defaultVendorPermissions() (allowPI bool, allowGeo bool, allowID bool, err error) {
	return false, false, false, nil
}

func (p *permissionsImpl) normalizeGDPR(gdprSignal Signal) Signal {
	if gdprSignal != SignalAmbiguous {
		return gdprSignal
	}

	if p.cfg.UsersyncIfAmbiguous {
		return SignalNo
	}

	return SignalYes
}

func (p *permissionsImpl) allowSync(ctx context.Context, vendorID uint16, consent string) (bool, error) {

	if consent == "" {
		return false, nil
	}

	parsedConsent, vendor, err := p.parseVendor(ctx, vendorID, consent)
	if err != nil {
		return false, err
	}

	if vendor == nil {
		return false, nil
	}

	// InfoStorageAccess is the same across TCF 1 and TCF 2
	if parsedConsent.Version() == 2 {
		if !p.cfg.TCF2.Purpose1.Enabled {
			// We are not enforcing purpose 1
			return true, nil
		}
		consent, ok := parsedConsent.(tcf2.ConsentMetadata)
		if !ok {
			err := fmt.Errorf("Unable to access TCF2 parsed consent")
			return false, err
		}
		return p.checkPurpose(consent, vendor, vendorID, consentconstants.InfoStorageAccess), nil
	}
	if vendor.Purpose(consentconstants.InfoStorageAccess) && parsedConsent.PurposeAllowed(consentconstants.InfoStorageAccess) && parsedConsent.VendorConsent(vendorID) {
		return true, nil
	}
	return false, nil
}

func (p *permissionsImpl) allowPI(ctx context.Context, vendorID uint16, consent string) (bool, bool, bool, error) {
	parsedConsent, vendor, err := p.parseVendor(ctx, vendorID, consent)
	if err != nil {
		return false, false, false, err
	}

	if vendor == nil {
		return false, false, false, nil
	}

	if parsedConsent.Version() == 2 {
		if p.cfg.TCF2.Enabled {
			return p.allowPITCF2(parsedConsent, vendor, vendorID)
		}
		if (vendor.Purpose(consentconstants.InfoStorageAccess) || vendor.LegitimateInterest(consentconstants.InfoStorageAccess)) && parsedConsent.PurposeAllowed(consentconstants.InfoStorageAccess) && (vendor.Purpose(consentconstants.PersonalizationProfile) || vendor.LegitimateInterest(consentconstants.PersonalizationProfile)) && parsedConsent.PurposeAllowed(consentconstants.PersonalizationProfile) && parsedConsent.VendorConsent(vendorID) {
			return true, true, true, nil
		}
	} else {
		if (vendor.Purpose(tcf1constants.InfoStorageAccess) || vendor.LegitimateInterest(tcf1constants.InfoStorageAccess)) && parsedConsent.PurposeAllowed(tcf1constants.InfoStorageAccess) && (vendor.Purpose(tcf1constants.AdSelectionDeliveryReporting) || vendor.LegitimateInterest(tcf1constants.AdSelectionDeliveryReporting)) && parsedConsent.PurposeAllowed(tcf1constants.AdSelectionDeliveryReporting) && parsedConsent.VendorConsent(vendorID) {
			return true, true, true, nil
		}
	}
	return false, false, false, nil
}

func (p *permissionsImpl) allowPITCF2(parsedConsent api.VendorConsents, vendor api.Vendor, vendorID uint16) (allowPI bool, allowGeo bool, allowID bool, err error) {
	consent, ok := parsedConsent.(tcf2.ConsentMetadata)
	err = nil
	allowPI = false
	allowGeo = false
	allowID = false
	if !ok {
		err = fmt.Errorf("Unable to access TCF2 parsed consent")
		return
	}
	if p.cfg.TCF2.SpecialPurpose1.Enabled {
		allowGeo = consent.SpecialFeatureOptIn(1) && vendor.SpecialPurpose(1)
	} else {
		allowGeo = true
	}
	for i := 2; i <= 10; i++ {
		if p.checkPurpose(consent, vendor, vendorID, tcf1constants.Purpose(i)) {
			allowID = true
			break
		}
	}
	// Set to true so any purpose check can flip it to false
	allowPI = true
	if p.cfg.TCF2.Purpose1.Enabled {
		allowPI = allowPI && p.checkPurpose(consent, vendor, vendorID, consentconstants.InfoStorageAccess)
	}
	if p.cfg.TCF2.Purpose2.Enabled {
		allowPI = allowPI && p.checkPurpose(consent, vendor, vendorID, consentconstants.BasicAdserving)
	}
	if p.cfg.TCF2.Purpose7.Enabled {
		allowPI = allowPI && p.checkPurpose(consent, vendor, vendorID, consentconstants.AdPerformance)
	}
	return
}

const pubRestrictNotAllowed = 0
const pubRestrictRequireConsent = 1
const pubRestrictRequireLegitInterest = 2

func (p *permissionsImpl) checkPurpose(consent tcf2.ConsentMetadata, vendor api.Vendor, vendorID uint16, purpose tcf1constants.Purpose) bool {
	if purpose == consentconstants.InfoStorageAccess && p.cfg.TCF2.PurposeOneTreatment.Enabled && consent.PurposeOneTreatment() {
		return p.cfg.TCF2.PurposeOneTreatment.AccessAllowed
	}
	if consent.CheckPubRestriction(uint8(purpose), pubRestrictNotAllowed, vendorID) {
		return false
	}
	if consent.CheckPubRestriction(uint8(purpose), pubRestrictRequireConsent, vendorID) {
		return vendor.PurposeStrict(purpose) && consent.PurposeAllowed(purpose) && consent.VendorConsent(vendorID)
	}
	if consent.CheckPubRestriction(uint8(purpose), pubRestrictRequireLegitInterest, vendorID) {
		// Need LITransparency here
		return vendor.LegitimateInterestStrict(purpose) && consent.PurposeLITransparency(purpose) && consent.VendorLegitInterest(vendorID)
	}
	purposeAllowed := vendor.Purpose(purpose) && consent.PurposeAllowed(purpose) && consent.VendorConsent(vendorID)
	legitInterest := vendor.LegitimateInterest(purpose) && consent.PurposeLITransparency(purpose) && consent.VendorLegitInterest(vendorID)

	return purposeAllowed || legitInterest
}

func (p *permissionsImpl) parseVendor(ctx context.Context, vendorID uint16, consent string) (parsedConsent api.VendorConsents, vendor api.Vendor, err error) {
	parsedConsent, err = vendorconsent.ParseString(consent)
	if err != nil {
		err = &ErrorMalformedConsent{
			consent: consent,
			cause:   err,
		}
		return
	}

	version := parsedConsent.Version()
	if version < 1 || version > 2 {
		return
	}
	vendorList, err := p.fetchVendorList[version](ctx, parsedConsent.VendorListVersion())
	if err != nil {
		return
	}

	vendor = vendorList.Vendor(vendorID)
	return
}

// AllowHostCookies represents a GDPR permissions policy with host cookie syncing always allowed
type AllowHostCookies struct {
	*permissionsImpl
}

// HostCookiesAllowed always returns true
func (p *AllowHostCookies) HostCookiesAllowed(ctx context.Context, gdprSignal Signal, consent string) (bool, error) {
	return true, nil
}

// Exporting to allow for easy test setups
type AlwaysAllow struct{}

func (a AlwaysAllow) HostCookiesAllowed(ctx context.Context, gdprSignal Signal, consent string) (bool, error) {
	return true, nil
}

func (a AlwaysAllow) BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName, gdprSignal Signal, consent string) (bool, error) {
	return true, nil
}

func (a AlwaysAllow) PersonalInfoAllowed(ctx context.Context, bidder openrtb_ext.BidderName, PublisherID string, gdprSignal Signal, consent string) (bool, bool, bool, error) {
	return true, true, true, nil
}

// Exporting to allow for easy test setups
type AlwaysFail struct{}

func (a AlwaysFail) HostCookiesAllowed(ctx context.Context, gdprSignal Signal, consent string) (bool, error) {
	return false, nil
}

func (a AlwaysFail) BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName, gdprSignal Signal, consent string) (bool, error) {
	return false, nil
}

func (a AlwaysFail) PersonalInfoAllowed(ctx context.Context, bidder openrtb_ext.BidderName, PublisherID string, gdprSignal Signal, consent string) (bool, bool, bool, error) {
	return false, false, false, nil
}
