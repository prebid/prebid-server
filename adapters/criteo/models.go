package criteo

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type criteoRequest struct {
	ID          string                   `json:"id,omitempty"`
	Publisher   criteoPublisher          `json:"publisher,omitempty"`
	User        criteoUser               `json:"user,omitempty"`
	GdprConsent criteoGdprConsent        `json:"gdprconsent,omitempty"`
	Slots       []criteoRequestSlot      `json:"slots,omitempty"`
	Eids        []openrtb_ext.ExtUserEid `json:"eids,omitempty"`
}

func newCriteoRequest(slotIDGenerator slotIDGenerator, request *openrtb2.BidRequest) (criteoRequest, []error) {
	var errs []error

	// request cannot be nil by design

	criteoRequest := criteoRequest{}

	criteoRequest.ID = request.ID

	// Extracting request slots
	if len(request.Imp) > 0 {
		criteoSlots, slotsErr := newCriteoRequestSlots(slotIDGenerator, request.Imp)
		if len(slotsErr) > 0 {
			return criteoRequest, slotsErr
		}
		criteoRequest.Slots = criteoSlots
	}

	var networkId *int64
	for _, criteoSlot := range criteoRequest.Slots {
		if networkId == nil && criteoSlot.NetworkID != nil && *criteoSlot.NetworkID > 0 {
			networkId = criteoSlot.NetworkID
		} else if networkId != nil && criteoSlot.NetworkID != nil && *criteoSlot.NetworkID != *networkId {
			return criteoRequest, []error{&errortypes.BadInput{
				Message: "Bid request has slots coming with several network IDs which is not allowed",
			}}
		}
	}

	criteoRequest.Publisher = newCriteoPublisher(networkId, request.App, request.Site)

	var regsExt *openrtb_ext.ExtRegs
	if request.Regs != nil && request.Regs.Ext != nil {
		if err := json.Unmarshal(request.Regs.Ext, &regsExt); err != nil {
			errs = append(errs, err)
		}
	}

	criteoRequest.User = newCriteoUser(request.User, request.Device, regsExt)

	if gdprConsent, err := newCriteoGdprConsent(request.User, regsExt); err != nil {
		errs = append(errs, err)
	} else {
		criteoRequest.GdprConsent = gdprConsent
	}

	if request.User != nil && request.User.Ext != nil {
		var extUser openrtb_ext.ExtUser
		if err := json.Unmarshal(request.User.Ext, &extUser); err != nil {
			errs = append(errs, err)
		} else {
			criteoRequest.Eids = extUser.Eids
		}
	}

	return criteoRequest, errs
}

type criteoPublisher struct {
	SiteID    string `json:"siteid,omitempty"`
	BundleID  string `json:"bundleid,omitempty"`
	URL       string `json:"url,omitempty"`
	NetworkID *int64 `json:"networkid,omitempty"`
}

func newCriteoPublisher(networkId *int64, app *openrtb2.App, site *openrtb2.Site) criteoPublisher {
	// Both app and site cannot be nil at the same time by design in PBS

	criteoPublisher := criteoPublisher{}

	if networkId != nil && *networkId > 0 {
		criteoPublisher.NetworkID = networkId
	}

	if app != nil {
		criteoPublisher.BundleID = app.Bundle
	}

	if site != nil {
		criteoPublisher.SiteID = site.ID
		criteoPublisher.URL = site.Page
	}

	return criteoPublisher
}

type criteoUser struct {
	DeviceID     string `json:"deviceid,omitempty"`
	DeviceOS     string `json:"deviceos,omitempty"`
	DeviceIDType string `json:"deviceidtype,omitempty"`
	CookieID     string `json:"cookieuid,omitempty"`
	UID          string `json:"uid,omitempty"`
	IP           string `json:"ip,omitempty"`
	IPv6         string `json:"ipv6,omitempty"`
	UA           string `json:"ua,omitempty"`
	UspIab       string `json:"uspIab,omitempty"`
}

func newCriteoUser(user *openrtb2.User, device *openrtb2.Device, regsExt *openrtb_ext.ExtRegs) criteoUser {
	criteoUser := criteoUser{}

	if user == nil && device == nil {
		return criteoUser
	}

	if user != nil {
		criteoUser.CookieID = user.BuyerUID
	}

	if device != nil {
		deviceType := getDeviceType(device.OS)
		criteoUser.DeviceIDType = deviceType

		criteoUser.DeviceOS = device.OS
		criteoUser.DeviceID = device.IFA
		criteoUser.IP = device.IP
		criteoUser.IPv6 = device.IPv6
		criteoUser.UA = device.UA
	}

	if regsExt != nil {
		criteoUser.UspIab = regsExt.USPrivacy // CCPA
	}

	return criteoUser
}

type criteoGdprConsent struct {
	GdprApplies *bool  `json:"gdprapplies,omitempty"`
	ConsentData string `json:"consentdata,omitempty"`
}

func newCriteoGdprConsent(user *openrtb2.User, regsExt *openrtb_ext.ExtRegs) (criteoGdprConsent, error) {
	consent := criteoGdprConsent{}

	if user == nil && regsExt == nil {
		return consent, nil
	}

	if user != nil && user.Ext != nil {
		var userExt *openrtb_ext.ExtUser
		if err := json.Unmarshal(user.Ext, &userExt); err != nil {
			return consent, err
		}
		consent.ConsentData = userExt.Consent
	}

	if regsExt != nil {
		if regsExt.GDPR != nil {
			gdprApplies := bool((*regsExt.GDPR & 1) == 1)
			consent.GdprApplies = &gdprApplies
		}
	}

	return consent, nil
}

type criteoRequestSlot struct {
	SlotID    string              `json:"slotid,omitempty"`
	ImpID     string              `json:"impid,omitempty"`
	ZoneID    *int64              `json:"zoneid,omitempty"`
	NetworkID *int64              `json:"networkid,omitempty"`
	Sizes     []criteoRequestSize `json:"sizes,omitempty"`
}

func newCriteoRequestSlots(slotIDGenerator slotIDGenerator, impressions []openrtb2.Imp) ([]criteoRequestSlot, []error) {
	var errs []error

	// `impressions` known not to be nil or empty by design, PBS checks it upstream.

	// Criteo slot should comes with any of (both are ok as well):
	//   - `zoneid`
	//   - `networkid`, `slotid`, `sizes`
	//
	// if not, criteo will reject the slot.

	var criteoSlots = make([]criteoRequestSlot, len(impressions))

	for i := 0; i < len(impressions); i++ {
		criteoSlots[i] = criteoRequestSlot{}

		criteoSlots[i].ImpID = impressions[i].ID

		// Generating a random slot ID
		generatedSlotID, err := slotIDGenerator.NewSlotID()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		criteoSlots[i].SlotID = generatedSlotID

		if impressions[i].Banner != nil {
			if impressions[i].Banner.Format != nil {
				criteoSlots[i].Sizes = make([]criteoRequestSize, len(impressions[i].Banner.Format))
				for idx, format := range impressions[i].Banner.Format {
					criteoSlots[i].Sizes[idx] = newCriteoRequestSize(format.W, format.H)
				}
			} else if impressions[i].Banner.W != nil && *impressions[i].Banner.W > 0 && impressions[i].Banner.H != nil && *impressions[i].Banner.H > 0 {
				criteoSlots[i].Sizes = make([]criteoRequestSize, 1)
				criteoSlots[i].Sizes[0] = newCriteoRequestSize(*impressions[i].Banner.W, *impressions[i].Banner.H)
			}
		}

		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(impressions[i].Ext, &bidderExt); err != nil {
			errs = append(errs, err)
			continue
		}

		if bidderExt.Bidder != nil {
			var criteoExt openrtb_ext.ExtImpCriteo
			if err := json.Unmarshal(bidderExt.Bidder, &criteoExt); err != nil {
				errs = append(errs, err)
				continue
			}
			if criteoExt.ZoneID > 0 {
				criteoSlots[i].ZoneID = &criteoExt.ZoneID
			}
			if criteoExt.NetworkID > 0 {
				criteoSlots[i].NetworkID = &criteoExt.NetworkID
			}
		}
	}

	return criteoSlots, errs
}

type criteoRequestSize = string

func newCriteoRequestSize(width int64, height int64) criteoRequestSize {
	return fmt.Sprintf("%dx%d", width, height)
}

var deviceType = map[string]string{
	"ios":     "idfa",
	"android": "gaid",
	"unknown": "unknown",
}

func getDeviceType(os string) string {
	if os != "" {
		if dtype, ok := deviceType[strings.ToLower(os)]; ok {
			return dtype
		}
	}

	return deviceType["unknown"]
}

type criteoResponse struct {
	ID    string               `json:"id,omitempty"`
	Slots []criteoResponseSlot `json:"slots,omitempty"`
}

func newCriteoResponseFromBytes(bytes []byte) (criteoResponse, error) {
	var err error
	var bidResponse criteoResponse

	if err = json.Unmarshal(bytes, &bidResponse); err != nil {
		return bidResponse, err
	}

	return bidResponse, nil
}

type criteoResponseSlot struct {
	ArbitrageID  string  `json:"arbitrageid,omitempty"`
	ImpID        string  `json:"impid,omitempty"`
	ZoneID       int64   `json:"zoneid,omitempty"`
	NetworkID    int64   `json:"networkid,omitempty"`
	CPM          float64 `json:"cpm,omitempty"`
	Currency     string  `json:"currency,omitempty"`
	Width        int64   `json:"width,omitempty"`
	Height       int64   `json:"height,omitempty"`
	Creative     string  `json:"creative,omitempty"`
	CreativeCode string  `json:"creativecode,omitempty"`
}
