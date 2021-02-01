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
