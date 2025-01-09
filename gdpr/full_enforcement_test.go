package gdpr

import (
	"testing"

	"github.com/prebid/go-gdpr/consentconstants"
	"github.com/prebid/go-gdpr/vendorconsent"
	tcf2 "github.com/prebid/go-gdpr/vendorconsent/tcf2"
	"github.com/prebid/go-gdpr/vendorlist"
	"github.com/prebid/go-gdpr/vendorlist2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"

	"github.com/stretchr/testify/assert"
)

func TestLegalBasisWithPubRestrictionAllowNone(t *testing.T) {
	var (
		appnexus   = string(openrtb_ext.BidderAppnexus)
		appnexusID = uint16(32)
	)

	NoConsentsWithP1P2P3V32RestrictionAllowNone := "CPfMKEAPfMKEAAAAAAENCgCAAAAAAAAAAAAAAQAAAAAAAIAAAAAAAGCAAgAgCAAQAQBgAIAIAAAA"
	P1P2P3PurposeConsentAndV32VendorConsentWithP1P2P3V32RestrictionAllowNone := "CPfMKEAPfMKEAAAAAAENCgCAAOAAAAAAAAAAAQAAAAAEAIAAAAAAAGCAAgAgCAAQAQBgAIAIAAAA"

	tests := []struct {
		description              string
		config                   purposeConfig
		consent                  string
		wantConsentPurposeResult bool
		wantLIPurposeResult      bool
		wantFlexPurposeResult    bool
	}{
		{
			description: "enforce purpose & vendors off",
			config: purposeConfig{
				EnforcePurpose: false,
				EnforceVendors: false,
			},
			consent:                  NoConsentsWithP1P2P3V32RestrictionAllowNone,
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
		},
		{
			description: "enforce purpose & vendors on, bidder is a vendor exception",
			config: purposeConfig{
				EnforcePurpose:     true,
				EnforceVendors:     true,
				VendorExceptionMap: map[string]struct{}{appnexus: {}},
			},
			consent:                  NoConsentsWithP1P2P3V32RestrictionAllowNone,
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
		},
		{
			description: "enforce purpose & vendors on, purpose consent Y, vendor consent Y",
			config: purposeConfig{
				EnforcePurpose: true,
				EnforceVendors: true,
			},
			consent:                  P1P2P3PurposeConsentAndV32VendorConsentWithP1P2P3V32RestrictionAllowNone,
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
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

		vendor := getVendorList(t).Vendor(appnexusID)
		vendorInfo := VendorInfo{vendorID: appnexusID, vendor: vendor}
		enforcer := FullEnforcement{cfg: tt.config}

		enforcer.cfg.PurposeID = consentconstants.Purpose(1)
		consentPurposeResult := enforcer.LegalBasis(vendorInfo, appnexus, consentMeta, Overrides{})
		assert.Equal(t, tt.wantConsentPurposeResult, consentPurposeResult, tt.description+" -- GVL consent purpose")

		enforcer.cfg.PurposeID = consentconstants.Purpose(2)
		LIPurposeresult := enforcer.LegalBasis(vendorInfo, appnexus, consentMeta, Overrides{})
		assert.Equal(t, tt.wantLIPurposeResult, LIPurposeresult, tt.description+" -- GVL LI purpose")

		enforcer.cfg.PurposeID = consentconstants.Purpose(3)
		flexPurposeResult := enforcer.LegalBasis(vendorInfo, appnexus, consentMeta, Overrides{})
		assert.Equal(t, tt.wantFlexPurposeResult, flexPurposeResult, tt.description+" -- GVL flex purpose")
	}
}

func TestLegalBasisWithNoPubRestrictionsAndWithPubRestrictionAllowAll(t *testing.T) {
	var (
		appnexus   = string(openrtb_ext.BidderAppnexus)
		appnexusID = uint16(32)
	)

	NoConsents := "CPfCRQAPfCRQAAAAAAENCgCAAAAAAAAAAAAAAAAAAAAA"
	P1P2P3PurposeConsent := "CPfCRQAPfCRQAAAAAAENCgCAAOAAAAAAAAAAAAAAAAAA"
	P1P2P3PurposeLI := "CPfCRQAPfCRQAAAAAAENCgCAAAAAAOAAAAAAAAAAAAAA"
	V32VendorConsent := "CPfCRQAPfCRQAAAAAAENCgCAAAAAAAAAAAAAAQAAAAAEAAAAAAAA"
	V32VendorLI := "CPfCRQAPfCRQAAAAAAENCgCAAAAAAAAAAAAAAQAAAAAAAIAAAAACAAAA"
	P1P2P3PurposeConsentAndV32VendorConsent := "CPfCRQAPfCRQAAAAAAENCgCAAOAAAAAAAAAAAQAAAAAEAIAAAAAAAAAA"
	P1P2P3PurposeLIAndV32VendorLI := "CPfCRQAPfCRQAAAAAAENCgCAAAAAAOAAAAAAAQAAAAAAAIAAAAACAAAA"

	NoConsentsWithP1P2P3V32RestrictionAllowAll := "CPfMKEAPfMKEAAAAAAENCgCAAAAAAAAAAAAAAQAAAAAAAIAAAAAAAGDgAgAgCwAQAQB4AIAIAAAA"
	P1P2P3PurposeConsentWithP1P2P3V32RestrictionAllowAll := "CPfMKEAPfMKEAAAAAAENCgCAAOAAAAAAAAAAAQAAAAAAAIAAAAAAAGDgAgAgCwAQAQB4AIAIAAAA"
	P1P2P3PurposeLIWithP1P2P3V32RestrictionAllowAll := "CPfMKEAPfMKEAAAAAAENCgCAAAAAAOAAAAAAAQAAAAAAAIAAAAAAAGDgAgAgCwAQAQB4AIAIAAAA"
	V32VendorConsentWithP1P2P3V32RestrictionAllowAll := "CPfMKEAPfMKEAAAAAAENCgCAAAAAAAAAAAAAAQAAAAAEAIAAAAAAAGDgAgAgCwAQAQB4AIAIAAAA"
	V32VendorLIWithP1P2P3V32RestrictionAllowAll := "CPfMKEAPfMKEAAAAAAENCgCAAAAAAAAAAAAAAQAAAAAAAIAAAAACAGDgAgAgCwAQAQB4AIAIAAAA"
	P1P2P3PurposeConsentAndV32VendorConsentWithP1P2P3V32RestrictionAllowAll := "CPfMKEAPfMKEAAAAAAENCgCAAOAAAAAAAAAAAQAAAAAEAIAAAAAAAGDgAgAgCwAQAQB4AIAIAAAA"
	P1P2P3PurposeLIAndV32VendorLIWithP1P2P3V32RestrictionAllowAll := "CPfMKEAPfMKEAAAAAAENCgCAAAAAAOAAAAAAAQAAAAAAAIAAAAACAGDgAgAgCwAQAQB4AIAIAAAA"

	tests := []struct {
		description               string
		config                    purposeConfig
		consentNoPubRestriction   string
		consentWithPubRestriction string
		overrides                 Overrides
		wantConsentPurposeResult  bool
		wantLIPurposeResult       bool
		wantFlexPurposeResult     bool
	}{
		{
			description: "enforce purpose & vendors off",
			config: purposeConfig{
				EnforcePurpose: false,
				EnforceVendors: false,
			},
			consentNoPubRestriction:   NoConsents,
			consentWithPubRestriction: NoConsentsWithP1P2P3V32RestrictionAllowAll,
			wantConsentPurposeResult:  true,
			wantLIPurposeResult:       true,
			wantFlexPurposeResult:     true,
		},
		{
			description: "enforce purpose on, purpose consent N, legit interest N",
			config: purposeConfig{
				EnforcePurpose: true,
			},
			consentNoPubRestriction:   NoConsents,
			consentWithPubRestriction: NoConsentsWithP1P2P3V32RestrictionAllowAll,
			wantConsentPurposeResult:  false,
			wantLIPurposeResult:       false,
			wantFlexPurposeResult:     false,
		},
		{
			description: "enforce purpose on, purpose consent Y",
			config: purposeConfig{
				EnforcePurpose: true,
			},
			consentNoPubRestriction:   P1P2P3PurposeConsent,
			consentWithPubRestriction: P1P2P3PurposeConsentWithP1P2P3V32RestrictionAllowAll,
			wantConsentPurposeResult:  true,
			wantLIPurposeResult:       false,
			wantFlexPurposeResult:     true,
		},
		{
			description: "enforce purpose on, legit interest Y",
			config: purposeConfig{
				EnforcePurpose: true,
			},
			consentNoPubRestriction:   P1P2P3PurposeLI,
			consentWithPubRestriction: P1P2P3PurposeLIWithP1P2P3V32RestrictionAllowAll,
			wantConsentPurposeResult:  false,
			wantLIPurposeResult:       true,
			wantFlexPurposeResult:     true,
		},
		{
			description: "enforce purpose on, purpose consent Y, enforce vendors off but overrides treats it as on",
			config: purposeConfig{
				EnforcePurpose: true,
			},
			consentNoPubRestriction:   P1P2P3PurposeConsent,
			consentWithPubRestriction: P1P2P3PurposeConsentWithP1P2P3V32RestrictionAllowAll,
			overrides:                 Overrides{enforceVendors: true},
			wantConsentPurposeResult:  false,
			wantLIPurposeResult:       false,
			wantFlexPurposeResult:     false,
		},
		{
			description: "enforce purpose on, purpose consent Y, vendor consent Y, enforce vendors off but overrides treats it as on",
			config: purposeConfig{
				EnforcePurpose: true,
			},
			consentNoPubRestriction:   P1P2P3PurposeConsentAndV32VendorConsent,
			consentWithPubRestriction: P1P2P3PurposeConsentAndV32VendorConsentWithP1P2P3V32RestrictionAllowAll,
			overrides:                 Overrides{enforceVendors: true},
			wantConsentPurposeResult:  true,
			wantLIPurposeResult:       false,
			wantFlexPurposeResult:     true,
		},
		{
			description: "enforce vendors on, vendor consent N, vendor legit interest N",
			config: purposeConfig{
				EnforceVendors: true,
			},
			consentNoPubRestriction:   NoConsents,
			consentWithPubRestriction: NoConsentsWithP1P2P3V32RestrictionAllowAll,
			wantConsentPurposeResult:  false,
			wantLIPurposeResult:       false,
			wantFlexPurposeResult:     false,
		},
		{
			description: "enforce vendors on, vendor consent Y",
			config: purposeConfig{
				EnforceVendors: true,
			},
			consentNoPubRestriction:   V32VendorConsent,
			consentWithPubRestriction: V32VendorConsentWithP1P2P3V32RestrictionAllowAll,
			wantConsentPurposeResult:  true,
			wantLIPurposeResult:       false,
			wantFlexPurposeResult:     true,
		},
		{
			description: "enforce vendors on, vendor legit interest Y",
			config: purposeConfig{
				EnforceVendors: true,
			},
			consentNoPubRestriction:   V32VendorLI,
			consentWithPubRestriction: V32VendorLIWithP1P2P3V32RestrictionAllowAll,
			wantConsentPurposeResult:  false,
			wantLIPurposeResult:       true,
			wantFlexPurposeResult:     true,
		},
		{
			description: "enforce vendors on, vendor consent Y, enforce purpose off but overrides treats it as on",
			config: purposeConfig{
				EnforceVendors: true,
			},
			consentNoPubRestriction:   V32VendorConsent,
			consentWithPubRestriction: V32VendorConsentWithP1P2P3V32RestrictionAllowAll,
			overrides:                 Overrides{enforcePurpose: true},
			wantConsentPurposeResult:  false,
			wantLIPurposeResult:       false,
			wantFlexPurposeResult:     false,
		},
		{
			description: "enforce vendors on, purpose consent Y, vendor consent Y, enforce purpose off but overrides treats it as on",
			config: purposeConfig{
				EnforceVendors: true,
			},
			consentNoPubRestriction:   P1P2P3PurposeConsentAndV32VendorConsent,
			consentWithPubRestriction: P1P2P3PurposeConsentAndV32VendorConsentWithP1P2P3V32RestrictionAllowAll,
			overrides:                 Overrides{enforcePurpose: true},
			wantConsentPurposeResult:  true,
			wantLIPurposeResult:       false,
			wantFlexPurposeResult:     true,
		},
		{
			description: "enforce purpose & vendors on, purpose consent Y, vendor consent N",
			config: purposeConfig{
				EnforcePurpose: true,
				EnforceVendors: true,
			},
			consentNoPubRestriction:   P1P2P3PurposeConsent,
			consentWithPubRestriction: P1P2P3PurposeConsentWithP1P2P3V32RestrictionAllowAll,
			wantConsentPurposeResult:  false,
			wantLIPurposeResult:       false,
			wantFlexPurposeResult:     false,
		},
		{
			description: "enforce purpose & vendors on, purpose consent N, vendor consent Y",
			config: purposeConfig{
				EnforcePurpose: true,
				EnforceVendors: true,
			},
			consentNoPubRestriction:   V32VendorConsent,
			consentWithPubRestriction: V32VendorConsentWithP1P2P3V32RestrictionAllowAll,
			wantConsentPurposeResult:  false,
			wantLIPurposeResult:       false,
			wantFlexPurposeResult:     false,
		},
		{
			description: "enforce purpose & vendors on, purpose consent Y, vendor consent Y",
			config: purposeConfig{
				EnforcePurpose: true,
				EnforceVendors: true,
			},
			consentNoPubRestriction:   P1P2P3PurposeConsentAndV32VendorConsent,
			consentWithPubRestriction: P1P2P3PurposeConsentAndV32VendorConsentWithP1P2P3V32RestrictionAllowAll,
			wantConsentPurposeResult:  true,
			wantLIPurposeResult:       false,
			wantFlexPurposeResult:     true,
		},
		{
			description: "enforce purpose & vendors on, legit interest Y, vendor legit interest N",
			config: purposeConfig{
				EnforcePurpose: true,
				EnforceVendors: true,
			},
			consentNoPubRestriction:   P1P2P3PurposeLI,
			consentWithPubRestriction: P1P2P3PurposeLIWithP1P2P3V32RestrictionAllowAll,
			wantConsentPurposeResult:  false,
			wantLIPurposeResult:       false,
			wantFlexPurposeResult:     false,
		},
		{
			description: "enforce purpose & vendors on, legit interest N, vendor legit interest Y",
			config: purposeConfig{
				EnforcePurpose: true,
				EnforceVendors: true,
			},
			consentNoPubRestriction:   V32VendorLI,
			consentWithPubRestriction: V32VendorLIWithP1P2P3V32RestrictionAllowAll,
			wantConsentPurposeResult:  false,
			wantLIPurposeResult:       false,
			wantFlexPurposeResult:     false,
		},
		{
			description: "enforce purpose & vendors on, legit interest Y, vendor legit interest Y",
			config: purposeConfig{
				EnforcePurpose: true,
				EnforceVendors: true,
			},
			consentNoPubRestriction:   P1P2P3PurposeLIAndV32VendorLI,
			consentWithPubRestriction: P1P2P3PurposeLIAndV32VendorLIWithP1P2P3V32RestrictionAllowAll,
			wantConsentPurposeResult:  false,
			wantLIPurposeResult:       true,
			wantFlexPurposeResult:     true,
		},
		{
			description: "enforce purpose & vendors on, bidder is a vendor exception",
			config: purposeConfig{
				EnforcePurpose:     true,
				EnforceVendors:     true,
				VendorExceptionMap: map[string]struct{}{appnexus: {}},
			},
			consentNoPubRestriction:   NoConsents,
			consentWithPubRestriction: NoConsentsWithP1P2P3V32RestrictionAllowAll,
			wantConsentPurposeResult:  true,
			wantLIPurposeResult:       true,
			wantFlexPurposeResult:     true,
		},
		{
			description: "enforce purpose & vendors on, bidder is a vendor exception but overrides disallow them",
			config: purposeConfig{
				EnforcePurpose:     true,
				EnforceVendors:     true,
				VendorExceptionMap: map[string]struct{}{appnexus: {}},
			},
			consentNoPubRestriction:   NoConsents,
			consentWithPubRestriction: NoConsentsWithP1P2P3V32RestrictionAllowAll,
			overrides:                 Overrides{blockVendorExceptions: true},
			wantConsentPurposeResult:  false,
			wantLIPurposeResult:       false,
			wantFlexPurposeResult:     false,
		},
	}

	for _, tt := range tests {
		consents := [2]string{tt.consentNoPubRestriction, tt.consentWithPubRestriction}

		for i := 0; i < len(consents); i++ {
			consent := consents[i]

			// convert consent string to TCF2 object
			parsedConsent, err := vendorconsent.ParseString(consent)
			if err != nil {
				t.Fatalf("Failed to parse consent %s: %s\n", consent, tt.description)
			}
			consentMeta, ok := parsedConsent.(tcf2.ConsentMetadata)
			if !ok {
				t.Fatalf("Failed to convert consent %s: %s\n", consent, tt.description)
			}

			vendor := getVendorList(t).Vendor(appnexusID)
			vendorInfo := VendorInfo{vendorID: appnexusID, vendor: vendor}
			enforcer := FullEnforcement{cfg: tt.config}

			enforcer.cfg.PurposeID = consentconstants.Purpose(1)
			consentPurposeResult := enforcer.LegalBasis(vendorInfo, appnexus, consentMeta, tt.overrides)
			assert.Equal(t, tt.wantConsentPurposeResult, consentPurposeResult, tt.description+" -- GVL consent purpose -- consent string %d of %d", i+1, len(consents))

			enforcer.cfg.PurposeID = consentconstants.Purpose(2)
			LIPurposeresult := enforcer.LegalBasis(vendorInfo, appnexus, consentMeta, tt.overrides)
			assert.Equal(t, tt.wantLIPurposeResult, LIPurposeresult, tt.description+" -- GVL LI purpose -- consent string %d of %d", i+1, len(consents))

			enforcer.cfg.PurposeID = consentconstants.Purpose(3)
			flexPurposeResult := enforcer.LegalBasis(vendorInfo, appnexus, consentMeta, tt.overrides)
			assert.Equal(t, tt.wantFlexPurposeResult, flexPurposeResult, tt.description+" -- GVL flex purpose -- consent string %d of %d", i+1, len(consents))
		}
	}
}

func TestLegalBasisWithPubRestrictionRequireConsent(t *testing.T) {
	var (
		appnexus   = string(openrtb_ext.BidderAppnexus)
		appnexusID = uint16(32)
	)

	NoConsentsWithP1P2P3V32RestrictionRequireConsent := "CPfFkMAPfFkMAAAAAAENCgCAAAAAAAAAAAAAAQAAAAAAAIAAAAAAAGCgAgAgCQAQAQBoAIAIAAAA"
	P1P2P3PurposeConsentWithP1P2P3V32RestrictionRequireConsent := "CPfFkMAPfFkMAAAAAAENCgCAAOAAAAAAAAAAAQAAAAAAAIAAAAAAAGCgAgAgCQAQAQBoAIAIAAAA"
	P1P2P3PurposeLIWithP1P2P3V32RestrictionRequireConsent := "CPfFkMAPfFkMAAAAAAENCgCAAAAAAOAAAAAAAQAAAAAAAIAAAAAAAGCgAgAgCQAQAQBoAIAIAAAA"
	V32VendorConsentWithP1P2P3V32RestrictionRequireConsent := "CPfFkMAPfFkMAAAAAAENCgCAAAAAAAAAAAAAAQAAAAAEAIAAAAAAAGCgAgAgCQAQAQBoAIAIAAAA"
	V32VendorLIWithP1P2P3V32RestrictionRequireConsent := "CPfFkMAPfFkMAAAAAAENCgCAAAAAAAAAAAAAAQAAAAAAAIAAAAACAGCgAgAgCQAQAQBoAIAIAAAA"
	P1P2P3PurposeConsentAndV32VendorConsentWithP1P2P3V32RestrictionRequireConsent := "CPfFkMAPfFkMAAAAAAENCgCAAOAAAAAAAAAAAQAAAAAEAIAAAAAAAGCgAgAgCQAQAQBoAIAIAAAA"
	P1P2P3PurposeLIAndV32VendorLIWithP1P2P3V32RestrictionRequireConsent := "CPfFkMAPfFkMAAAAAAENCgCAAAAAAOAAAAAAAQAAAAAAAIAAAAACAGCgAgAgCQAQAQBoAIAIAAAA"

	tests := []struct {
		description              string
		config                   purposeConfig
		consent                  string
		overrides                Overrides
		wantConsentPurposeResult bool
		wantLIPurposeResult      bool
		wantFlexPurposeResult    bool
	}{
		{
			description: "enforce purpose & vendors off",
			config: purposeConfig{
				EnforcePurpose: false,
				EnforceVendors: false,
			},
			consent:                  NoConsentsWithP1P2P3V32RestrictionRequireConsent,
			wantConsentPurposeResult: true,
			wantLIPurposeResult:      true,
			wantFlexPurposeResult:    true,
		},
		{
			description: "enforce purpose on, purpose consent N, legit interest N",
			config: purposeConfig{
				EnforcePurpose: true,
			},
			consent:                  NoConsentsWithP1P2P3V32RestrictionRequireConsent,
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
		},
		{
			description: "enforce purpose on, purpose consent Y",
			config: purposeConfig{
				EnforcePurpose: true,
			},
			consent:                  P1P2P3PurposeConsentWithP1P2P3V32RestrictionRequireConsent,
			wantConsentPurposeResult: true,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    true,
		},
		{
			description: "enforce purpose on, legit interest Y",
			config: purposeConfig{
				EnforcePurpose: true,
			},
			consent:                  P1P2P3PurposeLIWithP1P2P3V32RestrictionRequireConsent,
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
		},
		{
			description: "enforce purpose on, purpose consent Y, enforce vendors off but overrides treats it as on",
			config: purposeConfig{
				EnforcePurpose: true,
			},
			consent:                  P1P2P3PurposeConsentWithP1P2P3V32RestrictionRequireConsent,
			overrides:                Overrides{enforceVendors: true},
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
		},
		{
			description: "enforce purpose on, purpose consent Y, vendor consent Y, enforce vendors off but overrides treats it as on",
			config: purposeConfig{
				EnforcePurpose: true,
			},
			consent:                  P1P2P3PurposeConsentAndV32VendorConsentWithP1P2P3V32RestrictionRequireConsent,
			overrides:                Overrides{enforceVendors: true},
			wantConsentPurposeResult: true,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    true,
		},
		{
			description: "enforce vendors on, vendor consent N, vendor legit interest N",
			config: purposeConfig{
				EnforceVendors: true,
			},
			consent:                  NoConsentsWithP1P2P3V32RestrictionRequireConsent,
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
		},
		{
			description: "enforce vendors on, vendor consent Y",
			config: purposeConfig{
				EnforceVendors: true,
			},
			consent:                  V32VendorConsentWithP1P2P3V32RestrictionRequireConsent,
			wantConsentPurposeResult: true,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    true,
		},
		{
			description: "enforce vendors on, vendor legit interest Y",
			config: purposeConfig{
				EnforceVendors: true,
			},
			consent:                  V32VendorLIWithP1P2P3V32RestrictionRequireConsent,
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
		},
		{
			description: "enforce vendors on, vendor consent Y, enforce purpose off but overrides treats it as on",
			config: purposeConfig{
				EnforceVendors: true,
			},
			consent:                  V32VendorConsentWithP1P2P3V32RestrictionRequireConsent,
			overrides:                Overrides{enforcePurpose: true},
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
		},
		{
			description: "enforce vendors on, purpose consent Y, vendor consent Y, enforce purpose off but overrides treats it as on",
			config: purposeConfig{
				EnforceVendors: true,
			},
			consent:                  P1P2P3PurposeConsentAndV32VendorConsentWithP1P2P3V32RestrictionRequireConsent,
			overrides:                Overrides{enforcePurpose: true},
			wantConsentPurposeResult: true,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    true,
		},
		{
			description: "enforce purpose & vendors on, purpose consent Y, vendor consent N",
			config: purposeConfig{
				EnforcePurpose: true,
				EnforceVendors: true,
			},
			consent:                  P1P2P3PurposeConsentWithP1P2P3V32RestrictionRequireConsent,
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
		},
		{
			description: "enforce purpose & vendors on, purpose consent N, vendor consent Y",
			config: purposeConfig{
				EnforcePurpose: true,
				EnforceVendors: true,
			},
			consent:                  V32VendorConsentWithP1P2P3V32RestrictionRequireConsent,
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
		},
		{
			description: "enforce purpose & vendors on, purpose consent Y, vendor consent Y",
			config: purposeConfig{
				EnforcePurpose: true,
				EnforceVendors: true,
			},
			consent:                  P1P2P3PurposeConsentAndV32VendorConsentWithP1P2P3V32RestrictionRequireConsent,
			wantConsentPurposeResult: true,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    true,
		},
		{
			description: "enforce purpose & vendors on, legit interest Y, vendor legit interest N",
			config: purposeConfig{
				EnforcePurpose: true,
				EnforceVendors: true,
			},
			consent:                  P1P2P3PurposeLIWithP1P2P3V32RestrictionRequireConsent,
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
		},
		{
			description: "enforce purpose & vendors on, legit interest N, vendor legit interest Y",
			config: purposeConfig{
				EnforcePurpose: true,
				EnforceVendors: true,
			},
			consent:                  V32VendorLIWithP1P2P3V32RestrictionRequireConsent,
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
		},
		{
			description: "enforce purpose & vendors on, legit interest Y, vendor legit interest Y",
			config: purposeConfig{
				EnforcePurpose: true,
				EnforceVendors: true,
			},
			consent:                  P1P2P3PurposeLIAndV32VendorLIWithP1P2P3V32RestrictionRequireConsent,
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
		},
		{
			description: "enforce purpose & vendors on, bidder is a vendor exception",
			config: purposeConfig{
				EnforcePurpose:     true,
				EnforceVendors:     true,
				VendorExceptionMap: map[string]struct{}{appnexus: {}},
			},
			consent:                  NoConsentsWithP1P2P3V32RestrictionRequireConsent,
			wantConsentPurposeResult: true,
			wantLIPurposeResult:      true,
			wantFlexPurposeResult:    true,
		},
		{
			description: "enforce purpose & vendors on, bidder is a vendor exception but overrides disallow them",
			config: purposeConfig{
				EnforcePurpose:     true,
				EnforceVendors:     true,
				VendorExceptionMap: map[string]struct{}{appnexus: {}},
			},
			consent:                  NoConsentsWithP1P2P3V32RestrictionRequireConsent,
			overrides:                Overrides{blockVendorExceptions: true},
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
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

		vendor := getVendorList(t).Vendor(appnexusID)
		vendorInfo := VendorInfo{vendorID: appnexusID, vendor: vendor}
		enforcer := FullEnforcement{cfg: tt.config}

		enforcer.cfg.PurposeID = consentconstants.Purpose(1)
		consentPurposeResult := enforcer.LegalBasis(vendorInfo, appnexus, consentMeta, tt.overrides)
		assert.Equal(t, tt.wantConsentPurposeResult, consentPurposeResult, tt.description+" -- GVL consent purpose")

		enforcer.cfg.PurposeID = consentconstants.Purpose(2)
		LIPurposeresult := enforcer.LegalBasis(vendorInfo, appnexus, consentMeta, tt.overrides)
		assert.Equal(t, tt.wantLIPurposeResult, LIPurposeresult, tt.description+" -- GVL LI purpose")

		enforcer.cfg.PurposeID = consentconstants.Purpose(3)
		flexPurposeResult := enforcer.LegalBasis(vendorInfo, appnexus, consentMeta, tt.overrides)
		assert.Equal(t, tt.wantFlexPurposeResult, flexPurposeResult, tt.description+" -- GVL flex purpose")
	}
}

func TestLegalBasisWithPubRestrictionRequireLI(t *testing.T) {
	var (
		appnexus   = string(openrtb_ext.BidderAppnexus)
		appnexusID = uint16(32)
	)

	NoConsentsWithP1P2P3V32RestrictionRequireLI := "CPfFkMAPfFkMAAAAAAENCgCAAAAAAAAAAAAAAQAAAAAAAIAAAAAAAGDAAgAgCgAQAQBwAIAIAAAA"
	P1P2P3PurposeConsentWithP1P2P3V32RestrictionRequireLI := "CPfFkMAPfFkMAAAAAAENCgCAAOAAAAAAAAAAAQAAAAAAAIAAAAAAAGDAAgAgCgAQAQBwAIAIAAAA"
	P1P2P3PurposeLIWithP1P2P3V32RestrictionRequireLI := "CPfFkMAPfFkMAAAAAAENCgCAAAAAAOAAAAAAAQAAAAAAAIAAAAAAAGDAAgAgCgAQAQBwAIAIAAAA"
	V32VendorConsentWithP1P2P3V32RestrictionRequireLI := "CPfFkMAPfFkMAAAAAAENCgCAAAAAAAAAAAAAAQAAAAAEAIAAAAAAAGDAAgAgCgAQAQBwAIAIAAAA"
	V32VendorLIWithP1P2P3V32RestrictionRequireLI := "CPfFkMAPfFkMAAAAAAENCgCAAAAAAAAAAAAAAQAAAAAAAIAAAAACAGDAAgAgCgAQAQBwAIAIAAAA"
	P1P2P3PurposeConsentAndV32VendorConsentWithP1P2P3V32RestrictionRequireLI := "CPfFkMAPfFkMAAAAAAENCgCAAOAAAAAAAAAAAQAAAAAEAIAAAAAAAGDAAgAgCgAQAQBwAIAIAAAA"
	P1P2P3PurposeLIAndV32VendorLIWithP1P2P3V32RestrictionRequireLI := "CPfFkMAPfFkMAAAAAAENCgCAAAAAAOAAAAAAAQAAAAAAAIAAAAACAGDAAgAgCgAQAQBwAIAIAAAA"

	tests := []struct {
		description              string
		config                   purposeConfig
		consent                  string
		overrides                Overrides
		wantConsentPurposeResult bool
		wantLIPurposeResult      bool
		wantFlexPurposeResult    bool
	}{
		{
			description: "enforce purpose & vendors off",
			config: purposeConfig{
				EnforcePurpose: false,
				EnforceVendors: false,
			},
			consent:                  NoConsentsWithP1P2P3V32RestrictionRequireLI,
			wantConsentPurposeResult: true,
			wantLIPurposeResult:      true,
			wantFlexPurposeResult:    true,
		},
		{
			description: "enforce purpose on, purpose consent N, legit interest N",
			config: purposeConfig{
				EnforcePurpose: true,
			},
			consent:                  NoConsentsWithP1P2P3V32RestrictionRequireLI,
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
		},
		{
			description: "enforce purpose on, purpose consent Y",
			config: purposeConfig{
				EnforcePurpose: true,
			},
			consent:                  P1P2P3PurposeConsentWithP1P2P3V32RestrictionRequireLI,
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
		},
		{
			description: "enforce purpose on, legit interest Y",
			config: purposeConfig{
				EnforcePurpose: true,
			},
			consent:                  P1P2P3PurposeLIWithP1P2P3V32RestrictionRequireLI,
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      true,
			wantFlexPurposeResult:    true,
		},
		{
			description: "enforce purpose on, vendor legit interest Y, enforce vendors off but overrides treats it as on",
			config: purposeConfig{
				EnforcePurpose: true,
			},
			consent:                  P1P2P3PurposeLIWithP1P2P3V32RestrictionRequireLI,
			overrides:                Overrides{enforceVendors: true},
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
		},
		{
			description: "enforce purpose on, vendor legit interest Y, vendor consent Y, enforce vendors off but overrides treats it as on",
			config: purposeConfig{
				EnforcePurpose: true,
			},
			consent:                  P1P2P3PurposeLIAndV32VendorLIWithP1P2P3V32RestrictionRequireLI,
			overrides:                Overrides{enforceVendors: true},
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      true,
			wantFlexPurposeResult:    true,
		},
		{
			description: "enforce vendors on, vendor consent N, vendor legit interest N",
			config: purposeConfig{
				EnforceVendors: true,
			},
			consent:                  NoConsentsWithP1P2P3V32RestrictionRequireLI,
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
		},
		{
			description: "enforce vendors on, vendor consent Y",
			config: purposeConfig{
				EnforceVendors: true,
			},
			consent:                  V32VendorConsentWithP1P2P3V32RestrictionRequireLI,
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
		},
		{
			description: "enforce vendors on, vendor legit interest Y",
			config: purposeConfig{
				EnforceVendors: true,
			},
			consent:                  V32VendorLIWithP1P2P3V32RestrictionRequireLI,
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      true,
			wantFlexPurposeResult:    true,
		},
		{
			description: "enforce vendors on, vendor legit interest Y, enforce purpose off but overrides treats it as on",
			config: purposeConfig{
				EnforceVendors: true,
			},
			consent:                  V32VendorLIWithP1P2P3V32RestrictionRequireLI,
			overrides:                Overrides{enforcePurpose: true},
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
		},
		{
			description: "enforce vendors on, vendor legit interest Y, vendor consent Y, enforce purpose off but overrides treats it as on",
			config: purposeConfig{
				EnforceVendors: true,
			},
			consent:                  P1P2P3PurposeLIAndV32VendorLIWithP1P2P3V32RestrictionRequireLI,
			overrides:                Overrides{enforcePurpose: true},
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      true,
			wantFlexPurposeResult:    true,
		},
		{
			description: "enforce purpose & vendors on, purpose consent Y, vendor consent N",
			config: purposeConfig{
				EnforcePurpose: true,
				EnforceVendors: true,
			},
			consent:                  P1P2P3PurposeConsentWithP1P2P3V32RestrictionRequireLI,
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
		},
		{
			description: "enforce purpose & vendors on, purpose consent N, vendor consent Y",
			config: purposeConfig{
				EnforcePurpose: true,
				EnforceVendors: true,
			},
			consent:                  V32VendorConsentWithP1P2P3V32RestrictionRequireLI,
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
		},
		{
			description: "enforce purpose & vendors on, purpose consent Y, vendor consent Y",
			config: purposeConfig{
				EnforcePurpose: true,
				EnforceVendors: true,
			},
			consent:                  P1P2P3PurposeConsentAndV32VendorConsentWithP1P2P3V32RestrictionRequireLI,
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
		},
		{
			description: "enforce purpose & vendors on, legit interest Y, vendor legit interest N",
			config: purposeConfig{
				EnforcePurpose: true,
				EnforceVendors: true,
			},
			consent:                  P1P2P3PurposeLIWithP1P2P3V32RestrictionRequireLI,
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
		},
		{
			description: "enforce purpose & vendors on, legit interest N, vendor legit interest Y",
			config: purposeConfig{
				EnforcePurpose: true,
				EnforceVendors: true,
			},
			consent:                  V32VendorLIWithP1P2P3V32RestrictionRequireLI,
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
		},
		{
			description: "enforce purpose & vendors on, legit interest Y, vendor legit interest Y",
			config: purposeConfig{
				EnforcePurpose: true,
				EnforceVendors: true,
			},
			consent:                  P1P2P3PurposeLIAndV32VendorLIWithP1P2P3V32RestrictionRequireLI,
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      true,
			wantFlexPurposeResult:    true,
		},
		{
			description: "enforce purpose & vendors on, bidder is a vendor exception",
			config: purposeConfig{
				EnforcePurpose:     true,
				EnforceVendors:     true,
				VendorExceptionMap: map[string]struct{}{appnexus: {}},
			},
			consent:                  NoConsentsWithP1P2P3V32RestrictionRequireLI,
			wantConsentPurposeResult: true,
			wantLIPurposeResult:      true,
			wantFlexPurposeResult:    true,
		},
		{
			description: "enforce purpose & vendors on, bidder is a vendor exception but overrides disallow them",
			config: purposeConfig{
				EnforcePurpose:     true,
				EnforceVendors:     true,
				VendorExceptionMap: map[string]struct{}{appnexus: {}},
			},
			consent:                  NoConsentsWithP1P2P3V32RestrictionRequireLI,
			overrides:                Overrides{blockVendorExceptions: true},
			wantConsentPurposeResult: false,
			wantLIPurposeResult:      false,
			wantFlexPurposeResult:    false,
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

		vendor := getVendorList(t).Vendor(appnexusID)
		vendorInfo := VendorInfo{vendorID: appnexusID, vendor: vendor}
		enforcer := FullEnforcement{cfg: tt.config}

		enforcer.cfg.PurposeID = consentconstants.Purpose(1)
		consentPurposeResult := enforcer.LegalBasis(vendorInfo, appnexus, consentMeta, tt.overrides)
		assert.Equal(t, tt.wantConsentPurposeResult, consentPurposeResult, tt.description+" -- GVL consent purpose")

		enforcer.cfg.PurposeID = consentconstants.Purpose(2)
		LIPurposeresult := enforcer.LegalBasis(vendorInfo, appnexus, consentMeta, tt.overrides)
		assert.Equal(t, tt.wantLIPurposeResult, LIPurposeresult, tt.description+" -- GVL LI purpose")

		enforcer.cfg.PurposeID = consentconstants.Purpose(3)
		flexPurposeResult := enforcer.LegalBasis(vendorInfo, appnexus, consentMeta, tt.overrides)
		assert.Equal(t, tt.wantFlexPurposeResult, flexPurposeResult, tt.description+" -- GVL flex purpose")
	}
}

func TestLegalBasisWithoutVendor(t *testing.T) {
	appnexus := string(openrtb_ext.BidderAppnexus)
	P1P2P3PurposeConsent := "CPfCRQAPfCRQAAAAAAENCgCAAOAAAAAAAAAAAAAAAAAA"
	tests := []struct {
		name       string
		config     purposeConfig
		wantResult bool
	}{
		{
			name: "enforce_purpose_&_vendors_off",
			config: purposeConfig{
				EnforcePurpose: false,
				EnforceVendors: false,
			},
			wantResult: true,
		},
		{
			name: "enforce_purpose_on,_bidder_is_a_vendor_exception",
			config: purposeConfig{
				EnforcePurpose:     true,
				EnforceVendors:     false,
				VendorExceptionMap: map[string]struct{}{appnexus: {}},
			},
			wantResult: true,
		},
		{
			name: "enforce_purpose_on",
			config: purposeConfig{
				EnforcePurpose: true,
				EnforceVendors: false,
			},
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// convert consent string to TCF2 object
			parsedConsent, err := vendorconsent.ParseString(P1P2P3PurposeConsent)
			if err != nil {
				t.Fatalf("Failed to parse consent %s\n", P1P2P3PurposeConsent)
			}
			consentMeta, ok := parsedConsent.(tcf2.ConsentMetadata)
			if !ok {
				t.Fatalf("Failed to convert consent %s\n", P1P2P3PurposeConsent)
			}

			vendorInfo := VendorInfo{
				vendorID: 32,
				vendor:   nil,
			}

			enforcer := FullEnforcement{cfg: tt.config}
			enforcer.cfg.PurposeID = consentconstants.Purpose(3)

			result := enforcer.LegalBasis(vendorInfo, appnexus, consentMeta, Overrides{})
			assert.Equal(t, tt.wantResult, result)
		})
	}
}

func getVendorList(t *testing.T) vendorlist.VendorList {
	GVL := makeVendorList()

	marshaledGVL, err := jsonutil.Marshal(GVL)
	if err != nil {
		t.Fatalf("Failed to marshal GVL")
	}

	parsedGVL, err := vendorlist2.ParseEagerly(marshaledGVL)
	if err != nil {
		t.Fatalf("Failed to parse vendor list data. %v", err)
	}
	return parsedGVL
}

func makeVendorList() vendorList {
	return vendorList{
		VendorListVersion: 2,
		Vendors: map[string]*vendor{
			"32": {
				ID:               32,
				Purposes:         []int{1},
				LegIntPurposes:   []int{2},
				FlexiblePurposes: []int{3},
			},
		},
	}
}
