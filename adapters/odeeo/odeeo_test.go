package odeeo

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

const testEndpoint = "https://ads.exchange.odeeo.io/v1/bidrequest/prebids2s"

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderOdeeo, config.Adapter{Endpoint: testEndpoint}, config.Server{})
	require.NoError(t, buildErr)

	adapterstest.RunJSONBidderTest(t, "odeeotest", bidder)
}

func TestMakeBidsBidMeta(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderOdeeo, config.Adapter{Endpoint: testEndpoint}, config.Server{})
	require.NoError(t, buildErr)

	request := &openrtb2.BidRequest{
		ID:  "req-1",
		Imp: []openrtb2.Imp{{ID: "imp-1"}, {ID: "imp-2"}, {ID: "imp-3"}},
	}

	responseBody := []byte(`{
		"id": "req-1",
		"cur": "USD",
		"seatbid": [{
			"seat": "odeeo",
			"bid": [
				{
					"id": "bid-1",
					"impid": "imp-1",
					"price": 2.5,
					"adm": "<VAST version=\"4.0\"></VAST>",
					"adomain": ["example-advertiser.com"],
					"cat": ["IAB1-1", "IAB1-6", "IAB1-7"],
					"mtype": 3
				},
				{
					"id": "bid-2",
					"impid": "imp-2",
					"price": 1.0,
					"adm": "<div></div>",
					"mtype": 1
				},
				{
					"id": "bid-3",
					"impid": "imp-3",
					"price": 1.5,
					"adm": "<div></div>",
					"cat": ["IAB1-1"],
					"mtype": 1
				}
			]
		}]
	}`)

	bidderResponse, errs := bidder.MakeBids(request, &adapters.RequestData{}, &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       responseBody,
	})
	require.Empty(t, errs)
	require.NotNil(t, bidderResponse)
	assert.Equal(t, "USD", bidderResponse.Currency)
	require.Len(t, bidderResponse.Bids, 3)

	fullMeta := bidderResponse.Bids[0]
	assert.Equal(t, openrtb_ext.BidTypeAudio, fullMeta.BidType)
	assert.Equal(t, openrtb_ext.BidderName(""), fullMeta.Seat, "TypedBid.Seat must stay at the default (adapter name)")
	assert.Nil(t, fullMeta.BidVideo, "audio bids carry no video ext")
	assert.Equal(t, &openrtb_ext.ExtBidPrebidMeta{
		AdvertiserDomains:    []string{"example-advertiser.com"},
		PrimaryCategoryID:    "IAB1-1",
		SecondaryCategoryIDs: []string{"IAB1-6", "IAB1-7"},
		Seat:                 "odeeo",
		MediaType:            "audio",
	}, fullMeta.BidMeta)

	minimalMeta := bidderResponse.Bids[1]
	assert.Equal(t, openrtb_ext.BidTypeBanner, minimalMeta.BidType)
	assert.Nil(t, minimalMeta.BidVideo, "banner bids carry no video ext")
	assert.Equal(t, &openrtb_ext.ExtBidPrebidMeta{
		Seat:      "odeeo",
		MediaType: "banner",
	}, minimalMeta.BidMeta, "adomain/cat absent: only mediaType and seat are set")

	singleCategoryMeta := bidderResponse.Bids[2]
	assert.Equal(t, &openrtb_ext.ExtBidPrebidMeta{
		PrimaryCategoryID: "IAB1-1",
		Seat:              "odeeo",
		MediaType:         "banner",
	}, singleCategoryMeta.BidMeta, "single cat: primary set, no secondary")
}
