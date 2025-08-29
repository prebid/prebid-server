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

func TestGetBidMeta(t *testing.T) {
	buyerId := "123"
	dspId := "456"
	bradId := "789"

	testCases := []struct {
		ext          *oxBidExt
		expectedMeta *openrtb_ext.ExtBidPrebidMeta
	}{
		{
			&oxBidExt{BuyerId: &buyerId, DspId: &dspId, BrandId: &bradId},
			&openrtb_ext.ExtBidPrebidMeta{AdvertiserID: 123, NetworkID: 456, BrandID: 789},
		},
		{
			&oxBidExt{BuyerId: &buyerId, DspId: &dspId},
			&openrtb_ext.ExtBidPrebidMeta{AdvertiserID: 123, NetworkID: 456, BrandID: 0},
		},
		{
			&oxBidExt{BuyerId: &buyerId, BrandId: &bradId},
			&openrtb_ext.ExtBidPrebidMeta{AdvertiserID: 123, NetworkID: 0, BrandID: 789},
		},
		{
			&oxBidExt{DspId: &dspId, BrandId: &bradId},
			&openrtb_ext.ExtBidPrebidMeta{AdvertiserID: 0, NetworkID: 456, BrandID: 789},
		},
		{
			&oxBidExt{BuyerId: &buyerId},
			&openrtb_ext.ExtBidPrebidMeta{AdvertiserID: 123, NetworkID: 0, BrandID: 0},
		},
		{
			&oxBidExt{DspId: &dspId},
			&openrtb_ext.ExtBidPrebidMeta{AdvertiserID: 0, NetworkID: 456, BrandID: 0},
		},
		{
			&oxBidExt{BrandId: &bradId},
			&openrtb_ext.ExtBidPrebidMeta{AdvertiserID: 0, NetworkID: 0, BrandID: 789},
		},
	}

	for _, testCase := range testCases {
		marshaledExt, _ := json.Marshal(testCase.ext)
		bid := &openrtb2.Bid{Ext: marshaledExt}
		updatedMeta := getBidMeta(bid)
		assert.Equal(t, testCase.expectedMeta, updatedMeta)
	}
}

func TestOpenxAdapter_MakeBids_BidsMeta(t *testing.T) {
	responseBody := `{"id":"test-request-id","seatbid":[{"seat":"openx","bid":[{"id":"all-buyer-ext","impid":"all-buyer-ext-imp-id","price":0.5,"adm":"some-test-ad","crid":"crid_10","ext":{"dsp_id":"123","brand_id":"456","buyer_id":"789"},"h":90,"w":728,"mtype":1},{"id":"only-dspId","impid":"only-dspId-imp-id","price":0.6,"adm":"some-test-ad","crid":"crid_11","ext":{"dsp_id":"321"},"h":90,"w":728,"mtype":1}]}],"cur":"USD"}`
	allBuyerMeta := &openrtb_ext.ExtBidPrebidMeta{NetworkID: 123, BrandID: 456, AdvertiserID: 789}
	onlyDspIdMeta := &openrtb_ext.ExtBidPrebidMeta{NetworkID: 321}
	response := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(responseBody),
	}
	adapter := &OpenxAdapter{bidderName: "", endpoint: ""}
	bids, _ := adapter.MakeBids(&openrtb2.BidRequest{}, &adapters.RequestData{}, response)
	assert.Equal(t, len(bids.Bids), 2)
	assert.Equal(t, bids.Bids[0].BidMeta, allBuyerMeta)
	assert.Equal(t, bids.Bids[1].BidMeta, onlyDspIdMeta)
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
