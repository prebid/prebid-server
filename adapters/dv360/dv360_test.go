//go:build !integration
// +build !integration

package dv360

import (
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(test *testing.T) {
	adapterstest.RunJSONBidderTest(
		test,
		"dv360test",
		NewDV360Bidder(
			http.DefaultClient,
			"https://bid.g.doubleclick.net/xbbe/bid/tapjoy",
		),
	)
}
