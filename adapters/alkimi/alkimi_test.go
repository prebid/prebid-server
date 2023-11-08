package alkimi

import (
	"encoding/json"
	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"testing"

	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

const (
	alkimiTestEndpoint = "https://exchange.alkimi-onboarding.com/server/bid"
)

func TestEndpointEmpty(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderAlkimi, config.Adapter{
		Endpoint: ""}, config.Server{ExternalUrl: alkimiTestEndpoint, GvlID: 1, DataCenter: "2"})
	assert.Error(t, buildErr)
}

func TestEndpointMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderAlkimi, config.Adapter{
		Endpoint: " http://leading.space.is.invalid"}, config.Server{ExternalUrl: alkimiTestEndpoint, GvlID: 1, DataCenter: "2"})
	assert.Error(t, buildErr)
}

func TestBuilder(t *testing.T) {
	bidder, buildErr := buildBidder()
	if buildErr != nil {
		t.Fatalf("Failed to build bidder: %v", buildErr)
	}
	assert.NotNil(t, bidder)
}

func TestMakeRequests(t *testing.T) {
	// given
	bidder, _ := buildBidder()
	extImpAlkimi, _ := json.Marshal(extObj{AlkimiBidderExt: openrtb_ext.ExtImpAlkimi{BidFloor: 5, Instl: 1, Exp: 2}})
	bidRequest := openrtb2.BidRequest{
		Imp: []openrtb2.Imp{
			{
				BidFloor:    6,
				BidFloorCur: "",
				Ext:         extImpAlkimi,
			},
			{
				BidFloor:    -1,
				BidFloorCur: "USD",
				Ext:         extImpAlkimi,
			},
			{
				BidFloor:    10,
				BidFloorCur: "USD",
				Ext:         extImpAlkimi,
			},
		},
	}
	// when
	requestData, _ := bidder.MakeRequests(&bidRequest, nil)
	request := requestData[0]
	var updatedRequest openrtb2.BidRequest
	errUnmarshal := json.Unmarshal(request.Body, &updatedRequest)
	updatedImps := updatedRequest.Imp
	// then
	assert.Len(t, requestData, 1)
	if errUnmarshal != nil {
		t.Fatal("Corrupted updated request")
	}
	assert.Len(t, updatedImps, 3)

	assert.Equal(t, 5.0, updatedImps[0].BidFloor)
	assert.Equal(t, int8(1), updatedImps[0].Instl)
	assert.Equal(t, int64(2), updatedImps[0].Exp)

	assert.Equal(t, 5.0, updatedImps[1].BidFloor)
	assert.Equal(t, int8(1), updatedImps[1].Instl)
	assert.Equal(t, int64(2), updatedImps[1].Exp)

	assert.Equal(t, 10.0, updatedImps[2].BidFloor)
	assert.Equal(t, int8(1), updatedImps[2].Instl)
	assert.Equal(t, int64(2), updatedImps[2].Exp)
}

func TestMakeBidsShouldReturnErrorIfResponseBodyCouldNotBeParsed(t *testing.T) {
	// given
	bidder, _ := buildBidder()
	bid := openrtb2.Bid{
		ImpID: "impId-1",
		AdM:   "adm:${AUCTION_PRICE}",
		NURL:  "nurl:${AUCTION_PRICE}",
		Price: 1,
	}
	sb := openrtb2.SeatBid{Bid: []openrtb2.Bid{bid}}
	resp := openrtb2.BidResponse{SeatBid: []openrtb2.SeatBid{sb}}
	respJson, jsonErr := json.Marshal(resp)
	request := openrtb2.BidRequest{
		Imp: append(make([]openrtb2.Imp, 1), openrtb2.Imp{ID: "impId-1", Banner: &openrtb2.Banner{}}),
	}
	// when
	bids, errs := bidder.MakeBids(&request, nil, &adapters.ResponseData{
		StatusCode: 200,
		Body:       respJson,
	})
	// then
	if jsonErr != nil {
		t.Fatalf("Failed to serialize test bid %v: %v", bid, jsonErr)
	}
	if len(errs) > 0 {
		t.Fatalf("Failed to make bids: %v", errs)
	}
	assert.Len(t, bids.Bids, 1)
	assert.Equal(t, "nurl:1", bids.Bids[0].Bid.NURL)
	assert.Equal(t, "adm:1", bids.Bids[0].Bid.AdM)
}

func TestMakeBidsShouldReturnEmptyListIfBidResponseIsNull(t *testing.T) {
	// given
	bidder, _ := buildBidder()
	// when
	bids, errs := bidder.MakeBids(&openrtb2.BidRequest{}, nil, &adapters.ResponseData{
		StatusCode: 204,
	})
	// then
	if len(errs) > 0 {
		t.Fatalf("Failed to make bids: %v", errs)
	}
	assert.Nil(t, bids)
}

func TestMakeBidsShouldReturnEmptyListIfBidResponseIsError(t *testing.T) {
	// given
	bidder, _ := buildBidder()
	// when
	bids, errs := bidder.MakeBids(&openrtb2.BidRequest{}, nil, &adapters.ResponseData{
		StatusCode: 500,
	})
	// then
	assert.Len(t, errs, 1)
	assert.Nil(t, bids)
}

func TestMakeBidsShouldReturnBidWithResolvedMacros(t *testing.T) {
	// given
	bidder, _ := buildBidder()
	bid := openrtb2.Bid{
		ImpID: "impId-1",
		AdM:   "adm:${AUCTION_PRICE}",
		NURL:  "nurl:${AUCTION_PRICE}",
		Price: 1,
	}
	seatBid := openrtb2.SeatBid{Bid: []openrtb2.Bid{bid}}
	resp := openrtb2.BidResponse{SeatBid: []openrtb2.SeatBid{seatBid}}
	respJson, jsonErr := json.Marshal(resp)

	request := openrtb2.BidRequest{
		Imp: append(make([]openrtb2.Imp, 1), openrtb2.Imp{ID: "impId-1", Banner: &openrtb2.Banner{}}),
	}
	// when
	bids, errs := bidder.MakeBids(&request, nil, &adapters.ResponseData{
		StatusCode: 200,
		Body:       respJson,
	})
	// then
	if jsonErr != nil {
		t.Fatalf("Failed to serialize test bid %v: %v", bid, jsonErr)
	}
	if len(errs) > 0 {
		t.Fatalf("Failed to make bids: %v", errs)
	}
	assert.Len(t, bids.Bids, 1)
	assert.Equal(t, "nurl:1", bids.Bids[0].Bid.NURL)
	assert.Equal(t, "adm:1", bids.Bids[0].Bid.AdM)
}

func buildBidder() (adapters.Bidder, error) {
	return Builder(
		openrtb_ext.BidderAlkimi,
		config.Adapter{Endpoint: alkimiTestEndpoint},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"},
	)
}
