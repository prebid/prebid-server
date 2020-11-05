package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccountGDPREnabledForIntegrationType(t *testing.T) {
	trueValue, falseValue := true, false

	tests := []struct {
		description          string
		giveIntegrationType  IntegrationType
		giveGDPREnabled      *bool
		giveAMPGDPREnabled   *bool
		giveAppGDPREnabled   *bool
		giveVideoGDPREnabled *bool
		giveWebGDPREnabled   *bool
		wantEnabled          *bool
	}{
		{
			description:         "GDPR AMP integration enabled, general GDPR disabled",
			giveIntegrationType: IntegrationTypeAMP,
			giveGDPREnabled:     &falseValue,
			giveAMPGDPREnabled:  &trueValue,
			wantEnabled:         &trueValue,
		},
		{
			description:         "GDPR App integration enabled, general GDPR disabled",
			giveIntegrationType: IntegrationTypeApp,
			giveGDPREnabled:     &falseValue,
			giveAppGDPREnabled:  &trueValue,
			wantEnabled:         &trueValue,
		},
		{
			description:          "GDPR Video integration enabled, general GDPR disabled",
			giveIntegrationType:  IntegrationTypeVideo,
			giveGDPREnabled:      &falseValue,
			giveVideoGDPREnabled: &trueValue,
			wantEnabled:          &trueValue,
		},
		{
			description:         "GDPR Web integration enabled, general GDPR disabled",
			giveIntegrationType: IntegrationTypeWeb,
			giveGDPREnabled:     &falseValue,
			giveWebGDPREnabled:  &trueValue,
			wantEnabled:         &trueValue,
		},
		{
			description:         "Web integration enabled, general GDPR unspecified",
			giveIntegrationType: IntegrationTypeWeb,
			giveGDPREnabled:     nil,
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
			description:         "GDPR Web integration disabled, general GDPR unspecified",
			giveIntegrationType: IntegrationTypeWeb,
			giveGDPREnabled:     nil,
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
				IntegrationEnabled: AccountGDPRIntegration{
					AMP:   tt.giveAMPGDPREnabled,
					App:   tt.giveAppGDPREnabled,
					Video: tt.giveVideoGDPREnabled,
					Web:   tt.giveWebGDPREnabled,
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

func TestAccountCCPAEnabledForRequestType(t *testing.T) {
	trueValue, falseValue := true, false

	tests := []struct {
		description          string
		giveRequestType      RequestType
		giveCCPAEnabled      *bool
		giveAMPCCPAEnabled   *bool
		giveAppCCPAEnabled   *bool
		giveVideoCCPAEnabled *bool
		giveWebCCPAEnabled   *bool
		wantEnabled          *bool
	}{
		{
			description:        "CCPA AMP integration enabled, general CCPA disabled",
			giveRequestType:    RequestTypeAMP,
			giveCCPAEnabled:    &falseValue,
			giveAMPCCPAEnabled: &trueValue,
			wantEnabled:        &trueValue,
		},
		{
			description:        "CCPA App integration enabled, general CCPA disabled",
			giveRequestType:    RequestTypeApp,
			giveCCPAEnabled:    &falseValue,
			giveAppCCPAEnabled: &trueValue,
			wantEnabled:        &trueValue,
		},
		{
			description:          "CCPA Video integration enabled, general CCPA disabled",
			giveRequestType:      RequestTypeVideo,
			giveCCPAEnabled:      &falseValue,
			giveVideoCCPAEnabled: &trueValue,
			wantEnabled:          &trueValue,
		},
		{
			description:        "CCPA Web integration enabled, general CCPA disabled",
			giveRequestType:    RequestTypeWeb,
			giveCCPAEnabled:    &falseValue,
			giveWebCCPAEnabled: &trueValue,
			wantEnabled:        &trueValue,
		},
		{
			description:        "Web integration enabled, general CCPA unspecified",
			giveRequestType:    RequestTypeWeb,
			giveCCPAEnabled:    nil,
			giveWebCCPAEnabled: &trueValue,
			wantEnabled:        &trueValue,
		},
		{
			description:        "CCPA Web integration disabled, general CCPA enabled",
			giveRequestType:    RequestTypeWeb,
			giveCCPAEnabled:    &trueValue,
			giveWebCCPAEnabled: &falseValue,
			wantEnabled:        &falseValue,
		},
		{
			description:        "CCPA Web integration disabled, general CCPA unspecified",
			giveRequestType:    RequestTypeWeb,
			giveCCPAEnabled:    nil,
			giveWebCCPAEnabled: &falseValue,
			wantEnabled:        &falseValue,
		},
		{
			description:        "CCPA Web integration unspecified, general CCPA disabled",
			giveRequestType:    RequestTypeWeb,
			giveCCPAEnabled:    &falseValue,
			giveWebCCPAEnabled: nil,
			wantEnabled:        &falseValue,
		},
		{
			description:        "CCPA Web integration unspecified, general CCPA enabled",
			giveRequestType:    RequestTypeWeb,
			giveCCPAEnabled:    &trueValue,
			giveWebCCPAEnabled: nil,
			wantEnabled:        &trueValue,
		},
		{
			description:        "CCPA Web integration unspecified, general CCPA unspecified",
			giveRequestType:    RequestTypeWeb,
			giveCCPAEnabled:    nil,
			giveWebCCPAEnabled: nil,
			wantEnabled:        nil,
		},
	}

	for _, tt := range tests {
		account := Account{
			CCPA: AccountCCPA{
				Enabled: tt.giveCCPAEnabled,
				IntegrationEnabled: AccountCCPAIntegration{
					AMP:   tt.giveAMPCCPAEnabled,
					App:   tt.giveAppCCPAEnabled,
					Video: tt.giveVideoCCPAEnabled,
					Web:   tt.giveWebCCPAEnabled,
				},
			},
		}

		enabled := account.CCPA.EnabledForRequestType(tt.giveRequestType)

		if tt.wantEnabled == nil {
			assert.Nil(t, enabled, tt.description)
		} else {
			assert.NotNil(t, enabled, tt.description)
			assert.Equal(t, *tt.wantEnabled, *enabled, tt.description)
		}
	}
}
