package unicorn

import (
  "testing"

  "github.com/prebid/prebid-server/adapters/adapterstest"
  "github.com/prebid/prebid-server/config"
  "github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
  bidder, buildErr := Builder(openrtb_ext.BidderUnicorn, config.Adapter{
    Endpoint: "http://localhost:4000"})

  if buildErr != nil {
    t.Fatalf("Builder returned unexpected error %v", buildErr)
  }

  adapterstest.RunJSONBidderTest(t, "unicorntest", bidder)
}