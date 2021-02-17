package gdpr

import (
	"context"
	"net/http"
	"strconv"

	"github.com/prebid/go-gdpr/vendorlist"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
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
	PersonalInfoAllowed(ctx context.Context, bidder openrtb_ext.BidderName, PublisherID string, gdprSignal Signal, consent string) (bool, bool, bool, error)
}

// Versions of the GDPR TCF technical specification.
const (
	tcf1SpecVersion uint8 = 1
	tcf2SpecVersion uint8 = 2
)

// NewPermissions gets an instance of the Permissions for use elsewhere in the project.
func NewPermissions(ctx context.Context, cfg config.GDPR, vendorIDs map[openrtb_ext.BidderName]uint16, client *http.Client) Permissions {
	if !cfg.Enabled {
		return &AlwaysAllow{}
	}

	permissionsImpl := &permissionsImpl{
		cfg:       cfg,
		vendorIDs: vendorIDs,
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf1SpecVersion: newVendorListFetcherTCF1(cfg),
			tcf2SpecVersion: newVendorListFetcherTCF2(ctx, cfg, client, vendorListURLMaker)},
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
	consent string
	cause   error
}

func (e *ErrorMalformedConsent) Error() string {
	return "malformed consent string " + e.consent + ": " + e.cause.Error()
}

// SignalParse parses a raw signal and returns
func SignalParse(rawSignal string) (Signal, error) {
	if rawSignal == "" {
		return SignalAmbiguous, nil
	}

	i, err := strconv.Atoi(rawSignal)

	if err != nil {
		return SignalAmbiguous, err
	}
	if i != 0 && i != 1 {
		return SignalAmbiguous, &errortypes.BadInput{Message: "GDPR signal should be integer 0 or 1"}
	}

	return Signal(i), nil
}
