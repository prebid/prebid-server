package openx

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderOpenx, config.Adapter{
		Endpoint: "http://rtb.openx.net/prebid"})

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

func assertCurrencyInBidResponse(t *testing.T, expectedCurrency string, currency *string) {
	bidder, buildErr := Builder(openrtb_ext.BidderOpenx, config.Adapter{
		Endpoint: "http://rtb.openx.net/prebid"})

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
