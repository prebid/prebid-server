package gumgum

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/adapters/adapterstest"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderGumGum, config.Adapter{
		Endpoint: "https://g2.gumgum.com/providers/prbds2s/bid"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "gumgumtest", bidder)
}

func TestResponseWithCurrencies(t *testing.T) {
	// Test for USD currency
	assertCurrencyInBidResponse(t, "USD", "USD")

	// Test for EUR currency
	assertCurrencyInBidResponse(t, "EUR", "EUR")
}

func assertCurrencyInBidResponse(t *testing.T, expectedCurrency string, currency string) {
	// Create a GumGum bidder
	bidder, buildErr := Builder(openrtb_ext.BidderGumGum, config.Adapter{
		Endpoint: "https://g2.gumgum.com/providers/prbds2s/bid"}, config.Server{
		ExternalUrl: "http://hosturl.com",
		GvlID:       1,
		DataCenter:  "2",
	})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Create a mock BidRequest
	prebidRequest := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{},
	}

	// Create a mock BidResponse with or without currency
	mockedBidResponse := &openrtb2.BidResponse{}
	if currency != "" {
		mockedBidResponse.Cur = currency
	}

	// Marshal the mock bid response to JSON
	body, err := json.Marshal(mockedBidResponse)
	if err != nil {
		t.Fatalf("Failed to marshal mock bid response: %v", err)
	}

	// Create a mock ResponseData
	responseData := &adapters.ResponseData{
		StatusCode: 200,
		Body:       body,
	}

	// Call MakeBids
	bidResponse, errs := bidder.MakeBids(prebidRequest, nil, responseData)

	// Assert no errors
	if len(errs) != 0 {
		t.Fatalf("Failed to make bids %v", errs)
	}

	// Assert that the currency is correctly set
	assert.Equal(t, expectedCurrency, bidResponse.Currency, "Expected currency to be set to "+expectedCurrency)
}
