package gdpr

import (
	"context"
	"testing"

	"github.com/prebid/go-gdpr/vendorlist"
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
		vendorListFetcher := func(ctx context.Context, id uint16) (vendorlist.VendorList, error) {
			return nil, nil
		}

		perms := NewPermissions(config, &tcf2Config{}, vendorIDs, vendorListFetcher)

		assert.IsType(t, tt.wantType, perms, tt.description)
	}
}
