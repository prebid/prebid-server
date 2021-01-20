package pulsepoint

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/stretchr/testify/assert"
)

func TestOpenRTBRequest(t *testing.T) {
	bidder := new(PulsePointAdapter)

	request := &openrtb.BidRequest{
		ID: "request-12345",
		Imp: []openrtb.Imp{{
			ID: "banner-1",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{
					{W: 300, H: 250},
				},
			},
			Ext: json.RawMessage(`{"bidder": {
				"cp": 1234,
				"ct": 1001
			}}`),
		}, {
			ID: "test-imp-video-id",
			Video: &openrtb.Video{
				W:           640,
				H:           360,
				MIMEs:       []string{"video/mp4"},
				MinDuration: 15,
				MaxDuration: 30,
			},
			Ext: json.RawMessage(`{"bidder": {
				"cp": 1234,
				"ct": 2001
			}}`),
		}},
		Site: &openrtb.Site{
			Publisher: &openrtb.Publisher{
				Name: "publisher.com",
			},
		},
	}

	reqs, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs, "Got unexpected errors while building HTTP requests: %v", errs)
	assert.Equal(t, 1, len(reqs), "Unexpected number of HTTP requests. Got %d. Expected %d", len(reqs), 2)

	httpReq := reqs[0]
	assert.Equal(t, "POST", httpReq.Method, "Expected a POST message. Got %s", httpReq.Method)

	var ortbRequest openrtb.BidRequest
	if err := json.Unmarshal(httpReq.Body, &ortbRequest); err != nil {
		t.Fatalf("Failed to unmarshal HTTP request: %v", ortbRequest)
	}

	assert.Equal(t, request.ID, ortbRequest.ID, "Bad Request ID. Expected %s, Got %s", request.ID, ortbRequest.ID)
	assert.Equal(t, 2, len(ortbRequest.Imp), "Wrong len(request.Imp). Expected %d, Got %d", len(request.Imp), len(ortbRequest.Imp))
	assert.Equal(t, request.Imp[0].ID, ortbRequest.Imp[0].ID, "Bad Impression ID. Expected %s, Got %s", request.Imp[0].ID, ortbRequest.Imp[0].ID)
	assert.Equal(t, request.Imp[1].ID, ortbRequest.Imp[1].ID, "Bad Impression ID. Expected %s, Got %s", request.Imp[1].ID, ortbRequest.Imp[1].ID)
	assert.Equal(t, "1001", ortbRequest.Imp[0].TagID, "Bad Tag ID. Expected 1001, Got %s", ortbRequest.Imp[0].TagID)
	assert.Equal(t, "2001", ortbRequest.Imp[1].TagID, "Bad Tag ID. Expected 2001, Got %s", ortbRequest.Imp[1].TagID)
	assert.Equal(t, "1234", ortbRequest.Site.Publisher.ID, "Bad Publisher ID. Expected 1234, Got %s", ortbRequest.Site.Publisher.ID)
}

func TestOpenRTBRequestNoPubProvided(t *testing.T) {
	bidder := new(PulsePointAdapter)

	request := &openrtb.BidRequest{
		ID: "request-12345",
		Imp: []openrtb.Imp{{
			ID: "banner-1",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{
					{W: 300, H: 250},
				},
			},
			Ext: json.RawMessage(`{"bidder": {
				"cp": 1234,
				"ct": 1001
			}}`),
		},
		},
		App: &openrtb.App{
			ID: "com.pulsepoint.app",
		},
	}

	reqs, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs, "Got unexpected errors while building HTTP requests: %v", errs)
	assert.Equal(t, 1, len(reqs), "Unexpected number of HTTP requests. Got %d. Expected %d", len(reqs), 2)

	httpReq := reqs[0]
	assert.Equal(t, "POST", httpReq.Method, "Expected a POST message. Got %s", httpReq.Method)

	var ortbRequest openrtb.BidRequest
	if err := json.Unmarshal(httpReq.Body, &ortbRequest); err != nil {
		t.Fatalf("Failed to unmarshal HTTP request: %v", ortbRequest)
	}

	assert.Equal(t, request.ID, ortbRequest.ID, "Bad Request ID. Expected %s, Got %s", request.ID, ortbRequest.ID)
	assert.Equal(t, 1, len(ortbRequest.Imp), "Wrong len(request.Imp). Expected %d, Got %d", len(request.Imp), len(ortbRequest.Imp))
	assert.Equal(t, "1234", ortbRequest.App.Publisher.ID, "Bad Publisher ID. Expected 1234, Got %s", ortbRequest.App.Publisher.ID)
}

func TestMakeBids(t *testing.T) {
	bidder := new(PulsePointAdapter)

	request := &openrtb.BidRequest{
		ID: "request-1000",
		Imp: []openrtb.Imp{{
			ID: "imp-123",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{
					{W: 300, H: 250},
				},
			},
		},
		},
	}

	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"id":"request-1000","seatbid":[{"bid":[{"id":"1234567890","impid":"imp-123","price":2,"crid":"4122982","adm":"some ad","h":50,"w":320}]}]}`),
	}

	openrtbResponse, errs := bidder.MakeBids(request, nil, httpResp)
	assert.NotNil(t, openrtbResponse, "Expected not empty response")
	assert.Equal(t, 1, len(openrtbResponse.Bids), "Expected 1 bid. Got %d", len(openrtbResponse.Bids))
	assert.Empty(t, errs, "Expected 0 errors. Got %d", len(errs))
	assert.Equal(t, openrtb_ext.BidTypeBanner, openrtbResponse.Bids[0].BidType, "Expected bid type %s. Got %s", openrtb_ext.BidTypeBanner, openrtbResponse.Bids[0].BidType)
}

func TestMakeBidsVideo(t *testing.T) {
	bidder := new(PulsePointAdapter)

	request := &openrtb.BidRequest{
		ID: "request-1001",
		Imp: []openrtb.Imp{{
			ID: "imp-234",
			Video: &openrtb.Video{
				W: 640,
				H: 360,
			},
		},
		},
	}

	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"id":"request-1001","seatbid":[{"bid":[{"id":"1234567890","impid":"imp-234","price":2,"crid":"4122982","adm":"<vast></vast>"}]}]}`),
	}

	openrtbResponse, errs := bidder.MakeBids(request, nil, httpResp)
	assert.NotNil(t, openrtbResponse, "Expected not empty response")
	assert.Equal(t, 1, len(openrtbResponse.Bids), "Expected 1 bid. Got %d", len(openrtbResponse.Bids))
	assert.Empty(t, errs, "Expected 0 errors. Got %d", len(errs))
	assert.Equal(t, openrtb_ext.BidTypeVideo, openrtbResponse.Bids[0].BidType, "Expected bid type %s. Got %s", openrtb_ext.BidTypeVideo, openrtbResponse.Bids[0].BidType)
}

func TestUnknownImpId(t *testing.T) {
	bidder := new(PulsePointAdapter)

	request := &openrtb.BidRequest{
		ID: "request-1001",
		Imp: []openrtb.Imp{{
			ID: "imp-234",
			Video: &openrtb.Video{
				W: 640,
				H: 360,
			},
		},
		},
	}

	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"id":"request-1001","seatbid":[{"bid":[{"id":"1234567890","impid":"imp-345","price":2,"crid":"4122982","adm":"<vast></vast>"}]}]}`),
	}

	openrtbResponse, errs := bidder.MakeBids(request, nil, httpResp)
	assert.NotNil(t, openrtbResponse, "Expected not empty response")
	assert.Equal(t, 0, len(openrtbResponse.Bids), "Expected 1 bid. Got %d", len(openrtbResponse.Bids))
	assert.Empty(t, errs, "Expected 0 errors. Got %d", len(errs))
}
