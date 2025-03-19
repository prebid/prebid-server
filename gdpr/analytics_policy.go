package gdpr

import (
	"context"
	"fmt"

	"github.com/prebid/go-gdpr/api"
	"github.com/prebid/go-gdpr/consentconstants"
	"github.com/prebid/go-gdpr/vendorconsent"
	tcf2 "github.com/prebid/go-gdpr/vendorconsent/tcf2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// PrivacyPolicy indicates whether an analytics adapter is allowed to perform some activity
type PrivacyPolicy interface {
	Allow(ctx context.Context, name string, gvlID uint16) (bool, error)
}

// PrivacyPolicyBuilder is a builder function that generates an object that implements the PrivacyPolicy interface
type PrivacyPolicyBuilder func(TCF2ConfigReader, Signal, string) PrivacyPolicy

// NewAnalyticsPolicyBuilder returns an PrivacyPolicyBuilder that has all global dependencies needed
// to build a request-specific analytics privacy policy
func NewAnalyticsPolicyBuilder(cfg config.GDPR, gvlVendorIDs map[openrtb_ext.BidderName]int16, vendorListFetcher VendorListFetcher) PrivacyPolicyBuilder {
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
	vendorIDs              map[openrtb_ext.BidderName]int16
	// request-specific
	cfg        TCF2ConfigReader
	consent    string
	gdprSignal Signal
}

// Allow determines whether analytics are permitted for a given analytics module
func (ap *analyticsPolicy) Allow(ctx context.Context, name string) (bool, error) {
	//if vendorID, found = ap.vendorIDs[name]
	// set the vendorID to 0 if not found


	if ap.gdprSignal != SignalYes {
		return true, nil
	}
	if ap.consent == "" {
		return ap.defaultPermissions(), nil
	}

	_, weakVendorEnforcement := ap.cfg.BasicEnforcementVendors()[name]
	if vendorID == 0 && !weakVendorEnforcement {
		return false, nil
	}

	parsedConsent, vendor, err := ap.parseVendor(ctx, vendorID, ap.consent)
	if err != nil {
		return ap.defaultPermissions(), err
	}

	// vendor will be nil if not a valid TCF2 consent string
	if vendor == nil {
		if weakVendorEnforcement && parsedConsent.Version() == 2 {
			vendor = vendorTrue{}
		} else {
			return ap.defaultPermissions(), nil
		}
	}

	if !ap.cfg.IsEnabled() {
		return false, nil
	}

	consentMeta, ok := parsedConsent.(tcf2.ConsentMetadata)
	if !ok {
		err = fmt.Errorf("Unable to access TCF2 parsed consent")
		return ap.defaultPermissions(), err
	}

	enforcer := ap.purposeEnforcerBuilder(consentconstants.Purpose(7), name)

	vendorInfo := VendorInfo{vendorID: vendorID, vendor: vendor}
	return enforcer.LegalBasis(vendorInfo, name, consentMeta, Overrides{}), nil
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

// parseVendor parses the consent string and fetches the specified vendor's information from the GVL
func (ap *analyticsPolicy) parseVendor(ctx context.Context, vendorID uint16, consent string) (parsedConsent api.VendorConsents, vendor api.Vendor, err error) {
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

	vendorList, err := ap.fetchVendorList(ctx, uint16(specVersion), parsedConsent.VendorListVersion())
	if err != nil {
		return
	}

	vendor = vendorList.Vendor(vendorID)
	return
}

// AllowAllAnalytics implements the PrivacyPolicy interface representing a policy that always
// permits sending data to analytics adapters
type AllowAllAnalytics struct{}

// Allow satisfies the PrivacyPolicy interface always returning true
func (aaa *AllowAllAnalytics) Allow(ctx context.Context, name string, vendorID uint16) (bool, error) {
	return true, nil
}