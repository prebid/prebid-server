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

const noBidder openrtb_ext.BidderName = ""

// permissionsImpl contains global and request-specific GDPR config data and is used to determine
// whether various cookie sync and auction activities are permitted for a request
// permissionsImpl implements the Permissions interface
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

// HostCookiesAllowed determines whether the host is allowed to set cookies on the user's device
func (p *permissionsImpl) HostCookiesAllowed(ctx context.Context) (bool, error) {
	if p.gdprSignal != SignalYes {
		return true, nil
	}

	return p.allowSync(ctx, uint16(p.hostVendorID), noBidder, false)
}

// BidderSyncAllowed determines whether a given bidder is allowed to perform a cookie sync
func (p *permissionsImpl) BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName) (bool, error) {
	if p.gdprSignal != SignalYes {
		return true, nil
	}

	id, ok := p.vendorIDs[bidder]
	if ok {
		vendorExceptions := p.cfg.PurposeVendorExceptions(consentconstants.Purpose(1))
		_, vendorException := vendorExceptions[bidder]
		return p.allowSync(ctx, id, bidder, vendorException)
	}

	return false, nil
}

// AuctionActivitiesAllowed determines whether auction activities are permitted for a given bidder
func (p *permissionsImpl) AuctionActivitiesAllowed(ctx context.Context, bidderCoreName openrtb_ext.BidderName, bidder openrtb_ext.BidderName) (permissions AuctionPermissions, err error) {
	if _, ok := p.nonStandardPublishers[p.publisherID]; ok {
		return AllowAll, nil
	}

	if p.gdprSignal != SignalYes {
		return AllowAll, nil
	}

	if p.consent == "" {
		return p.defaultPermissions(), nil
	}

	// note the bidder here is guaranteed to be enabled
	vendorID, vendorFound := p.resolveVendorID(bidderCoreName, bidder)
	basicEnforcementVendors := p.cfg.BasicEnforcementVendors()
	_, weakVendorEnforcement := basicEnforcementVendors[string(bidder)]

	if !vendorFound && !weakVendorEnforcement {
		return DenyAll, nil
	}

	parsedConsent, vendor, err := p.parseVendor(ctx, vendorID, p.consent)
	if err != nil {
		return p.defaultPermissions(), err
	}

	// vendor will be nil if not a valid TCF2 consent string
	if vendor == nil {
		if weakVendorEnforcement && parsedConsent.Version() == 2 {
			vendor = vendorTrue{}
		} else {
			return p.defaultPermissions(), nil
		}
	}

	if !p.cfg.IsEnabled() {
		return AllowBidRequestOnly, nil
	}

	consentMeta, ok := parsedConsent.(tcf2.ConsentMetadata)
	if !ok {
		err = fmt.Errorf("Unable to access TCF2 parsed consent")
		return p.defaultPermissions(), err
	}

	vendorInfo := VendorInfo{vendorID: vendorID, vendor: vendor}
	permissions = AuctionPermissions{}
	permissions.AllowBidRequest = p.allowBidRequest(bidderCoreName, consentMeta, vendorInfo)
	permissions.PassGeo = p.allowGeo(bidderCoreName, consentMeta, vendor)
	permissions.PassID = p.allowID(bidderCoreName, consentMeta, vendorInfo)

	return permissions, nil
}

// defaultPermissions returns a permissions object that denies passing user IDs while
// allowing passing geo information and sending bid requests based on whether purpose 2
// and feature one are enforced respectively
// if the consent string is empty or malformed we should use the default permissions
func (p *permissionsImpl) defaultPermissions() AuctionPermissions {
	perms := AuctionPermissions{}

	if !p.cfg.PurposeEnforced(consentconstants.Purpose(2)) {
		perms.AllowBidRequest = true
	}
	if !p.cfg.FeatureOneEnforced() {
		perms.PassGeo = true
	}
	return perms
}

// resolveVendorID gets the vendor ID for the specified bidder from either the alias GVL IDs
// provided in the request or from the bidder configs loaded at startup
func (p *permissionsImpl) resolveVendorID(bidderCoreName openrtb_ext.BidderName, bidder openrtb_ext.BidderName) (id uint16, ok bool) {
	if id, ok = p.aliasGVLIDs[string(bidder)]; ok {
		return id, ok
	}

	id, ok = p.vendorIDs[bidderCoreName]

	return id, ok
}

// allowSync computes cookie sync activity legal basis for a given bidder using the enforcement
// algorithms selected by the purpose enforcer builder
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
	if enforcer.LegalBasis(vendorInfo, bidder, consentMeta, Overrides{blockVendorExceptions: !vendorException}) {
		return true, nil
	}
	return false, nil
}

// allowBidRequest computes legal basis for a given bidder using the enforcement algorithms selected
// by the purpose enforcer builder
func (p *permissionsImpl) allowBidRequest(bidder openrtb_ext.BidderName, consentMeta tcf2.ConsentMetadata, vendorInfo VendorInfo) bool {
	enforcer := p.purposeEnforcerBuilder(consentconstants.Purpose(2), bidder)

	overrides := Overrides{}
	if _, ok := enforcer.(*BasicEnforcement); ok {
		overrides.allowLITransparency = true
	}
	return enforcer.LegalBasis(vendorInfo, bidder, consentMeta, overrides)
}

// allowGeo computes legal basis for a given bidder using the configs, consent and GVL pertaining to
// feature one
func (p *permissionsImpl) allowGeo(bidder openrtb_ext.BidderName, consentMeta tcf2.ConsentMetadata, vendor api.Vendor) bool {
	if !p.cfg.FeatureOneEnforced() {
		return true
	}
	if p.cfg.FeatureOneVendorException(bidder) {
		return true
	}

	basicEnforcementVendors := p.cfg.BasicEnforcementVendors()
	_, weakVendorEnforcement := basicEnforcementVendors[string(bidder)]
	return consentMeta.SpecialFeatureOptIn(1) && (vendor.SpecialFeature(1) || weakVendorEnforcement)
}

// allowID computes the pass user ID activity legal basis for a given bidder using the enforcement algorithms
// selected by the purpose enforcer builder. For the user ID activity, the selected enforcement algorithm must
// always assume we are enforcing the purpose.
// If the purpose for which we are computing legal basis is purpose 2, the algorithm should allow LI transparency.
func (p *permissionsImpl) allowID(bidder openrtb_ext.BidderName, consentMeta tcf2.ConsentMetadata, vendorInfo VendorInfo) bool {
	for i := 2; i <= 10; i++ {
		purpose := consentconstants.Purpose(i)
		enforcer := p.purposeEnforcerBuilder(purpose, bidder)

		overrides := Overrides{enforcePurpose: true, enforceVendors: true}
		if _, ok := enforcer.(*BasicEnforcement); ok && purpose == consentconstants.Purpose(2) {
			overrides.allowLITransparency = true
		}
		if enforcer.LegalBasis(vendorInfo, bidder, consentMeta, overrides) {
			return true
		}
	}

	return false
}

// parseVendor parses the consent string and fetches the specified vendor's information from the GVL
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

	policyVersion := parsedConsent.TCFPolicyVersion()
	specVersion, err := getSpecVersion(policyVersion)
	if err != nil {
		err = &ErrorMalformedConsent{
			Consent: consent,
			Cause:   err,
		}
		return
	}

	vendorList, err := p.fetchVendorList(ctx, uint16(specVersion), parsedConsent.VendorListVersion())
	if err != nil {
		return
	}

	vendor = vendorList.Vendor(vendorID)
	return
}

// getSpecVersion looks at the TCF policy version and determines the corresponding GVL specification
// version that should be used to calculate legal basis
func getSpecVersion(policyVersion uint8) (uint16, error) {
	var specVersion uint16 = 3

	if policyVersion > 4 {
		return 0, fmt.Errorf("invalid TCF policy version %d", policyVersion)
	}
	if policyVersion < 4 {
		specVersion = 2
	}
	return specVersion, nil
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
