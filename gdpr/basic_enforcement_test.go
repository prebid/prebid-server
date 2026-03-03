package gdpr

import (
	"testing"

	"github.com/prebid/go-gdpr/consentconstants"
	"github.com/prebid/go-gdpr/vendorconsent"
	tcf2 "github.com/prebid/go-gdpr/vendorconsent/tcf2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"

	"github.com/stretchr/testify/assert"
)

func TestBasicLegalBasis(t *testing.T) {
	var (
		appnexus   = string(openrtb_ext.BidderAppnexus)
		appnexusID = uint16(32)
	)

	noConsents := "CPerMsAPerMsAAAAAAENCfCAAAAAAAAAAAAAAAAAAAAA"
	purpose2Consent := "CPerMsAPerMsAAAAAAENCfCAAEAAAAAAAAAAAAAAAAAA"
	purpose2LI := "CPerMsAPerMsAAAAAAENCfCAAAAAAEAAAAAAAAAAAAAA"
	vendor32Consent := "CPerMsAPerMsAAAAAAENCfCAAAAAAAAAAAAAAQAAAAAEAAAAAAAA"
	purpose2AndVendor32Consent := "CPerMsAPerMsAAAAAAENCfCAAEAAAAAAAAAAAQAAAAAEAAAAAAAA"

	tests := []struct {
		description string
		config      purposeConfig
		consent     string
		overrides   Overrides
		wantResult  bool
	}{
		{
			description: "enforce purpose & vendors are off",
			consent:     noConsents,
			config: purposeConfig{
				PurposeID:      consentconstants.Purpose(2),
				EnforcePurpose: false,
				EnforceVendors: false,
			},
			wantResult: true,
		},
		{
			description: "enforce purpose on, purpose consent N",
			consent:     noConsents,
			config: purposeConfig{
				PurposeID:      consentconstants.Purpose(2),
				EnforcePurpose: true,
				EnforceVendors: false,
			},
			wantResult: false,
		},
		{
			description: "enforce purpose on, purpose consent Y",
			consent:     purpose2Consent,
			config: purposeConfig{
				PurposeID:      consentconstants.Purpose(2),
				EnforcePurpose: true,
				EnforceVendors: false,
			},
			wantResult: true,
		},
		{
			description: "enforce purpose on, purpose consent Y, enforce vendors off but overrides treats it as on",
			consent:     purpose2Consent,
			config: purposeConfig{
				PurposeID:      consentconstants.Purpose(2),
				EnforcePurpose: true,
				EnforceVendors: false,
			},
			overrides:  Overrides{enforceVendors: true},
			wantResult: false,
		},
		{
			description: "enforce purpose on, purpose consent Y, vendor consent Y, enforce vendors off but overrides treats it as on",
			consent:     purpose2AndVendor32Consent,
			config: purposeConfig{
				PurposeID:      consentconstants.Purpose(2),
				EnforcePurpose: true,
				EnforceVendors: false,
			},
			overrides:  Overrides{enforceVendors: true},
			wantResult: true,
		},
		{
			description: "enforce purpose on, purpose LI Transparency Y",
			consent:     purpose2LI,
			config: purposeConfig{
				PurposeID:      consentconstants.Purpose(2),
				EnforcePurpose: true,
				EnforceVendors: false,
			},
			wantResult: false,
		},
		{
			description: "enforce purpose on, purpose LI Transparency Y but overrides allow it",
			consent:     purpose2LI,
			config: purposeConfig{
				PurposeID:      consentconstants.Purpose(2),
				EnforcePurpose: true,
				EnforceVendors: false,
			},
			overrides:  Overrides{allowLITransparency: true},
			wantResult: true,
		},
		{
			description: "enforce vendors on, vendor consent N",
			consent:     noConsents,
			config: purposeConfig{
				PurposeID:      consentconstants.Purpose(2),
				EnforcePurpose: false,
				EnforceVendors: true,
			},
			wantResult: false,
		},
		{
			description: "enforce vendors on, vendor consent Y",
			consent:     vendor32Consent,
			config: purposeConfig{
				PurposeID:      consentconstants.Purpose(2),
				EnforcePurpose: false,
				EnforceVendors: true,
			},
			wantResult: true,
		},
		{
			description: "enforce vendors on, vendor consent Y, enforce purpose off but overrides treats it as on",
			consent:     vendor32Consent,
			config: purposeConfig{
				PurposeID:      consentconstants.Purpose(2),
				EnforcePurpose: false,
				EnforceVendors: true,
			},
			overrides:  Overrides{enforcePurpose: true},
			wantResult: false,
		},
		{
			description: "enforce vendors on, purpose consent Y, vendor consent Y, enforce purpose off but overrides treats it as on",
			consent:     purpose2AndVendor32Consent,
			config: purposeConfig{
				PurposeID:      consentconstants.Purpose(2),
				EnforcePurpose: false,
				EnforceVendors: true,
			},
			overrides:  Overrides{enforcePurpose: true},
			wantResult: true,
		},
		{
			description: "enforce vendors on, vendor consent N, bidder is a basic vendor",
			consent:     noConsents,
			config: purposeConfig{
				PurposeID:                  consentconstants.Purpose(2),
				EnforcePurpose:             false,
				EnforceVendors:             true,
				BasicEnforcementVendorsMap: map[string]struct{}{appnexus: {}},
			},
			wantResult: true,
		},
		{
			description: "enforce purpose & vendors are on",
			consent:     noConsents,
			config: purposeConfig{
				PurposeID:      consentconstants.Purpose(2),
				EnforcePurpose: true,
				EnforceVendors: true,
			},
			wantResult: false,
		},
		{
			description: "enforce purpose & vendors are on, bidder is a vendor exception",
			consent:     noConsents,
			config: purposeConfig{
				PurposeID:          consentconstants.Purpose(2),
				EnforcePurpose:     true,
				EnforceVendors:     true,
				VendorExceptionMap: map[string]struct{}{appnexus: {}},
			},
			wantResult: true,
		},
		{
			description: "enforce purpose & vendors are on, bidder is a vendor exception but overrides disallow them",
			consent:     noConsents,
			config: purposeConfig{
				PurposeID:          consentconstants.Purpose(2),
				EnforcePurpose:     true,
				EnforceVendors:     true,
				VendorExceptionMap: map[string]struct{}{appnexus: {}},
			},
			overrides:  Overrides{blockVendorExceptions: true},
			wantResult: false,
		},
		{
			description: "enforce purpose & vendors are on, purpose consent Y, vendor consent N",
			consent:     purpose2Consent,
			config: purposeConfig{
				PurposeID:      consentconstants.Purpose(2),
				EnforcePurpose: true,
				EnforceVendors: true,
			},
			wantResult: false,
		},
		{
			description: "enforce purpose & vendors are on, purpose consent Y, vendor consent N, bidder is a basic vendor",
			consent:     purpose2Consent,
			config: purposeConfig{
				PurposeID:                  consentconstants.Purpose(2),
				EnforcePurpose:             true,
				EnforceVendors:             true,
				BasicEnforcementVendorsMap: map[string]struct{}{appnexus: {}},
			},
			wantResult: true,
		},
		{
			description: "enforce purpose & vendors are on, purpose consent Y, vendor consent Y",
			consent:     purpose2AndVendor32Consent,
			config: purposeConfig{
				PurposeID:      consentconstants.Purpose(2),
				EnforcePurpose: true,
				EnforceVendors: true,
			},
			wantResult: true,
		},
	}

	for _, tt := range tests {
		// convert consent string to TCF2 object
		parsedConsent, err := vendorconsent.ParseString(tt.consent)
		if err != nil {
			t.Fatalf("Failed to parse consent %s: %s\n", tt.consent, tt.description)
		}
		consentMeta, ok := parsedConsent.(tcf2.ConsentMetadata)
		if !ok {
			t.Fatalf("Failed to convert consent %s: %s\n", tt.consent, tt.description)
		}

		enforcer := BasicEnforcement{cfg: tt.config}

		vendorInfo := VendorInfo{vendorID: appnexusID, vendor: nil}
		result := enforcer.LegalBasis(vendorInfo, appnexus, consentMeta, tt.overrides)

		assert.Equal(t, tt.wantResult, result, tt.description)
	}
}
