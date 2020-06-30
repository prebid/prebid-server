package openrtb_ext

import (
	"strconv"

	"github.com/PubMatic-OpenWrap/prebid-server/errortypes"
	"github.com/buger/jsonparser"
)

// PrebidExtKey represents the prebid extension key used in requests
const PrebidExtKey = "prebid"

// ExtDevice defines the contract for bidrequest.device.ext
type ExtDevice struct {
	Prebid ExtDevicePrebid `json:"prebid"`
}

// Pointer to interstitial so we do not force it to exist
type ExtDevicePrebid struct {
	Interstitial *ExtDeviceInt `json:"interstitial"`
}

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
