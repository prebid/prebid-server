package gdpr

import (
	"context"
	"testing"

	"github.com/prebid/go-gdpr/consentconstants"
	"github.com/prebid/go-gdpr/vendorlist"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"

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
		vendorListFetcher := func(ctx context.Context, specVersion, listVersion uint16) (vendorlist.VendorList, error) {
			return nil, nil
		}

		fakePurposeEnforcerBuilder := fakePurposeEnforcerBuilder{
			purposeEnforcer: nil,
		}.Builder
		perms := NewPermissions(config, &tcf2Config{}, vendorIDs, vendorListFetcher, fakePurposeEnforcerBuilder, RequestInfo{})

		assert.IsType(t, tt.wantType, perms, tt.description)
	}
}

type fakePurposeEnforcerBuilder struct {
	purposeEnforcer PurposeEnforcer
}

func (fpeb fakePurposeEnforcerBuilder) Builder(consentconstants.Purpose, string) PurposeEnforcer {
	return fpeb.purposeEnforcer
}
