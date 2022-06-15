package gdpr

import (
	"context"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type Permissions interface {
	// Determines whether or not the host company is allowed to read/write cookies.
	//
	// If the consent string was nonsensical, the returned error will be an ErrorMalformedConsent.
	HostCookiesAllowed(ctx context.Context, gdprSignal Signal, consent string) (bool, error)

	// Determines whether or not the given bidder is allowed to user personal info for ad targeting.
	//
	// If the consent string was nonsensical, the returned error will be an ErrorMalformedConsent.
	BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName, gdprSignal Signal, consent string) (bool, error)

	// Determines whether or not to send PI information to a bidder, or mask it out.
	//
	// If the consent string was nonsensical, the returned error will be an ErrorMalformedConsent.
	AuctionActivitiesAllowed(ctx context.Context, bidderCoreName openrtb_ext.BidderName, bidder openrtb_ext.BidderName, PublisherID string, gdprSignal Signal, consent string, aliasGVLIDs map[string]uint16) (permissions AuctionPermissions, err error)
}

type PermissionsBuilder func(TCF2ConfigReader) Permissions

// NewPermissionsBuilder is of type PermissionsBuilderMaker
// It takes host config data used to configure the builder function it returns
func NewPermissionsBuilder(cfg config.GDPR, gvlVendorIDs map[openrtb_ext.BidderName]uint16, vendorListFetcher VendorListFetcher) PermissionsBuilder {
	return func(tcf2Cfg TCF2ConfigReader) Permissions {
		return NewPermissions(cfg, tcf2Cfg, gvlVendorIDs, vendorListFetcher)
	}
}

// NewPermissions gets a per-request Permissions object that can then be used to check GDPR permissions for a given bidder.
func NewPermissions(cfg config.GDPR, tcf2Config TCF2ConfigReader, vendorIDs map[openrtb_ext.BidderName]uint16, fetcher VendorListFetcher) Permissions {
	if !cfg.Enabled {
		return &AlwaysAllow{}
	}

	permissionsImpl := &permissionsImpl{
		fetchVendorList:       fetcher,
		gdprDefaultValue:      cfg.DefaultValue,
		hostVendorID:          cfg.HostVendorID,
		nonStandardPublishers: cfg.NonStandardPublisherMap,
		cfg:                   tcf2Config,
		vendorIDs:             vendorIDs,
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
