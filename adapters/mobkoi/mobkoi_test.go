package mobkoi

import (
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderMobkoi, config.Adapter{
		Endpoint: "http://dev.mobkoi.com/bid"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "mobkoitest", bidder)
}

func TestSeatLeftEmptyForAliasSupport(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderMobkoi, config.Adapter{
		Endpoint: "http://dev.mobkoi.com/bid"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	require.NoError(t, buildErr)

	body := []byte(`{"id":"resp","cur":"USD","seatbid":[{"seat":"mobkoi","bid":[{"id":"1","impid":"imp1","price":1.5,"crid":"20"}]}]}`)
	resp, errs := bidder.MakeBids(
		&openrtb2.BidRequest{},
		&adapters.RequestData{},
		&adapters.ResponseData{StatusCode: http.StatusOK, Body: body},
	)

	require.Empty(t, errs)
	require.NotNil(t, resp)
	require.Len(t, resp.Bids, 1)
	assert.Empty(t, resp.Bids[0].Seat, "adapter must leave TypedBid.Seat empty so aliases pass the bidder-code-spoofing guard")
}
