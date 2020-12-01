package config

import (
	"testing"

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
