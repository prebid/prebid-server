package gdpr

import (
	"context"
	"testing"

	"github.com/prebid/go-gdpr/consentconstants"
	"github.com/prebid/go-gdpr/vendorlist"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestNewAnalyticsPolicy(t *testing.T) {
	tests := []struct {
		name        string
		gdprEnabled bool
		expectType  PrivacyPolicy
	}{
		{
			name:        "gdpr-disabled-returns-allowall",
			gdprEnabled: false,
			expectType:  &AllowAllAnalytics{},
		},
		{
			name:        "gdpr-enabled-returns-analytics-policy",
			gdprEnabled: true,
			expectType:  &analyticsPolicy{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tcf2Cfg := &mockTCF2ConfigReader{}
			vendorIDs := map[openrtb_ext.BidderName]uint16{}

			vendorListFetcher := func(ctx context.Context, specVersion, listVersion uint16) (vendorlist.VendorList, error) {
				return nil, nil
			}
			purposeEnforcerBuilder := NewPurposeEnforcerBuilder(tcf2Cfg)

			gdprConfig := config.GDPR{
				Enabled: tt.gdprEnabled,
			}

			policy := NewAnalyticsPolicy(gdprConfig, tcf2Cfg, vendorIDs, vendorListFetcher, purposeEnforcerBuilder, SignalYes, "some-consent")

			assert.NotNil(t, policy)
			assert.IsType(t, tt.expectType, policy)
		})
	}
}

func TestAnalyticsPolicy_Allow(t *testing.T) {
	// appnexus consents to all purposes and features
	validAppNexusConsentToAll := "CQT56BeQT56BeAEAAAENCZCAAP_AAH_AAAAAAQAAAAAEAIAAAAACAAA"
	// appnexus does not consent to any purposes or features
	validAppNexusConsentToNone := "CQT56BeQT56BeAEAAAENCZCAAAAAAAAAAAAAAQAAAAAAAIAAAAAAAAA"
	// appnexus consent with bad version
	invalidAppNexusConsentBadVersion := "DPuKGCPPuKGCPNEAAAENCZCAAAAAAAAAAAAAAAAAAAAA"

	tests := []struct {
		name            string
		consent         string
		gdprSignal      Signal
		enforcementAlgo config.TCF2EnforcementAlgo
		purposeEnforced bool
		expectAllow     bool
		fetchError      bool
	}{
		{
			name:        "test-signal-no",
			gdprSignal:  SignalNo,
			consent:     validAppNexusConsentToAll,
			expectAllow: true,
		},
		{
			name:        "test-signal-ambiguous",
			gdprSignal:  SignalAmbiguous,
			consent:     validAppNexusConsentToAll,
			expectAllow: true,
		},
		{
			name:            "test-empty-consent-when-purpose-enforced",
			gdprSignal:      SignalYes,
			consent:         "",
			purposeEnforced: true,
			expectAllow:     false,
		},
		{
			name:            "test-empty-consent-when-purpose-not-enforced",
			gdprSignal:      SignalYes,
			consent:         "",
			purposeEnforced: false,
			expectAllow:     true,
		},
		{
			name:            "test-consent-error-when-purpose-enforced",
			gdprSignal:      SignalYes,
			consent:         invalidAppNexusConsentBadVersion,
			purposeEnforced: true,
			expectAllow:     false,
		},
		{
			name:            "test-consent-error-when-purpose-not-enforced",
			gdprSignal:      SignalYes,
			consent:         invalidAppNexusConsentBadVersion,
			purposeEnforced: false,
			expectAllow:     true,
		},
		{
			name:            "test-fetch-vendor-error",
			gdprSignal:      SignalYes,
			consent:         validAppNexusConsentToAll,
			purposeEnforced: true,
			fetchError:      true,
			expectAllow:     false,
		},
		{
			name:            "test-basic-enforcement",
			gdprSignal:      SignalYes,
			consent:         validAppNexusConsentToAll,
			enforcementAlgo: config.TCF2BasicEnforcement,
			purposeEnforced: true,
			expectAllow:     true,
		},
		{
			name:            "test-full-enforcement-allowed",
			gdprSignal:      SignalYes,
			consent:         validAppNexusConsentToAll,
			purposeEnforced: true,
			expectAllow:     true,
		},
		{
			name:            "test-full-enforcement-not-allowed",
			gdprSignal:      SignalYes,
			consent:         validAppNexusConsentToNone,
			purposeEnforced: true,
			expectAllow:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := mockTCF2ConfigReader{
				enforcementAlgo:      tt.enforcementAlgo,
				purposeEnforcedValue: tt.purposeEnforced,
			}
			purposeEnforcerBuilder := NewPurposeEnforcerBuilder(&cfg)

			vendorIDs := map[openrtb_ext.BidderName]uint16{openrtb_ext.BidderAppnexus: 32}

			// vendor list setup
			vendorListData := MarshalVendorList(vendorList{
				VendorListVersion: 2,
				Vendors: map[string]*vendor{
					"32": {
						ID:               32,
						Purposes:         []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
						LegIntPurposes:   []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
						FlexiblePurposes: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
					},
				},
			})
			fetcher := listFetcher(map[uint16]map[uint16]vendorlist.VendorList{
				2: {
					153: parseVendorListDataV2(t, vendorListData),
				},
			})
			if tt.fetchError {
				fetcher = listFetcher(map[uint16]map[uint16]vendorlist.VendorList{})
			}

			analyticsPolicy := &analyticsPolicy{
				fetchVendorList:        fetcher,
				purposeEnforcerBuilder: purposeEnforcerBuilder,
				vendorIDs:              vendorIDs,
				cfg:                    &cfg,
				consent:                tt.consent,
				gdprSignal:             tt.gdprSignal,
				ctx:                    context.Background(),
			}
			result := analyticsPolicy.Allow("appnexus")
			assert.Equal(t, tt.expectAllow, result)
		})
	}
}

// Mock implementations for testing

type mockTCF2ConfigReader struct {
	enforcementAlgo      config.TCF2EnforcementAlgo
	purposeAllowed       bool
	purposeEnforcedValue bool
}

func (m *mockTCF2ConfigReader) BasicEnforcementVendors() map[string]struct{} {
	return make(map[string]struct{})
}

func (m *mockTCF2ConfigReader) ChannelEnabled(channel config.ChannelType) bool {
	return false
}

func (m *mockTCF2ConfigReader) IsEnabled() bool {
	return true
}

func (m *mockTCF2ConfigReader) FeatureOneEnforced() bool {
	return false
}

func (m *mockTCF2ConfigReader) FeatureOneVendorException(vendorName openrtb_ext.BidderName) bool {
	return false
}

func (m *mockTCF2ConfigReader) PurposeEnforced(purposeID consentconstants.Purpose) bool {
	return m.purposeEnforcedValue
}

func (m *mockTCF2ConfigReader) PurposeEnforcementAlgo(consentconstants.Purpose) config.TCF2EnforcementAlgo {
	if m.enforcementAlgo == config.TCF2BasicEnforcement {
		return config.TCF2BasicEnforcement
	}
	return config.TCF2FullEnforcement
}

func (m *mockTCF2ConfigReader) PurposeEnforcingVendors(purposeID consentconstants.Purpose) bool {
	return true
}

func (m *mockTCF2ConfigReader) PurposeVendorExceptions(purposeID consentconstants.Purpose) map[string]struct{} {
	return make(map[string]struct{})
}

func (m *mockTCF2ConfigReader) PurposeLITransparency(purposeID consentconstants.Purpose) bool {
	return false
}

func (m *mockTCF2ConfigReader) PurposeOneTreatmentEnabled() bool {
	return false
}

func (m *mockTCF2ConfigReader) PurposeOneTreatmentAccessAllowed() bool {
	return false
}
