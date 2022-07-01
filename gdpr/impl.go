package gdpr

import (
	"context"
	"errors"
	"fmt"

	"github.com/prebid/go-gdpr/api"
	"github.com/prebid/go-gdpr/consentconstants"
	tcf2ConsentConstants "github.com/prebid/go-gdpr/consentconstants/tcf2"
	"github.com/prebid/go-gdpr/vendorconsent"
	tcf2 "github.com/prebid/go-gdpr/vendorconsent/tcf2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type permissionsImpl struct {
	// global
	fetchVendorList       VendorListFetcher
	gdprDefaultValue      string
	hostVendorID          int
	nonStandardPublishers map[string]struct{}
	vendorIDs             map[openrtb_ext.BidderName]uint16
	// request-specific
	aliasGVLIDs map[string]uint16
	cfg         TCF2ConfigReader
	consent     string
	gdprSignal  Signal
	publisherID string
	// cached enforcers
	// purposeEnforcers      map[consentconstants.Purpose]map[TCF2Enforcement]PurposeEnforcer
}

// called from /cookie_sync & /setuid
// vendor exception should not apply -- should we allow a host to specify themselves as a vendor exception for purpose 1?
// if yes, should we validate at startup and throw a warning?
// if no, should we add validation logic that fails hard? That seems like the right place to do it rather than ignoring the value in the GDPR logic
func (p *permissionsImpl) HostCookiesAllowed(ctx context.Context) (bool, error) {
	if p.gdprSignal != SignalYes {
		return true, nil
	}

	return p.allowSync(ctx, uint16(p.hostVendorID), false)
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
		return p.allowSync(ctx, id, vendorException)
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
	// if vendor == nil && parsedConsent.Version() != 2 {
	// 	return DenyAll, nil
	// }
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

func (p *permissionsImpl) allowSync(ctx context.Context, vendorID uint16, vendorException bool) (bool, error) {
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

	enforceVendors := p.cfg.PurposeEnforcingVendors(tcf2ConsentConstants.InfoStorageAccess)
	return p.checkPurpose(consentMeta, vendor, vendorID, tcf2ConsentConstants.InfoStorageAccess, enforceVendors, vendorException, false), nil
	// purpose := consentconstants.Purpose(1)
	// enforcer := p.getPurposeEnforcer(purpose, bidder, true)

	// vendorInfo := VendorInfo{vendorID: vendorID, vendor: vendor}
	// if enforcer.LegalBasis(vendorInfo, BidderInfo{bidder: bidder}, consentMeta) {
	// 	return true, nil
	// }
	// return false, nil
}

func (p *permissionsImpl) allowBidRequest(bidder openrtb_ext.BidderName, consentMeta tcf2.ConsentMetadata, vendorInfo VendorInfo) bool {
	purpose := consentconstants.Purpose(2)
	enforcer := p.getPurposeEnforcer(purpose, bidder, true) //TODO: true

	// this function will return true if purpose 2 is NOT enforced
	if enforcer.LegalBasis(vendorInfo, BidderInfo{bidder: bidder}, consentMeta) {
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
		enforcer := p.getPurposeEnforcer(purpose, bidder, true) //TODO: true value should be set based on the value of p.VendorList

		if enforcer.PurposeEnforced() && enforcer.LegalBasis(vendorInfo, BidderInfo{bidder: bidder}, consentMeta) {
			return true
		}
	}

	return false
}

func (p *permissionsImpl) getPurposeEnforcer(purpose consentconstants.Purpose, bidder openrtb_ext.BidderName, haveGVL bool) PurposeEnforcer {
	// use cached enforcer if already exists
	// if enforcer, ok := ts.purposeEnforcers[purpose]; ok { //TODO: need to consider when both enforcers are needed for a purpose because some vendors require basic enforcement
	// 	return enforcer
	// }

	cfg := purposeConfig{
		PurposeID:                  purpose,
		EnforcePurpose:             p.cfg.PurposeEnforcementType(purpose),
		EnforceVendors:             p.cfg.PurposeEnforcingVendors(purpose),
		VendorExceptionMap:         p.cfg.PurposeVendorExceptions(purpose),
		BasicEnforcementVendorsMap: p.cfg.BasicEnforcementVendors(),
	}

	downgraded := p.isDowngraded(cfg.EnforcePurpose, p.cfg.BasicEnforcementVendor(bidder), haveGVL)
	enforcer := NewPurposeEnforcer(cfg, downgraded)

	//cache the enforcer
	//TODO: uncomment
	// ts.purposeEnforcers[purpose] := enforcer

	return enforcer
}

func (p *permissionsImpl) isDowngraded(enforcePurpose string, basicEnforcementVendor bool, haveGVL bool) bool {
	if enforcePurpose == TCF2FullEnforcement && basicEnforcementVendor {
		return true
	}
	if enforcePurpose == TCF2FullEnforcement && !haveGVL {
		return true
	}
	// When no enforcement algorithm has been specified, we are not enforcing the purpose which resorts to using the full algorithm by default
	// resulting in publisher restriction checks and possible vendor checks if we are enforcing vendors for the purpose. If the GVL is not
	// available though, we need to drop down to basic
	if enforcePurpose == TCF2NoEnforcement && !haveGVL {
		return true
	}
	return false
}

const pubRestrictNotAllowed = 0
const pubRestrictRequireConsent = 1
const pubRestrictRequireLegitInterest = 2

func (p *permissionsImpl) checkPurpose(consent tcf2.ConsentMetadata, vendor api.Vendor, vendorID uint16, purpose consentconstants.Purpose, enforceVendors, vendorException, weakVendorEnforcement bool) bool {
	if consent.CheckPubRestriction(uint8(purpose), pubRestrictNotAllowed, vendorID) {
		return false
	}

	if vendorException {
		return true
	}

	purposeAllowed := p.consentEstablished(consent, vendor, vendorID, purpose, enforceVendors, weakVendorEnforcement)
	legitInterest := p.legitInterestEstablished(consent, vendor, vendorID, purpose, enforceVendors, weakVendorEnforcement)

	if consent.CheckPubRestriction(uint8(purpose), pubRestrictRequireConsent, vendorID) {
		return purposeAllowed
	}
	if consent.CheckPubRestriction(uint8(purpose), pubRestrictRequireLegitInterest, vendorID) {
		// Need LITransparency here
		return legitInterest
	}

	return purposeAllowed || legitInterest
}

func (p *permissionsImpl) consentEstablished(consent tcf2.ConsentMetadata, vendor api.Vendor, vendorID uint16, purpose consentconstants.Purpose, enforceVendors, weakVendorEnforcement bool) bool {
	if !consent.PurposeAllowed(purpose) {
		return false
	}
	if weakVendorEnforcement {
		return true
	}
	if !enforceVendors {
		return true
	}
	if vendor.Purpose(purpose) && consent.VendorConsent(vendorID) {
		return true
	}
	return false
}

func (p *permissionsImpl) legitInterestEstablished(consent tcf2.ConsentMetadata, vendor api.Vendor, vendorID uint16, purpose consentconstants.Purpose, enforceVendors, weakVendorEnforcement bool) bool {
	if !consent.PurposeLITransparency(purpose) {
		return false
	}
	if weakVendorEnforcement {
		return true
	}
	if !enforceVendors {
		return true
	}
	if vendor.LegitimateInterest(purpose) && consent.VendorLegitInterest(vendorID) {
		return true
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
