package adnuntius

import (
	"testing"
	"time"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdnuntius, config.Adapter{
		Endpoint:         "http://whatever.url",
		ExtraAdapterInfo: "http://gdpr.url",
	},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}
	assertTzo(t, bidder)
	AssignDefaultValues(bidder)

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

func AssignDefaultValues(bidder adapters.Bidder) {
	bidderAdnuntius, _ := bidder.(*adapter)
	bidderAdnuntius.time = &FakeTime{
		time: time.Date(2016, 1, 1, 12, 30, 15, 0, time.UTC),
	}
}
