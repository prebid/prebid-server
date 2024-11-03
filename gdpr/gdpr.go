package gdpr

import (
	"context"

	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

type Permissions interface {
	// Determines whether or not the host company is allowed to read/write cookies.
	//
	// If the consent string was nonsensical, the returned error will be an ErrorMalformedConsent.
	HostCookiesAllowed(ctx context.Context) (bool, error)

	// Determines whether or not the given bidder is allowed to user personal info for ad targeting.
	//
	// If the consent string was nonsensical, the returned error will be an ErrorMalformedConsent.
	BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName) (bool, error)

	// Determines whether or not to send PI information to a bidder, or mask it out.
	//
	// If the consent string was nonsensical, the no permissions are granted.
	AuctionActivitiesAllowed(ctx context.Context, bidderCoreName openrtb_ext.BidderName, bidder openrtb_ext.BidderName) AuctionPermissions
}

type PermissionsBuilder func(TCF2ConfigReader, RequestInfo) Permissions

type RequestInfo struct {
	AliasGVLIDs map[string]uint16
	Consent     string
	GDPRSignal  Signal
	PublisherID string
}

// NewPermissionsBuilder takes host config data used to configure the builder function it returns
func NewPermissionsBuilder(cfg config.GDPR, gvlVendorIDs map[openrtb_ext.BidderName]uint16, vendorListFetcher VendorListFetcher) PermissionsBuilder {
	return func(tcf2Cfg TCF2ConfigReader, requestInfo RequestInfo) Permissions {
		purposeEnforcerBuilder := NewPurposeEnforcerBuilder(tcf2Cfg)

		return NewPermissions(cfg, tcf2Cfg, gvlVendorIDs, vendorListFetcher, purposeEnforcerBuilder, requestInfo)
	}
}

// NewPermissions gets a per-request Permissions object that can then be used to check GDPR permissions for a given bidder.
func NewPermissions(cfg config.GDPR, tcf2Config TCF2ConfigReader, vendorIDs map[openrtb_ext.BidderName]uint16, fetcher VendorListFetcher, purposeEnforcerBuilder PurposeEnforcerBuilder, requestInfo RequestInfo) Permissions {
	if !cfg.Enabled {
		return &AlwaysAllow{}
	}

	permissionsImpl := &permissionsImpl{
		fetchVendorList:        fetcher,
		gdprDefaultValue:       cfg.DefaultValue,
		hostVendorID:           cfg.HostVendorID,
		nonStandardPublishers:  cfg.NonStandardPublisherMap,
		cfg:                    tcf2Config,
		vendorIDs:              vendorIDs,
		publisherID:            requestInfo.PublisherID,
		gdprSignal:             SignalNormalize(requestInfo.GDPRSignal, cfg.DefaultValue),
		consent:                requestInfo.Consent,
		aliasGVLIDs:            requestInfo.AliasGVLIDs,
		purposeEnforcerBuilder: purposeEnforcerBuilder,
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
