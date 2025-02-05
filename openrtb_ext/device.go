package openrtb_ext

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/buger/jsonparser"
	"github.com/prebid/prebid-server/v3/errortypes"
)

// PrebidExtKey represents the prebid extension key used in requests
const PrebidExtKey = "prebid"

// PrebidExtBidderKey represents the field name within request.imp.ext.prebid reserved for bidder params.
const PrebidExtBidderKey = "bidder"

// ExtDevice defines the contract for bidrequest.device.ext
type ExtDevice struct {
	// Attribute:
	//   atts
	// Type:
	//   integer; optional - iOS Only
	// Description:
	//   iOS app tracking authorization status.
	// Extension Spec:
	//   https://github.com/InteractiveAdvertisingBureau/openrtb/blob/master/extensions/community_extensions/skadnetwork.md
	ATTS *IOSAppTrackingStatus `json:"atts"`

	// Attribute:
	//   prebid
	// Type:
	//   object; optional
	// Description:
	//   Prebid extensions for the Device object.
	Prebid ExtDevicePrebid `json:"prebid"`
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

// IsKnownIOSAppTrackingStatus returns true if the value is a known iOS app tracking authorization status.
func IsKnownIOSAppTrackingStatus(v int64) bool {
	switch IOSAppTrackingStatus(v) {
	case IOSAppTrackingStatusNotDetermined:
		return true
	case IOSAppTrackingStatusRestricted:
		return true
	case IOSAppTrackingStatusDenied:
		return true
	case IOSAppTrackingStatusAuthorized:
		return true
	default:
		return false
	}
}

// ExtDevicePrebid defines the contract for bidrequest.device.ext.prebid
type ExtDevicePrebid struct {
	Interstitial *ExtDeviceInt `json:"interstitial"`
}

// ExtDeviceInt defines the contract for bidrequest.device.ext.prebid.interstitial
type ExtDeviceInt struct {
	MinWidthPerc  int64 `json:"minwidthperc"`
	MinHeightPerc int64 `json:"minheightperc"`
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
		edi.MinWidthPerc = int64(perc)
	}
	if value, dataType, _, _ := jsonparser.Get(b, "minheightperc"); dataType != jsonparser.Number {
		return &errortypes.BadInput{Message: "request.device.ext.prebid.interstitial.minheightperc must be a number between 0 and 100"}
	} else {
		perc, err := strconv.Atoi(string(value))
		if err != nil || perc < 0 || perc > 100 {
			return &errortypes.BadInput{Message: "request.device.ext.prebid.interstitial.minheightperc must be a number between 0 and 100"}
		}
		edi.MinHeightPerc = int64(perc)
	}
	return nil
}

// ParseDeviceExtATTS parses the ATTS value from the request.device.ext OpenRTB field.
func ParseDeviceExtATTS(deviceExt json.RawMessage) (*IOSAppTrackingStatus, error) {
	v, err := jsonparser.GetInt(deviceExt, "atts")

	// node not found error
	if err == jsonparser.KeyPathNotFoundError {
		return nil, nil
	}

	// unexpected parse error
	if err != nil {
		return nil, err
	}

	// invalid value error
	if !IsKnownIOSAppTrackingStatus(v) {
		return nil, errors.New("invalid status")
	}

	status := IOSAppTrackingStatus(v)
	return &status, nil
}
