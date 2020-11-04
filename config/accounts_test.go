package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccountGDPREnabledForRequestType(t *testing.T) {
	tests := []struct {
		description          string
		giveRequestType      RequestType
		giveGDPREnabled      *bool
		giveAMPGDPREnabled   *bool
		giveAppGDPREnabled   *bool
		giveVideoGDPREnabled *bool
		giveWebGDPREnabled   *bool
		wantEnabled          *bool
	}{
		{
			description:        "GDPR AMP integration enabled, general GDPR disabled",
			giveRequestType:    RequestTypeAMP,
			giveGDPREnabled:    &[]bool{false}[0],
			giveAMPGDPREnabled: &[]bool{true}[0],
			wantEnabled:        &[]bool{true}[0],
		},
		{
			description:        "GDPR App integration enabled, general GDPR disabled",
			giveRequestType:    RequestTypeApp,
			giveGDPREnabled:    &[]bool{false}[0],
			giveAppGDPREnabled: &[]bool{true}[0],
			wantEnabled:        &[]bool{true}[0],
		},
		{
			description:          "GDPR Video integration enabled, general GDPR disabled",
			giveRequestType:      RequestTypeVideo,
			giveGDPREnabled:      &[]bool{false}[0],
			giveVideoGDPREnabled: &[]bool{true}[0],
			wantEnabled:          &[]bool{true}[0],
		},
		{
			description:        "GDPR Web integration enabled, general GDPR disabled",
			giveRequestType:    RequestTypeWeb,
			giveGDPREnabled:    &[]bool{false}[0],
			giveWebGDPREnabled: &[]bool{true}[0],
			wantEnabled:        &[]bool{true}[0],
		},
		{
			description:        "Web integration enabled, general GDPR unspecified",
			giveRequestType:    RequestTypeWeb,
			giveGDPREnabled:    nil,
			giveWebGDPREnabled: &[]bool{true}[0],
			wantEnabled:        &[]bool{true}[0],
		},
		{
			description:        "GDPR Web integration disabled, general GDPR enabled",
			giveRequestType:    RequestTypeWeb,
			giveGDPREnabled:    &[]bool{true}[0],
			giveWebGDPREnabled: &[]bool{false}[0],
			wantEnabled:        &[]bool{false}[0],
		},
		{
			description:        "GDPR Web integration disabled, general GDPR unspecified",
			giveRequestType:    RequestTypeWeb,
			giveGDPREnabled:    nil,
			giveWebGDPREnabled: &[]bool{false}[0],
			wantEnabled:        &[]bool{false}[0],
		},
		{
			description:        "GDPR Web integration unspecified, general GDPR disabled",
			giveRequestType:    RequestTypeWeb,
			giveGDPREnabled:    &[]bool{false}[0],
			giveWebGDPREnabled: nil,
			wantEnabled:        &[]bool{false}[0],
		},
		{
			description:        "GDPR Web integration unspecified, general GDPR enabled",
			giveRequestType:    RequestTypeWeb,
			giveGDPREnabled:    &[]bool{true}[0],
			giveWebGDPREnabled: nil,
			wantEnabled:        &[]bool{true}[0],
		},
		{
			description:        "GDPR Web integration unspecified, general GDPR unspecified",
			giveRequestType:    RequestTypeWeb,
			giveGDPREnabled:    nil,
			giveWebGDPREnabled: nil,
			wantEnabled:        nil,
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

		enabled := account.GDPR.EnabledForRequestType(tt.giveRequestType)

		if tt.wantEnabled == nil {
			assert.Nil(t, enabled, tt.description)
		} else {
			assert.NotNil(t, enabled, tt.description)
			assert.Equal(t, *tt.wantEnabled, *enabled, tt.description)
		}
	}
}
