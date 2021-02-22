package gdpr

import (
	"context"
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/stretchr/testify/assert"
)

func TestNewPermissions(t *testing.T) {
	tests := []struct {
		description  string
		gdprEnabled  bool
		hostVendorID int
		wantType     Permissions
	}{
		{
			gdprEnabled:  false,
			hostVendorID: 32,
			wantType:     &AlwaysAllow{},
		},
		{
			gdprEnabled:  true,
			hostVendorID: 0,
			wantType:     &AllowHostCookies{},
		},
		{
			gdprEnabled:  true,
			hostVendorID: 32,
			wantType:     &permissionsImpl{},
		},
	}

	for _, tt := range tests {

		config := config.GDPR{
			Enabled:      tt.gdprEnabled,
			HostVendorID: tt.hostVendorID,
		}
		vendorIDs := map[openrtb_ext.BidderName]uint16{}

		perms := NewPermissions(context.Background(), config, vendorIDs, &http.Client{})

		assert.IsType(t, tt.wantType, perms, tt.description)
	}
}

func TestSignalParse(t *testing.T) {
	tests := []struct {
		description string
		rawSignal   string
		wantSignal  Signal
		wantError   bool
	}{
		{
			description: "valid raw signal is 0",
			rawSignal:   "0",
			wantSignal:  SignalNo,
			wantError:   false,
		},
		{
			description: "Valid signal - raw signal is 1",
			rawSignal:   "1",
			wantSignal:  SignalYes,
			wantError:   false,
		},
		{
			description: "Valid signal - raw signal is empty",
			rawSignal:   "",
			wantSignal:  SignalAmbiguous,
			wantError:   false,
		},
		{
			description: "Invalid signal - raw signal is -1",
			rawSignal:   "-1",
			wantSignal:  SignalAmbiguous,
			wantError:   true,
		},
		{
			description: "Invalid signal - raw signal is abc",
			rawSignal:   "abc",
			wantSignal:  SignalAmbiguous,
			wantError:   true,
		},
	}

	for _, tt := range tests {
		signal, err := SignalParse(tt.rawSignal)

		assert.Equal(t, tt.wantSignal, signal, tt.description)

		if tt.wantError {
			assert.NotNil(t, err, tt.description)
		} else {
			assert.Nil(t, err, tt.description)
		}
	}
}
