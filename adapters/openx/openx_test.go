package openx

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "openxtest", NewOpenxBidder("http://rtb.openx.net/prebid"))
}

func TestResponseWithCurrencies(t *testing.T) {
	assertCurrencyInBidResponse(t, "USD", nil)

	currency := "USD"
	assertCurrencyInBidResponse(t, "USD", &currency)

	currency = "EUR"
	assertCurrencyInBidResponse(t, "EUR", &currency)
}

func assertCurrencyInBidResponse(t *testing.T, expectedCurrency string, currency *string) {

	bidder := NewOpenxBidder("http://rtb.openx.net/prebid")
	prebidRequest := &openrtb.BidRequest{
		Imp: []openrtb.Imp{},
	}
	mockedBidResponse := &openrtb.BidResponse{}
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
