package ownadx

import (
	"testing"

	"github.com/prebid/prebid-server/v2/adapters/adapterstest"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderOwnAdx, config.Adapter{
		Endpoint: "https://pbs.prebid-ownadx.com/bidder/bid/{{.AccountID}}/{{.ZoneID}}?token={{.SourceId}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.NoError(t, buildErr)
	adapterstest.RunJSONBidderTest(t, "ownadxtest", bidder)
}
