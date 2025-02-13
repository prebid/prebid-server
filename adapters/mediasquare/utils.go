package mediasquare

import (
	"encoding/json"
	"fmt"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

var headerList = map[string][]string{
	"Content-Type": {"application/json;charset=utf-8"},
	"Accept":       {"application/json"},
}

// mType: Returns the openrtb2.MarkupType from an msqResponseBids.
func (msqBids *msqResponseBids) mType() openrtb2.MarkupType {
	switch {
	case msqBids.Video != nil:
		return openrtb2.MarkupVideo
	case msqBids.Native != nil:
		return openrtb2.MarkupNative
	default:
		return openrtb2.MarkupBanner
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
		if bb, _ := jsonutil.Marshal(extBid); len(bb) > 0 {
			raw = json.RawMessage(bb)
		}
	}
	return
}

// loadExtBid: Extracts the ExtBid from msqBids as (openrtb_ext.ExtBid, []error).
func (msqBids *msqResponseBids) loadExtBid() (extBid openrtb_ext.ExtBid, errs []error) {
	if msqBids.Dsa != nil {
		bb, err := jsonutil.Marshal(msqBids.Dsa)
		if err != nil {
			errs = append(errs, err)
		}
		if len(bb) > 0 {
			var dsa openrtb_ext.ExtBidDSA
			if err = jsonutil.Unmarshal(bb, &dsa); err != nil {
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

// errorWriter: Returns a Custom error message.
func errorWriter(referer string, err error, isEmpty bool) error {
	if isEmpty {
		return fmt.Errorf("%s: is empty.", referer)
	}
	return fmt.Errorf("%s: %s", referer, err.Error())
}
