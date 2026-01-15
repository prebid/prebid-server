package iqiyi

import (
	"encoding/json"
	"math"
	"net/http"
	"testing"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderIqiyi, config.Adapter{
		Endpoint: "https://cupid.iqiyi.net/bid?a={{.AccountID}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "iqiyitest", bidder)
}

func TestBuilder(t *testing.T) {
	t.Run("Invalid template syntax", func(t *testing.T) {
		_, err := Builder(openrtb_ext.BidderIqiyi, config.Adapter{
			Endpoint: "{{.InvalidField"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unable to parse endpoint url template")
	})

	t.Run("Template execution error - invalid pipeline function", func(t *testing.T) {
		_, err := Builder(openrtb_ext.BidderIqiyi, config.Adapter{
			Endpoint: "https://cupid.iqiyi.net/bid?a={{.AccountID | nonexistent}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unable to parse endpoint url template")
	})

	t.Run("Valid template", func(t *testing.T) {
		bidder, err := Builder(openrtb_ext.BidderIqiyi, config.Adapter{
			Endpoint: "https://cupid.iqiyi.net/bid?a={{.AccountID}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
		assert.NoError(t, err)
		assert.NotNil(t, bidder)
	})
}

func TestSelectCurrency(t *testing.T) {
	t.Run("Response currency is set", func(t *testing.T) {
		req := &openrtb2.BidRequest{Cur: []string{"EUR"}}
		resp := &openrtb2.BidResponse{Cur: "CNY"}
		result := selectCurrency(req, resp)
		assert.Equal(t, "CNY", result)
	})

	t.Run("Request currency is set when response currency is empty", func(t *testing.T) {
		req := &openrtb2.BidRequest{Cur: []string{"EUR"}}
		resp := &openrtb2.BidResponse{Cur: ""}
		result := selectCurrency(req, resp)
		assert.Equal(t, "EUR", result)
	})

	t.Run("Default to USD when both are empty", func(t *testing.T) {
		req := &openrtb2.BidRequest{Cur: []string{}}
		resp := &openrtb2.BidResponse{Cur: ""}
		result := selectCurrency(req, resp)
		assert.Equal(t, "USD", result)
	})

	t.Run("Default to USD when request currency is empty string", func(t *testing.T) {
		req := &openrtb2.BidRequest{Cur: []string{""}}
		resp := &openrtb2.BidResponse{Cur: ""}
		result := selectCurrency(req, resp)
		assert.Equal(t, "USD", result)
	})
}

func TestMakeRequests(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderIqiyi, config.Adapter{
		Endpoint: "https://cupid.iqiyi.net/bid?a={{.AccountID}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	t.Run("Invalid ext JSON", func(t *testing.T) {
		request := &openrtb2.BidRequest{
			ID: "test-request-id",
			Imp: []openrtb2.Imp{{
				ID:  "test-imp-id",
				Ext: json.RawMessage(`invalid json`),
			}},
		}
		reqs, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})
		assert.Nil(t, reqs)
		assert.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "error unmarshalling impression ext")
	})

	t.Run("Invalid bidder ext JSON", func(t *testing.T) {
		request := &openrtb2.BidRequest{
			ID: "test-request-id",
			Imp: []openrtb2.Imp{{
				ID:  "test-imp-id",
				Ext: json.RawMessage(`{"bidder":"invalid json"}`),
			}},
		}
		reqs, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})
		assert.Nil(t, reqs)
		assert.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "error unmarshalling Iqiyi bidder params")
	})

	t.Run("JSON marshal error", func(t *testing.T) {
		request := &openrtb2.BidRequest{
			ID: "test-request-id",
			Imp: []openrtb2.Imp{{
				ID:       "test-imp-id",
				Ext:      json.RawMessage(`{"bidder":{"accountid":"test-account"}}`),
				BidFloor: math.Inf(1), // Cannot be marshalled
			}},
		}
		reqs, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})
		assert.Nil(t, reqs)
		assert.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "json")
	})

	t.Run("Banner with missing dimensions uses Format", func(t *testing.T) {
		w := int64(0)
		h := int64(0)
		request := &openrtb2.BidRequest{
			ID: "test-request-id",
			Imp: []openrtb2.Imp{{
				ID:  "test-imp-id",
				Ext: json.RawMessage(`{"bidder":{"accountid":"test-account"}}`),
				Banner: &openrtb2.Banner{
					W: &w,
					H: &h,
					Format: []openrtb2.Format{
						{W: 320, H: 50},
					},
				},
			}},
		}
		reqs, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})
		assert.Empty(t, errs)
		assert.NotNil(t, reqs)
		assert.Len(t, reqs, 1)
		// Verify that Banner dimensions were set from Format
		var unmarshaledRequest openrtb2.BidRequest
		json.Unmarshal(reqs[0].Body, &unmarshaledRequest)
		assert.NotNil(t, unmarshaledRequest.Imp[0].Banner)
		assert.Equal(t, int64(320), *unmarshaledRequest.Imp[0].Banner.W)
		assert.Equal(t, int64(50), *unmarshaledRequest.Imp[0].Banner.H)
	})

	t.Run("BidFloorCur set to USD when empty and BidFloor > 0", func(t *testing.T) {
		request := &openrtb2.BidRequest{
			ID: "test-request-id",
			Imp: []openrtb2.Imp{{
				ID:          "test-imp-id",
				Ext:         json.RawMessage(`{"bidder":{"accountid":"test-account"}}`),
				BidFloor:    1.5,
				BidFloorCur: "",
			}},
		}
		reqs, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})
		assert.Empty(t, errs)
		assert.NotNil(t, reqs)
		assert.Len(t, reqs, 1)
		// Verify that BidFloorCur was set to USD
		var unmarshaledRequest openrtb2.BidRequest
		json.Unmarshal(reqs[0].Body, &unmarshaledRequest)
		assert.Equal(t, "USD", unmarshaledRequest.Imp[0].BidFloorCur)
	})

	t.Run("buildEndpointURL error", func(t *testing.T) {
		// Create a template that will fail during Execute by using a function that doesn't exist
		// We need to use a template that parses successfully but fails during execution
		// The simplest way is to create a template with a custom function map that's empty,
		// then use a function that requires custom definition
		// However, since we can't use undefined functions in Parse, we'll use a different approach:
		// Create a valid template first, then we'll manually trigger an error by using a nil template
		// Actually, let's use a template that will cause Execute to fail by trying to call a method
		invalidTemplate, err := template.New("invalid").Parse("https://example.com/{{call .AccountID \"method\"}}")
		if err != nil {
			// If call syntax doesn't work, create a template that will fail during Execute
			// by trying to use a function that requires arguments in wrong way
			invalidTemplate, err = template.New("invalid").Parse("https://example.com/{{printf}}")
			if err != nil {
				t.Fatalf("Failed to create template: %v", err)
			}
		}

		badBidder := &adapter{
			endpoint: invalidTemplate,
		}

		request := &openrtb2.BidRequest{
			ID: "test-request-id",
			Imp: []openrtb2.Imp{{
				ID:  "test-imp-id",
				Ext: json.RawMessage(`{"bidder":{"accountid":"test-account"}}`),
			}},
		}
		reqs, errs := badBidder.MakeRequests(request, &adapters.ExtraRequestInfo{})
		assert.Nil(t, reqs)
		assert.Len(t, errs, 1)
		assert.NotNil(t, errs[0])
	})
}

func TestMakeBids(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderIqiyi, config.Adapter{
		Endpoint: "https://cupid.iqiyi.net/bid?a={{.AccountID}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	internalRequest := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "test-imp-id",
		}},
	}

	t.Run("StatusNoContent", func(t *testing.T) {
		response := &adapters.ResponseData{
			StatusCode: http.StatusNoContent,
			Body:       []byte{},
		}
		bidResponse, errs := bidder.MakeBids(internalRequest, &adapters.RequestData{}, response)
		assert.Nil(t, bidResponse)
		assert.Nil(t, errs)
	})

	t.Run("Unexpected status code", func(t *testing.T) {
		response := &adapters.ResponseData{
			StatusCode: http.StatusBadRequest,
			Body:       []byte{},
		}
		bidResponse, errs := bidder.MakeBids(internalRequest, &adapters.RequestData{}, response)
		assert.Nil(t, bidResponse)
		assert.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "Unexpected http status code")
	})

	t.Run("Invalid JSON response", func(t *testing.T) {
		response := &adapters.ResponseData{
			StatusCode: http.StatusOK,
			Body:       []byte(`invalid json`),
		}
		bidResponse, errs := bidder.MakeBids(internalRequest, &adapters.RequestData{}, response)
		assert.Nil(t, bidResponse)
		assert.Len(t, errs, 1)
	})

	t.Run("Unsupported mtype", func(t *testing.T) {
		response := &adapters.ResponseData{
			StatusCode: http.StatusOK,
			Body: json.RawMessage(`{
				"id": "test-response-id",
				"seatbid": [{
					"bid": [{
						"id": "test-bid-id",
						"impid": "test-imp-id",
						"price": 1.0,
						"mtype": 3
					}]
				}]
			}`),
		}
		bidResponse, errs := bidder.MakeBids(internalRequest, &adapters.RequestData{}, response)
		assert.Nil(t, bidResponse)
		assert.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "Unsupported mtype")
	})

	t.Run("Successful response with Banner bid", func(t *testing.T) {
		response := &adapters.ResponseData{
			StatusCode: http.StatusOK,
			Body: json.RawMessage(`{
				"id": "test-response-id",
				"cur": "CNY",
				"seatbid": [{
					"bid": [{
						"id": "test-bid-id",
						"impid": "test-imp-id",
						"price": 1.0,
						"mtype": 1
					}]
				}]
			}`),
		}
		bidResponse, errs := bidder.MakeBids(internalRequest, &adapters.RequestData{}, response)
		assert.Empty(t, errs)
		assert.NotNil(t, bidResponse)
		assert.Equal(t, "CNY", bidResponse.Currency)
		assert.Len(t, bidResponse.Bids, 1)
		assert.Equal(t, openrtb_ext.BidTypeBanner, bidResponse.Bids[0].BidType)
	})

	t.Run("Successful response with Video bid", func(t *testing.T) {
		response := &adapters.ResponseData{
			StatusCode: http.StatusOK,
			Body: json.RawMessage(`{
				"id": "test-response-id",
				"seatbid": [{
					"bid": [{
						"id": "test-bid-id",
						"impid": "test-imp-id",
						"price": 1.0,
						"mtype": 2
					}]
				}]
			}`),
		}
		internalRequestWithCur := &openrtb2.BidRequest{
			ID:  "test-request-id",
			Cur: []string{"EUR"},
			Imp: []openrtb2.Imp{{
				ID: "test-imp-id",
			}},
		}
		bidResponse, errs := bidder.MakeBids(internalRequestWithCur, &adapters.RequestData{}, response)
		assert.Empty(t, errs)
		assert.NotNil(t, bidResponse)
		assert.Equal(t, "EUR", bidResponse.Currency) // Should use request currency when response currency is empty
		assert.Len(t, bidResponse.Bids, 1)
		assert.Equal(t, openrtb_ext.BidTypeVideo, bidResponse.Bids[0].BidType)
	})

	t.Run("Successful response with Native bid", func(t *testing.T) {
		response := &adapters.ResponseData{
			StatusCode: http.StatusOK,
			Body: json.RawMessage(`{
				"id": "test-response-id",
				"seatbid": [{
					"bid": [{
						"id": "test-bid-id",
						"impid": "test-imp-id",
						"price": 1.0,
						"mtype": 4
					}]
				}]
			}`),
		}
		bidResponse, errs := bidder.MakeBids(internalRequest, &adapters.RequestData{}, response)
		assert.Empty(t, errs)
		assert.NotNil(t, bidResponse)
		assert.Equal(t, "USD", bidResponse.Currency) // Should default to USD
		assert.Len(t, bidResponse.Bids, 1)
		assert.Equal(t, openrtb_ext.BidTypeNative, bidResponse.Bids[0].BidType)
	})
}

func TestGetMediaTypeForImp(t *testing.T) {
	t.Run("Banner type", func(t *testing.T) {
		bid := openrtb2.Bid{
			ID:    "test-bid-id",
			MType: openrtb2.MarkupBanner,
		}
		mediaType, err := getMediaTypeForImp(bid)
		assert.NoError(t, err)
		assert.Equal(t, openrtb_ext.BidTypeBanner, mediaType)
	})

	t.Run("Video type", func(t *testing.T) {
		bid := openrtb2.Bid{
			ID:    "test-bid-id",
			MType: openrtb2.MarkupVideo,
		}
		mediaType, err := getMediaTypeForImp(bid)
		assert.NoError(t, err)
		assert.Equal(t, openrtb_ext.BidTypeVideo, mediaType)
	})

	t.Run("Native type", func(t *testing.T) {
		bid := openrtb2.Bid{
			ID:    "test-bid-id",
			MType: openrtb2.MarkupNative,
		}
		mediaType, err := getMediaTypeForImp(bid)
		assert.NoError(t, err)
		assert.Equal(t, openrtb_ext.BidTypeNative, mediaType)
	})

	t.Run("Unsupported mtype", func(t *testing.T) {
		bid := openrtb2.Bid{
			ID:    "test-bid-id",
			MType: 3,
		}
		mediaType, err := getMediaTypeForImp(bid)
		assert.Error(t, err)
		assert.Equal(t, openrtb_ext.BidType(""), mediaType)
		assert.Contains(t, err.Error(), "Unsupported mtype")
	})
}
