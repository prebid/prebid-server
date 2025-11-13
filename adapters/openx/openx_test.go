package openx

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderOpenx, config.Adapter{
		Endpoint: "http://rtb.openx.net/prebid"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "openxtest", bidder)
}

func TestResponseWithCurrencies(t *testing.T) {
	assertCurrencyInBidResponse(t, "USD", nil)

	currency := "USD"
	assertCurrencyInBidResponse(t, "USD", &currency)

	currency = "EUR"
	assertCurrencyInBidResponse(t, "EUR", &currency)
}

func TestOpenxAdapter_GetBidMeta(t *testing.T) {
	testCases := []struct {
		bid          *openrtb2.Bid
		expectedMeta *openrtb_ext.ExtBidPrebidMeta
	}{
		{
			&openrtb2.Bid{Ext: json.RawMessage(`malformed`)},
			nil,
		},
		{
			&openrtb2.Bid{Ext: json.RawMessage(`{}`)},
			nil,
		},
		{
			&openrtb2.Bid{Ext: json.RawMessage(`{"dsp_id":"456","brand_id":"789","buyer_id":"123"}`)},
			&openrtb_ext.ExtBidPrebidMeta{AdvertiserID: 123, NetworkID: 456, BrandID: 789},
		},
		{
			&openrtb2.Bid{Ext: json.RawMessage(`{"dsp_id":"456","brand_id":"malformed","buyer_id":"123"}`)},
			nil,
		},
		{
			&openrtb2.Bid{Ext: json.RawMessage(`{"dsp_id":"456","brand_id":"789","buyer_id":"123-456"}`)},
			&openrtb_ext.ExtBidPrebidMeta{AdvertiserID: 0, NetworkID: 456, BrandID: 789},
		},
		{
			&openrtb2.Bid{Ext: json.RawMessage(`{"buyer_id":"123","dsp_id":"456"}`)},
			&openrtb_ext.ExtBidPrebidMeta{AdvertiserID: 123, NetworkID: 456, BrandID: 0},
		},
		{
			&openrtb2.Bid{Ext: json.RawMessage(`{"buyer_id":"123","brand_id":"789"}`)},
			&openrtb_ext.ExtBidPrebidMeta{AdvertiserID: 123, NetworkID: 0, BrandID: 789},
		},
		{
			&openrtb2.Bid{Ext: json.RawMessage(`{"dsp_id":"456","brand_id":"789"}`)},
			&openrtb_ext.ExtBidPrebidMeta{AdvertiserID: 0, NetworkID: 456, BrandID: 789},
		},
		{
			&openrtb2.Bid{Ext: json.RawMessage(`{"buyer_id":"123"}`)},
			&openrtb_ext.ExtBidPrebidMeta{AdvertiserID: 123, NetworkID: 0, BrandID: 0},
		},
		{
			&openrtb2.Bid{Ext: json.RawMessage(`{"dsp_id":"456"}`)},
			&openrtb_ext.ExtBidPrebidMeta{AdvertiserID: 0, NetworkID: 456, BrandID: 0},
		},
		{
			&openrtb2.Bid{Ext: json.RawMessage(`{"brand_id":"789"}`)},
			&openrtb_ext.ExtBidPrebidMeta{AdvertiserID: 0, NetworkID: 0, BrandID: 789},
		},
		{
			&openrtb2.Bid{Ext: json.RawMessage(`{"buyer_id":"123","dsp_id":"456"}`)},
			&openrtb_ext.ExtBidPrebidMeta{AdvertiserID: 123, NetworkID: 456, BrandID: 0},
		},
		{
			&openrtb2.Bid{Ext: json.RawMessage(`{"buyer_id":"malformed","dsp_id":"malformed","brand_id":"malformed"}`)},
			nil,
		},
	}

	for _, testCase := range testCases {
		updatedMeta := getBidMeta(testCase.bid)
		assert.Equal(t, testCase.expectedMeta, updatedMeta)
	}
}

func TestOpenxAdapter_MakeBids(t *testing.T) {
	responseBody := `{"id":"test-request-id","seatbid":[{"seat":"openx","bid":[{"id":"all-buyer-ext","impid":"all-buyer-ext-imp-id","price":0.5,"adm":"some-test-ad","crid":"crid_10","ext":{"dsp_id":"123","brand_id":"456","buyer_id":"789"},"h":90,"w":728,"mtype":1},{"id":"only-dspId","impid":"only-dspId-imp-id","price":0.6,"adm":"some-test-ad","crid":"crid_11","ext":{"dsp_id":"321"},"h":90,"w":728,"mtype":1}]}],"cur":"USD"}`
	response := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(responseBody),
	}
	adapter := &OpenxAdapter{bidderName: "", endpoint: ""}
	bids, _ := adapter.MakeBids(&openrtb2.BidRequest{}, &adapters.RequestData{}, response)
	expectedBidderResponse := adapters.BidderResponse{
		Currency: "USD",
		Bids: []*adapters.TypedBid{
			{
				Bid: &openrtb2.Bid{
					ID:    "all-buyer-ext",
					ImpID: "all-buyer-ext-imp-id",
					Price: 0.5,
					AdM:   "some-test-ad",
					CrID:  "crid_10",
					W:     728,
					H:     90,
					MType: 1,
					Ext:   json.RawMessage(`{"dsp_id":"123","brand_id":"456","buyer_id":"789"}`),
				},
				BidMeta:  &openrtb_ext.ExtBidPrebidMeta{NetworkID: 123, BrandID: 456, AdvertiserID: 789},
				BidType:  "banner",
				BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 0, PrimaryCategory: ""},
			},
			{
				Bid: &openrtb2.Bid{
					ID:    "only-dspId",
					ImpID: "only-dspId-imp-id",
					Price: 0.6,
					AdM:   "some-test-ad",
					CrID:  "crid_11",
					W:     728,
					H:     90,
					MType: 1,
					Ext:   json.RawMessage(`{"dsp_id":"321"}`),
				},
				BidMeta:  &openrtb_ext.ExtBidPrebidMeta{NetworkID: 321},
				BidType:  "banner",
				BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 0, PrimaryCategory: ""},
			},
		},
	}
	assert.Equal(t, expectedBidderResponse, *bids)
}

func assertCurrencyInBidResponse(t *testing.T, expectedCurrency string, currency *string) {
	bidder, buildErr := Builder(openrtb_ext.BidderOpenx, config.Adapter{
		Endpoint: "http://rtb.openx.net/prebid"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	prebidRequest := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{},
	}
	mockedBidResponse := &openrtb2.BidResponse{}
	if currency != nil {
		mockedBidResponse.Cur = *currency
	}
	body, _ := json.Marshal(mockedBidResponse)
	responseData := &adapters.ResponseData{
		StatusCode: 200,
		Body:       body,
	}
	bidResponse, errs := bidder.MakeBids(prebidRequest, nil, responseData)

	if errs != nil {
		t.Fatalf("Failed to make bids %v", errs)
	}
	assert.Equal(t, expectedCurrency, bidResponse.Currency)
}
