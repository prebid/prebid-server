package adgeneration

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
	bidder, buildErr := Builder(openrtb_ext.BidderAdgeneration, config.Adapter{
		Endpoint: "https://d.socdm.com/adsv/v1"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "adgenerationtest", bidder)
}

func TestGetRequestUri(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdgeneration, config.Adapter{
		Endpoint: "https://d.socdm.com/adsv/v1"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	bidderAdgeneration, _ := bidder.(*AdgenerationAdapter)

	// Test items
	failedRequest := &openrtb2.BidRequest{
		ID: "test-failed-bid-request",
		Imp: []openrtb2.Imp{
			{ID: "extImpBidder-failed-test", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}, Ext: json.RawMessage(`{{ "id": "58278" }}`)},
			{ID: "extImpBidder-failed-test", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}, Ext: json.RawMessage(`{"_bidder": { "id": "58278" }}`)},
			{ID: "extImpAdgeneration-failed-test", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}, Ext: json.RawMessage(`{"bidder": { "_id": "58278" }}`)},
		},
		Source: &openrtb2.Source{TID: "SourceTID"},
		Device: &openrtb2.Device{UA: "testUA", IP: "testIP"},
		Site:   &openrtb2.Site{Page: "https://supership.com"},
		User:   &openrtb2.User{BuyerUID: "buyerID"},
	}
	successRequest := &openrtb2.BidRequest{
		ID: "test-success-bid-request",
		Imp: []openrtb2.Imp{
			{ID: "bidRequest-success-test", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}, Ext: json.RawMessage(`{"bidder": { "id": "58278" }}`)},
		},
		Source: &openrtb2.Source{TID: "SourceTID"},
		Device: &openrtb2.Device{UA: "testUA", IP: "testIP"},
		Site:   &openrtb2.Site{Page: "https://supership.com"},
		User:   &openrtb2.User{BuyerUID: "buyerID"},
	}

	numRequests := len(failedRequest.Imp)
	for index := 0; index < numRequests; index++ {
		httpRequests, err := bidderAdgeneration.getRequestUri(failedRequest, index)
		if err == nil {
			t.Errorf("getRequestUri: %v did not throw an error", failedRequest.Imp[index])
		}
		if httpRequests != "" {
			t.Errorf("getRequestUri: %v did return Request: %s", failedRequest.Imp[index], httpRequests)
		}
	}
	numRequests = len(successRequest.Imp)
	for index := 0; index < numRequests; index++ {
		// getRawQuery Test.
		adgExt, err := unmarshalExtImpAdgeneration(&successRequest.Imp[index])
		if err != nil {
			t.Errorf("unmarshalExtImpAdgeneration: %v did throw an error: %v", successRequest.Imp[index], err)
		}
		rawQuery := bidderAdgeneration.getRawQuery(adgExt.Id, successRequest, &successRequest.Imp[index])
		expectQueries := map[string]string{
			"posall":        "SSPLOC",
			"id":            adgExt.Id,
			"sdktype":       "0",
			"hb":            "true",
			"currency":      bidderAdgeneration.getCurrency(successRequest),
			"sdkname":       "prebidserver",
			"adapterver":    bidderAdgeneration.version,
			"sizes":         getSizes(&successRequest.Imp[index]),
			"tp":            successRequest.Site.Page,
			"transactionid": successRequest.Source.TID,
		}
		for key, expectedValue := range expectQueries {
			actualValue := rawQuery.Get(key)
			if actualValue != expectedValue {
				t.Errorf("getRawQuery: %s value does not match expected %s, actual %s", key, expectedValue, actualValue)
			}
		}

		// RequestUri Test.
		actualUri, err := bidderAdgeneration.getRequestUri(successRequest, index)
		if err != nil {
			t.Errorf("getRequestUri: %v did throw an error: %v", successRequest.Imp[index], err)
		}
		expectedUri := "https://d.socdm.com/adsv/v1?adapterver=" + bidderAdgeneration.version + "&currency=JPY&hb=true&id=58278&posall=SSPLOC&sdkname=prebidserver&sdktype=0&sizes=300x250&t=json3&tp=https%3A%2F%2Fsupership.com&transactionid=SourceTID"
		if actualUri != expectedUri {
			t.Errorf("getRequestUri: does not match expected %s, actual %s", expectedUri, actualUri)
		}
	}
}

func TestGetSizes(t *testing.T) {
	// Test items
	var request *openrtb2.Imp
	var size string
	multiFormatBanner := &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 320, H: 50}}}
	noFormatBanner := &openrtb2.Banner{Format: []openrtb2.Format{}}
	nativeFormat := &openrtb2.Native{}

	request = &openrtb2.Imp{Banner: multiFormatBanner}
	size = getSizes(request)
	if size != "300x250,320x50" {
		t.Errorf("%v does not match size.", multiFormatBanner)
	}
	request = &openrtb2.Imp{Banner: noFormatBanner}
	size = getSizes(request)
	if size != "" {
		t.Errorf("%v does not match size.", noFormatBanner)
	}
	request = &openrtb2.Imp{Native: nativeFormat}
	size = getSizes(request)
	if size != "" {
		t.Errorf("%v does not match size.", nativeFormat)
	}
}

func TestGetCurrency(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdgeneration, config.Adapter{
		Endpoint: "https://d.socdm.com/adsv/v1"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	bidderAdgeneration, _ := bidder.(*AdgenerationAdapter)

	// Test items
	var request *openrtb2.BidRequest
	var currency string
	innerDefaultCur := []string{"USD", "JPY"}
	usdCur := []string{"USD", "EUR"}

	request = &openrtb2.BidRequest{Cur: innerDefaultCur}
	currency = bidderAdgeneration.getCurrency(request)
	if currency != "JPY" {
		t.Errorf("%v does not match currency.", innerDefaultCur)
	}
	request = &openrtb2.BidRequest{Cur: usdCur}
	currency = bidderAdgeneration.getCurrency(request)
	if currency != "USD" {
		t.Errorf("%v does not match currency.", usdCur)
	}
}

func TestCreateAd(t *testing.T) {
	// Test items
	adgBannerImpId := "test-banner-imp"
	adgBannerResponse := adgServerResponse{
		Ad:         "<!DOCTYPE html>\n<head>\n<meta charset=\"UTF-8\">\n<script src=\"test.com\"></script>\n<body>\n<div id=\"medibasspContainer\">\n<iframe src=\"https://dummy-iframe.com></iframe>\n</div>\n</body>\n",
		Beacon:     "<img src=\"https://dummy-beacon.com\">",
		Beaconurl:  "https://dummy-beacon.com",
		Cpm:        50,
		Creativeid: "DummyDsp_SdkTeam_supership.jp",
		H:          300,
		W:          250,
		Ttl:        10,
		LandingUrl: "",
		Scheduleid: "111111",
	}
	matchBannerTag := "<div id=\"medibasspContainer\">\n<iframe src=\"https://dummy-iframe.com></iframe>\n</div>\n<img src=\"https://dummy-beacon.com\">"

	adgVastImpId := "test-vast-imp"
	adgVastResponse := adgServerResponse{
		Ad:         "<!DOCTYPE html>\n<head>\n<meta charset=\"UTF-8\">\n<script src=\"test.com\"></script>\n<body>\n<div id=\"medibasspContainer\">\n<iframe src=\"https://dummy-iframe.com></iframe>\n</div>\n</body>\n",
		Beacon:     "<img src=\"https://dummy-beacon.com\">",
		Beaconurl:  "https://dummy-beacon.com",
		Cpm:        50,
		Creativeid: "DummyDsp_SdkTeam_supership.jp",
		H:          300,
		W:          250,
		Ttl:        10,
		LandingUrl: "",
		Vastxml:    "<?xml version=\"1.0\"><VAST version=\"3.0\"</VAST>",
		Scheduleid: "111111",
	}
	matchVastTag := "<div id=\"apvad-test-vast-imp\"></div><script type=\"text/javascript\" id=\"apv\" src=\"https://cdn.apvdr.com/js/VideoAd.min.js\"></script><script type=\"text/javascript\"> (function(){ new APV.VideoAd({s:\"test-vast-imp\"}).load('<?xml version=\"1.0\"><VAST version=\"3.0\"</VAST>'); })(); </script><img src=\"https://dummy-beacon.com\">"

	bannerAd := createAd(&adgBannerResponse, adgBannerImpId)
	if bannerAd != matchBannerTag {
		t.Errorf("%v does not match createAd.", adgBannerResponse)
	}
	vastAd := createAd(&adgVastResponse, adgVastImpId)
	if vastAd != matchVastTag {
		t.Errorf("%v does not match createAd.", adgVastResponse)
	}
}

func TestMakeBids(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdgeneration, config.Adapter{
		Endpoint: "https://d.socdm.com/adsv/v1"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	bidderAdgeneration, _ := bidder.(*AdgenerationAdapter)

	internalRequest := &openrtb2.BidRequest{
		ID: "test-success-bid-request",
		Imp: []openrtb2.Imp{
			{ID: "bidRequest-success-test", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}, Ext: json.RawMessage(`{"bidder": { "id": "58278" }}`)},
		},
		Device: &openrtb2.Device{UA: "testUA", IP: "testIP"},
		Site:   &openrtb2.Site{Page: "https://supership.com"},
		User:   &openrtb2.User{BuyerUID: "buyerID"},
	}
	externalRequest := adapters.RequestData{}
	response := adapters.ResponseData{
		StatusCode: 200,
		Body:       ([]byte)("{\n \"ad\": \"testAd\",\n \"cpm\": 30,\n \"creativeid\": \"Dummy_supership.jp\",\n \"h\": 250,\n \"locationid\": \"58278\",\n \"results\": [{}],\n \"dealid\": \"test-deal-id\",\n \"w\": 300\n }"),
	}
	// default Currency InternalRequest
	defaultCurBidderResponse, errs := bidder.MakeBids(internalRequest, &externalRequest, &response)
	if len(errs) > 0 {
		t.Errorf("MakeBids return errors. errors: %v", errs)
	}
	checkBidResponse(t, defaultCurBidderResponse, bidderAdgeneration.defaultCurrency)

	// Specified Currency InternalRequest
	usdCur := "USD"
	internalRequest.Cur = []string{usdCur}
	specifiedCurBidderResponse, errs := bidder.MakeBids(internalRequest, &externalRequest, &response)
	if len(errs) > 0 {
		t.Errorf("MakeBids return errors. errors: %v", errs)
	}
	checkBidResponse(t, specifiedCurBidderResponse, usdCur)

}

func checkBidResponse(t *testing.T, bidderResponse *adapters.BidderResponse, expectedCurrency string) {
	if bidderResponse == nil {
		t.Errorf("actual bidResponse is nil.")
	}

	// AdM is assured by TestCreateAd and JSON tests
	var expectedAdM string = "testAd"
	var expectedID string = "58278"
	var expectedImpID = "bidRequest-success-test"
	var expectedPrice float64 = 30.0
	var expectedW int64 = 300
	var expectedH int64 = 250
	var expectedCrID string = "Dummy_supership.jp"
	var extectedDealID string = "test-deal-id"

	//nolint: staticcheck // false positive SA5011: possible nil pointer dereference
	assert.Equal(t, expectedCurrency, bidderResponse.Currency)
	assert.Equal(t, 1, len(bidderResponse.Bids))
	assert.Equal(t, expectedID, bidderResponse.Bids[0].Bid.ID)
	assert.Equal(t, expectedImpID, bidderResponse.Bids[0].Bid.ImpID)
	assert.Equal(t, expectedAdM, bidderResponse.Bids[0].Bid.AdM)
	assert.Equal(t, expectedPrice, bidderResponse.Bids[0].Bid.Price)
	assert.Equal(t, expectedW, bidderResponse.Bids[0].Bid.W)
	assert.Equal(t, expectedH, bidderResponse.Bids[0].Bid.H)
	assert.Equal(t, expectedCrID, bidderResponse.Bids[0].Bid.CrID)
	assert.Equal(t, extectedDealID, bidderResponse.Bids[0].Bid.DealID)
}
