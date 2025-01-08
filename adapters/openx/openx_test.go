package openx

import (
	"encoding/json"
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

func TestGetMediaTypeForImp(t *testing.T) {
	imps := []openrtb2.Imp{
		{ID: "1", Banner: &openrtb2.Banner{}},
		{ID: "2", Video: &openrtb2.Video{}},
		{ID: "3", Native: &openrtb2.Native{}},
		{ID: "4", Video: &openrtb2.Video{}, Native: &openrtb2.Native{}},
		{ID: "5", Banner: &openrtb2.Banner{}, Video: &openrtb2.Video{}, Native: &openrtb2.Native{}},
	}

	assert.Equal(t, getMediaTypeForImp("1", imps), openrtb_ext.BidTypeBanner)
	assert.Equal(t, getMediaTypeForImp("2", imps), openrtb_ext.BidTypeVideo)
	assert.Equal(t, getMediaTypeForImp("3", imps), openrtb_ext.BidTypeNative)
	assert.Equal(t, getMediaTypeForImp("4", imps), openrtb_ext.BidTypeVideo)
	assert.Equal(t, getMediaTypeForImp("5", imps), openrtb_ext.BidTypeBanner)

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
