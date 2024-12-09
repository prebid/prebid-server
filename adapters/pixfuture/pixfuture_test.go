package pixfuture

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestPixfutureAdapter_MakeRequests(t *testing.T) {
	adapter := &PixfutureAdapter{
		endpoint: "http://mock-pixfuture-endpoint.com",
	}

	// Prepare a mock BidRequest
	bidRequest := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{
			{
				ID:     "test-imp-id",
				Banner: &openrtb2.Banner{W: int64Ptr(300), H: int64Ptr(250)},
				Ext:    jsonRawExt(`{"bidder":{"siteId":"123"}}`),
			},
		},
	}

	// Call MakeRequests
	requests, errs := adapter.MakeRequests(bidRequest, nil)

	// Assert no errors
	assert.Empty(t, errs, "unexpected errors in MakeRequests")

	// Assert single request created
	assert.Equal(t, 1, len(requests), "expected exactly one request")

	// Assert request data
	request := requests[0]
	assert.Equal(t, "POST", request.Method, "unexpected HTTP method")
	assert.Equal(t, "http://mock-pixfuture-endpoint.com", request.Uri, "unexpected request URI")
	assert.Contains(t, string(request.Body), `"id":"test-request-id"`, "unexpected request body")
	assert.Equal(t, "application/json", request.Headers.Get("Content-Type"), "unexpected content-type")
}

func TestPixfutureAdapter_MakeBids(t *testing.T) {
	adapter := &PixfutureAdapter{}

	// Mock HTTP response
	responseData := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body: []byte(`{
			"id": "test-response-id",
			"seatbid": [{
				"bid": [{
					"id": "test-bid-id",
					"impid": "test-imp-id",
					"price": 1.23,
					"adm": "<html>Ad Content</html>",
					"crid": "creative-123",
					"w": 300,
					"h": 250,
					"ext": {"prebid":{"type":"banner"}}
				}]
			}],
			"cur": "USD"
		}`),
	}

	bidRequest := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{
			{ID: "test-imp-id"},
		},
	}

	// Call MakeBids
	bidResponse, errs := adapter.MakeBids(bidRequest, nil, responseData)

	// Assert no errors
	assert.Empty(t, errs, "unexpected errors in MakeBids")

	// Assert bid response
	assert.NotNil(t, bidResponse, "expected bid response")
	assert.Equal(t, "USD", bidResponse.Currency, "unexpected currency")
	assert.Equal(t, 1, len(bidResponse.Bids), "expected one bid")

	// Assert bid details
	bid := bidResponse.Bids[0]
	assert.Equal(t, "test-bid-id", bid.Bid.ID, "unexpected bid ID")
	assert.Equal(t, "test-imp-id", bid.Bid.ImpID, "unexpected impression ID")
	assert.Equal(t, 1.23, bid.Bid.Price, "unexpected bid price")
	assert.Equal(t, openrtb_ext.BidTypeBanner, bid.BidType, "unexpected bid type")
}

func int64Ptr(i int64) *int64 {
	return &i
}

func jsonRawExt(jsonStr string) json.RawMessage {
	return json.RawMessage(jsonStr)
}
