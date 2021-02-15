package adyoulike

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/mxmCherry/openrtb"
	"github.com/stretchr/testify/assert"
)

const testsBidderEndpoint = "https://localhost/bid/4"

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdyoulike, config.Adapter{
		Endpoint: testsBidderEndpoint})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "adyouliketest", bidder)
}

func TestMakeRequestNoImp(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdyoulike, config.Adapter{
		Endpoint: testsBidderEndpoint})

	assert.Nil(t, buildErr, "buildErr must be nil")

	var reqInfo adapters.ExtraRequestInfo
	var req openrtb.BidRequest
	req.ID = "test_id"

	bids, errs := bidder.MakeRequests(&req, &reqInfo)

	assert.EqualError(t, errs[0], "No impression in the bid request")
	assert.Len(t, bids, 0)
}

func TestMakeRequestInvalidParams(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdyoulike, config.Adapter{
		Endpoint: testsBidderEndpoint})

	assert.Nil(t, buildErr, "buildErr must be nil")

	var reqInfo adapters.ExtraRequestInfo
	var req openrtb.BidRequest
	req.ID = "test_id"

	reqExt := `{"prebid":{}}`
	impExt := `{"bidder":{"placementId":123}}`
	req.Ext = []byte(reqExt)

	req.Imp = append(req.Imp, openrtb.Imp{ID: "1_0", Ext: []byte(impExt)})

	bids, errs := bidder.MakeRequests(&req, &reqInfo)

	assert.EqualError(t, errs[0], "Key path not found")
	assert.Len(t, bids, 0)
}

func TestMakeRequestTagId(t *testing.T) {

	bidder, buildErr := Builder(openrtb_ext.BidderAdyoulike, config.Adapter{
		Endpoint: testsBidderEndpoint})

	assert.Nil(t, buildErr, "buildErr must be nil")

	var reqInfo adapters.ExtraRequestInfo
	var req openrtb.BidRequest
	req.ID = "test_id"

	reqExt := `{"prebid":{}}`
	impExt1 := `{"bidder":{"placement":"placementid1"}}`
	impExt2 := `{"bidder":{"placement":"placementid2"}}`
	req.Ext = []byte(reqExt)

	req.Imp = append(req.Imp, openrtb.Imp{ID: "1_0", Ext: []byte(impExt1)})
	req.Imp = append(req.Imp, openrtb.Imp{ID: "1_1", Ext: []byte(impExt2)})

	requests, errs := bidder.MakeRequests(&req, &reqInfo)

	assert.Len(t, errs, 0)

	var request *openrtb.BidRequest
	json.Unmarshal(requests[0].Body, &request)

	for _, imp := range request.Imp {

		assert.True(t, imp.ID == "1_0" || imp.ID == "1_1")

		if imp.ID == "1_0" {
			assert.Equal(t, imp.TagID, "placementid1")
		} else if imp.ID == "1_1" {
			assert.Equal(t, imp.TagID, "placementid2")
		}
	}
}

func TestOpenRTBStandardResponse(t *testing.T) {

	responseBody, _ := json.Marshal(openrtb.BidResponse{
		ID: "123",
		SeatBid: []openrtb.SeatBid{
			{
				Bid: []openrtb.Bid{
					{
						ID:    "12340",
						ImpID: "10",
						Price: 300.00,
						NURL:  "http://example.com/winnoticeurl0",
						AdM:   "%3C%3Fxml%20version%3D%221.0%22%20encod%2Fhtml%3E",
					},
					{
						ID:    "12341",
						ImpID: "11",
						Price: 301.00,
						NURL:  "http://example.com/winnoticeurl1",
						AdM:   "%3C%3Fxml%20version%3D%221.0%22%20encod%2FVAST%3E",
					},
					{
						ID:    "12342",
						ImpID: "12",
						Price: 302.00,
						NURL:  "http://example.com/winnoticeurl2",
						AdM:   "{'json':'response','for':'native'}",
					},
				},
			},
		},
	})

	request := openrtb.BidRequest{
		Imp: []openrtb.Imp{
			{
				ID:     "10",
				Banner: &openrtb.Banner{},
			},
			{
				ID:    "11",
				Video: &openrtb.Video{},
			},
			{
				ID:     "12",
				Native: &openrtb.Native{},
			},
		},
	}

	expectedResponse := adapters.BidderResponse{
		Currency: "",
		Bids: []*adapters.TypedBid{
			{
				Bid: &openrtb.Bid{
					ID:    "12340",
					ImpID: "10",
					Price: 300,
					NURL:  "http://example.com/winnoticeurl0",
					AdM:   "%3C%3Fxml%20version%3D%221.0%22%20encod%2Fhtml%3E",
				},
				BidType: "banner",
			},
			{
				Bid: &openrtb.Bid{
					ID:    "12341",
					ImpID: "11",
					Price: 301,
					NURL:  "http://example.com/winnoticeurl1",
					AdM:   "%3C%3Fxml%20version%3D%221.0%22%20encod%2FVAST%3E",
				},
				BidType: "video",
			},
			{
				Bid: &openrtb.Bid{
					ID:    "12342",
					ImpID: "12",
					Price: 302,
					NURL:  "http://example.com/winnoticeurl2",
					AdM:   "{'json':'response','for':'native'}",
				},
				BidType: "native",
			},
		},
	}

	bidder, buildErr := Builder(openrtb_ext.BidderAdyoulike, config.Adapter{
		Endpoint: testsBidderEndpoint,
	})

	assert.Nil(t, buildErr, "buildErr must be nil")

	httpResponse := &adapters.ResponseData{StatusCode: http.StatusOK, Body: responseBody}
	bidResponse, errs := bidder.MakeBids(&request, nil, httpResponse)

	if len(bidResponse.Bids) != 3 {
		t.Fatalf("Expected 3 bids. Got %d", len(bidResponse.Bids))
	}
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors. Got %d", len(errs))
	}

	for i, typedBid := range bidResponse.Bids {

		expected := expectedResponse.Bids[i].Bid

		assert.Equal(t, expected.ID, typedBid.Bid.ID, "Incorrect Bid.id")
		assert.Equal(t, expected.ImpID, typedBid.Bid.ImpID, "Incorrect Bid.impid")
		assert.Equal(t, expected.Price, typedBid.Bid.Price, "Incorrect Bid.price")
		assert.Equal(t, expected.NURL, typedBid.Bid.NURL, "Incorrect Bid.nurl")
		assert.Equal(t, expected.AdM, typedBid.Bid.AdM, "Incorrect Bid.adm")

		assert.Equal(t, expectedResponse.Bids[i].BidType, typedBid.BidType, "Incorrect BidType")
	}

}

func TestOpenRTBSurpriseResponse(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdyoulike, config.Adapter{
		Endpoint: testsBidderEndpoint})

	assert.Nil(t, buildErr, "buildErr must be nil")

	bidResponse, errs := bidder.MakeBids(nil, nil,
		&adapters.ResponseData{StatusCode: http.StatusNoContent, Body: []byte("")})
	if bidResponse != nil && errs != nil {
		t.Fatalf("Expected no bids and no errors. Got %d bids and %d", len(bidResponse.Bids), len(errs))
	}

	bidResponse, errs = bidder.MakeBids(nil, nil,
		&adapters.ResponseData{StatusCode: http.StatusServiceUnavailable, Body: []byte("")})
	if bidResponse != nil || len(errs) != 1 {
		t.Fatalf("Expected one error and no bids. Got %d bids and %d", len(bidResponse.Bids), len(errs))
	}

	bidResponse, errs = bidder.MakeBids(nil, nil,
		&adapters.ResponseData{StatusCode: http.StatusOK, Body: []byte("{:'not-valid-json'}")})
	if bidResponse != nil || len(errs) != 1 {
		t.Fatalf("Expected one error and no bids. Got %d bids and %d", len(bidResponse.Bids), len(errs))
	}
}
