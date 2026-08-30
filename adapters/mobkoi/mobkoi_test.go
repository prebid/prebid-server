package mobkoi

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
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

// TestMakeBidsDoesNotHardcodeSeat guards against re-introducing a hardcoded
// "mobkoi" bid seat. Aliases of the mobkoi bidder (e.g. a publisher-specific
// alias configured in request.ext.prebid.aliases) rely on TypedBid.Seat being
// left empty so the exchange labels the bid with the requested bidder code
// (see exchange/bidder.go, which only overrides the alias-derived bidder name
// when Seat is non-empty) instead of always reporting it under the core
// "mobkoi" seat.
func TestMakeBidsDoesNotHardcodeSeat(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderMobkoi, config.Adapter{
		Endpoint: "http://dev.mobkoi.com/bid"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	response := `{
		"id": "resp-id",
		"cur": "USD",
		"seatbid": [{
			"seat": "mobkoi",
			"type": "banner",
			"bid": [{"id": "bid-1", "impid": "imp-1", "price": 1.5}]
		}]
	}`

	bidderResponse, errs := bidder.MakeBids(
		&openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "imp-1"}}},
		&adapters.RequestData{},
		&adapters.ResponseData{StatusCode: 200, Body: []byte(response)},
	)

	assert.Empty(t, errs)
	if assert.Len(t, bidderResponse.Bids, 1) {
		assert.Empty(t, bidderResponse.Bids[0].Seat, "Seat must be left empty so aliases are labeled with the requested bidder code")
	}
}
