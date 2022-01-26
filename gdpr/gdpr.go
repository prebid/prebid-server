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
	HostCookiesAllowed(ctx context.Context, cfg TCF2ConfigReader, gdprSignal Signal, consent string) (bool, error)

	// Determines whether or not the given bidder is allowed to user personal info for ad targeting.
	//
	// If the consent string was nonsensical, the returned error will be an ErrorMalformedConsent.
	BidderSyncAllowed(ctx context.Context, cfg TCF2ConfigReader, bidder openrtb_ext.BidderName, gdprSignal Signal, consent string) (bool, error)

	// Determines whether or not to send PI information to a bidder, or mask it out.
	//
	// If the consent string was nonsensical, the returned error will be an ErrorMalformedConsent.
	AuctionActivitiesAllowed(ctx context.Context, cfg TCF2ConfigReader, bidder openrtb_ext.BidderName, PublisherID string, gdprSignal Signal, consent string) (allowBidReq bool, passGeo bool, passID bool, err error)
}

// Versions of the GDPR TCF technical specification.
const (
	tcf2SpecVersion uint8 = 2
)

// NewPermissions gets an instance of the Permissions for use elsewhere in the project.
func NewPermissions(ctx context.Context, cfg config.GDPR, vendorIDs map[openrtb_ext.BidderName]uint16, client *http.Client) Permissions {
	if !cfg.Enabled {
		return &AlwaysAllow{}
	}

	permissionsImpl := &permissionsImpl{
		gdprDefaultValue:      cfg.DefaultValue,
		hostVendorID:          cfg.HostVendorID,
		nonStandardPublishers: cfg.NonStandardPublisherMap,
		vendorIDs:             vendorIDs,
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf2SpecVersion: newVendorListFetcher(ctx, cfg, client, vendorListURLMaker)},
	}

	if cfg.HostVendorID == 0 {
		return &AllowHostCookies{
			permissionsImpl: permissionsImpl,
		}
	}

	return permissionsImpl
}

// An ErrorMalformedConsent will be returned by the Permissions interface if
// the consent string argument was the reason for the failure.
type ErrorMalformedConsent struct {
	Consent string
	Cause   error
}

func (e *ErrorMalformedConsent) Error() string {
	return "malformed consent string " + e.Consent + ": " + e.Cause.Error()
}
