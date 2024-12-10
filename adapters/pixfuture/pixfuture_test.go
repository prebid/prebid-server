package pixfuture

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/stretchr/testify/assert"
)

func TestPixfutureAdapter_MakeRequests(t *testing.T) {
	adapter := &PixfutureAdapter{endpoint: "http://mock-pixfuture-endpoint.com"}

	t.Run("Valid Request", func(t *testing.T) {
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

		requests, errs := adapter.MakeRequests(bidRequest, nil)
		assert.Empty(t, errs, "unexpected errors in MakeRequests")
		assert.Equal(t, 1, len(requests), "expected exactly one request")
	})

	t.Run("Empty Impressions", func(t *testing.T) {
		bidRequest := &openrtb2.BidRequest{}

		requests, errs := adapter.MakeRequests(bidRequest, nil)
		assert.NotEmpty(t, errs, "expected error for no impressions")
		assert.Nil(t, requests, "expected no requests")
	})

	t.Run("Marshal Error", func(t *testing.T) {
		originalMarshal := json.Marshal
		defer func() { json.Marshal = originalMarshal }()
		json.Marshal = func(v interface{}) ([]byte, error) {
			return nil, errors.New("mock marshal error")
		}

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

		requests, errs := adapter.MakeRequests(bidRequest, nil)
		assert.NotEmpty(t, errs, "expected marshal error")
		assert.Nil(t, requests, "expected no requests")
	})
}

func TestPixfutureAdapter_MakeBids(t *testing.T) {
	adapter := &PixfutureAdapter{}

	t.Run("No Content Response", func(t *testing.T) {
		responseData := &adapters.ResponseData{StatusCode: http.StatusNoContent}
		bidResponse, errs := adapter.MakeBids(nil, nil, responseData)

		assert.Nil(t, bidResponse, "expected no bid response")
		assert.Empty(t, errs, "expected no errors for 204 status")
	})

	t.Run("Bad Request Response", func(t *testing.T) {
		responseData := &adapters.ResponseData{StatusCode: http.StatusBadRequest}
		bidResponse, errs := adapter.MakeBids(nil, nil, responseData)

		assert.Nil(t, bidResponse, "expected no bid response")
		assert.NotEmpty(t, errs, "expected error for 400 status")
		assert.IsType(t, &errortypes.BadInput{}, errs[0], "expected BadInput error")
	})

	t.Run("Unexpected Status Code", func(t *testing.T) {
		responseData := &adapters.ResponseData{StatusCode: http.StatusInternalServerError}
		bidResponse, errs := adapter.MakeBids(nil, nil, responseData)

		assert.Nil(t, bidResponse, "expected no bid response")
		assert.NotEmpty(t, errs, "expected error for 500 status")
		assert.IsType(t, &errortypes.BadServerResponse{}, errs[0], "expected BadServerResponse error")
	})

	t.Run("Unmarshal Error", func(t *testing.T) {
		responseData := &adapters.ResponseData{
			StatusCode: http.StatusOK,
			Body:       []byte(`invalid-json`),
		}
		bidResponse, errs := adapter.MakeBids(nil, nil, responseData)

		assert.Nil(t, bidResponse, "expected no bid response")
		assert.NotEmpty(t, errs, "expected unmarshal error")
	})

	t.Run("Error in MediaType Parsing", func(t *testing.T) {
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
						"ext": {}
					}]
				}],
				"cur": "USD"
			}`),
		}

		bidRequest := &openrtb2.BidRequest{ID: "test-request-id", Imp: []openrtb2.Imp{{ID: "test-imp-id"}}}
		bidResponse, errs := adapter.MakeBids(bidRequest, nil, responseData)

		assert.NotNil(t, bidResponse, "expected bid response")
		assert.NotEmpty(t, errs, "expected error in media type parsing")
		assert.Empty(t, bidResponse.Bids, "expected no valid bids")
	})
}

func int64Ptr(i int64) *int64 {
	return &i
}

func jsonRawExt(jsonStr string) json.RawMessage {
	return json.RawMessage(jsonStr)
}
