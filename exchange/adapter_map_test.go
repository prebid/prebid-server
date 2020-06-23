package exchange

import (
	"strings"
	"testing"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	metricsConfig "github.com/prebid/prebid-server/pbsmetrics/config"
)

func TestNewAdapterMap(t *testing.T) {
	cfg := &config.Configuration{Adapters: blankAdapterConfig(openrtb_ext.BidderList())}
	adapterMap := newAdapterMap(nil, cfg, adapters.ParseBidderInfos(cfg.Adapters, "../static/bidder-info", openrtb_ext.BidderList()), &metricsConfig.DummyMetricsEngine{})
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
	cfgAdapters := blankAdapterConfig(openrtb_ext.BidderList())
	disabledList := []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus, openrtb_ext.BidderBrightroll, openrtb_ext.BidderOpenx}
	for _, d := range disabledList {
		tmp := cfgAdapters[string(d)]
		tmp.Disabled = true
		cfgAdapters[string(d)] = tmp
		for i, b := range bidderList {
			if b == d {
				bidderList = append(bidderList[:i], bidderList[i+1:]...)
			}
		}
	}
	adapterMap := newAdapterMap(nil, &config.Configuration{Adapters: cfgAdapters}, adapters.ParseBidderInfos(cfgAdapters, "../static/bidder-info", bidderList), &metricsConfig.DummyMetricsEngine{})
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

func blankAdapterConfig(bidderList []openrtb_ext.BidderName) map[string]config.Adapter {
	adapters := make(map[string]config.Adapter)
	for _, b := range bidderList {
		adapters[strings.ToLower(string(b))] = config.Adapter{}
	}
	return adapters
}
