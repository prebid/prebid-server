package nextmillennium

import (
	"testing"

	"github.com/prebid/prebid-server/v2/adapters/adapterstest"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderNextMillennium, config.Adapter{
		Endpoint: "https://pbs.nextmillmedia.com/openrtb2/auction"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "nextmillenniumtest", bidder)
}
func TestWithExtraInfo(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderNextMillennium, config.Adapter{
		Endpoint:         "https://pbs.nextmillmedia.com/openrtb2/auction",
		ExtraAdapterInfo: "{\"nmmFlags\":[\"flag1\",\"flag2\"]}",
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}
	bidderNextMillennium, _ := bidder.(*adapter)
	assert.Equal(t, bidderNextMillennium.nmmFlags, []string{"flag1", "flag2"})
}
