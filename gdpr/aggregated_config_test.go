package gdpr

import (
	"testing"

	"github.com/prebid/go-gdpr/consentconstants"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/stretchr/testify/assert"
)

func MakeTCF2ConfigPurposeMaps(tcf2Config *tcf2Config) {
	tcf2Config.AccountConfig.PurposeConfigs = map[consentconstants.Purpose]*config.AccountGDPRPurpose{
		1:  &tcf2Config.AccountConfig.Purpose1,
		2:  &tcf2Config.AccountConfig.Purpose2,
		3:  &tcf2Config.AccountConfig.Purpose3,
		4:  &tcf2Config.AccountConfig.Purpose4,
		5:  &tcf2Config.AccountConfig.Purpose5,
		6:  &tcf2Config.AccountConfig.Purpose6,
		7:  &tcf2Config.AccountConfig.Purpose7,
		8:  &tcf2Config.AccountConfig.Purpose8,
		9:  &tcf2Config.AccountConfig.Purpose9,
		10: &tcf2Config.AccountConfig.Purpose10,
	}

	tcf2Config.HostConfig.PurposeConfigs = map[consentconstants.Purpose]*config.TCF2Purpose{
		1:  &tcf2Config.HostConfig.Purpose1,
		2:  &tcf2Config.HostConfig.Purpose2,
		3:  &tcf2Config.HostConfig.Purpose3,
		4:  &tcf2Config.HostConfig.Purpose4,
		5:  &tcf2Config.HostConfig.Purpose5,
		6:  &tcf2Config.HostConfig.Purpose6,
		7:  &tcf2Config.HostConfig.Purpose7,
		8:  &tcf2Config.HostConfig.Purpose8,
		9:  &tcf2Config.HostConfig.Purpose9,
		10: &tcf2Config.HostConfig.Purpose10,
	}
}

func TestIntegrationEnabled(t *testing.T) {
	tests := []struct {
		description            string
		giveHostGDPREnabled    bool
		giveAccountGDPREnabled *bool
		wantIntegrationEnabled bool
	}{
		{
			description:            "Set at account level - use account setting false",
			giveHostGDPREnabled:    true,
			giveAccountGDPREnabled: &[]bool{false}[0],
			wantIntegrationEnabled: false,
		},
		{
			description:            "Set at account level - use account setting true",
			giveHostGDPREnabled:    false,
			giveAccountGDPREnabled: &[]bool{true}[0],
			wantIntegrationEnabled: true,
		},
		{
			description:            "Not set at account level - use host setting false",
			giveHostGDPREnabled:    false,
			giveAccountGDPREnabled: nil,
			wantIntegrationEnabled: false,
		},
		{
			description:            "Not set at account level - use host setting true",
			giveHostGDPREnabled:    true,
			giveAccountGDPREnabled: nil,
			wantIntegrationEnabled: true,
		},
	}

	for _, tt := range tests {
		cfg := tcf2Config{
			AccountConfig: config.AccountGDPR{
				Enabled: tt.giveAccountGDPREnabled,
			},
			HostConfig: config.TCF2{
				Enabled: tt.giveHostGDPREnabled,
			},
		}

		result := cfg.IntegrationEnabled(config.IntegrationTypeWeb)

		assert.Equal(t, tt.wantIntegrationEnabled, result, tt.description)
	}
}

func TestPurposeEnforced(t *testing.T) {
	tests := []struct {
		description                    string
		givePurpose1HostEnforcement    string
		givePurpose1AccountEnforcement string
		givePurpose2HostEnforcement    string
		givePurpose2AccountEnforcement string
		givePurpose                    consentconstants.Purpose
		wantEnforced                   bool
	}{
		{
			description:                    "Purpose 1 set at account level - use account setting false",
			givePurpose1HostEnforcement:    config.TCF2FullEnforcement,
			givePurpose1AccountEnforcement: config.TCF2NoEnforcement,
			givePurpose:                    1,
			wantEnforced:                   false,
		},
		{
			description:                    "Purpose 1 set at account level - use account setting true",
			givePurpose1HostEnforcement:    config.TCF2NoEnforcement,
			givePurpose1AccountEnforcement: config.TCF2FullEnforcement,
			givePurpose:                    1,
			wantEnforced:                   true,
		},
		{
			description:                    "Purpose 1 not set at account level - use host setting false",
			givePurpose1HostEnforcement:    config.TCF2NoEnforcement,
			givePurpose1AccountEnforcement: "",
			givePurpose:                    1,
			wantEnforced:                   false,
		},
		{
			description:                    "Purpose 1 not set at account level - use host setting true",
			givePurpose1HostEnforcement:    config.TCF2FullEnforcement,
			givePurpose1AccountEnforcement: "",
			givePurpose:                    1,
			wantEnforced:                   true,
		},
		{
			description:                    "Some other purpose set at account level - use account setting true",
			givePurpose2HostEnforcement:    config.TCF2NoEnforcement,
			givePurpose2AccountEnforcement: config.TCF2FullEnforcement,
			givePurpose:                    2,
			wantEnforced:                   true,
		},
	}

	for _, tt := range tests {
		cfg := tcf2Config{
			AccountConfig: config.AccountGDPR{
				Purpose1: config.AccountGDPRPurpose{
					EnforcePurpose: tt.givePurpose1AccountEnforcement,
				},
				Purpose2: config.AccountGDPRPurpose{
					EnforcePurpose: tt.givePurpose2AccountEnforcement,
				},
			},
			HostConfig: config.TCF2{
				Purpose1: config.TCF2Purpose{
					EnforcePurpose: tt.givePurpose1HostEnforcement,
				},
				Purpose2: config.TCF2Purpose{
					EnforcePurpose: tt.givePurpose2HostEnforcement,
				},
			},
		}
		MakeTCF2ConfigPurposeMaps(&cfg)

		result := cfg.PurposeEnforced(consentconstants.Purpose(tt.givePurpose))

		assert.Equal(t, tt.wantEnforced, result, tt.description)
	}
}

func TestPurposeEnforcingVendors(t *testing.T) {
	tests := []struct {
		description                  string
		givePurpose1HostEnforcing    bool
		givePurpose1AccountEnforcing *bool
		givePurpose2HostEnforcing    bool
		givePurpose2AccountEnforcing *bool
		givePurpose                  consentconstants.Purpose
		wantEnforcing                bool
	}{
		{
			description:                  "Purpose 1 set at account level - use account setting false",
			givePurpose1HostEnforcing:    true,
			givePurpose1AccountEnforcing: &[]bool{false}[0],
			givePurpose:                  1,
			wantEnforcing:                false,
		},
		{
			description:                  "Purpose 1 set at account level - use account setting true",
			givePurpose1HostEnforcing:    false,
			givePurpose1AccountEnforcing: &[]bool{true}[0],
			givePurpose:                  1,
			wantEnforcing:                true,
		},
		{
			description:                  "Purpose 1 not set at account level - use host setting false",
			givePurpose1HostEnforcing:    false,
			givePurpose1AccountEnforcing: nil,
			givePurpose:                  1,
			wantEnforcing:                false,
		},
		{
			description:                  "Purpose 1 not set at account level - use host setting true",
			givePurpose1HostEnforcing:    true,
			givePurpose1AccountEnforcing: nil,
			givePurpose:                  1,
			wantEnforcing:                true,
		},
		{
			description:                  "Some other purpose set at account level - use account setting true",
			givePurpose2HostEnforcing:    false,
			givePurpose2AccountEnforcing: &[]bool{true}[0],
			givePurpose:                  2,
			wantEnforcing:                true,
		},
	}

	for _, tt := range tests {
		cfg := tcf2Config{
			AccountConfig: config.AccountGDPR{
				Purpose1: config.AccountGDPRPurpose{
					EnforceVendors: tt.givePurpose1AccountEnforcing,
				},
				Purpose2: config.AccountGDPRPurpose{
					EnforceVendors: tt.givePurpose2AccountEnforcing,
				},
			},
			HostConfig: config.TCF2{
				Purpose1: config.TCF2Purpose{
					EnforceVendors: tt.givePurpose1HostEnforcing,
				},
				Purpose2: config.TCF2Purpose{
					EnforceVendors: tt.givePurpose2HostEnforcing,
				},
			},
		}
		MakeTCF2ConfigPurposeMaps(&cfg)

		result := cfg.PurposeEnforcingVendors(consentconstants.Purpose(tt.givePurpose))

		assert.Equal(t, tt.wantEnforcing, result, tt.description)
	}
}

func TestPurposeVendorException(t *testing.T) {
	tests := []struct {
		description                           string
		givePurpose1HostVendorExceptionMap    map[openrtb_ext.BidderName]struct{}
		givePurpose1AccountVendorExceptionMap map[openrtb_ext.BidderName]struct{}
		givePurpose2HostVendorExceptionMap    map[openrtb_ext.BidderName]struct{}
		givePurpose2AccountVendorExceptionMap map[openrtb_ext.BidderName]struct{}
		givePurpose                           consentconstants.Purpose
		giveBidder                            openrtb_ext.BidderName
		wantVendorException                   bool
	}{
		{
			description:                           "Purpose 1 exception list set at account level - vendor found",
			givePurpose1HostVendorExceptionMap:    map[openrtb_ext.BidderName]struct{}{},
			givePurpose1AccountVendorExceptionMap: map[openrtb_ext.BidderName]struct{}{"appnexus": {}, "rubicon": {}},
			givePurpose:                           1,
			giveBidder:                            "appnexus",
			wantVendorException:                   true,
		},
		{
			description:                           "Purpose 1 exception list set at account level - vendor not found",
			givePurpose1HostVendorExceptionMap:    map[openrtb_ext.BidderName]struct{}{},
			givePurpose1AccountVendorExceptionMap: map[openrtb_ext.BidderName]struct{}{"rubicon": {}},
			givePurpose:                           1,
			giveBidder:                            "appnexus",
			wantVendorException:                   false,
		},
		{
			description:                           "Purpose 1 exception list not set at account level - vendor found in host list",
			givePurpose1HostVendorExceptionMap:    map[openrtb_ext.BidderName]struct{}{"appnexus": {}, "rubicon": {}},
			givePurpose1AccountVendorExceptionMap: nil,
			givePurpose:                           1,
			giveBidder:                            "appnexus",
			wantVendorException:                   true,
		},
		{
			description:                           "Purpose 1 exception list not set at account level - vendor not found in host list",
			givePurpose1HostVendorExceptionMap:    map[openrtb_ext.BidderName]struct{}{"rubicon": {}},
			givePurpose1AccountVendorExceptionMap: nil,
			givePurpose:                           1,
			giveBidder:                            "appnexus",
			wantVendorException:                   false,
		},
		{
			description:                           "Purpose 1 exception list not set at account level or host level - vendor not found",
			givePurpose1HostVendorExceptionMap:    nil,
			givePurpose1AccountVendorExceptionMap: nil,
			givePurpose:                           1,
			giveBidder:                            "appnexus",
			wantVendorException:                   false,
		},
		{
			description:                           "Some other purpose exception list set at account level - vendor found",
			givePurpose2HostVendorExceptionMap:    map[openrtb_ext.BidderName]struct{}{},
			givePurpose2AccountVendorExceptionMap: map[openrtb_ext.BidderName]struct{}{"appnexus": {}, "rubicon": {}},
			givePurpose:                           2,
			giveBidder:                            "appnexus",
			wantVendorException:                   true,
		},
	}

	for _, tt := range tests {
		cfg := tcf2Config{
			AccountConfig: config.AccountGDPR{
				Purpose1: config.AccountGDPRPurpose{
					VendorExceptionMap: tt.givePurpose1AccountVendorExceptionMap,
				},
				Purpose2: config.AccountGDPRPurpose{
					VendorExceptionMap: tt.givePurpose2AccountVendorExceptionMap,
				},
			},
			HostConfig: config.TCF2{
				Purpose1: config.TCF2Purpose{
					VendorExceptionMap: tt.givePurpose1HostVendorExceptionMap,
				},
				Purpose2: config.TCF2Purpose{
					VendorExceptionMap: tt.givePurpose2HostVendorExceptionMap,
				},
			},
		}
		MakeTCF2ConfigPurposeMaps(&cfg)

		result := cfg.PurposeVendorException(consentconstants.Purpose(tt.givePurpose), tt.giveBidder)

		assert.Equal(t, tt.wantVendorException, result, tt.description)
	}
}

func TestFeatureOneEnforced(t *testing.T) {
	tests := []struct {
		description          string
		giveHostEnforcing    bool
		giveAccountEnforcing *bool
		wantEnforcing        bool
		wantEnabled          bool
	}{
		{
			description:          "Feature 1 enforced set at account level - use account setting false",
			giveHostEnforcing:    true,
			giveAccountEnforcing: &[]bool{false}[0],
			wantEnforcing:        false,
		},
		{
			description:          "Feature 1 enforced set at account level - use account setting true",
			giveHostEnforcing:    false,
			giveAccountEnforcing: &[]bool{true}[0],
			wantEnforcing:        true,
		},
		{
			description:          "Feature 1 enforced not set at account level - use host setting false",
			giveHostEnforcing:    false,
			giveAccountEnforcing: nil,
			wantEnforcing:        false,
		},
		{
			description:          "Feature 1 enforced not set at account level - use host setting true",
			giveHostEnforcing:    true,
			giveAccountEnforcing: nil,
			wantEnforcing:        true,
		},
	}

	for _, tt := range tests {
		cfg := tcf2Config{
			AccountConfig: config.AccountGDPR{
				SpecialFeature1: config.AccountGDPRSpecialFeature{
					Enforce: tt.giveAccountEnforcing,
				},
			},
			HostConfig: config.TCF2{
				SpecialFeature1: config.TCF2SpecialFeature{
					Enforce: tt.giveHostEnforcing,
				},
			},
		}

		result := cfg.FeatureOneEnforced()

		assert.Equal(t, tt.wantEnforcing, result, tt.description)
	}
}

func TestFeatureOneVendorException(t *testing.T) {
	tests := []struct {
		description                   string
		giveHostVendorExceptionMap    map[openrtb_ext.BidderName]struct{}
		giveAccountVendorExceptionMap map[openrtb_ext.BidderName]struct{}
		giveBidder                    openrtb_ext.BidderName
		wantVendorException           bool
	}{
		{
			description:                   "Feature 1 exception list set at account level - vendor found",
			giveHostVendorExceptionMap:    map[openrtb_ext.BidderName]struct{}{},
			giveAccountVendorExceptionMap: map[openrtb_ext.BidderName]struct{}{"appnexus": {}, "rubicon": {}},
			giveBidder:                    "appnexus",
			wantVendorException:           true,
		},
		{
			description:                   "Feature 1 exception list set at account level - vendor not found",
			giveHostVendorExceptionMap:    map[openrtb_ext.BidderName]struct{}{},
			giveAccountVendorExceptionMap: map[openrtb_ext.BidderName]struct{}{"rubicon": {}},
			giveBidder:                    "appnexus",
			wantVendorException:           false,
		},
		{
			description:                   "Feature 1 exception list not set at account level - vendor found in host list",
			giveHostVendorExceptionMap:    map[openrtb_ext.BidderName]struct{}{"appnexus": {}, "rubicon": {}},
			giveAccountVendorExceptionMap: nil,
			giveBidder:                    "appnexus",
			wantVendorException:           true,
		},
		{
			description:                   "Feature 1 exception list not set at account level - vendor not found in host list",
			giveHostVendorExceptionMap:    map[openrtb_ext.BidderName]struct{}{"rubicon": {}},
			giveAccountVendorExceptionMap: nil,
			giveBidder:                    "appnexus",
			wantVendorException:           false,
		},
		{
			description:                   "Feature 1 exception list not set at account level or host level - vendor not found",
			giveHostVendorExceptionMap:    nil,
			giveAccountVendorExceptionMap: nil,
			giveBidder:                    "appnexus",
			wantVendorException:           false,
		},
	}

	for _, tt := range tests {
		cfg := tcf2Config{
			AccountConfig: config.AccountGDPR{
				SpecialFeature1: config.AccountGDPRSpecialFeature{
					VendorExceptionMap: tt.giveAccountVendorExceptionMap,
				},
			},
			HostConfig: config.TCF2{
				SpecialFeature1: config.TCF2SpecialFeature{
					VendorExceptionMap: tt.giveHostVendorExceptionMap,
				},
			},
		}

		result := cfg.FeatureOneVendorException(tt.giveBidder)

		assert.Equal(t, tt.wantVendorException, result, tt.description)
	}
}

func TestPurposeOneTreatmentEnabled(t *testing.T) {
	tests := []struct {
		description        string
		giveHostEnabled    bool
		giveAccountEnabled *bool
		wantEnabled        bool
	}{
		{
			description:        "Purpose 1 treatment enabled set at account level - use account setting false",
			giveHostEnabled:    true,
			giveAccountEnabled: &[]bool{false}[0],
			wantEnabled:        false,
		},
		{
			description:        "Purpose 1 treatment enabled set at account level - use account setting true",
			giveHostEnabled:    false,
			giveAccountEnabled: &[]bool{true}[0],
			wantEnabled:        true,
		},
		{
			description:        "Purpose 1 treatment enabled not set at account level - use host setting false",
			giveHostEnabled:    false,
			giveAccountEnabled: nil,
			wantEnabled:        false,
		},
		{
			description:        "Purpose 1 treatment enabled not set at account level - use host setting true",
			giveHostEnabled:    true,
			giveAccountEnabled: nil,
			wantEnabled:        true,
		},
	}

	for _, tt := range tests {
		cfg := tcf2Config{
			AccountConfig: config.AccountGDPR{
				PurposeOneTreatment: config.AccountGDPRPurposeOneTreatment{
					Enabled: tt.giveAccountEnabled,
				},
			},
			HostConfig: config.TCF2{
				PurposeOneTreatment: config.TCF2PurposeOneTreatment{
					Enabled: tt.giveHostEnabled,
				},
			},
		}

		result := cfg.PurposeOneTreatmentEnabled()

		assert.Equal(t, tt.wantEnabled, result, tt.description)
	}
}

func TestPurposeOneTreatmentAllowed(t *testing.T) {
	tests := []struct {
		description              string
		giveHostAccessAllowed    bool
		giveAccountAccessAllowed *bool
		wantAccessAllowed        bool
	}{
		{
			description:              "Purpose 1 treatment access allowed set at account level - use account setting false",
			giveHostAccessAllowed:    true,
			giveAccountAccessAllowed: &[]bool{false}[0],
			wantAccessAllowed:        false,
		},
		{
			description:              "Purpose 1 treatment access allowed set at account level - use account setting true",
			giveHostAccessAllowed:    false,
			giveAccountAccessAllowed: &[]bool{true}[0],
			wantAccessAllowed:        true,
		},
		{
			description:              "Purpose 1 treatment access allowed not set at account level - use host setting false",
			giveHostAccessAllowed:    false,
			giveAccountAccessAllowed: nil,
			wantAccessAllowed:        false,
		},
		{
			description:              "Purpose 1 treatment access allowed not set at account level - use host setting true",
			giveHostAccessAllowed:    true,
			giveAccountAccessAllowed: nil,
			wantAccessAllowed:        true,
		},
	}

	for _, tt := range tests {
		cfg := tcf2Config{
			AccountConfig: config.AccountGDPR{
				PurposeOneTreatment: config.AccountGDPRPurposeOneTreatment{
					AccessAllowed: tt.giveAccountAccessAllowed,
				},
			},
			HostConfig: config.TCF2{
				PurposeOneTreatment: config.TCF2PurposeOneTreatment{
					AccessAllowed: tt.giveHostAccessAllowed,
				},
			},
		}

		result := cfg.PurposeOneTreatmentAccessAllowed()

		assert.Equal(t, tt.wantAccessAllowed, result, tt.description)
	}
}

func TestBasicEnforcementVendor(t *testing.T) {
	tests := []struct {
		description                          string
		giveAccountBasicEnforcementVendorMap map[string]struct{}
		giveBidder                           openrtb_ext.BidderName
		wantBasicEnforcement                 bool
	}{
		{
			description:                          "Basic enforcement vendor list set at account level - vendor found",
			giveAccountBasicEnforcementVendorMap: map[string]struct{}{"appnexus": {}, "rubicon": {}},
			giveBidder:                           "appnexus",
			wantBasicEnforcement:                 true,
		},
		{
			description:                          "Basic enforcement vendor list set at account level - vendor not found",
			giveAccountBasicEnforcementVendorMap: map[string]struct{}{"rubicon": {}},
			giveBidder:                           "appnexus",
			wantBasicEnforcement:                 false,
		},
		{
			description:                          "Basic enforcement vendor list not set at account level - vendor not found",
			giveAccountBasicEnforcementVendorMap: nil,
			giveBidder:                           "appnexus",
			wantBasicEnforcement:                 false,
		},
	}

	for _, tt := range tests {
		cfg := tcf2Config{
			AccountConfig: config.AccountGDPR{
				BasicEnforcementVendorsMap: tt.giveAccountBasicEnforcementVendorMap,
			},
		}

		result := cfg.BasicEnforcementVendor(tt.giveBidder)

		assert.Equal(t, tt.wantBasicEnforcement, result, tt.description)
	}
}
