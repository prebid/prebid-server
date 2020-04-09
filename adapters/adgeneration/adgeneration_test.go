package adgeneration

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "adgenerationtest", NewAdgenerationAdapter("https://d.socdm.com/adsv/v1"))
}

func TestGetRequestUri(t *testing.T) {
	bidder := NewAdgenerationAdapter("https://d.socdm.com/adsv/v1")
	// Test items
	failedRequest := &openrtb.BidRequest{
		ID: "test-failed-bid-request",
		Imp: []openrtb.Imp{
			{ID: "extImpBidder-failed-test", Banner: &openrtb.Banner{Format: []openrtb.Format{{W: 300, H: 250}}}, Ext: json.RawMessage(`{{ "id": "58278" }}`)},
			{ID: "extImpBidder-failed-test", Banner: &openrtb.Banner{Format: []openrtb.Format{{W: 300, H: 250}}}, Ext: json.RawMessage(`{"_bidder": { "id": "58278" }}`)},
			{ID: "extImpAdgeneration-failed-test", Banner: &openrtb.Banner{Format: []openrtb.Format{{W: 300, H: 250}}}, Ext: json.RawMessage(`{"bidder": { "_id": "58278" }}`)},
		},
		Device: &openrtb.Device{UA: "testUA", IP: "testIP"},
		Site:   &openrtb.Site{Page: "https://supership.com"},
		User:   &openrtb.User{BuyerUID: "buyerID"},
	}
	successRequest := &openrtb.BidRequest{
		ID: "test-success-bid-request",
		Imp: []openrtb.Imp{
			{ID: "bidRequest-success-test", Banner: &openrtb.Banner{Format: []openrtb.Format{{W: 300, H: 250}}}, Ext: json.RawMessage(`{"bidder": { "id": "58278" }}`)},
		},
		Device: &openrtb.Device{UA: "testUA", IP: "testIP"},
		Site:   &openrtb.Site{Page: "https://supership.com"},
		User:   &openrtb.User{BuyerUID: "buyerID"},
	}

	numRequests := len(failedRequest.Imp)
	for index := 0; index < numRequests; index++ {
		httpRequests, err := bidder.getRequestUri(failedRequest, index)
		if err == nil {
			t.Errorf("%v did not throw an error", failedRequest.Imp[index])
		}
		if httpRequests != "" {
			t.Errorf("%v did return Request: %s", failedRequest.Imp[index], httpRequests)
		}
	}
	numRequests = len(successRequest.Imp)
	for index := 0; index < numRequests; index++ {
		httpRequests, err := bidder.getRequestUri(successRequest, index)
		if err != nil {
			t.Errorf("%v did throw an error", successRequest.Imp[index])
		}
		if httpRequests == "?posall=SSPLOC&id=58278&sdktype=0&hb=true&t=json3&currency=JPY&sdkname=prebidserver&adapterver="+bidder.version {
			t.Errorf("%v did return Request: %s", successRequest.Imp[index], httpRequests)
		}
	}
}

func TestGetSizes(t *testing.T) {
	bidder := NewAdgenerationAdapter("https://d.socdm.com/adsv/v1")
	// Test items
	var request *openrtb.Imp
	var size string
	multiFormatBanner := &openrtb.Banner{Format: []openrtb.Format{{W: 300, H: 250}, {W: 320, H: 50}}}
	noFormatBanner := &openrtb.Banner{Format: []openrtb.Format{}}
	nativeFormat := &openrtb.Native{}

	request = &openrtb.Imp{Banner: multiFormatBanner}
	size = bidder.getSizes(request)
	if size != "300×250,320×50" {
		t.Errorf("%v does not match size.", multiFormatBanner)
	}
	request = &openrtb.Imp{Banner: noFormatBanner}
	size = bidder.getSizes(request)
	if size != "" {
		t.Errorf("%v does not match size.", noFormatBanner)
	}
	request = &openrtb.Imp{Native: nativeFormat}
	size = bidder.getSizes(request)
	if size != "" {
		t.Errorf("%v does not match size.", nativeFormat)
	}
}

func TestGetCurrency(t *testing.T) {
	bidder := NewAdgenerationAdapter("https://d.socdm.com/adsv/v1")
	// Test items
	var request *openrtb.BidRequest
	var currency string
	innerDefaultCur := []string{"USD", "JPY"}
	usdCur := []string{"USD", "EUR"}

	request = &openrtb.BidRequest{Cur: innerDefaultCur}
	currency = bidder.getCurrency(request)
	if currency != "JPY" {
		t.Errorf("%v does not match currency.", innerDefaultCur)
	}
	request = &openrtb.BidRequest{Cur: usdCur}
	currency = bidder.getCurrency(request)
	if currency != "USD" {
		t.Errorf("%v does not match currency.", usdCur)
	}
}

func TestCreateAd(t *testing.T) {
	bidder := NewAdgenerationAdapter("https://d.socdm.com/adsv/v1")
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

	bannerAd := bidder.createAd(&adgBannerResponse, adgBannerImpId)
	if bannerAd != matchBannerTag {
		t.Errorf("%v does not match createAd.", adgBannerResponse)
	}
	vastAd := bidder.createAd(&adgVastResponse, adgVastImpId)
	if vastAd != matchVastTag {
		t.Errorf("%v does not match createAd.", adgVastResponse)
	}
}
