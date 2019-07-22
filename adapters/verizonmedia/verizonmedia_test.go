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

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "verizonmediatest", new(VerizonMediaAdapter))
}
