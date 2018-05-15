package gdpr

import (
	"context"
	"net/http"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type Permissions interface {
	HostCookiesAllowed(ctx context.Context, consent string) (bool, error)
	BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName, consent string) (bool, error)
}

// NewPermissions gets an instance of the Permissions for use elsewhere in the project.
func NewPermissions(ctx context.Context, cfg config.GDPR, vendorIDs map[openrtb_ext.BidderName]uint16, client *http.Client) Permissions {
	return &permissionsImpl{
		cfg:             cfg,
		vendorIDs:       vendorIDs,
		fetchVendorList: newVendorListFetcher(ctx, client, vendorListURLMaker),
	}
}
