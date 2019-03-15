package onemobile

import (
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"testing"
)

/**
 * Verify adapter names are setup correctly.
 */
func TestOneMobileAdapterNames(t *testing.T) {
	adapter := NewOneMobileAdapter(adapters.DefaultHTTPAdapterConfig, "http://localhost/bid")
	adapterstest.VerifyStringValue(adapter.Name(), "onemobile", t)
}

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "onemobiletest", new(OneMobileAdapter))
}
