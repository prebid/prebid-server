package config

import (
	"testing"

	"github.com/prebid/go-gdpr/consentconstants"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestAccountGDPREnabledForIntegrationType(t *testing.T) {
	trueValue, falseValue := true, false

	tests := []struct {
		description         string
		giveIntegrationType IntegrationType
		giveGDPREnabled     *bool
		giveWebGDPREnabled  *bool
		wantEnabled         *bool
	}{
		{
			description:         "GDPR Web integration enabled, general GDPR disabled",
			giveIntegrationType: IntegrationTypeWeb,
			giveGDPREnabled:     &falseValue,
			giveWebGDPREnabled:  &trueValue,
			wantEnabled:         &trueValue,
		},
		{
			description:         "GDPR Web integration disabled, general GDPR enabled",
			giveIntegrationType: IntegrationTypeWeb,
			giveGDPREnabled:     &trueValue,
			giveWebGDPREnabled:  &falseValue,
			wantEnabled:         &falseValue,
		},
		{
			description:         "GDPR Web integration unspecified, general GDPR disabled",
			giveIntegrationType: IntegrationTypeWeb,
			giveGDPREnabled:     &falseValue,
			giveWebGDPREnabled:  nil,
			wantEnabled:         &falseValue,
		},
		{
			description:         "GDPR Web integration unspecified, general GDPR enabled",
			giveIntegrationType: IntegrationTypeWeb,
			giveGDPREnabled:     &trueValue,
			giveWebGDPREnabled:  nil,
			wantEnabled:         &trueValue,
		},
		{
			description:         "GDPR Web integration unspecified, general GDPR unspecified",
			giveIntegrationType: IntegrationTypeWeb,
			giveGDPREnabled:     nil,
			giveWebGDPREnabled:  nil,
			wantEnabled:         nil,
		},
	}

	for _, tt := range tests {
		account := Account{
			GDPR: AccountGDPR{
				Enabled: tt.giveGDPREnabled,
				IntegrationEnabled: AccountIntegration{
					Web: tt.giveWebGDPREnabled,
				},
			},
		}

		enabled := account.GDPR.EnabledForIntegrationType(tt.giveIntegrationType)

		if tt.wantEnabled == nil {
			assert.Nil(t, enabled, tt.description)
		} else {
			assert.NotNil(t, enabled, tt.description)
			assert.Equal(t, *tt.wantEnabled, *enabled, tt.description)
		}
	}
}

func TestAccountCCPAEnabledForIntegrationType(t *testing.T) {
	trueValue, falseValue := true, false

	tests := []struct {
		description         string
		giveIntegrationType IntegrationType
		giveCCPAEnabled     *bool
		giveWebCCPAEnabled  *bool
		wantEnabled         *bool
	}{
		{
			description:         "CCPA Web integration enabled, general CCPA disabled",
			giveIntegrationType: IntegrationTypeWeb,
			giveCCPAEnabled:     &falseValue,
			giveWebCCPAEnabled:  &trueValue,
			wantEnabled:         &trueValue,
		},
		{
			description:         "CCPA Web integration disabled, general CCPA enabled",
			giveIntegrationType: IntegrationTypeWeb,
			giveCCPAEnabled:     &trueValue,
			giveWebCCPAEnabled:  &falseValue,
			wantEnabled:         &falseValue,
		},
		{
			description:         "CCPA Web integration unspecified, general CCPA disabled",
			giveIntegrationType: IntegrationTypeWeb,
			giveCCPAEnabled:     &falseValue,
			giveWebCCPAEnabled:  nil,
			wantEnabled:         &falseValue,
		},
		{
			description:         "CCPA Web integration unspecified, general CCPA enabled",
			giveIntegrationType: IntegrationTypeWeb,
			giveCCPAEnabled:     &trueValue,
			giveWebCCPAEnabled:  nil,
			wantEnabled:         &trueValue,
		},
		{
			description:         "CCPA Web integration unspecified, general CCPA unspecified",
			giveIntegrationType: IntegrationTypeWeb,
			giveCCPAEnabled:     nil,
			giveWebCCPAEnabled:  nil,
			wantEnabled:         nil,
		},
	}

	for _, tt := range tests {
		account := Account{
			CCPA: AccountCCPA{
				Enabled: tt.giveCCPAEnabled,
				IntegrationEnabled: AccountIntegration{
					Web: tt.giveWebCCPAEnabled,
				},
			},
		}

		enabled := account.CCPA.EnabledForIntegrationType(tt.giveIntegrationType)

		if tt.wantEnabled == nil {
			assert.Nil(t, enabled, tt.description)
		} else {
			assert.NotNil(t, enabled, tt.description)
			assert.Equal(t, *tt.wantEnabled, *enabled, tt.description)
		}
	}
}

func TestAccountIntegrationGetByIntegrationType(t *testing.T) {
	trueValue, falseValue := true, false

	tests := []struct {
		description         string
		giveAMPEnabled      *bool
		giveAppEnabled      *bool
		giveVideoEnabled    *bool
		giveWebEnabled      *bool
		giveIntegrationType IntegrationType
		wantEnabled         *bool
	}{
		{
			description:         "AMP integration setting unspecified, returns nil",
			giveIntegrationType: IntegrationTypeAMP,
			wantEnabled:         nil,
		},
		{
			description:         "AMP integration disabled, returns false",
			giveAMPEnabled:      &falseValue,
			giveIntegrationType: IntegrationTypeAMP,
			wantEnabled:         &falseValue,
		},
		{
			description:         "AMP integration enabled, returns true",
			giveAMPEnabled:      &trueValue,
			giveIntegrationType: IntegrationTypeAMP,
			wantEnabled:         &trueValue,
		},
		{
			description:         "App integration setting unspecified, returns nil",
			giveIntegrationType: IntegrationTypeApp,
			wantEnabled:         nil,
		},
		{
			description:         "App integration disabled, returns false",
			giveAppEnabled:      &falseValue,
			giveIntegrationType: IntegrationTypeApp,
			wantEnabled:         &falseValue,
		},
		{
			description:         "App integration enabled, returns true",
			giveAppEnabled:      &trueValue,
			giveIntegrationType: IntegrationTypeApp,
			wantEnabled:         &trueValue,
		},
		{
			description:         "Video integration setting unspecified, returns nil",
			giveIntegrationType: IntegrationTypeVideo,
			wantEnabled:         nil,
		},
		{
			description:         "Video integration disabled, returns false",
			giveVideoEnabled:    &falseValue,
			giveIntegrationType: IntegrationTypeVideo,
			wantEnabled:         &falseValue,
		},
		{
			description:         "Video integration enabled, returns true",
			giveVideoEnabled:    &trueValue,
			giveIntegrationType: IntegrationTypeVideo,
			wantEnabled:         &trueValue,
		},
		{
			description:         "Web integration setting unspecified, returns nil",
			giveIntegrationType: IntegrationTypeWeb,
			wantEnabled:         nil,
		},
		{
			description:         "Web integration disabled, returns false",
			giveWebEnabled:      &falseValue,
			giveIntegrationType: IntegrationTypeWeb,
			wantEnabled:         &falseValue,
		},
		{
			description:         "Web integration enabled, returns true",
			giveWebEnabled:      &trueValue,
			giveIntegrationType: IntegrationTypeWeb,
			wantEnabled:         &trueValue,
		},
	}

	for _, tt := range tests {
		accountIntegration := AccountIntegration{
			AMP:   tt.giveAMPEnabled,
			App:   tt.giveAppEnabled,
			Video: tt.giveVideoEnabled,
			Web:   tt.giveWebEnabled,
		}

		result := accountIntegration.GetByIntegrationType(tt.giveIntegrationType)
		if tt.wantEnabled == nil {
			assert.Nil(t, result, tt.description)
		} else {
			assert.NotNil(t, result, tt.description)
			assert.Equal(t, *tt.wantEnabled, *result, tt.description)
		}
	}
}

func TestPurposeEnforced(t *testing.T) {
	tests := []struct {
		description          string
		givePurposeConfigNil bool
		givePurpose1Enforced string
		givePurpose2Enforced string
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
			givePurpose1Enforced: "",
			givePurpose:          1,
			wantEnforced:         true,
			wantEnforcedSet:      false,
		},
		{
			description:          "Purpose 1 Enforced set to full enforcement",
			givePurpose1Enforced: TCF2FullEnforcement,
			givePurpose:          1,
			wantEnforced:         true,
			wantEnforcedSet:      true,
		},
		{
			description:          "Purpose 1 Enforced set to no enforcement",
			givePurpose1Enforced: TCF2NoEnforcement,
			givePurpose:          1,
			wantEnforced:         false,
			wantEnforcedSet:      true,
		},
		{
			description:          "Purpose 2 Enforced set to full enforcement",
			givePurpose2Enforced: TCF2FullEnforcement,
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

func TestPurposeVendorException(t *testing.T) {
	tests := []struct {
		description              string
		givePurposeConfigNil     bool
		givePurpose1ExceptionMap map[openrtb_ext.BidderName]struct{}
		givePurpose2ExceptionMap map[openrtb_ext.BidderName]struct{}
		givePurpose              consentconstants.Purpose
		giveBidder               openrtb_ext.BidderName
		wantIsVendorException    bool
		wantVendorExceptionSet   bool
	}{
		{
			description:            "Purpose config is nil",
			givePurposeConfigNil:   true,
			givePurpose:            1,
			giveBidder:             "appnexus",
			wantIsVendorException:  false,
			wantVendorExceptionSet: false,
		},
		{
			description:            "Nil - exception map not defined for purpose",
			givePurpose:            1,
			giveBidder:             "appnexus",
			wantIsVendorException:  false,
			wantVendorExceptionSet: false,
		},
		{
			description:              "Empty - exception map empty for purpose",
			givePurpose:              1,
			givePurpose1ExceptionMap: map[openrtb_ext.BidderName]struct{}{},
			giveBidder:               "appnexus",
			wantIsVendorException:    false,
			wantVendorExceptionSet:   true,
		},
		{
			description:              "One - bidder found in purpose exception map containing one entry",
			givePurpose:              1,
			givePurpose1ExceptionMap: map[openrtb_ext.BidderName]struct{}{"appnexus": {}},
			giveBidder:               "appnexus",
			wantIsVendorException:    true,
			wantVendorExceptionSet:   true,
		},
		{
			description:              "Many - bidder found in purpose exception map containing multiple entries",
			givePurpose:              1,
			givePurpose1ExceptionMap: map[openrtb_ext.BidderName]struct{}{"rubicon": {}, "appnexus": {}, "index": {}},
			giveBidder:               "appnexus",
			wantIsVendorException:    true,
			wantVendorExceptionSet:   true,
		},
		{
			description:              "Many - bidder not found in purpose exception map containing multiple entries",
			givePurpose:              1,
			givePurpose1ExceptionMap: map[openrtb_ext.BidderName]struct{}{"rubicon": {}, "appnexus": {}, "index": {}},
			givePurpose2ExceptionMap: map[openrtb_ext.BidderName]struct{}{"rubicon": {}, "appnexus": {}, "openx": {}},
			giveBidder:               "openx",
			wantIsVendorException:    false,
			wantVendorExceptionSet:   true,
		},
		{
			description:              "Many - bidder found in different purpose exception map containing multiple entries",
			givePurpose:              2,
			givePurpose1ExceptionMap: map[openrtb_ext.BidderName]struct{}{"rubicon": {}, "appnexus": {}, "index": {}},
			givePurpose2ExceptionMap: map[openrtb_ext.BidderName]struct{}{"rubicon": {}, "appnexus": {}, "openx": {}},
			giveBidder:               "openx",
			wantIsVendorException:    true,
			wantVendorExceptionSet:   true,
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

		value, present := accountGDPR.PurposeVendorException(tt.givePurpose, tt.giveBidder)

		assert.Equal(t, tt.wantIsVendorException, value, tt.description)
		assert.Equal(t, tt.wantVendorExceptionSet, present, tt.description)
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

func TestBasicEnforcementVendor(t *testing.T) {
	tests := []struct {
		description        string
		giveBasicVendorMap map[string]struct{}
		giveBidder         openrtb_ext.BidderName
		wantBasicVendorSet bool
		wantIsBasicVendor  bool
	}{
		{
			description:        "Nil - basic vendor map not defined",
			giveBidder:         "appnexus",
			wantBasicVendorSet: false,
			wantIsBasicVendor:  false,
		},
		{
			description:        "Empty - basic vendor map empty",
			giveBasicVendorMap: map[string]struct{}{},
			giveBidder:         "appnexus",
			wantBasicVendorSet: true,
			wantIsBasicVendor:  false,
		},
		{
			description:        "One - bidder found in basic vendor map containing one entry",
			giveBasicVendorMap: map[string]struct{}{"appnexus": {}},
			giveBidder:         "appnexus",
			wantBasicVendorSet: true,
			wantIsBasicVendor:  true,
		},
		{
			description:        "Many - bidder found in basic vendor map containing multiple entries",
			giveBasicVendorMap: map[string]struct{}{"rubicon": {}, "appnexus": {}, "index": {}},
			giveBidder:         "appnexus",
			wantBasicVendorSet: true,
			wantIsBasicVendor:  true,
		},
		{
			description:        "Many - bidder not found in basic vendor map containing multiple entries",
			giveBasicVendorMap: map[string]struct{}{"rubicon": {}, "appnexus": {}, "index": {}},
			giveBidder:         "openx",
			wantBasicVendorSet: true,
			wantIsBasicVendor:  false,
		},
	}

	for _, tt := range tests {
		accountGDPR := AccountGDPR{
			BasicEnforcementVendorsMap: tt.giveBasicVendorMap,
		}

		value, present := accountGDPR.BasicEnforcementVendor(tt.giveBidder)

		assert.Equal(t, tt.wantIsBasicVendor, value, tt.description)
		assert.Equal(t, tt.wantBasicVendorSet, present, tt.description)
	}
}
