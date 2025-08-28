package openx

import (
	"encoding/json"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
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

func TestUpdateBidExtMeta(t *testing.T) {
	buyerId := "123"
	dspId := "456"
	bradId := "789"

	testCases := []struct {
		ext             *oxBidExt
		expectedBuyerId int
		expectedDspId   int
		expectedBrandId int
	}{
		{
			&oxBidExt{BuyerId: &buyerId, DspId: &dspId, BrandId: &bradId},
			123,
			456,
			789,
		},
		{
			&oxBidExt{BuyerId: &buyerId, DspId: &dspId},
			123,
			456,
			0,
		},
		{
			&oxBidExt{BuyerId: &buyerId, BrandId: &bradId},
			123,
			0,
			789,
		},
		{
			&oxBidExt{DspId: &dspId, BrandId: &bradId},
			0,
			456,
			789,
		},
		{
			&oxBidExt{BuyerId: &buyerId},
			123,
			0,
			0,
		},
		{
			&oxBidExt{DspId: &dspId},
			0,
			456,
			0,
		},
		{
			&oxBidExt{BrandId: &bradId},
			0,
			0,
			789,
		},
	}

	for _, testCase := range testCases {
		marshaledExt, _ := json.Marshal(testCase.ext)
		bid := &openrtb2.Bid{Ext: marshaledExt}

		bid.Ext = updateBidExtMeta(bid)

		var updatedExt *openrtb_ext.ExtBidPrebid
		_ = jsonutil.Unmarshal(bid.Ext, &updatedExt)

		assert.Equal(t, testCase.expectedBuyerId, updatedExt.Meta.NetworkID)
		assert.Equal(t, testCase.expectedDspId, updatedExt.Meta.AdvertiserID)
		assert.Equal(t, testCase.expectedBrandId, updatedExt.Meta.BrandID)
	}
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
