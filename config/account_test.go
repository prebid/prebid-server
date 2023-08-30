package config

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/prebid/go-gdpr/consentconstants"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestAccountGDPREnabledForChannelType(t *testing.T) {
	trueValue, falseValue := true, false

	tests := []struct {
		description                      string
		giveChannelType                  ChannelType
		giveGDPREnabled                  *bool
		giveWebGDPREnabled               *bool
		giveWebGDPREnabledForIntegration *bool
		wantEnabled                      *bool
	}{
		{
			description:                      "GDPR Web channel enabled, general GDPR disabled",
			giveChannelType:                  ChannelWeb,
			giveGDPREnabled:                  &falseValue,
			giveWebGDPREnabled:               &trueValue,
			giveWebGDPREnabledForIntegration: nil,
			wantEnabled:                      &trueValue,
		},
		{
			description:                      "GDPR Web channel disabled, general GDPR enabled",
			giveChannelType:                  ChannelWeb,
			giveGDPREnabled:                  &trueValue,
			giveWebGDPREnabled:               &falseValue,
			giveWebGDPREnabledForIntegration: nil,
			wantEnabled:                      &falseValue,
		},
		{
			description:                      "GDPR Web channel unspecified, general GDPR disabled",
			giveChannelType:                  ChannelWeb,
			giveGDPREnabled:                  &falseValue,
			giveWebGDPREnabled:               nil,
			giveWebGDPREnabledForIntegration: nil,
			wantEnabled:                      &falseValue,
		},
		{
			description:                      "GDPR Web channel unspecified, general GDPR enabled",
			giveChannelType:                  ChannelWeb,
			giveGDPREnabled:                  &trueValue,
			giveWebGDPREnabled:               nil,
			giveWebGDPREnabledForIntegration: nil,
			wantEnabled:                      &trueValue,
		},
		{
			description:                      "GDPR Web channel unspecified, general GDPR unspecified",
			giveChannelType:                  ChannelWeb,
			giveGDPREnabled:                  nil,
			giveWebGDPREnabled:               nil,
			giveWebGDPREnabledForIntegration: nil,
			wantEnabled:                      nil,
		},
		{
			description:                      "Inegration Enabled is set, and channel enabled isn't",
			giveChannelType:                  ChannelWeb,
			giveGDPREnabled:                  &falseValue,
			giveWebGDPREnabled:               nil,
			giveWebGDPREnabledForIntegration: &trueValue,
			wantEnabled:                      &trueValue,
		},
		{
			description:                      "Inegration Enabled is set, and channel enabled is set, channel should have precedence",
			giveChannelType:                  ChannelWeb,
			giveGDPREnabled:                  &falseValue,
			giveWebGDPREnabled:               &trueValue,
			giveWebGDPREnabledForIntegration: &falseValue,
			wantEnabled:                      &trueValue,
		},
	}

	for _, tt := range tests {
		account := Account{
			GDPR: AccountGDPR{
				Enabled: tt.giveGDPREnabled,
				ChannelEnabled: AccountChannel{
					Web: tt.giveWebGDPREnabled,
				},
				IntegrationEnabled: AccountChannel{
					Web: tt.giveWebGDPREnabledForIntegration,
				},
			},
		}

		enabled := account.GDPR.EnabledForChannelType(tt.giveChannelType)

		if tt.wantEnabled == nil {
			assert.Nil(t, enabled, tt.description)
		} else {
			assert.NotNil(t, enabled, tt.description)
			assert.Equal(t, *tt.wantEnabled, *enabled, tt.description)
		}
	}
}

func TestAccountCCPAEnabledForChannelType(t *testing.T) {
	trueValue, falseValue := true, false

	tests := []struct {
		description                      string
		giveChannelType                  ChannelType
		giveCCPAEnabled                  *bool
		giveWebCCPAEnabled               *bool
		giveWebCCPAEnabledForIntegration *bool
		wantEnabled                      *bool
	}{
		{
			description:                      "CCPA Web channel enabled, general CCPA disabled",
			giveChannelType:                  ChannelWeb,
			giveCCPAEnabled:                  &falseValue,
			giveWebCCPAEnabled:               &trueValue,
			giveWebCCPAEnabledForIntegration: nil,
			wantEnabled:                      &trueValue,
		},
		{
			description:                      "CCPA Web channel disabled, general CCPA enabled",
			giveChannelType:                  ChannelWeb,
			giveCCPAEnabled:                  &trueValue,
			giveWebCCPAEnabled:               &falseValue,
			giveWebCCPAEnabledForIntegration: nil,
			wantEnabled:                      &falseValue,
		},
		{
			description:                      "CCPA Web channel unspecified, general CCPA disabled",
			giveChannelType:                  ChannelWeb,
			giveCCPAEnabled:                  &falseValue,
			giveWebCCPAEnabled:               nil,
			giveWebCCPAEnabledForIntegration: nil,
			wantEnabled:                      &falseValue,
		},
		{
			description:                      "CCPA Web channel unspecified, general CCPA enabled",
			giveChannelType:                  ChannelWeb,
			giveCCPAEnabled:                  &trueValue,
			giveWebCCPAEnabled:               nil,
			giveWebCCPAEnabledForIntegration: nil,
			wantEnabled:                      &trueValue,
		},
		{
			description:                      "CCPA Web channel unspecified, general CCPA unspecified",
			giveChannelType:                  ChannelWeb,
			giveCCPAEnabled:                  nil,
			giveWebCCPAEnabled:               nil,
			giveWebCCPAEnabledForIntegration: nil,
			wantEnabled:                      nil,
		},
		{
			description:                      "Inegration Enabled is set, and channel enabled isn't",
			giveChannelType:                  ChannelWeb,
			giveCCPAEnabled:                  &falseValue,
			giveWebCCPAEnabled:               nil,
			giveWebCCPAEnabledForIntegration: &trueValue,
			wantEnabled:                      &trueValue,
		},
		{
			description:                      "Inegration Enabled is set, and channel enabled is set, channel should have precedence",
			giveChannelType:                  ChannelWeb,
			giveCCPAEnabled:                  &falseValue,
			giveWebCCPAEnabled:               &trueValue,
			giveWebCCPAEnabledForIntegration: &falseValue,
			wantEnabled:                      &trueValue,
		},
	}

	for _, tt := range tests {
		account := Account{
			CCPA: AccountCCPA{
				Enabled: tt.giveCCPAEnabled,
				ChannelEnabled: AccountChannel{
					Web: tt.giveWebCCPAEnabled,
				},
				IntegrationEnabled: AccountChannel{
					Web: tt.giveWebCCPAEnabledForIntegration,
				},
			},
		}

		enabled := account.CCPA.EnabledForChannelType(tt.giveChannelType)

		if tt.wantEnabled == nil {
			assert.Nil(t, enabled, tt.description)
		} else {
			assert.NotNil(t, enabled, tt.description)
			assert.Equal(t, *tt.wantEnabled, *enabled, tt.description)
		}
	}
}

func TestAccountChannelGetByChannelType(t *testing.T) {
	trueValue, falseValue := true, false

	tests := []struct {
		description      string
		giveAMPEnabled   *bool
		giveAppEnabled   *bool
		giveVideoEnabled *bool
		giveWebEnabled   *bool
		giveChannelType  ChannelType
		wantEnabled      *bool
	}{
		{
			description:     "AMP channel setting unspecified, returns nil",
			giveChannelType: ChannelAMP,
			wantEnabled:     nil,
		},
		{
			description:     "AMP channel disabled, returns false",
			giveAMPEnabled:  &falseValue,
			giveChannelType: ChannelAMP,
			wantEnabled:     &falseValue,
		},
		{
			description:     "AMP channel enabled, returns true",
			giveAMPEnabled:  &trueValue,
			giveChannelType: ChannelAMP,
			wantEnabled:     &trueValue,
		},
		{
			description:     "App channel setting unspecified, returns nil",
			giveChannelType: ChannelApp,
			wantEnabled:     nil,
		},
		{
			description:     "App channel disabled, returns false",
			giveAppEnabled:  &falseValue,
			giveChannelType: ChannelApp,
			wantEnabled:     &falseValue,
		},
		{
			description:     "App channel enabled, returns true",
			giveAppEnabled:  &trueValue,
			giveChannelType: ChannelApp,
			wantEnabled:     &trueValue,
		},
		{
			description:     "Video channel setting unspecified, returns nil",
			giveChannelType: ChannelVideo,
			wantEnabled:     nil,
		},
		{
			description:      "Video channel disabled, returns false",
			giveVideoEnabled: &falseValue,
			giveChannelType:  ChannelVideo,
			wantEnabled:      &falseValue,
		},
		{
			description:      "Video channel enabled, returns true",
			giveVideoEnabled: &trueValue,
			giveChannelType:  ChannelVideo,
			wantEnabled:      &trueValue,
		},
		{
			description:     "Web channel setting unspecified, returns nil",
			giveChannelType: ChannelWeb,
			wantEnabled:     nil,
		},
		{
			description:     "Web channel disabled, returns false",
			giveWebEnabled:  &falseValue,
			giveChannelType: ChannelWeb,
			wantEnabled:     &falseValue,
		},
		{
			description:     "Web channel enabled, returns true",
			giveWebEnabled:  &trueValue,
			giveChannelType: ChannelWeb,
			wantEnabled:     &trueValue,
		},
	}

	for _, tt := range tests {
		accountChannel := AccountChannel{
			AMP:   tt.giveAMPEnabled,
			App:   tt.giveAppEnabled,
			Video: tt.giveVideoEnabled,
			Web:   tt.giveWebEnabled,
		}

		result := accountChannel.GetByChannelType(tt.giveChannelType)
		if tt.wantEnabled == nil {
			assert.Nil(t, result, tt.description)
		} else {
			assert.NotNil(t, result, tt.description)
			assert.Equal(t, *tt.wantEnabled, *result, tt.description)
		}
	}
}

func TestPurposeEnforced(t *testing.T) {
	True := true
	False := false

	tests := []struct {
		description          string
		givePurposeConfigNil bool
		givePurpose1Enforced *bool
		givePurpose2Enforced *bool
		givePurpose          consentconstants.Purpose
		wantEnforced         bool
		wantEnforcedSet      bool
	}{
		{
			description:          "Purpose config is nil",
			givePurposeConfigNil: true,
			givePurpose:          1,
			wantEnforced:         true,
			wantEnforcedSet:      false,
		},
		{
			description:          "Purpose 1 Enforced not set",
			givePurpose1Enforced: nil,
			givePurpose:          1,
			wantEnforced:         true,
			wantEnforcedSet:      false,
		},
		{
			description:          "Purpose 1 Enforced set to full enforcement",
			givePurpose1Enforced: &True,
			givePurpose:          1,
			wantEnforced:         true,
			wantEnforcedSet:      true,
		},
		{
			description:          "Purpose 1 Enforced set to no enforcement",
			givePurpose1Enforced: &False,
			givePurpose:          1,
			wantEnforced:         false,
			wantEnforcedSet:      true,
		},
		{
			description:          "Purpose 2 Enforced set to full enforcement",
			givePurpose2Enforced: &True,
			givePurpose:          2,
			wantEnforced:         true,
			wantEnforcedSet:      true,
		},
	}

	for _, tt := range tests {
		accountGDPR := AccountGDPR{}

		if !tt.givePurposeConfigNil {
			accountGDPR.PurposeConfigs = map[consentconstants.Purpose]*AccountGDPRPurpose{
				1: {
					EnforcePurpose: tt.givePurpose1Enforced,
				},
				2: {
					EnforcePurpose: tt.givePurpose2Enforced,
				},
			}
		}

		value, present := accountGDPR.PurposeEnforced(tt.givePurpose)

		assert.Equal(t, tt.wantEnforced, value, tt.description)
		assert.Equal(t, tt.wantEnforcedSet, present, tt.description)
	}
}

func TestPurposeEnforcementAlgo(t *testing.T) {
	tests := []struct {
		description          string
		givePurposeConfigNil bool
		givePurpose1Algo     TCF2EnforcementAlgo
		givePurpose2Algo     TCF2EnforcementAlgo
		givePurpose          consentconstants.Purpose
		wantAlgo             TCF2EnforcementAlgo
		wantAlgoSet          bool
	}{
		{
			description:          "Purpose config is nil",
			givePurposeConfigNil: true,
			givePurpose:          1,
			wantAlgo:             TCF2UndefinedEnforcement,
			wantAlgoSet:          false,
		},
		{
			description:      "Purpose 1 enforcement algo is undefined",
			givePurpose1Algo: TCF2UndefinedEnforcement,
			givePurpose:      1,
			wantAlgo:         TCF2UndefinedEnforcement,
			wantAlgoSet:      false,
		},
		{
			description:      "Purpose 1 enforcement algo set to basic",
			givePurpose1Algo: TCF2BasicEnforcement,
			givePurpose:      1,
			wantAlgo:         TCF2BasicEnforcement,
			wantAlgoSet:      true,
		},
		{
			description:      "Purpose 1 enforcement algo set to full",
			givePurpose1Algo: TCF2FullEnforcement,
			givePurpose:      1,
			wantAlgo:         TCF2FullEnforcement,
			wantAlgoSet:      true,
		},
		{
			description:      "Purpose 2 Enforcement algo set to basic",
			givePurpose2Algo: TCF2BasicEnforcement,
			givePurpose:      2,
			wantAlgo:         TCF2BasicEnforcement,
			wantAlgoSet:      true,
		},
	}

	for _, tt := range tests {
		accountGDPR := AccountGDPR{}

		if !tt.givePurposeConfigNil {
			accountGDPR.PurposeConfigs = map[consentconstants.Purpose]*AccountGDPRPurpose{
				1: {
					EnforceAlgoID: tt.givePurpose1Algo,
				},
				2: {
					EnforceAlgoID: tt.givePurpose2Algo,
				},
			}
		}

		value, present := accountGDPR.PurposeEnforcementAlgo(tt.givePurpose)

		assert.Equal(t, tt.wantAlgo, value, tt.description)
		assert.Equal(t, tt.wantAlgoSet, present, tt.description)
	}
}

func TestPurposeEnforcingVendors(t *testing.T) {
	tests := []struct {
		description           string
		givePurposeConfigNil  bool
		givePurpose1Enforcing *bool
		givePurpose2Enforcing *bool
		givePurpose           consentconstants.Purpose
		wantEnforcing         bool
		wantEnforcingSet      bool
	}{
		{
			description:          "Purpose config is nil",
			givePurposeConfigNil: true,
			givePurpose:          1,
			wantEnforcing:        true,
			wantEnforcingSet:     false,
		},
		{
			description:           "Purpose 1 Enforcing not set",
			givePurpose1Enforcing: nil,
			givePurpose:           1,
			wantEnforcing:         true,
			wantEnforcingSet:      false,
		},
		{
			description:           "Purpose 1 Enforcing set to true",
			givePurpose1Enforcing: &[]bool{true}[0],
			givePurpose:           1,
			wantEnforcing:         true,
			wantEnforcingSet:      true,
		},
		{
			description:           "Purpose 1 Enforcing set to false",
			givePurpose1Enforcing: &[]bool{false}[0],
			givePurpose:           1,
			wantEnforcing:         false,
			wantEnforcingSet:      true,
		},
		{
			description:           "Purpose 2 Enforcing set to true",
			givePurpose2Enforcing: &[]bool{true}[0],
			givePurpose:           2,
			wantEnforcing:         true,
			wantEnforcingSet:      true,
		},
	}

	for _, tt := range tests {
		accountGDPR := AccountGDPR{}

		if !tt.givePurposeConfigNil {
			accountGDPR.PurposeConfigs = map[consentconstants.Purpose]*AccountGDPRPurpose{
				1: {
					EnforceVendors: tt.givePurpose1Enforcing,
				},
				2: {
					EnforceVendors: tt.givePurpose2Enforcing,
				},
			}
		}

		value, present := accountGDPR.PurposeEnforcingVendors(tt.givePurpose)

		assert.Equal(t, tt.wantEnforcing, value, tt.description)
		assert.Equal(t, tt.wantEnforcingSet, present, tt.description)
	}
}

func TestPurposeVendorExceptions(t *testing.T) {
	tests := []struct {
		description              string
		givePurposeConfigNil     bool
		givePurpose1ExceptionMap map[openrtb_ext.BidderName]struct{}
		givePurpose2ExceptionMap map[openrtb_ext.BidderName]struct{}
		givePurpose              consentconstants.Purpose
		wantExceptionMap         map[openrtb_ext.BidderName]struct{}
		wantExceptionMapSet      bool
	}{
		{
			description:          "Purpose config is nil",
			givePurposeConfigNil: true,
			givePurpose:          1,
			// wantExceptionMap:     map[openrtb_ext.BidderName]struct{}{},
			wantExceptionMap:    nil,
			wantExceptionMapSet: false,
		},
		{
			description: "Nil - exception map not defined for purpose",
			givePurpose: 1,
			// wantExceptionMap:    map[openrtb_ext.BidderName]struct{}{},
			wantExceptionMap:    nil,
			wantExceptionMapSet: false,
		},
		{
			description:              "Empty - exception map empty for purpose",
			givePurpose:              1,
			givePurpose1ExceptionMap: map[openrtb_ext.BidderName]struct{}{},
			wantExceptionMap:         map[openrtb_ext.BidderName]struct{}{},
			wantExceptionMapSet:      true,
		},
		{
			description:              "Nonempty - exception map with multiple entries for purpose",
			givePurpose:              1,
			givePurpose1ExceptionMap: map[openrtb_ext.BidderName]struct{}{"rubicon": {}, "appnexus": {}, "index": {}},
			wantExceptionMap:         map[openrtb_ext.BidderName]struct{}{"rubicon": {}, "appnexus": {}, "index": {}},
			wantExceptionMapSet:      true,
		},
		{
			description:              "Nonempty - exception map with multiple entries for different purpose",
			givePurpose:              2,
			givePurpose1ExceptionMap: map[openrtb_ext.BidderName]struct{}{"rubicon": {}, "appnexus": {}, "index": {}},
			givePurpose2ExceptionMap: map[openrtb_ext.BidderName]struct{}{"rubicon": {}, "appnexus": {}, "openx": {}},
			wantExceptionMap:         map[openrtb_ext.BidderName]struct{}{"rubicon": {}, "appnexus": {}, "openx": {}},
			wantExceptionMapSet:      true,
		},
	}

	for _, tt := range tests {
		accountGDPR := AccountGDPR{}

		if !tt.givePurposeConfigNil {
			accountGDPR.PurposeConfigs = map[consentconstants.Purpose]*AccountGDPRPurpose{
				1: {
					VendorExceptionMap: tt.givePurpose1ExceptionMap,
				},
				2: {
					VendorExceptionMap: tt.givePurpose2ExceptionMap,
				},
			}
		}

		value, present := accountGDPR.PurposeVendorExceptions(tt.givePurpose)

		assert.Equal(t, tt.wantExceptionMap, value, tt.description)
		assert.Equal(t, tt.wantExceptionMapSet, present, tt.description)
	}
}

func TestFeatureOneEnforced(t *testing.T) {
	tests := []struct {
		description     string
		giveEnforce     *bool
		wantEnforcedSet bool
		wantEnforced    bool
	}{
		{
			description:     "Special feature 1 enforce not set",
			giveEnforce:     nil,
			wantEnforcedSet: false,
			wantEnforced:    true,
		},
		{
			description:     "Special feature 1 enforce set to true",
			giveEnforce:     &[]bool{true}[0],
			wantEnforcedSet: true,
			wantEnforced:    true,
		},
		{
			description:     "Special feature 1 enforce set to false",
			giveEnforce:     &[]bool{false}[0],
			wantEnforcedSet: true,
			wantEnforced:    false,
		},
	}

	for _, tt := range tests {
		accountGDPR := AccountGDPR{
			SpecialFeature1: AccountGDPRSpecialFeature{
				Enforce: tt.giveEnforce,
			},
		}

		value, present := accountGDPR.FeatureOneEnforced()

		assert.Equal(t, tt.wantEnforced, value, tt.description)
		assert.Equal(t, tt.wantEnforcedSet, present, tt.description)
	}
}

func TestFeatureOneVendorException(t *testing.T) {
	tests := []struct {
		description            string
		giveExceptionMap       map[openrtb_ext.BidderName]struct{}
		giveBidder             openrtb_ext.BidderName
		wantVendorExceptionSet bool
		wantIsVendorException  bool
	}{
		{
			description:            "Nil - exception map not defined",
			giveBidder:             "appnexus",
			wantVendorExceptionSet: false,
			wantIsVendorException:  false,
		},
		{
			description:            "Empty - exception map empty",
			giveExceptionMap:       map[openrtb_ext.BidderName]struct{}{},
			giveBidder:             "appnexus",
			wantVendorExceptionSet: true,
			wantIsVendorException:  false,
		},
		{
			description:            "One - bidder found in exception map containing one entry",
			giveExceptionMap:       map[openrtb_ext.BidderName]struct{}{"appnexus": {}},
			giveBidder:             "appnexus",
			wantVendorExceptionSet: true,
			wantIsVendorException:  true,
		},
		{
			description:            "Many - bidder found in exception map containing multiple entries",
			giveExceptionMap:       map[openrtb_ext.BidderName]struct{}{"rubicon": {}, "appnexus": {}, "index": {}},
			giveBidder:             "appnexus",
			wantVendorExceptionSet: true,
			wantIsVendorException:  true,
		},
		{
			description:            "Many - bidder not found in exception map containing multiple entries",
			giveExceptionMap:       map[openrtb_ext.BidderName]struct{}{"rubicon": {}, "appnexus": {}, "index": {}},
			giveBidder:             "openx",
			wantVendorExceptionSet: true,
			wantIsVendorException:  false,
		},
	}

	for _, tt := range tests {
		accountGDPR := AccountGDPR{
			SpecialFeature1: AccountGDPRSpecialFeature{
				VendorExceptionMap: tt.giveExceptionMap,
			},
		}

		value, present := accountGDPR.FeatureOneVendorException(tt.giveBidder)

		assert.Equal(t, tt.wantIsVendorException, value, tt.description)
		assert.Equal(t, tt.wantVendorExceptionSet, present, tt.description)
	}
}

func TestPurposeOneTreatmentEnabled(t *testing.T) {
	tests := []struct {
		description    string
		giveEnabled    *bool
		wantEnabledSet bool
		wantEnabled    bool
	}{
		{
			description:    "Purpose one treatment enabled not set",
			giveEnabled:    nil,
			wantEnabledSet: false,
			wantEnabled:    true,
		},
		{
			description:    "Purpose one treatment enabled set to true",
			giveEnabled:    &[]bool{true}[0],
			wantEnabledSet: true,
			wantEnabled:    true,
		},
		{
			description:    "Purpose one treatment enabled set to false",
			giveEnabled:    &[]bool{false}[0],
			wantEnabledSet: true,
			wantEnabled:    false,
		},
	}

	for _, tt := range tests {
		accountGDPR := AccountGDPR{
			PurposeOneTreatment: AccountGDPRPurposeOneTreatment{
				Enabled: tt.giveEnabled,
			},
		}

		value, present := accountGDPR.PurposeOneTreatmentEnabled()

		assert.Equal(t, tt.wantEnabled, value, tt.description)
		assert.Equal(t, tt.wantEnabledSet, present, tt.description)
	}
}

func TestPurposeOneTreatmentAccessAllowed(t *testing.T) {
	tests := []struct {
		description    string
		giveAllowed    *bool
		wantAllowedSet bool
		wantAllowed    bool
	}{
		{
			description:    "Purpose one treatment access allowed not set",
			giveAllowed:    nil,
			wantAllowedSet: false,
			wantAllowed:    true,
		},
		{
			description:    "Purpose one treatment access allowed set to true",
			giveAllowed:    &[]bool{true}[0],
			wantAllowedSet: true,
			wantAllowed:    true,
		},
		{
			description:    "Purpose one treatment access allowed set to false",
			giveAllowed:    &[]bool{false}[0],
			wantAllowedSet: true,
			wantAllowed:    false,
		},
	}

	for _, tt := range tests {
		accountGDPR := AccountGDPR{
			PurposeOneTreatment: AccountGDPRPurposeOneTreatment{
				AccessAllowed: tt.giveAllowed,
			},
		}

		value, present := accountGDPR.PurposeOneTreatmentAccessAllowed()

		assert.Equal(t, tt.wantAllowed, value, tt.description)
		assert.Equal(t, tt.wantAllowedSet, present, tt.description)
	}
}

func TestModulesGetConfig(t *testing.T) {
	modules := AccountModules{
		"acme": {
			"foo":     json.RawMessage(`{"foo": "bar"}`),
			"foo.bar": json.RawMessage(`{"foo": "bar"}`),
		},
		"acme.foo": {
			"baz": json.RawMessage(`{"foo": "bar"}`),
		},
	}

	testCases := []struct {
		description    string
		givenId        string
		givenModules   AccountModules
		expectedConfig json.RawMessage
		expectedError  error
	}{
		{
			description:    "Returns module config if found by ID",
			givenId:        "acme.foo",
			givenModules:   modules,
			expectedConfig: json.RawMessage(`{"foo": "bar"}`),
			expectedError:  nil,
		},
		{
			description:    "Returns module config if found by ID",
			givenId:        "acme.foo.bar",
			givenModules:   modules,
			expectedConfig: json.RawMessage(`{"foo": "bar"}`),
			expectedError:  nil,
		},
		{
			description:    "Returns nil config if wrong ID provided",
			givenId:        "invalid_id",
			givenModules:   modules,
			expectedConfig: nil,
			expectedError:  errors.New("ID must consist of vendor and module names separated by dot, got: invalid_id"),
		},
		{
			description:    "Returns nil config if no matching module exists",
			givenId:        "acme.bar",
			givenModules:   modules,
			expectedConfig: nil,
			expectedError:  nil,
		},
		{
			description:    "Returns nil config if no matching module exists",
			givenId:        "acme.foo.baz",
			givenModules:   modules,
			expectedConfig: nil,
			expectedError:  nil,
		},
		{
			description:    "Returns nil config if no module configs defined in account",
			givenId:        "acme.foo",
			givenModules:   nil,
			expectedConfig: nil,
			expectedError:  nil,
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			gotConfig, err := test.givenModules.ModuleConfig(test.givenId)
			assert.Equal(t, test.expectedError, err)
			assert.Equal(t, test.expectedConfig, gotConfig)
		})
	}
}

func TestAccountChannelIsSet(t *testing.T) {
	trueBool := true
	falseBool := false

	testCases := []struct {
		name                string
		givenAccountChannel *AccountChannel
		expected            bool
	}{
		{
			name:                "AccountChannelSetAllFields",
			givenAccountChannel: &AccountChannel{AMP: &trueBool, App: &falseBool, Video: &falseBool, Web: &falseBool},
			expected:            true,
		},
		{
			name:                "AccountChannelNotSet",
			givenAccountChannel: &AccountChannel{},
			expected:            false,
		},
		{
			name:                "AccountChannelSetAmpOnly",
			givenAccountChannel: &AccountChannel{AMP: &trueBool},
			expected:            true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.givenAccountChannel.IsSet())
		})
	}
}

func TestAccountPriceFloorsValidate(t *testing.T) {

	tests := []struct {
		description string
		pf          *AccountPriceFloors
		want        []error
	}{
		{
			description: "valid configuration",
			pf: &AccountPriceFloors{
				EnforceFloorsRate: 100,
				MaxRule:           200,
				MaxSchemaDims:     10,
			},
		},
		{
			description: "Invalid configuration: EnforceFloorRate:110",
			pf: &AccountPriceFloors{
				EnforceFloorsRate: 110,
			},
			want: []error{errors.New("account_defaults.price_floors.enforce_floors_rate should be between 0 and 100")},
		},
		{
			description: "Invalid configuration: EnforceFloorRate:-10",
			pf: &AccountPriceFloors{
				EnforceFloorsRate: -10,
			},
			want: []error{errors.New("account_defaults.price_floors.enforce_floors_rate should be between 0 and 100")},
		},
		{
			description: "Invalid configuration: MaxRule:-20",
			pf: &AccountPriceFloors{
				MaxRule: -20,
			},
			want: []error{errors.New("account_defaults.price_floors.max_rules should be between 0 and 2147483647")},
		},
		{
			description: "Invalid configuration: MaxSchemaDims:100",
			pf: &AccountPriceFloors{
				MaxSchemaDims: 100,
			},
			want: []error{errors.New("account_defaults.price_floors.max_schema_dims should be between 0 and 20")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			var errs []error
			got := tt.pf.validate(errs)
			assert.ElementsMatch(t, got, tt.want)
		})
	}
}
