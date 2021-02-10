package openrtb_ext

import (
	"encoding/json"
	"strconv"

	"github.com/buger/jsonparser"
	"github.com/prebid/prebid-server/errortypes"
)

// PrebidExtKey represents the prebid extension key used in requests
const PrebidExtKey = "prebid"

// ExtDevice defines the contract for bidrequest.device.ext
type ExtDevice struct {
	// Attribute:
	//   atts
	// Type:
	//   integer; optional - iOS Only
	// Description:
	//   iOS app tracking authorization status.
	// Extention Spec:
	//   https://github.com/InteractiveAdvertisingBureau/openrtb/blob/master/extensions/community_extensions/skadnetwork.md
	ATTS   *IOSAppTrackingStatus `json:"atts"`
	Prebid ExtDevicePrebid       `json:"prebid"`
}

// IOSAppTrackingStatus describes the values for iOS app tracking authorization status.
type IOSAppTrackingStatus int

// Values of the IOSAppTrackingStatus enumeration.
const (
	IOSAppTrackingStatusNotDetermined IOSAppTrackingStatus = 0
	IOSAppTrackingStatusRestricted    IOSAppTrackingStatus = 1
	IOSAppTrackingStatusDenied        IOSAppTrackingStatus = 2
	IOSAppTrackingStatusAuthorized    IOSAppTrackingStatus = 3
)

// ExtDevicePrebid defines the contract for bidrequest.device.ext.prebid
type ExtDevicePrebid struct {
	Interstitial *ExtDeviceInt `json:"interstitial"`
}

// ExtDeviceInt defines the contract for bidrequest.device.ext.prebid.interstitial
type ExtDeviceInt struct {
	MinWidthPerc  uint64 `json:"minwidtheperc"`
	MinHeightPerc uint64 `json:"minheightperc"`
}

func (edi *ExtDeviceInt) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return &errortypes.BadInput{Message: "request.device.ext.prebid.interstitial must have some data in it"}
	}
	if value, dataType, _, _ := jsonparser.Get(b, "minwidthperc"); dataType != jsonparser.Number {
		return &errortypes.BadInput{Message: "request.device.ext.prebid.interstitial.minwidthperc must be a number between 0 and 100"}
	} else {
		perc, err := strconv.Atoi(string(value))
		if err != nil || perc < 0 || perc > 100 {
			return &errortypes.BadInput{Message: "request.device.ext.prebid.interstitial.minwidthperc must be a number between 0 and 100"}
		}
		edi.MinWidthPerc = uint64(perc)
	}
	if value, dataType, _, _ := jsonparser.Get(b, "minheightperc"); dataType != jsonparser.Number {
		return &errortypes.BadInput{Message: "request.device.ext.prebid.interstitial.minheightperc must be a number between 0 and 100"}
	} else {
		perc, err := strconv.Atoi(string(value))
		if err != nil || perc < 0 || perc > 100 {
			return &errortypes.BadInput{Message: "request.device.ext.prebid.interstitial.minheightperc must be a number between 0 and 100"}
		}
		edi.MinHeightPerc = uint64(perc)
	}
	return nil
}

func ParseDeviceExtATTS(deviceExt json.RawMessage) (*IOSAppTrackingStatus, error) {
	return nil, nil
}
