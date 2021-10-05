package adnuntius

import (
	"testing"
	"time"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdnuntius, config.Adapter{
		Endpoint: "http://whatever.url"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}
	assertTzo(t, bidder)
	replaceTzoWithKnownTime(bidder)

	adapterstest.RunJSONBidderTest(t, "adnuntiustest", bidder)
}

func assertTzo(t *testing.T, bidder adapters.Bidder) {
	bidderAdnuntius, _ := bidder.(*adapter)
	assert.NotNil(t, bidderAdnuntius.tzo)
}

func replaceTzoWithKnownTime(bidder adapters.Bidder) {
	bidderAdnuntius, _ := bidder.(*adapter)
	bidderAdnuntius.tzo = knownTzo(time.Date(2016, 1, 1, 12, 30, 15, 0, time.UTC))
}
