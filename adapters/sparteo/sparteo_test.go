package sparteo

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func int64Ptr(i int64) *int64 {
	return &i
}

// TestBuilder verifies that the Builder function correctly creates a bidder instance.
// It checks for errors, ensures the returned bidder is not nil, and confirms that the endpoint
// in the adapter is set according to the configuration.
func TestBuilder(t *testing.T) {
	cfg := config.Adapter{Endpoint: "https://test-bid.sparteo.com/auction"}
	bidder, err := Builder(openrtb_ext.BidderName("sparteo"), cfg, config.Server{})

	require.NoError(t, err, "Builder returned an error")
	assert.NotNil(t, bidder, "Bidder is nil")

	sparteoAdapter, ok := bidder.(*adapter)
	require.True(t, ok, "Expected *adapter, got %T", bidder)

	assert.Equal(t, "https://test-bid.sparteo.com/auction", sparteoAdapter.endpoint, "Endpoint is not correctly set")
}

// TestMakeRequests_NoImpressions checks that MakeRequests returns no valid request and produces an error
// when the bid request does not include any impressions.
func TestMakeRequests_NoImpressions(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderName("sparteo"), config.Adapter{}, config.Server{})
	req := &openrtb2.BidRequest{}

	requests, errs := bidder.MakeRequests(req, &adapters.ExtraRequestInfo{})
	assert.Empty(t, requests, "Expected no requests")
	assert.NotEmpty(t, errs, "Expected an error")
}

// TestMakeRequests_Valid verifies that MakeRequests processes a valid bid request correctly.
// It checks that the method sets the right HTTP method, URL, and that the request body
// is properly modified with the required bidder parameters.
func TestMakeRequests_Valid(t *testing.T) {
	bidder, _ := Builder(
		openrtb_ext.BidderName("sparteo"),
		config.Adapter{Endpoint: "https://test-bid.sparteo.com/auction"},
		config.Server{},
	)

	// Create a valid impression with a banner and proper bidder extension (networkId).
	imp := openrtb2.Imp{
		ID: "imp1",
		Banner: &openrtb2.Banner{
			W: int64Ptr(300),
			H: int64Ptr(250),
		},
		Ext: json.RawMessage(`{
            "bidder": {
                "networkId": "net123"
            }
        }`),
	}

	// Build the bid request including a Site with Publisher information.
	req := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{imp},
		Site: &openrtb2.Site{
			Domain: "dev.sparteo.com",
			Publisher: &openrtb2.Publisher{
				Domain: "dev.sparteo.com",
			},
			Page: "dev.sparteo.com",
			Ref:  "dev.sparteo.com",
		},
	}

	// Execute MakeRequests.
	requests, errs := bidder.MakeRequests(req, &adapters.ExtraRequestInfo{})
	assert.NotEmpty(t, requests, "Expected one request")
	assert.Empty(t, errs, "Expected no errors")

	if len(requests) == 1 {
		// Confirm that the HTTP method is POST and the endpoint URL is correct.
		assert.Equal(t, "POST", requests[0].Method, "Method should be POST")
		assert.Contains(t, requests[0].Uri, "sparteo.com", "Endpoint should be the sparteo endpoint")
		assert.Contains(t, requests[0].ImpIDs, "imp1", "ImpID should be included")

		// Unmarshal the modified request body.
		var updatedReq openrtb2.BidRequest
		err := json.Unmarshal(requests[0].Body, &updatedReq)
		require.NoError(t, err, "Failed to unmarshal request body")
		require.Len(t, updatedReq.Imp, 1, "Expected 1 imp in the updated request")

		// Verify that the imp extension was updated with a new "sparteo" object.
		var impExtMap map[string]interface{}
		err = json.Unmarshal(updatedReq.Imp[0].Ext, &impExtMap)
		require.NoError(t, err, "Failed to unmarshal updated imp.Ext")

		// Validate that the Site's Publisher extension has been updated with the networkId.
		require.NotNil(t, updatedReq.Site, "Expected site to be non-nil")
		require.NotNil(t, updatedReq.Site.Publisher, "Expected publisher to be non-nil")
		require.NotNil(t, updatedReq.Site.Publisher.Ext, "Expected publisher ext to be set")

		var pubExt map[string]interface{}
		err = json.Unmarshal(updatedReq.Site.Publisher.Ext, &pubExt)
		require.NoError(t, err, "Failed to unmarshal publisher ext")

		pubParams, ok := pubExt["params"].(map[string]interface{})
		require.True(t, ok, "Expected 'params' object in publisher ext")
		assert.Equal(t, "net123", pubParams["networkId"], "Expected networkId from bidder ext to be set in publisher ext")
	}
}

// TestMakeRequests_InvalidBidderExt verifies that if the bidder extension is not formatted correctly (i.e. not an object),
// then MakeRequests returns an error during parsing.
func TestMakeRequests_InvalidBidderExt(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderName("sparteo"), config.Adapter{}, config.Server{})

	imp := openrtb2.Imp{
		ID:     "imp1",
		Banner: &openrtb2.Banner{W: int64Ptr(300), H: int64Ptr(250)},
		Ext: json.RawMessage(`{
            "bidder": "not an object"
        }`),
	}

	req := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{imp},
	}

	_, errs := bidder.MakeRequests(req, &adapters.ExtraRequestInfo{})
	require.NotEmpty(t, errs, "Expected an error from parseExt because bidder is invalid")
	assert.Contains(t, errs[0].Error(), "error while decoding impExt")
}

// TestMakeBids_NoContent checks that when the HTTP response has a 204 (No Content) status,
// MakeBids returns no bid response and no errors.
func TestMakeBids_NoContent(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderName("sparteo"), config.Adapter{}, config.Server{})

	req := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{
			{
				ID:     "imp1",
				Banner: &openrtb2.Banner{W: int64Ptr(300), H: int64Ptr(250)},
			},
		},
	}

	respData := &adapters.ResponseData{
		StatusCode: http.StatusNoContent,
	}

	bidResponse, errs := bidder.MakeBids(req, nil, respData)
	assert.Nil(t, bidResponse, "No response expected")
	assert.Empty(t, errs, "No errors expected")
}

// TestMakeBids_BadRequest verifies that MakeBids returns an error when the response has a 400 (Bad Request) status.
func TestMakeBids_BadRequest(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderName("sparteo"), config.Adapter{}, config.Server{})
	req := &openrtb2.BidRequest{}

	respData := &adapters.ResponseData{
		StatusCode: http.StatusBadRequest,
	}

	bidResponse, errs := bidder.MakeBids(req, nil, respData)
	assert.Nil(t, bidResponse)
	assert.NotEmpty(t, errs, "Expected an error for 400 status")
}

// TestMakeBids_BadServerResponse tests that a server error (HTTP 500) results in an error and no bid response.
func TestMakeBids_BadServerResponse(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderName("sparteo"), config.Adapter{}, config.Server{})
	req := &openrtb2.BidRequest{}

	respData := &adapters.ResponseData{
		StatusCode: 500,
	}

	bidResponse, errs := bidder.MakeBids(req, nil, respData)
	assert.Nil(t, bidResponse)
	assert.NotEmpty(t, errs, "Expected an error for 500 status")
}

// TestMakeBids_ValidResponse ensures that a valid bid response is parsed correctly.
// It checks that bids are assigned the correct bid types based on the impression type (banner vs. video).
func TestMakeBids_ValidResponse(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderName("sparteo"), config.Adapter{}, config.Server{})

	req := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{
			{ID: "banner-imp", Banner: &openrtb2.Banner{W: int64Ptr(300), H: int64Ptr(250)}},
			{ID: "video-imp", Video: &openrtb2.Video{W: int64Ptr(640), H: int64Ptr(360)}},
		},
	}

	bids := []openrtb2.Bid{
		{
			ID:    "bid1",
			ImpID: "banner-imp",
			Price: 1.0,
			Ext:   json.RawMessage(`{"prebid":{"type":"banner"}}`),
		},
		{
			ID:    "bid2",
			ImpID: "video-imp",
			Price: 2.0,
			Ext:   json.RawMessage(`{"prebid":{"type":"video"}}`),
		},
	}

	seatBid := []openrtb2.SeatBid{
		{Bid: bids},
	}

	bidResp := openrtb2.BidResponse{
		Cur:     "EUR",
		SeatBid: seatBid,
	}

	respBody, err := json.Marshal(bidResp)
	require.NoError(t, err)

	respData := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       respBody,
	}

	bidResponse, errs := bidder.MakeBids(req, nil, respData)
	assert.Empty(t, errs, "Expected no errors")
	assert.NotNil(t, bidResponse, "Expected a valid bid response")
	assert.Equal(t, "EUR", bidResponse.Currency)

	if len(bidResponse.Bids) == 2 {
		assert.Equal(t, openrtb_ext.BidTypeBanner, bidResponse.Bids[0].BidType, "Expected first bid to be Banner")
		assert.Equal(t, openrtb_ext.BidTypeVideo, bidResponse.Bids[1].BidType, "Expected second bid to be Video")
	} else {
		t.Errorf("Expected 2 bids, got %d", len(bidResponse.Bids))
	}
}

// TestMakeBids_FilterInvalidBid verifies that MakeBids filters out bids
// that have an invalid or unknown bid type extension.
// In this test, a bid with an unknown type "foobar" is provided,
// and it should be filtered out (i.e. not included in the bid response).
func TestMakeBids_FilterInvalidBid(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderName("sparteo"), config.Adapter{}, config.Server{})

	req := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{
			{ID: "imp1", Banner: &openrtb2.Banner{W: int64Ptr(300), H: int64Ptr(250)}},
		},
	}

	bids := []openrtb2.Bid{
		{
			ID:    "bid-invalid",
			ImpID: "imp1",
			Price: 1.0,
			Ext:   json.RawMessage(`{"prebid":{"type":"foobar"}}`),
		},
	}

	seatBid := []openrtb2.SeatBid{
		{Bid: bids},
	}

	bidResp := openrtb2.BidResponse{
		Cur:     "USD",
		SeatBid: seatBid,
	}

	respBody, err := json.Marshal(bidResp)
	require.NoError(t, err)

	respData := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       respBody,
	}

	bidResponse, errs := bidder.MakeBids(req, nil, respData)

	assert.Empty(t, errs, "Expected no errors when filtering out invalid bids")
	assert.NotNil(t, bidResponse, "Expected a valid bid response")
	assert.Equal(t, "USD", bidResponse.Currency)
	assert.Len(t, bidResponse.Bids, 0, "Expected no bids to be returned because the bid type is unknown")
}

// TestMakeBids_EmptySeatBids tests that an empty seat bid array in the response produces no errors
// and returns a valid (but empty) bid response.
func TestMakeBids_EmptySeatBids(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderName("sparteo"), config.Adapter{}, config.Server{})

	req := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{
			{ID: "imp1", Banner: &openrtb2.Banner{W: int64Ptr(300), H: int64Ptr(250)}},
		},
	}

	bidResp := openrtb2.BidResponse{
		Cur:     "EUR",
		SeatBid: []openrtb2.SeatBid{},
	}

	respBody, _ := json.Marshal(bidResp)
	respData := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       respBody,
	}

	bidResponse, errs := bidder.MakeBids(req, nil, respData)
	assert.Empty(t, errs, "No errors expected from an empty seatBid")
	assert.NotNil(t, bidResponse, "Should still get a BidderResponse object, though it will have no bids")
	assert.Empty(t, bidResponse.Bids, "No bids expected since seatBid was empty")
}

// TestMakeBids_SeatBidNoBids ensures that if a SeatBid exists but its Bid array is empty,
// the adapter returns a valid response with zero bids.
func TestMakeBids_SeatBidNoBids(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderName("sparteo"), config.Adapter{}, config.Server{})

	req := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{
			{ID: "imp1", Banner: &openrtb2.Banner{W: int64Ptr(300), H: int64Ptr(250)}},
		},
	}

	bidResp := openrtb2.BidResponse{
		Cur: "EUR",
		SeatBid: []openrtb2.SeatBid{
			{Bid: []openrtb2.Bid{}},
		},
	}

	respBody, _ := json.Marshal(bidResp)
	respData := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       respBody,
	}

	bidResponse, errs := bidder.MakeBids(req, nil, respData)
	assert.Empty(t, errs)
	assert.NotNil(t, bidResponse)
	assert.Empty(t, bidResponse.Bids, "Expect zero typed bids because seatBid[0].Bid is empty")
}

// TestMakeBids_UnknownImpID verifies that MakeBids handles a bid with an unknown impID
// by using the bid's extension to determine the bid type (falling back to banner).
// In this test, the bid's ImpID ("imp2") does not match any imp in the request (which only has "imp1"),
// but because a valid extension is provided, the adapter should still process the bid and assign it a banner type.
func TestMakeBids_UnknownImpID(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderName("sparteo"), config.Adapter{}, config.Server{})

	req := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{
			{ID: "imp1", Banner: &openrtb2.Banner{W: int64Ptr(300), H: int64Ptr(250)}},
		},
	}

	seatBid := []openrtb2.SeatBid{
		{
			Bid: []openrtb2.Bid{
				{
					ID:    "bid1",
					ImpID: "imp2",
					Price: 1.0,
					Ext:   json.RawMessage(`{"prebid":{"type": "banner"}}`),
				},
			},
		},
	}

	bidResp := openrtb2.BidResponse{Cur: "EUR", SeatBid: seatBid}
	respBody, _ := json.Marshal(bidResp)
	respData := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       respBody,
	}

	bidResponse, errs := bidder.MakeBids(req, nil, respData)
	require.Empty(t, errs, "Should not throw an error for unknown impID; it should fallback to banner")
	require.Len(t, bidResponse.Bids, 1, "Expected one bid even for an unknown impID")
	assert.Equal(t, openrtb_ext.BidTypeBanner, bidResponse.Bids[0].BidType, "Expected fallback bid type Banner for unknown impID")
}

// TestParseExt_Success tests that parseExt correctly decodes a valid bidder extension from an impression.
func TestParseExt_Success(t *testing.T) {
	imp := openrtb2.Imp{
		ID:  "imp1",
		Ext: json.RawMessage(`{"bidder":{"networkId":"netABC"}}`),
	}
	res, err := parseExt(&imp)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, "netABC", res.NetworkId)
}

// TestParseExt_FailDecodeBidderExt ensures that parseExt returns an error when the JSON in the impression extension is invalid.
func TestParseExt_FailDecodeBidderExt(t *testing.T) {
	imp := openrtb2.Imp{
		ID:  "imp1",
		Ext: json.RawMessage(`this: definitely-not-valid-json`),
	}
	res, err := parseExt(&imp)
	assert.Nil(t, res)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error while decoding extImpBidder")
}

// TestGetNetworkId_HasNetworkId verifies that the networkId is correctly extracted when it is provided.
func TestGetNetworkId_HasNetworkId(t *testing.T) {
	imp := openrtb2.Imp{
		ID:  "impWithNetwork",
		Ext: json.RawMessage(`{"bidder": {"networkId": "netXYZ"}}`),
	}
	extImp, err := parseExt(&imp)
	require.NoError(t, err)
	assert.Equal(t, "netXYZ", extImp.NetworkId)
}

// TestGetNetworkId_MissingNetworkId checks that if networkId is missing from the bidder extension,
// it is returned as empty.
func TestGetNetworkId_MissingNetworkId(t *testing.T) {
	imp := openrtb2.Imp{
		ID:  "impWithoutNetwork",
		Ext: json.RawMessage(`{"bidder": {}}`),
	}
	extImp, err := parseExt(&imp)
	require.NoError(t, err)
	assert.Empty(t, extImp.NetworkId, "Expected networkId to be empty when missing")
}

// TestMakeBids_InvalidJSONResponse tests that when the response body is invalid JSON,
// MakeBids returns an error and no bid response.
func TestMakeBids_InvalidJSONResponse(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderName("sparteo"), config.Adapter{}, config.Server{})

	req := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{
			{
				ID:     "imp1",
				Banner: &openrtb2.Banner{W: int64Ptr(300), H: int64Ptr(250)},
			},
		},
	}

	respData := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte("this is not valid json"),
	}

	bidResponse, errs := bidder.MakeBids(req, nil, respData)
	require.NotNil(t, errs, "Expected an error when response body is invalid JSON")
	assert.Nil(t, bidResponse, "Expected no bid response on invalid JSON")
}

// TestGetMediaType_Video verifies that getMediaType returns BidTypeVideo
// when the extension JSON contains {"prebid":{"type":"video"}} and no error is returned.
func TestGetMediaType_Video(t *testing.T) {
	adapter := &adapter{}
	bid := &openrtb2.Bid{
		Ext: json.RawMessage(`{"prebid":{"type":"video"}}`),
	}
	result, err := adapter.getMediaType(bid)
	assert.NoError(t, err, "Expected no error for valid video type")
	assert.Equal(t, openrtb_ext.BidTypeVideo, result, "Expected media type to be Video")
}

// TestGetMediaType_Banner verifies that getMediaType returns BidTypeBanner
// when the extension JSON contains {"prebid":{"type":"banner"}} and no error is returned.
func TestGetMediaType_Banner(t *testing.T) {
	adapter := &adapter{}
	bid := &openrtb2.Bid{
		Ext: json.RawMessage(`{"prebid":{"type":"banner"}}`),
	}
	result, err := adapter.getMediaType(bid)
	assert.NoError(t, err, "Expected no error for valid banner type")
	assert.Equal(t, openrtb_ext.BidTypeBanner, result, "Expected media type to be Banner")
}

// TestGetMediaType_Native verifies that getMediaType returns BidTypeNative
// when the extension JSON contains {"prebid":{"type":"native"}} and no error is returned.
func TestGetMediaType_Native(t *testing.T) {
	adapter := &adapter{}
	bid := &openrtb2.Bid{
		Ext: json.RawMessage(`{"prebid":{"type":"native"}}`),
	}
	result, err := adapter.getMediaType(bid)
	assert.NoError(t, err, "Expected no error for valid native type")
	assert.Equal(t, openrtb_ext.BidTypeNative, result, "Expected media type to be Native")
}

// TestGetMediaType_Unknown verifies that getMediaType returns an error and an empty result
// when the extension JSON contains an unknown type (i.e. not "video", "banner").
func TestGetMediaType_Unknown(t *testing.T) {
	adapter := &adapter{}
	bid := &openrtb2.Bid{
		Ext: json.RawMessage(`{"prebid":{"type":"audio"}}`),
	}
	result, err := adapter.getMediaType(bid)
	assert.Error(t, err, "Expected error for unknown bid type")
	assert.Equal(t, openrtb_ext.BidType(""), result, "Expected empty result for unknown bid type")
}

// TestGetMediaType_InvalidJSON verifies that getMediaType returns an error and an empty result
// when the extension JSON is invalid.
func TestGetMediaType_InvalidJSON(t *testing.T) {
	adapter := &adapter{}
	bid := &openrtb2.Bid{
		Ext: json.RawMessage(`invalid-json`),
	}
	result, err := adapter.getMediaType(bid)
	assert.Error(t, err, "Expected error for invalid JSON")
	assert.Equal(t, openrtb_ext.BidType(""), result, "Expected empty result for invalid JSON")
}

// TestGetMediaType_EmptyType verifies that getMediaType returns an error and an empty result
// when the extension JSON is valid but the "type" field is empty.
func TestGetMediaType_EmptyType(t *testing.T) {
	adapter := &adapter{}
	bid := &openrtb2.Bid{
		Ext: json.RawMessage(`{"prebid":{"type":""}}`),
	}
	result, err := adapter.getMediaType(bid)
	assert.Error(t, err, "Expected error for empty type")
	assert.Equal(t, openrtb_ext.BidType(""), result, "Expected empty result for empty type")
}

// TestGetMediaType_NilExt verifies that getMediaType returns an error and an empty result
// when the bid's extension is nil.
func TestGetMediaType_NilExt(t *testing.T) {
	adapter := &adapter{}
	bid := &openrtb2.Bid{
		Ext: nil,
	}
	result, err := adapter.getMediaType(bid)
	assert.Error(t, err, "Expected error for nil extension")
	assert.Equal(t, openrtb_ext.BidType(""), result, "Expected empty result for nil extension")
}

// TestMakeRequests_MergeBidderParams verifies that all parameters from ext.bidder
// are merged into ext.sparteo.params and that the original ext.bidder key is removed.
func TestMakeRequests_MergeBidderParams(t *testing.T) {
	bidder, _ := Builder(
		openrtb_ext.BidderName("sparteo"),
		config.Adapter{Endpoint: "https://test-bid.sparteo.com/auction"},
		config.Server{},
	)

	imp := openrtb2.Imp{
		ID: "imp-merge",
		Banner: &openrtb2.Banner{
			W: int64Ptr(300),
			H: int64Ptr(250),
		},
		Ext: json.RawMessage(`{
			"bidder": {
				"custom1": "value1",
				"networkId": "netABC",
				"unknown": "unknownValue"
			}
		}`),
	}

	req := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{imp},
		Site: &openrtb2.Site{
			Domain: "test.sparteo.com",
			Publisher: &openrtb2.Publisher{
				Domain: "test.sparteo.com",
			},
		},
	}

	requests, errs := bidder.MakeRequests(req, &adapters.ExtraRequestInfo{})
	require.Empty(t, errs, "Expected no errors")
	require.Len(t, requests, 1, "Expected one request")

	var updatedReq openrtb2.BidRequest
	err := json.Unmarshal(requests[0].Body, &updatedReq)
	require.NoError(t, err, "Failed to unmarshal request body")
	require.Len(t, updatedReq.Imp, 1, "Expected one imp in the updated request")

	var extMap map[string]interface{}
	err = json.Unmarshal(updatedReq.Imp[0].Ext, &extMap)
	require.NoError(t, err, "Failed to unmarshal updated imp.Ext")

	_, bidderExists := extMap["bidder"]
	assert.False(t, bidderExists, "ext should not contain the 'bidder' key after merge")

	sparteoMap, ok := extMap["sparteo"].(map[string]interface{})
	require.True(t, ok, "Expected 'sparteo' object in imp.Ext")
	paramsMap, ok := sparteoMap["params"].(map[string]interface{})
	require.True(t, ok, "Expected 'params' object in ext.sparteo")

	assert.Equal(t, "value1", paramsMap["custom1"], "Expected custom1 to be merged")
	assert.Equal(t, "unknownValue", paramsMap["unknown"], "Expected unknown to be merged")

	var pubExt map[string]interface{}
	err = json.Unmarshal(updatedReq.Site.Publisher.Ext, &pubExt)
	require.NoError(t, err, "Failed to unmarshal publisher ext")
	pubParams, ok := pubExt["params"].(map[string]interface{})
	require.True(t, ok, "Expected 'params' object in publisher ext")
	assert.Equal(t, "netABC", pubParams["networkId"], "Expected publisher params networkId to match merged networkId")
}

// TestMakeRequests_PublisherExt_UnmarshalError verifies that when the publisher extension JSON is invalid (i.e. not an object),
// the adapter resets it to an empty map and correctly merges the networkId from the impression bidder extension.
func TestMakeRequests_PublisherExt_UnmarshalError(t *testing.T) {
	bidder, _ := Builder(
		openrtb_ext.BidderName("sparteo"),
		config.Adapter{Endpoint: "https://test-bid.sparteo.com/auction"},
		config.Server{},
	)

	imp := openrtb2.Imp{
		ID: "imp1",
		Banner: &openrtb2.Banner{
			W: int64Ptr(300),
			H: int64Ptr(250),
		},
		Ext: json.RawMessage(`{"bidder":{"networkId":"netPub"}}`),
	}

	req := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{imp},
		Site: &openrtb2.Site{
			Domain: "test.sparteo.com",
			Publisher: &openrtb2.Publisher{
				Domain: "test.sparteo.com",
				Ext:    json.RawMessage(`"not an object"`),
			},
		},
	}

	requests, _ := bidder.MakeRequests(req, &adapters.ExtraRequestInfo{})

	assert.NotNil(t, requests, "Expected a valid request despite publisher ext issue")
	require.NotNil(t, req.Site.Publisher.Ext, "Publisher ext should be set")

	var pubExt map[string]interface{}
	err := json.Unmarshal(req.Site.Publisher.Ext, &pubExt)
	require.NoError(t, err, "Updated publisher ext should unmarshal")
	params, ok := pubExt["params"].(map[string]interface{})
	require.True(t, ok, "Expected publisher ext 'params' to be a map")
	assert.Equal(t, "netPub", params["networkId"], "Expected networkId to be set from imp bidder ext")
}

// TestMakeRequests_PublisherExt_ParamsNotMap verifies that when the publisher extension's "params" field is not a map,
// the adapter replaces it with a new map and correctly merges the networkId from the bidder extension.
func TestMakeRequests_PublisherExt_ParamsNotMap(t *testing.T) {
	bidder, _ := Builder(
		openrtb_ext.BidderName("sparteo"),
		config.Adapter{Endpoint: "https://test-bid.sparteo.com/auction"},
		config.Server{},
	)

	imp := openrtb2.Imp{
		ID: "imp1",
		Banner: &openrtb2.Banner{
			W: int64Ptr(300),
			H: int64Ptr(250),
		},
		Ext: json.RawMessage(`{"bidder":{"networkId":"net123"}}`),
	}

	req := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{imp},
		Site: &openrtb2.Site{
			Domain: "test.sparteo.com",
			Publisher: &openrtb2.Publisher{
				Domain: "test.sparteo.com",
				Ext:    json.RawMessage(`{"params": "should be an object"}`),
			},
		},
	}

	requests, errs := bidder.MakeRequests(req, &adapters.ExtraRequestInfo{})
	require.Empty(t, errs, "Expected no errors from publisher ext processing")
	require.Len(t, requests, 1, "Expected one request")

	require.NotNil(t, req.Site.Publisher.Ext, "Publisher ext should be set")
	var pubExt map[string]interface{}
	err := json.Unmarshal(req.Site.Publisher.Ext, &pubExt)
	require.NoError(t, err, "Updated publisher ext should unmarshal")
	params, ok := pubExt["params"].(map[string]interface{})
	require.True(t, ok, "Expected publisher ext 'params' to be a map after type correction")
	assert.Equal(t, "net123", params["networkId"], "Expected networkId to be merged into publisher ext")
}