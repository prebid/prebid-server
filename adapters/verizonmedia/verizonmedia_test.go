package verizonmedia

import (
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"testing"
)

/**
 * Verify adapter names are setup correctly.
 */
func TestVerizonMediaAdapterNames(t *testing.T) {
	adapter := NewVerizonMediaAdapter(adapters.DefaultHTTPAdapterConfig, "http://localhost/bid")
	adapterstest.VerifyStringValue(adapter.Name(), "verizonmedia", t)
}

/**
 * Verify adapter SkipNoCookie is correct.
 */
func TestVerizonMediaAdapterSkipNoCookieFlag(t *testing.T) {
	adapter := NewVerizonMediaAdapter(adapters.DefaultHTTPAdapterConfig, "http://localhost/bid")
	adapterstest.VerifyBoolValue(adapter.SkipNoCookies(), false, t)
}

/**
 * Verify bidder is created with the provided endpoint.
 */
func TestVerizonMediaBidderEndpointConfig(t *testing.T) {
	bidder := NewVerizonMediaBidder(nil, "http://localhost/bid")
	adapterstest.VerifyStringValue(bidder.URI, "http://localhost/bid", t)
}

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "verizonmediatest", new(VerizonMediaAdapter))
}
