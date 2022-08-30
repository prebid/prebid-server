package gdpr

import (
	"context"
	"errors"
	"fmt"

	"github.com/prebid/go-gdpr/api"
	"github.com/prebid/go-gdpr/consentconstants"
	"github.com/prebid/go-gdpr/vendorconsent"
	tcf2 "github.com/prebid/go-gdpr/vendorconsent/tcf2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const noBidder = ""

type permissionsImpl struct {
	// global
	fetchVendorList        VendorListFetcher
	gdprDefaultValue       string
	hostVendorID           int
	nonStandardPublishers  map[string]struct{}
	purposeEnforcerBuilder PurposeEnforcerBuilder
	vendorIDs              map[openrtb_ext.BidderName]uint16
	// request-specific
	aliasGVLIDs map[string]uint16
	cfg         TCF2ConfigReader
	consent     string
	gdprSignal  Signal
	publisherID string
}

// called from /cookie_sync & /setuid
// vendor exception should not apply -- should we allow a host to specify themselves as a vendor exception for purpose 1?
// if yes, should we validate at startup and throw a warning?
// if no, should we add validation logic that fails hard? That seems like the right place to do it rather than ignoring the value in the GDPR logic
func (p *permissionsImpl) HostCookiesAllowed(ctx context.Context) (bool, error) {
	if p.gdprSignal != SignalYes {
		return true, nil
	}

	return p.allowSync(ctx, uint16(p.hostVendorID), noBidder, false)
}

// called from /cookie_sync
// yes vendor exception applies
func (p *permissionsImpl) BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName) (bool, error) {
	if p.gdprSignal != SignalYes {
		return true, nil
	}

	id, ok := p.vendorIDs[bidder]
	if ok {
		vendorException := p.cfg.PurposeVendorException(consentconstants.Purpose(1), bidder)
		return p.allowSync(ctx, id, bidder, vendorException)
	}

	return false, nil
}

// AuctionActivitiesAllowed implements the Permissions interface
// It determines whether auction activities are allowed for a given bidder
func (p *permissionsImpl) AuctionActivitiesAllowed(ctx context.Context, bidderCoreName openrtb_ext.BidderName, bidder openrtb_ext.BidderName) (permissions AuctionPermissions, err error) {
	if _, ok := p.nonStandardPublishers[p.publisherID]; ok {
		return AllowAll, nil
	}

	if p.gdprSignal != SignalYes {
		return AllowAll, nil
	}

	if p.consent == "" {
		return DenyAll, nil
	}

	// note the bidder here is guaranteed to be enabled
	vendorID, vendorFound := p.resolveVendorId(bidderCoreName, bidder)
	weakVendorEnforcement := p.cfg.BasicEnforcementVendor(bidder)

	if !vendorFound && !weakVendorEnforcement {
		return DenyAll, nil
	}

	parsedConsent, vendor, err := p.parseVendor(ctx, vendorID, p.consent)
	if err != nil {
		return DenyAll, err
	}

	// vendor will be nil if not a valid TCF2 consent string
	if vendor == nil {
		if weakVendorEnforcement && parsedConsent.Version() == 2 {
			vendor = vendorTrue{}
		} else {
			return DenyAll, nil
		}
	}

	if !p.cfg.IsEnabled() {
		return AllowBidRequestOnly, nil
	}

	consentMeta, ok := parsedConsent.(tcf2.ConsentMetadata)
	if !ok {
		err = fmt.Errorf("Unable to access TCF2 parsed consent")
		return DenyAll, err
	}

	vendorInfo := VendorInfo{vendorID: vendorID, vendor: vendor}
	permissions = AuctionPermissions{}
	permissions.AllowBidRequest = p.allowBidRequest(bidderCoreName, consentMeta, vendorInfo)
	permissions.PassGeo = p.allowGeo(bidderCoreName, consentMeta, vendor)
	permissions.PassID = p.allowID(bidderCoreName, consentMeta, vendorInfo)

	return permissions, nil
}

func (p *permissionsImpl) resolveVendorId(bidderCoreName openrtb_ext.BidderName, bidder openrtb_ext.BidderName) (id uint16, ok bool) {
	if id, ok = p.aliasGVLIDs[string(bidder)]; ok {
		return id, ok
	}

	id, ok = p.vendorIDs[bidderCoreName]

	return id, ok
}

func (p *permissionsImpl) allowSync(ctx context.Context, vendorID uint16, bidder openrtb_ext.BidderName, vendorException bool) (bool, error) {
	if p.consent == "" {
		return false, nil
	}

	parsedConsent, vendor, err := p.parseVendor(ctx, vendorID, p.consent)
	if err != nil {
		return false, err
	}

	if vendor == nil {
		return false, nil
	}

	if !p.cfg.PurposeEnforced(consentconstants.Purpose(1)) {
		return true, nil
	}
	consentMeta, ok := parsedConsent.(tcf2.ConsentMetadata)
	if !ok {
		err := errors.New("Unable to access TCF2 parsed consent")
		return false, err
	}

	if p.cfg.PurposeOneTreatmentEnabled() && consentMeta.PurposeOneTreatment() {
		return p.cfg.PurposeOneTreatmentAccessAllowed(), nil
	}

	purpose := consentconstants.Purpose(1)
	enforcer := p.purposeEnforcerBuilder(purpose, bidder)

	vendorInfo := VendorInfo{vendorID: vendorID, vendor: vendor}
	if enforcer.LegalBasis(vendorInfo, bidder, consentMeta) { //-->pass vendorexception? or dont care
		return true, nil
	}
	return false, nil
}

func (p *permissionsImpl) allowBidRequest(bidder openrtb_ext.BidderName, consentMeta tcf2.ConsentMetadata, vendorInfo VendorInfo) bool {
	purpose := consentconstants.Purpose(2)
	enforcer := p.purposeEnforcerBuilder(purpose, bidder)

	// this function will return true if purpose 2 is NOT enforced
	if enforcer.LegalBasis(vendorInfo, bidder, consentMeta) {
		return true
	}
	return false
}

func (p *permissionsImpl) allowGeo(bidder openrtb_ext.BidderName, consentMeta tcf2.ConsentMetadata, vendor api.Vendor) bool {
	if !p.cfg.FeatureOneEnforced() {
		return true
	}
	if p.cfg.FeatureOneVendorException(bidder) {
		return true
	}

	weakVendorEnforcement := p.cfg.BasicEnforcementVendor(bidder)
	return consentMeta.SpecialFeatureOptIn(1) && (vendor.SpecialFeature(1) || weakVendorEnforcement)
}

func (p *permissionsImpl) allowID(bidder openrtb_ext.BidderName, consentMeta tcf2.ConsentMetadata, vendorInfo VendorInfo) bool {
	for i := 2; i <= 10; i++ {
		purpose := consentconstants.Purpose(i)
		enforcer := p.purposeEnforcerBuilder(purpose, bidder)

		if p.cfg.PurposeEnforced(purpose) && enforcer.LegalBasis(vendorInfo, bidder, consentMeta) {
			return true
		}
	}

	return false
}

func (p *permissionsImpl) parseVendor(ctx context.Context, vendorID uint16, consent string) (parsedConsent api.VendorConsents, vendor api.Vendor, err error) {
	parsedConsent, err = vendorconsent.ParseString(consent)
	if err != nil {
		err = &ErrorMalformedConsent{
			Consent: consent,
			Cause:   err,
		}
		return
	}

	version := parsedConsent.Version()
	if version != 2 {
		return
	}

	vendorList, err := p.fetchVendorList(ctx, parsedConsent.VendorListVersion())
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
func (p *AllowHostCookies) HostCookiesAllowed(ctx context.Context) (bool, error) {
	return true, nil
}

// Exporting to allow for easy test setups
type AlwaysAllow struct{}

func (a AlwaysAllow) HostCookiesAllowed(ctx context.Context) (bool, error) {
	return true, nil
}
func (a AlwaysAllow) BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName) (bool, error) {
	return true, nil
}
func (a AlwaysAllow) AuctionActivitiesAllowed(ctx context.Context, bidderCoreName openrtb_ext.BidderName, bidder openrtb_ext.BidderName) (permissions AuctionPermissions, err error) {
	return AllowAll, nil
}

// vendorTrue claims everything.
type vendorTrue struct{}

func (v vendorTrue) Purpose(purposeID consentconstants.Purpose) bool {
	return true
}
func (v vendorTrue) PurposeStrict(purposeID consentconstants.Purpose) bool {
	return true
}
func (v vendorTrue) LegitimateInterest(purposeID consentconstants.Purpose) bool {
	return true
}
func (v vendorTrue) LegitimateInterestStrict(purposeID consentconstants.Purpose) bool {
	return true
}
func (v vendorTrue) SpecialFeature(featureID consentconstants.SpecialFeature) (hasSpecialFeature bool) {
	return true
}
func (v vendorTrue) SpecialPurpose(purposeID consentconstants.Purpose) (hasSpecialPurpose bool) {
	return true
}
