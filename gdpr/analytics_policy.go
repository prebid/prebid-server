package gdpr

import (
	"context"

	"github.com/prebid/go-gdpr/api"
	"github.com/prebid/go-gdpr/consentconstants"
	tcf2 "github.com/prebid/go-gdpr/vendorconsent/tcf2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// PrivacyPolicy indicates whether an analytics adapter is allowed to perform some activity
type PrivacyPolicy interface {
	SetContext(ctx context.Context)
	Allow(name string) bool
}

// PrivacyPolicyBuilder is a builder function that generates an object that implements the PrivacyPolicy interface
type PrivacyPolicyBuilder func(TCF2ConfigReader, Signal, string) PrivacyPolicy

// NewAnalyticsPolicyBuilder returns an PrivacyPolicyBuilder that has all global dependencies needed
// to build a request-specific analytics privacy policy
func NewAnalyticsPolicyBuilder(cfg config.GDPR, gvlVendorIDs map[openrtb_ext.BidderName]uint16, vendorListFetcher VendorListFetcher) PrivacyPolicyBuilder {
	return func(tcf2Cfg TCF2ConfigReader, gdprSignal Signal, consent string) PrivacyPolicy {
		purposeEnforcerBuilder := NewPurposeEnforcerBuilder(tcf2Cfg)

		return NewAnalyticsPolicy(cfg, tcf2Cfg, gvlVendorIDs, vendorListFetcher, purposeEnforcerBuilder, gdprSignal, consent)
	}
}

// NewAnalyticsPolicy returns a request-specific analytics privacy policy
func NewAnalyticsPolicy(cfg config.GDPR, tcf2Cfg TCF2ConfigReader, vendorIDs map[openrtb_ext.BidderName]uint16, fetcher VendorListFetcher, purposeEnforcerBuilder PurposeEnforcerBuilder, gdprSignal Signal, consent string) PrivacyPolicy {
	if !cfg.Enabled {
		return &AllowAllAnalytics{}
	}

	return &analyticsPolicy{
		fetchVendorList:        fetcher,
		purposeEnforcerBuilder: purposeEnforcerBuilder,
		vendorIDs:              vendorIDs,
		cfg:                    tcf2Cfg,
		consent:                consent,
		gdprSignal:             SignalNormalize(gdprSignal, cfg.DefaultValue),
	}
}

// analyticsPolicy contains global and request-specific GDPR config data and is used to determine
// whether an analytics adapter may be sent information for a given request.
// analyticsPolicy implements the PrivacyPolicy interface
type analyticsPolicy struct {
	// global
	fetchVendorList        VendorListFetcher
	purposeEnforcerBuilder PurposeEnforcerBuilder
	vendorIDs              map[openrtb_ext.BidderName]uint16
	// request-specific
	cfg        TCF2ConfigReader
	consent    string
	gdprSignal Signal
	ctx        context.Context
}

func (ap *analyticsPolicy) SetContext(ctx context.Context) {
	ap.ctx = ctx
}

// Allow determines whether analytics are permitted for a given analytics module
func (ap *analyticsPolicy) Allow(name string) (bool) {
	if ap.gdprSignal != SignalYes {
		return true
	}
	if ap.consent == "" {
		return ap.defaultPermissions()
	}

	pc, err := parseConsent(ap.consent)
	if err != nil {
		return ap.defaultPermissions()
	}

	// returns 0 if not found
	vendorID := ap.vendorIDs[openrtb_ext.BidderName(name)]

	vendor, err := ap.getVendor(ap.ctx, vendorID, *pc)
	if err != nil {
		return ap.defaultPermissions()
	}

	enforcer := ap.purposeEnforcerBuilder(consentconstants.Purpose(7), name)

	vendorInfo := VendorInfo{vendorID: vendorID, vendor: vendor}
	return enforcer.LegalBasis(vendorInfo, name, pc.consentMeta, Overrides{})
}

// defaultPermissions denies sending information to an analytics adapter when purpose 7 is
// enabled; otherwise it is permitted.
// if the consent string is empty or malformed we should use the default permissions
func (ap *analyticsPolicy) defaultPermissions() (allow bool) {
	if !ap.cfg.PurposeEnforced(consentconstants.Purpose(7)) {
		return true
	}
	return false
}

// getVendor retrieves the GVL vendor information for a particular bidder
func (ap *analyticsPolicy) getVendor(ctx context.Context, vendorID uint16, pc parsedConsent) (api.Vendor, error) {
	vendorList, err := ap.fetchVendorList(ctx, pc.specVersion, pc.listVersion)
	if err != nil {
		return nil, err
	}
	return vendorList.Vendor(vendorID), nil
}

// AllowAllAnalytics implements the PrivacyPolicy interface representing a policy that always
// permits sending data to analytics adapters
type AllowAllAnalytics struct{}

func (aaa *AllowAllAnalytics) SetContext(ctx context.Context) {
	return
}

// Allow satisfies the PrivacyPolicy interface always returning true
func (aaa *AllowAllAnalytics) Allow(name string) (bool) {
	return true
}