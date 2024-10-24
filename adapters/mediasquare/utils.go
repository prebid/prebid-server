package mediasquare

import (
	"encoding/json"
	"fmt"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

var headerList = map[string][]string{
	"Content-Type": {"application/json;charset=utf-8"},
	"Accept":       {"application/json"},
}

var mediaTypeList = map[openrtb_ext.BidType]openrtb2.MarkupType{
	"banner": openrtb2.MarkupBanner,
	"video":  openrtb2.MarkupVideo,
	"audio":  openrtb2.MarkupAudio,
	"native": openrtb2.MarkupNative,
}

// mType: Returns the openrtb2.MarkupType from an msqResponseBids.
func (msqBids *msqResponseBids) mType() openrtb2.MarkupType {
	switch {
	case msqBids.Video != nil:
		return mediaTypeList["video"]
	case msqBids.Native != nil:
		return mediaTypeList["native"]
	default:
		return mediaTypeList["banner"]
	}
}

// bidType: Returns the openrtb_ext.BidType from an msqResponseBids.
func (msqBids *msqResponseBids) bidType() openrtb_ext.BidType {
	switch {
	case msqBids.Video != nil:
		return "video"
	case msqBids.Native != nil:
		return "native"
	default:
		return "banner"
	}
}

// extBid: Extracts the ExtBid from msqBids formated as (json.RawMessage).
func (msqBids *msqResponseBids) extBid() (raw json.RawMessage) {
	extBid, _ := msqBids.loadExtBid()
	if extBid.DSA != nil || extBid.Prebid != nil {
		if bb, _ := json.Marshal(extBid); len(bb) > 0 {
			raw = json.RawMessage(bb)
		}
	}
	return
}

// loadExtBid: Extracts the ExtBid from msqBids as (openrtb_ext.ExtBid, []error).
func (msqBids *msqResponseBids) loadExtBid() (extBid openrtb_ext.ExtBid, errs []error) {
	if msqBids.Dsa != nil {
		bb, err := json.Marshal(msqBids.Dsa)
		if err != nil {
			errs = append(errs, err)
		}
		if len(bb) > 0 {
			var dsa openrtb_ext.ExtBidDSA
			if err = json.Unmarshal(bb, &dsa); err != nil {
				errs = append(errs, err)
			} else {
				extBid.DSA = &dsa
			}
		}
	}
	return
}

// extBidPrebidMeta: Extracts the ExtBidPrebidMeta from msqBids as (*openrtb_ext.ExtBidPrebidMeta).
func (msqBids *msqResponseBids) extBidPrebidMeta() *openrtb_ext.ExtBidPrebidMeta {
	var extBidMeta openrtb_ext.ExtBidPrebidMeta
	if msqBids.ADomain != nil {
		extBidMeta.AdvertiserDomains = msqBids.ADomain
	}
	extBidMeta.MediaType = string(msqBids.bidType())
	return &extBidMeta
}

// ptrInt8ToBool: Returns (TRUE) when i equals 1.
func ptrInt8ToBool(i *int8) bool {
	if i != nil {
		return (*i == int8(1))
	}
	return false
}

// intToPtrInt: Returns a ptr_int(*int) for which *ptr_int = i.
func intToPtrInt(i int) *int {
	val := int(i)
	return &val
}

// errorWritter: Returns a Custom error message.
func errorWritter(referer string, err error, isEmpty bool) error {
	if isEmpty {
		return fmt.Errorf("%s: is empty.", referer)
	}
	return fmt.Errorf("%s: %s", referer, err.Error())
}
