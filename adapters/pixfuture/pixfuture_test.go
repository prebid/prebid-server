package pixfuture

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestBuilder(t *testing.T) {
	adapter, err := Builder("pixfuture", config.Adapter{Endpoint: "https://mock-endpoint.com"}, config.Server{})
	assert.NoError(t, err, "unexpected error during Builder execution")
	assert.NotNil(t, adapter, "expected a non-nil adapter instance")
}

func TestAdapter_MakeRequests(t *testing.T) {
	adapter := &adapter{endpoint: "https://mock-pixfuture-endpoint.com"}

	t.Run("Valid Request", func(t *testing.T) {
		bidRequest := &openrtb2.BidRequest{
			ID: "test-request-id",
			Imp: []openrtb2.Imp{
				{
					ID:     "test-imp-id",
					Banner: &openrtb2.Banner{W: int64Ptr(300), H: int64Ptr(250)},
					Ext:    json.RawMessage(`{"bidder":{"siteId":"123", "placementId":"456"}}`),
				},
			},
		}

		requests, errs := adapter.MakeRequests(bidRequest, nil)
		assert.Empty(t, errs, "unexpected errors in MakeRequests")
		assert.Equal(t, 1, len(requests), "expected exactly one request")

		request := requests[0]
		assert.Equal(t, "POST", request.Method, "unexpected HTTP method")
		assert.Equal(t, "https://mock-pixfuture-endpoint.com", request.Uri, "unexpected request URI")
		assert.Contains(t, string(request.Body), `"id":"test-request-id"`, "unexpected request body")
		assert.Equal(t, "application/json", request.Headers.Get("Content-Type"), "unexpected content-type")
	})

	t.Run("No Impressions", func(t *testing.T) {
		bidRequest := &openrtb2.BidRequest{ID: "test-request-id"}

		requests, errs := adapter.MakeRequests(bidRequest, nil)
		assert.NotEmpty(t, errs, "expected error for request with no impressions")
		assert.Nil(t, requests, "expected no requests for request with no impressions")
	})

	t.Run("Malformed BidRequest", func(t *testing.T) {
		bidRequest := &openrtb2.BidRequest{}

		requests, errs := adapter.MakeRequests(bidRequest, nil)
		assert.NotEmpty(t, errs, "expected error for malformed request")
		assert.Nil(t, requests, "expected no requests for malformed request")
	})
}

func TestAdapter_MakeBids(t *testing.T) {
	adapter := &adapter{}

	t.Run("Valid Response", func(t *testing.T) {
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

		bidRequest := &openrtb2.BidRequest{ID: "test-request-id", Imp: []openrtb2.Imp{{ID: "test-imp-id"}}}
		bidResponse, errs := adapter.MakeBids(bidRequest, nil, responseData)

		assert.Empty(t, errs, "unexpected errors in MakeBids")
		assert.NotNil(t, bidResponse, "expected bid response")
		assert.Equal(t, "USD", bidResponse.Currency, "unexpected currency")
		assert.Equal(t, 1, len(bidResponse.Bids), "expected one bid")

		bid := bidResponse.Bids[0]
		assert.Equal(t, "test-bid-id", bid.Bid.ID, "unexpected bid ID")
		assert.Equal(t, "test-imp-id", bid.Bid.ImpID, "unexpected impression ID")
		assert.Equal(t, 1.23, bid.Bid.Price, "unexpected bid price")
		assert.Equal(t, openrtb_ext.BidTypeBanner, bid.BidType, "unexpected bid type")
	})

	t.Run("No Content Response", func(t *testing.T) {
		responseData := &adapters.ResponseData{StatusCode: http.StatusNoContent}
		bidRequest := &openrtb2.BidRequest{}
		bidResponse, errs := adapter.MakeBids(bidRequest, nil, responseData)
		assert.Nil(t, bidResponse, "expected no bid response")
		assert.Empty(t, errs, "unexpected errors for no content response")
	})

	t.Run("Bad Request Response", func(t *testing.T) {
		responseData := &adapters.ResponseData{StatusCode: http.StatusBadRequest}
		bidRequest := &openrtb2.BidRequest{}
		bidResponse, errs := adapter.MakeBids(bidRequest, nil, responseData)
		assert.Nil(t, bidResponse, "expected no bid response")
		assert.NotEmpty(t, errs, "expected errors for bad request response")
	})

	t.Run("Unexpected Status Code", func(t *testing.T) {
		responseData := &adapters.ResponseData{StatusCode: http.StatusInternalServerError}
		bidRequest := &openrtb2.BidRequest{}
		bidResponse, errs := adapter.MakeBids(bidRequest, nil, responseData)
		assert.Nil(t, bidResponse, "expected no bid response")
		assert.NotEmpty(t, errs, "expected errors for unexpected status code")
	})

	t.Run("Malformed Response Body", func(t *testing.T) {
		responseData := &adapters.ResponseData{
			StatusCode: http.StatusOK,
			Body:       []byte(`malformed response`),
		}
		bidRequest := &openrtb2.BidRequest{}
		bidResponse, errs := adapter.MakeBids(bidRequest, nil, responseData)
		assert.Nil(t, bidResponse, "expected no bid response")
		assert.NotEmpty(t, errs, "expected errors for malformed response body")
	})
}

func TestGetMediaTypeForBid(t *testing.T) {
	t.Run("Valid Bid Ext", func(t *testing.T) {
		bid := openrtb2.Bid{
			ID:  "test-bid",
			Ext: json.RawMessage(`{"prebid":{"type":"banner"}}`),
		}
		bidType, err := getMediaTypeForBid(bid)
		assert.NoError(t, err, "unexpected error in getMediaTypeForBid")
		assert.Equal(t, openrtb_ext.BidTypeBanner, bidType, "unexpected bid type")
	})

	t.Run("Invalid Bid Ext", func(t *testing.T) {
		bid := openrtb2.Bid{
			ID:  "test-bid",
			Ext: json.RawMessage(`{"invalid":"data"}`),
		}
		bidType, err := getMediaTypeForBid(bid)
		assert.Error(t, err, "expected error for invalid bid ext")
		assert.Equal(t, openrtb_ext.BidType(""), bidType, "expected empty bid type for invalid bid ext")
	})
}

func int64Ptr(i int64) *int64 {
	return &i
}

func jsonRawExt(jsonStr string) json.RawMessage {
	return json.RawMessage(jsonStr)
}
