package eplanning

import (
	"encoding/json"
	"net/http"

	"bytes"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"testing"
	"time"
)

type aBidInfo struct {
	deviceIP string
	deviceUA string
	tags     []aTagInfo
	width    uint64
	height   uint64
	buyerUID string
	delay    time.Duration
}

type aTagInfo struct {
	code     string
	bidfloor float64
	instl    int8
	price    float64
	content  string
	dealId   string
}

func TestOpenRTBRequest(t *testing.T) {
	bidder := new(EPlanningAdapter)
	bidder.URI = "http://e-planning-net"
	testData := createTestData()
	request := createOpenRtbRequest(testData)

	httpRequests, errs := bidder.MakeRequests(request)

	if len(errs) > 0 {
		t.Errorf("Got unexpected errors while building HTTP requests: %v", errs)
	}
	if len(httpRequests) != 1 {
		t.Fatalf("Unexpected number of HTTP requests. Got %d. Expected %d", len(httpRequests), 1)
	}

	r, err := http.NewRequest(httpRequests[0].Method, httpRequests[0].Uri, bytes.NewReader(httpRequests[0].Body))
	if err != nil {
		t.Fatalf("Unexpected request. Got %v", err)
	}
	r.Header = httpRequests[0].Headers
}

func TestOpenRTBStandardResponse(t *testing.T) {
	testData := createTestData()
	request := createOpenRtbRequest(testData)

	responseBody, err := createEPlanningServerResponse(*testData)
	if err != nil {
		t.Fatalf("Unable to create server response: %v", err)
		return
	}
	httpResponse := &adapters.ResponseData{StatusCode: http.StatusOK, Body: responseBody}

	bidder := new(EPlanningAdapter)
	bids, errs := bidder.MakeBids(request, nil, httpResponse)

	if len(bids) != 2 {
		t.Fatalf("Expected 2 bids. Got %d", len(bids))
	}
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors. Got %d", len(errs))
	}

	for _, typeBid := range bids {
		bid := typeBid.Bid
		matched := false

		for _, tag := range testData.tags {
			if bid.ID == tag.code {
				matched = true
				if bid.Price != tag.price {
					t.Errorf("Incorrect bid price '%.2f' expected '%.2f'", bid.Price, tag.price)
				}
				if bid.W != testData.width || bid.H != testData.height {
					t.Errorf("Incorrect bid size %dx%d, expected %dx%d", bid.W, bid.H, testData.width, testData.height)
				}
				if bid.DealID != tag.dealId {
					t.Errorf("Incorrect deal id '%s' expected '%s'", bid.DealID, tag.dealId)
				}
			}
		}
		if !matched {
			t.Errorf("Received bid with unknown id '%s'", bid.ID)
		}
	}
}

func TestOpenRTBSurpriseResponse(t *testing.T) {
	bidder := new(EPlanningAdapter)

	bids, errs := bidder.MakeBids(nil, nil,
		&adapters.ResponseData{StatusCode: http.StatusNoContent, Body: []byte("")})
	if bids != nil && errs != nil {
		t.Fatalf("Expected no bids and no errors. Got %d bids and %d", len(bids), len(errs))
	}

	bids, errs = bidder.MakeBids(nil, nil,
		&adapters.ResponseData{StatusCode: http.StatusServiceUnavailable, Body: []byte("")})
	if bids != nil || len(errs) != 1 {
		t.Fatalf("Expected one error and no bids. Got %d bids and %d", len(bids), len(errs))
	}

	bids, errs = bidder.MakeBids(nil, nil,
		&adapters.ResponseData{StatusCode: http.StatusOK, Body: []byte("{:'not-valid-json'}")})
	if bids != nil || len(errs) != 1 {
		t.Fatalf("Expected one error and no bids. Got %d bids and %d", len(bids), len(errs))
	}
}

func createTestData() *aBidInfo {
	testData := &aBidInfo{
		deviceIP: "111.111.111.111",
		deviceUA: "Mozilla/5.0 (iPhone; CPU iPhone OS 10_3_1 like Mac OS X) AppleWebKit/603.1.30 (KHTML, like Gecko) Mobile/14E8301",
		buyerUID: "user-id",
		tags: []aTagInfo{
			{code: "code1", price: 1.23, content: "banner-content1", dealId: "dealId1", bidfloor: 1, instl: 0},
			{code: "code2"}, // no bid for ad unit
			{code: "code3", price: 1.24, content: "banner-content2", dealId: "dealId2", bidfloor: 10, instl: 1},
		},
	}
	return testData
}

func createOpenRtbRequest(testData *aBidInfo) *openrtb.BidRequest {
	bidRequest := &openrtb.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb.Imp{
			{
				ID:       testData.tags[0].code,
				Banner:   &openrtb.Banner{},
				BidFloor: testData.tags[0].bidfloor,
				Instl:    testData.tags[0].instl,
				Ext:      openrtb.RawJSON(`{"bidder": { "ssp_espacio_id": "32344" }}`),
			},
			{
				ID:     testData.tags[1].code,
				Banner: &openrtb.Banner{},
				Ext:    openrtb.RawJSON(`{"bidder": { "ssp_espacio_id": "32345" }}`),
			},
			{
				ID:       testData.tags[2].code,
				Banner:   &openrtb.Banner{},
				BidFloor: testData.tags[2].bidfloor,
				Instl:    testData.tags[2].instl,
				Ext:      openrtb.RawJSON(`{"bidder": { "ssp_espacio_id": "32346" }}`),
			},
		},
		Site: &openrtb.Site{},
		Device: &openrtb.Device{
			UA: testData.deviceUA,
			IP: testData.deviceIP,
		},
		Source: &openrtb.Source{},
		User: &openrtb.User{
			BuyerUID: testData.buyerUID,
		},
	}
	return bidRequest
}

func createEPlanningServerResponse(testData aBidInfo) ([]byte, error) {
	bids := []EPlanningBid{
		{
			Id:     testData.tags[0].code,
			Price:  testData.tags[0].price,
			Width:  testData.width,
			Height: testData.height,
			DealId: testData.tags[0].dealId,
		},
		{},
		{
			Id:     testData.tags[2].code,
			Price:  testData.tags[2].price,
			Width:  testData.width,
			Height: testData.height,
			DealId: testData.tags[2].dealId,
		},
	}
	ePlanningServerResponse, err := json.Marshal(bids)
	return ePlanningServerResponse, err
}
