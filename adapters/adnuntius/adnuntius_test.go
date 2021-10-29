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
	replaceRealTimeWithKnownTime(bidder)

	adapterstest.RunJSONBidderTest(t, "adnuntiustest", bidder)
}

func assertTzo(t *testing.T, bidder adapters.Bidder) {
	bidderAdnuntius, _ := bidder.(*adapter)
	assert.NotNil(t, bidderAdnuntius.time)
}

// FakeTime implements the Time interface
type FakeTime struct {
	time time.Time
}

func (ft *FakeTime) Now() time.Time {
	return ft.time
}

func replaceRealTimeWithKnownTime(bidder adapters.Bidder) {
	bidderAdnuntius, _ := bidder.(*adapter)
	bidderAdnuntius.time = &FakeTime{
		time: time.Date(2016, 1, 1, 12, 30, 15, 0, time.UTC),
	}
}
