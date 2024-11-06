package gdpr

import (
	"testing"

	"github.com/prebid/go-gdpr/consentconstants"

	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"

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

		result := cfg.ChannelEnabled(config.ChannelWeb)

		assert.Equal(t, tt.wantIntegrationEnabled, result, tt.description)
	}
}

func TestPurposeEnforced(t *testing.T) {
	False := false
	True := true

	tests := []struct {
		description                    string
		givePurpose1HostEnforcement    bool
		givePurpose1AccountEnforcement *bool
		givePurpose2HostEnforcement    bool
		givePurpose2AccountEnforcement *bool
		givePurpose                    consentconstants.Purpose
		wantEnforced                   bool
	}{
		{
			description:                    "Purpose 1 set at account level - use account setting false",
			givePurpose1HostEnforcement:    true,
			givePurpose1AccountEnforcement: &False,
			givePurpose:                    1,
			wantEnforced:                   false,
		},
		{
			description:                    "Purpose 1 set at account level - use account setting true",
			givePurpose1HostEnforcement:    false,
			givePurpose1AccountEnforcement: &True,
			givePurpose:                    1,
			wantEnforced:                   true,
		},
		{
			description:                    "Purpose 1 not set at account level - use host setting false",
			givePurpose1HostEnforcement:    false,
			givePurpose1AccountEnforcement: nil,
			givePurpose:                    1,
			wantEnforced:                   false,
		},
		{
			description:                    "Purpose 1 not set at account level - use host setting true",
			givePurpose1HostEnforcement:    true,
			givePurpose1AccountEnforcement: nil,
			givePurpose:                    1,
			wantEnforced:                   true,
		},
		{
			description:                    "Some other purpose set at account level - use account setting true",
			givePurpose2HostEnforcement:    false,
			givePurpose2AccountEnforcement: &True,
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

func TestPurposeEnforcementAlgo(t *testing.T) {

	tests := []struct {
		description             string
		givePurpose1HostAlgo    config.TCF2EnforcementAlgo
		givePurpose1AccountAlgo config.TCF2EnforcementAlgo
		givePurpose2HostAlgo    config.TCF2EnforcementAlgo
		givePurpose2AccountAlgo config.TCF2EnforcementAlgo
		givePurpose             consentconstants.Purpose
		wantAlgo                config.TCF2EnforcementAlgo
	}{
		{
			description:             "Purpose 1 set at account level - use account setting basic",
			givePurpose1HostAlgo:    config.TCF2FullEnforcement,
			givePurpose1AccountAlgo: config.TCF2BasicEnforcement,
			givePurpose:             1,
			wantAlgo:                config.TCF2BasicEnforcement,
		},
		{
			description:             "Purpose 1 set at account level - use account setting full",
			givePurpose1HostAlgo:    config.TCF2BasicEnforcement,
			givePurpose1AccountAlgo: config.TCF2FullEnforcement,
			givePurpose:             1,
			wantAlgo:                config.TCF2FullEnforcement,
		},
		{
			description:             "Purpose 1 not set at account level - use host setting basic",
			givePurpose1HostAlgo:    config.TCF2BasicEnforcement,
			givePurpose1AccountAlgo: config.TCF2UndefinedEnforcement,
			givePurpose:             1,
			wantAlgo:                config.TCF2BasicEnforcement,
		},
		{
			description:             "Purpose 1 not set at account level - use host setting full",
			givePurpose1HostAlgo:    config.TCF2FullEnforcement,
			givePurpose1AccountAlgo: config.TCF2UndefinedEnforcement,
			givePurpose:             1,
			wantAlgo:                config.TCF2FullEnforcement,
		},
		{
			description:             "Some other purpose set at account level - use account setting basic",
			givePurpose2HostAlgo:    config.TCF2FullEnforcement,
			givePurpose2AccountAlgo: config.TCF2BasicEnforcement,
			givePurpose:             2,
			wantAlgo:                config.TCF2BasicEnforcement,
		},
	}

	for _, tt := range tests {
		cfg := tcf2Config{
			AccountConfig: config.AccountGDPR{
				Purpose1: config.AccountGDPRPurpose{
					EnforceAlgoID: tt.givePurpose1AccountAlgo,
				},
				Purpose2: config.AccountGDPRPurpose{
					EnforceAlgoID: tt.givePurpose2AccountAlgo,
				},
			},
			HostConfig: config.TCF2{
				Purpose1: config.TCF2Purpose{
					EnforceAlgoID: tt.givePurpose1HostAlgo,
				},
				Purpose2: config.TCF2Purpose{
					EnforceAlgoID: tt.givePurpose2HostAlgo,
				},
			},
		}
		MakeTCF2ConfigPurposeMaps(&cfg)

		result := cfg.PurposeEnforcementAlgo(consentconstants.Purpose(tt.givePurpose))

		assert.Equal(t, tt.wantAlgo, result, tt.description)
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

func TestPurposeVendorExceptions(t *testing.T) {
	tests := []struct {
		description                     string
		givePurpose1HostExceptionMap    map[string]struct{}
		givePurpose1AccountExceptionMap map[string]struct{}
		givePurpose2HostExceptionMap    map[string]struct{}
		givePurpose2AccountExceptionMap map[string]struct{}
		givePurpose                     consentconstants.Purpose
		wantExceptionMap                map[string]struct{}
	}{
		{
			description:                     "Purpose 1 exception list set at account level - use empty account list",
			givePurpose1HostExceptionMap:    map[string]struct{}{},
			givePurpose1AccountExceptionMap: map[string]struct{}{},
			givePurpose:                     1,
			wantExceptionMap:                map[string]struct{}{},
		},
		{
			description:                     "Purpose 1 exception list set at account level - use nonempty account list",
			givePurpose1HostExceptionMap:    map[string]struct{}{},
			givePurpose1AccountExceptionMap: map[string]struct{}{"appnexus": {}, "rubicon": {}},
			givePurpose:                     1,
			wantExceptionMap:                map[string]struct{}{"appnexus": {}, "rubicon": {}},
		},
		{
			description:                     "Purpose 1 exception list not set at account level - use empty host list",
			givePurpose1HostExceptionMap:    map[string]struct{}{},
			givePurpose1AccountExceptionMap: nil,
			givePurpose:                     1,
			wantExceptionMap:                map[string]struct{}{},
		},
		{
			description:                     "Purpose 1 exception list not set at account level - use nonempty host list",
			givePurpose1HostExceptionMap:    map[string]struct{}{"appnexus": {}, "rubicon": {}},
			givePurpose1AccountExceptionMap: nil,
			givePurpose:                     1,
			wantExceptionMap:                map[string]struct{}{"appnexus": {}, "rubicon": {}},
		},
		{
			description:                     "Purpose 1 exception list not set at account level or host level",
			givePurpose1HostExceptionMap:    nil,
			givePurpose1AccountExceptionMap: nil,
			givePurpose:                     1,
			wantExceptionMap:                map[string]struct{}{},
		},
		{
			description:                     "Some other purpose exception list set at account level",
			givePurpose2HostExceptionMap:    map[string]struct{}{},
			givePurpose2AccountExceptionMap: map[string]struct{}{"appnexus": {}, "rubicon": {}},
			givePurpose:                     2,
			wantExceptionMap:                map[string]struct{}{"appnexus": {}, "rubicon": {}},
		},
	}

	for _, tt := range tests {
		cfg := tcf2Config{
			AccountConfig: config.AccountGDPR{
				Purpose1: config.AccountGDPRPurpose{
					VendorExceptionMap: tt.givePurpose1AccountExceptionMap,
				},
				Purpose2: config.AccountGDPRPurpose{
					VendorExceptionMap: tt.givePurpose2AccountExceptionMap,
				},
			},
			HostConfig: config.TCF2{
				Purpose1: config.TCF2Purpose{
					VendorExceptionMap: tt.givePurpose1HostExceptionMap,
				},
				Purpose2: config.TCF2Purpose{
					VendorExceptionMap: tt.givePurpose2HostExceptionMap,
				},
			},
		}
		MakeTCF2ConfigPurposeMaps(&cfg)

		result := cfg.PurposeVendorExceptions(consentconstants.Purpose(tt.givePurpose))

		assert.Equal(t, tt.wantExceptionMap, result, tt.description)
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

func TestBasicEnforcementVendors(t *testing.T) {
	tests := []struct {
		description               string
		giveAccountBasicVendorMap map[string]struct{}
		wantBasicVendorMap        map[string]struct{}
	}{
		{
			description:               "Purpose 1 basic exception vendor list not set at account level",
			giveAccountBasicVendorMap: nil,
			wantBasicVendorMap:        map[string]struct{}{},
		},
		{
			description:               "Purpose 1 basic exception vendor list set at account level as empty list",
			giveAccountBasicVendorMap: map[string]struct{}{},
			wantBasicVendorMap:        map[string]struct{}{},
		},
		{
			description:               "Purpose 1 basic exception vendor list not set at account level as nonempty list",
			giveAccountBasicVendorMap: map[string]struct{}{"appnexus": {}, "rubicon": {}},
			wantBasicVendorMap:        map[string]struct{}{"appnexus": {}, "rubicon": {}},
		},
	}

	for _, tt := range tests {
		cfg := tcf2Config{
			AccountConfig: config.AccountGDPR{
				BasicEnforcementVendorsMap: tt.giveAccountBasicVendorMap,
			},
		}
		MakeTCF2ConfigPurposeMaps(&cfg)

		result := cfg.BasicEnforcementVendors()

		assert.Equal(t, tt.wantBasicVendorMap, result, tt.description)
	}
}
