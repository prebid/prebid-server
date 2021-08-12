package gdpr

import (
	"context"
	"net/http"
	"strconv"

	"github.com/prebid/go-gdpr/consentconstants"
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
	AuctionActivitiesAllowed(ctx context.Context, bidder openrtb_ext.BidderName, PublisherID string, gdprSignal Signal, consent string, weakVendorEnforcement bool) (allowBidReq bool, passGeo bool, passID bool, err error)
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

	gdprDefaultValue := SignalYes
	if cfg.DefaultValue == "0" {
		gdprDefaultValue = SignalNo
	}

	purposeConfigs := map[consentconstants.Purpose]config.TCF2Purpose{
		1:  cfg.TCF2.Purpose1,
		2:  cfg.TCF2.Purpose2,
		3:  cfg.TCF2.Purpose3,
		4:  cfg.TCF2.Purpose4,
		5:  cfg.TCF2.Purpose5,
		6:  cfg.TCF2.Purpose6,
		7:  cfg.TCF2.Purpose7,
		8:  cfg.TCF2.Purpose8,
		9:  cfg.TCF2.Purpose9,
		10: cfg.TCF2.Purpose10,
	}

	permissionsImpl := &permissionsImpl{
		cfg:              cfg,
		gdprDefaultValue: gdprDefaultValue,
		purposeConfigs:   purposeConfigs,
		vendorIDs:        vendorIDs,
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
