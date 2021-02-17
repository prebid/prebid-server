package verizonmedia

import (
	"testing"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adapterstest"
	"github.com/PubMatic-OpenWrap/prebid-server/config"
	"github.com/PubMatic-OpenWrap/prebid-server/openrtb_ext"
	"github.com/influxdata/influxdb/pkg/testing/assert"
)

func TestVerizonMediaBidderEndpointConfig(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderVerizonMedia, config.Adapter{
		Endpoint: "http://localhost/bid",
	})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	bidderVerizonMedia := bidder.(*VerizonMediaAdapter)

	assert.Equal(t, "http://localhost/bid", bidderVerizonMedia.URI)
}

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderVerizonMedia, config.Adapter{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "verizonmediatest", bidder)
}
