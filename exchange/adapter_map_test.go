package exchange

import (
	"testing"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestNewAdapterMap(t *testing.T) {
	adapterMap := newAdapterMap(nil, &config.Configuration{}, adapters.ParseBidderInfos("../static/bidder-info", openrtb_ext.BidderList()))
	for _, bidderName := range openrtb_ext.BidderMap {
		if bidder, ok := adapterMap[bidderName]; bidder == nil || !ok {
			t.Errorf("adapterMap missing expected Bidder: %s", string(bidderName))
		}
	}
	for bidder := range adapterMap {
		if !inList(openrtb_ext.BidderList(), bidder) {
			t.Errorf("adapterMap includes Bidder \"%s\" which is not found in the BidderList", string(bidder))
		}
	}
}

func TestNewAdapterMapDisabledAdapters(t *testing.T) {
	bidderList := openrtb_ext.BidderList()
	disabledList := []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus, openrtb_ext.BidderBrightroll, openrtb_ext.BidderOpenx}
	for _, d := range disabledList {
		for i, b := range bidderList {
			if b == d {
				bidderList = append(bidderList[:i], bidderList[i+1:]...)
			}
		}
	}
	adapterMap := newAdapterMap(nil, &config.Configuration{}, adapters.ParseBidderInfos("../static/bidder-info", bidderList))
	for _, bidderName := range openrtb_ext.BidderMap {
		if bidder, ok := adapterMap[bidderName]; bidder == nil || !ok {
			if inList(bidderList, bidderName) {
				t.Errorf("adapterMap missing expected Bidder: %s", string(bidderName))
			}
		} else {
			if inList(disabledList, bidderName) {
				t.Errorf("adapterMap contains disabled Bidder: %s", string(bidderName))
			}
		}
	}
}

func inList(list []openrtb_ext.BidderName, name openrtb_ext.BidderName) bool {
	for _, v := range list {
		if v == name {
			return true
		}
	}
	return false
}
