package gdpr

import (
	"context"
	"net/http"

	"github.com/prebid/go-gdpr/vendorlist"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type Permissions interface {
	// Determines whether or not the host company is allowed to read/write cookies.
	//
	// If the consent string was nonsensical, the returned error will be an ErrorMalformedConsent.
	HostCookiesAllowed(ctx context.Context, consent string) (bool, error)

	// Determines whether or not the given bidder is allowed to user personal info for ad targeting.
	//
	// If the consent string was nonsensical, the returned error will be an ErrorMalformedConsent.
	BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName, consent string) (bool, error)

	// Determines whether or not to send PI information to a bidder, or mask it out.
	//
	// If the consent string was nonsensical, the returned error will be an ErrorMalformedConsent.
	PersonalInfoAllowed(ctx context.Context, bidder openrtb_ext.BidderName, PublisherID string, consent string) (bool, bool, error)

	// Exposes the AMP execption flag
	AMPException() bool
}

const (
	tCF1 uint8 = 1
	tCF2 uint8 = 2
)

// NewPermissions gets an instance of the Permissions for use elsewhere in the project.
func NewPermissions(ctx context.Context, cfg config.GDPR, vendorIDs map[openrtb_ext.BidderName]uint16, client *http.Client) Permissions {
	// If the host doesn't buy into the IAB GDPR consent framework, then save some cycles and let all syncs happen.
	if cfg.HostVendorID == 0 {
		return AlwaysAllow{}
	}

	return &permissionsImpl{
		cfg:       cfg,
		vendorIDs: vendorIDs,
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tCF1: newVendorListFetcher(ctx, cfg, client, vendorListURLMaker, tCF1),
			tCF2: newVendorListFetcher(ctx, cfg, client, vendorListURLMaker, tCF2)},
	}
}

// An ErrorMalformedConsent will be returned by the Permissions interface if
// the consent string argument was the reason for the failure.
type ErrorMalformedConsent struct {
	consent string
	cause   error
}

func (e *ErrorMalformedConsent) Error() string {
	return "malformed consent string " + e.consent + ": " + e.cause.Error()
}
