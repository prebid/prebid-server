package aps

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
	"strconv"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/openrtb/v20/openrtb3"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyAPSResponse(t *testing.T) {
	tests := []struct {
		name        string
		rctx        models.RequestCtx
		bidResponse *openrtb2.BidResponse
		expected    *openrtb2.BidResponse
		description string
	}{
		{
			name: "Non-APS endpoint should return original response",
			rctx: models.RequestCtx{
				Endpoint: models.EndpointGoogleSDK,
				APS:      models.APS{Reject: false},
			},
			bidResponse: &openrtb2.BidResponse{
				ID:    "test-id",
				BidID: "bid-id",
				Cur:   "USD",
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{
							{ID: "bid-1", AdM: "<ad>test</ad>"},
						},
					},
				},
			},
			expected: &openrtb2.BidResponse{
				ID:    "test-id",
				BidID: "bid-id",
				Cur:   "USD",
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{
							{ID: "bid-1", AdM: "<ad>test</ad>"},
						},
					},
				},
			},
			description: "When endpoint is not APS, function should return original bidResponse unchanged",
		},
		{
			name: "APS endpoint with NBR should return original response",
			rctx: models.RequestCtx{
				Endpoint: models.EndpointAPS,
				APS:      models.APS{Reject: false},
			},
			bidResponse: &openrtb2.BidResponse{
				ID:    "test-id",
				BidID: "bid-id",
				Cur:   "USD",
				NBR:   openrtb3.NoBidUnknownError.Ptr(),
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{
							{ID: "bid-1", AdM: "<ad>test</ad>"},
						},
					},
				},
			},
			expected: &openrtb2.BidResponse{
				ID:    "test-id",
				BidID: "bid-id",
				Cur:   "USD",
				NBR:   openrtb3.NoBidUnknownError.Ptr(),
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{
							{ID: "bid-1", AdM: "<ad>test</ad>"},
						},
					},
				},
			},
			description: "When NBR is not nil, function should return original bidResponse unchanged",
		},
		{
			name: "APS endpoint with Reject flag should return original response",
			rctx: models.RequestCtx{
				Endpoint: models.EndpointAPS,
				APS:      models.APS{Reject: true},
			},
			bidResponse: &openrtb2.BidResponse{
				ID:    "test-id",
				BidID: "bid-id",
				Cur:   "USD",
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{
							{ID: "bid-1", AdM: "<ad>test</ad>"},
						},
					},
				},
			},
			expected: &openrtb2.BidResponse{
				ID:    "test-id",
				BidID: "bid-id",
				Cur:   "USD",
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{
							{ID: "bid-1", AdM: "<ad>test</ad>"},
						},
					},
				},
			},
			description: "When APS.Reject is true, function should return original bidResponse unchanged",
		},
		{
			name: "APS endpoint with empty SeatBid should return original response",
			rctx: models.RequestCtx{
				Endpoint: models.EndpointAPS,
				APS:      models.APS{Reject: false},
			},
			bidResponse: &openrtb2.BidResponse{
				ID:      "test-id",
				BidID:   "bid-id",
				Cur:     "USD",
				SeatBid: []openrtb2.SeatBid{},
			},
			expected: &openrtb2.BidResponse{
				ID:      "test-id",
				BidID:   "bid-id",
				Cur:     "USD",
				SeatBid: []openrtb2.SeatBid{},
			},
			description: "When SeatBid is empty, function should return original bidResponse unchanged",
		},
		{
			name: "APS endpoint with empty Bid array should return original response",
			rctx: models.RequestCtx{
				Endpoint: models.EndpointAPS,
				APS:      models.APS{Reject: false},
			},
			bidResponse: &openrtb2.BidResponse{
				ID:    "test-id",
				BidID: "bid-id",
				Cur:   "USD",
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{},
					},
				},
			},
			expected: &openrtb2.BidResponse{
				ID:    "test-id",
				BidID: "bid-id",
				Cur:   "USD",
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{},
					},
				},
			},
			description: "When Bid array is empty, function should return original bidResponse unchanged",
		},
		{
			name: "Valid APS endpoint should transform response",
			rctx: models.RequestCtx{
				Endpoint: models.EndpointAPS,
				APS:      models.APS{Reject: false},
			},
			bidResponse: &openrtb2.BidResponse{
				ID:    "test-id",
				BidID: "bid-id",
				Cur:   "USD",
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{
							{
								ID:    "bid-1",
								ImpID: "imp-1",
								AdM:   "<ad>test</ad>",
								Price: 1.23,
								Ext:   json.RawMessage(`{"custom": "data"}`),
							},
						},
					},
				},
			},
			expected: &openrtb2.BidResponse{
				ID:    "test-id",
				BidID: "bid-1",
				Cur:   "USD",
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{
							{
								ID:    "bid-1",
								ImpID: "imp-1",
								AdM:   "", // Will be set to compressed response
								Ext:   json.RawMessage(`{"custom": "data"}`),
							},
						},
					},
				},
			},
			description: "Valid APS request should compress response and transform structure; bid Ext is preserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of bidResponse for comparison
			originalResponse := *tt.bidResponse

			result := ApplyAPSResponse(tt.rctx, tt.bidResponse)

			// For the transformation case, we need to handle the compressed AdM specially
			if tt.name == "Valid APS endpoint should transform response" {
				assert.Equal(t, tt.expected.ID, result.ID)
				assert.Equal(t, tt.expected.BidID, result.BidID)
				assert.Equal(t, tt.expected.Cur, result.Cur)
				assert.Len(t, result.SeatBid, 1)
				assert.Len(t, result.SeatBid[0].Bid, 1)
				assert.Equal(t, tt.expected.SeatBid[0].Bid[0].ID, result.SeatBid[0].Bid[0].ID)
				assert.NotEmpty(t, result.SeatBid[0].Bid[0].AdM, "AdM should contain compressed data")
				assert.JSONEq(t, `{"custom": "data"}`, string(result.SeatBid[0].Bid[0].Ext))
			} else {
				// For all other cases, the response should remain unchanged
				assert.Equal(t, &originalResponse, result, tt.description)
			}
		})
	}
}

// TestApplyAPSResponse_AdmRoundTrip decodes bid.adm (gzip+base64) and checks it matches the
// jsoniter-marshaled BidResponse from before the in-place AdM mutation inside getBids.
func TestApplyAPSResponse_AdmRoundTrip(t *testing.T) {
	rctx := models.RequestCtx{
		Endpoint:     models.EndpointAPS,
		APS:          models.APS{Reject: false},
		PubIDStr:     "5890",
		ProfileIDStr: "1234",
	}
	br := &openrtb2.BidResponse{
		ID:    "resp-outer",
		BidID: "legacy-bid-id",
		Cur:   "EUR",
		Ext:   json.RawMessage(`{"prebid":{"auctiontimestamp":123}}`),
		SeatBid: []openrtb2.SeatBid{
			{
				Bid: []openrtb2.Bid{
					{
						ID:    "bid-inner",
						ImpID: "imp-9",
						Price: 2.5,
						AdM:   "<html>creative</html>",
						Ext:   json.RawMessage(`{"x":1}`),
					},
				},
			},
		},
	}
	require.NoError(t, setBidResponseExtForAdm(br, rctx.PubIDStr, rctx.ProfileIDStr))
	wantJSON, err := jsoniter.Marshal(br)
	require.NoError(t, err)

	out := ApplyAPSResponse(rctx, br)
	require.Len(t, out.SeatBid, 1)
	require.Len(t, out.SeatBid[0].Bid, 1)
	assert.Equal(t, "resp-outer", out.ID)
	assert.Equal(t, "bid-inner", out.BidID)
	assert.Equal(t, "EUR", out.Cur)
	assert.JSONEq(t, `{"x":1}`, string(out.SeatBid[0].Bid[0].Ext))

	adm := out.SeatBid[0].Bid[0].AdM
	raw, err := base64.StdEncoding.DecodeString(adm)
	require.NoError(t, err)
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	require.NoError(t, err)
	decodedBytes, err := io.ReadAll(zr)
	require.NoError(t, err)
	require.NoError(t, zr.Close())

	assert.Equal(t, string(wantJSON), string(decodedBytes))
	assert.JSONEq(t, `{"prebid":{"auctiontimestamp":123},"publisherid":"5890","profileid":1234}`, extractBidResponseExt(t, decodedBytes))
}

func extractBidResponseExt(t *testing.T, decodedBytes []byte) string {
	t.Helper()
	var decoded openrtb2.BidResponse
	require.NoError(t, json.Unmarshal(decodedBytes, &decoded))
	return string(decoded.Ext)
}

func TestSetBidResponseExtForAdm(t *testing.T) {
	tests := []struct {
		name        string
		bidResponse *openrtb2.BidResponse
		publisherID string
		profileID   string
		expectedExt string
		wantErr     bool
		description string
	}{
		{
			name: "sets publisherid as string and profileid as integer",
			bidResponse: &openrtb2.BidResponse{
				Ext: json.RawMessage(`{"prebid":{"auctiontimestamp":123}}`),
			},
			publisherID: "5890",
			profileID:   "1234",
			expectedExt: `{"prebid":{"auctiontimestamp":123},"publisherid":"5890","profileid":1234}`,
			description: "profileid must be encoded as a JSON number",
		},
		{
			name:        "initializes empty ext before setting fields",
			bidResponse: &openrtb2.BidResponse{},
			publisherID: "5890",
			profileID:   "1234",
			expectedExt: `{"publisherid":"5890","profileid":1234}`,
			description: "empty ext should be initialized before fields are set",
		},
		{
			name: "returns error when ext is not a json object",
			bidResponse: &openrtb2.BidResponse{
				Ext: json.RawMessage(`[]`),
			},
			publisherID: "5890",
			profileID:   "1234",
			wantErr:     true,
			description: "non-object ext should return an error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := setBidResponseExtForAdm(tt.bidResponse, tt.publisherID, tt.profileID)
			if tt.wantErr {
				require.Error(t, err, tt.description)
				return
			}

			require.NoError(t, err, tt.description)
			assert.JSONEq(t, tt.expectedExt, string(tt.bidResponse.Ext), tt.description)

			var ext map[string]any
			require.NoError(t, json.Unmarshal(tt.bidResponse.Ext, &ext))

			publisherID, ok := ext["publisherid"]
			require.True(t, ok)
			assert.IsType(t, "", publisherID, "publisherid must be a JSON string")
			assert.Equal(t, tt.publisherID, publisherID)

			profileID, ok := ext["profileid"]
			require.True(t, ok)
			assert.IsType(t, float64(0), profileID, "profileid must be a JSON number")
			expectedProfileID, parseErr := strconv.Atoi(tt.profileID)
			require.NoError(t, parseErr)
			assert.Equal(t, float64(expectedProfileID), profileID)
		})
	}
}

func TestGetBids(t *testing.T) {
	rctx := models.RequestCtx{
		PubIDStr:     "5890",
		ProfileIDStr: "1234",
	}
	tests := []struct {
		name        string
		bidResponse *openrtb2.BidResponse
		expectedLen int
		description string
	}{
		{
			name: "Valid bid response should return one bid",
			bidResponse: &openrtb2.BidResponse{
				ID:    "test-id",
				BidID: "bid-id",
				Cur:   "USD",
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{
							{
								ID:    "bid-1",
								ImpID: "imp-1",
								AdM:   "<ad>test</ad>",
								Price: 1.23,
							},
						},
					},
				},
			},
			expectedLen: 1,
			description: "Valid response should return exactly one bid",
		},
		{
			name: "Empty bid response should return nil",
			bidResponse: &openrtb2.BidResponse{
				ID:      "test-id",
				BidID:   "bid-id",
				Cur:     "USD",
				SeatBid: []openrtb2.SeatBid{},
			},
			expectedLen: 0,
			description: "Empty SeatBid should return nil",
		},
		{
			name: "Bid response with empty bids should return nil",
			bidResponse: &openrtb2.BidResponse{
				ID:    "test-id",
				BidID: "bid-id",
				Cur:   "USD",
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{},
					},
				},
			},
			expectedLen: 0,
			description: "Empty Bid array should return nil",
		},
		{
			name: "Complex bid response should compress properly",
			bidResponse: &openrtb2.BidResponse{
				ID:    "test-id",
				BidID: "bid-id",
				Cur:   "USD",
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{
							{
								ID:    "bid-1",
								AdM:   `<VAST version="4.2"><Ad><InLine><Creatives><Creative><Linear><MediaFiles><MediaFile delivery="progressive" type="application/javascript" width="300" height="250">console.log("test");</MediaFile></MediaFiles></Linear></Creative></Creatives></InLine></Ad></VAST>`,
								Price: 1.23,
								Ext:   json.RawMessage(`{"custom": "data"}`),
							},
						},
					},
				},
			},
			expectedLen: 1,
			description: "Complex AdM should be compressed; bid Ext is preserved",
		},
		{
			name: "Invalid ext should return nil",
			bidResponse: &openrtb2.BidResponse{
				Ext: json.RawMessage(`[]`),
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{
							{ID: "bid-1", AdM: "<ad>test</ad>"},
						},
					},
				},
			},
			expectedLen: 0,
			description: "Non-object ext should fail setBidResponseExtForAdm and return nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBids(rctx, tt.bidResponse)

			if tt.expectedLen == 0 {
				assert.Nil(t, result, tt.description)
			} else {
				assert.Len(t, result, tt.expectedLen, tt.description)
				if len(result) > 0 {
					assert.Equal(t, tt.bidResponse.SeatBid[0].Bid[0].ID, result[0].ID)
					assert.NotEmpty(t, result[0].AdM, "AdM should contain compressed data")
					assert.Equal(t, tt.bidResponse.SeatBid[0].Bid[0].Ext, result[0].Ext, "Ext should be preserved from source bid")
				}
			}
		})
	}
}

func TestCompressResponse(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		description string
	}{
		{
			name:        "Empty data should be base64 encoded",
			data:        []byte{},
			description: "Empty input should be base64 encoded",
		},
		{
			name:        "Simple data should be compressed",
			data:        []byte("simple test data"),
			description: "Simple string should be gzip compressed and base64 encoded",
		},
		{
			name:        "Large data should be compressed",
			data:        []byte("this is a longer string that should benefit from gzip compression to test the compression functionality"),
			description: "Larger string should be effectively compressed",
		},
		{
			name:        "JSON data should be compressed",
			data:        []byte(`{"test": "data", "number": 123, "array": [1, 2, 3]}`),
			description: "JSON data should be compressed and base64 encoded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compressResponse(tt.data)

			// Result should be base64 encoded
			assert.NotEmpty(t, result, "Compressed result should not be empty")

			// Verify it's valid base64
			decoded, err := base64.StdEncoding.DecodeString(string(result))
			assert.NoError(t, err, "Result should be valid base64")

			// Decompress and compare to original (gzip path); empty input yields empty gzip payload
			decompressed, err := gzip.NewReader(bytes.NewReader(decoded))
			require.NoError(t, err)
			defer decompressed.Close()
			decompressedBytes, err := io.ReadAll(decompressed)
			require.NoError(t, err)
			assert.Equal(t, tt.data, decompressedBytes, "Decompressed data should match original")
		})
	}
}

func TestSetAPSResponseReject(t *testing.T) {
	tests := []struct {
		name        string
		rctx        models.RequestCtx
		bidResponse *openrtb2.BidResponse
		expected    bool
		description string
	}{
		{
			name: "Non-APS endpoint should return false",
			rctx: models.RequestCtx{
				Endpoint: models.EndpointGoogleSDK,
			},
			bidResponse: &openrtb2.BidResponse{
				SeatBid: []openrtb2.SeatBid{
					{Bid: []openrtb2.Bid{{ID: "bid-1"}}},
				},
			},
			expected:    false,
			description: "Non-APS endpoint should always return false",
		},
		{
			name: "APS endpoint with NBR and debug=false should return true",
			rctx: models.RequestCtx{
				Endpoint: models.EndpointAPS,
				Debug:    false,
			},
			bidResponse: &openrtb2.BidResponse{
				NBR: openrtb3.NoBidUnknownError.Ptr(),
				SeatBid: []openrtb2.SeatBid{
					{Bid: []openrtb2.Bid{{ID: "bid-1"}}},
				},
			},
			expected:    true,
			description: "NBR with debug=false should reject",
		},
		{
			name: "APS endpoint with NBR and debug=true should return false",
			rctx: models.RequestCtx{
				Endpoint: models.EndpointAPS,
				Debug:    true,
			},
			bidResponse: &openrtb2.BidResponse{
				NBR: openrtb3.NoBidUnknownError.Ptr(),
				SeatBid: []openrtb2.SeatBid{
					{Bid: []openrtb2.Bid{{ID: "bid-1"}}},
				},
			},
			expected:    false,
			description: "NBR with debug=true should not reject",
		},
		{
			name: "APS endpoint with empty SeatBid should return true",
			rctx: models.RequestCtx{
				Endpoint: models.EndpointAPS,
				Debug:    false,
			},
			bidResponse: &openrtb2.BidResponse{
				SeatBid: []openrtb2.SeatBid{},
			},
			expected:    true,
			description: "Empty SeatBid should reject",
		},
		{
			name: "APS endpoint with empty Bid array should return true",
			rctx: models.RequestCtx{
				Endpoint: models.EndpointAPS,
				Debug:    false,
			},
			bidResponse: &openrtb2.BidResponse{
				SeatBid: []openrtb2.SeatBid{
					{Bid: []openrtb2.Bid{}},
				},
			},
			expected:    true,
			description: "Empty Bid array should reject",
		},
		{
			name: "Valid APS response should return false",
			rctx: models.RequestCtx{
				Endpoint: models.EndpointAPS,
				Debug:    false,
			},
			bidResponse: &openrtb2.BidResponse{
				SeatBid: []openrtb2.SeatBid{
					{Bid: []openrtb2.Bid{{ID: "bid-1"}}},
				},
			},
			expected:    false,
			description: "Valid response should not reject",
		},
		{
			name: "APS endpoint first seat with nil Bid slice should return true",
			rctx: models.RequestCtx{
				Endpoint: models.EndpointAPS,
				Debug:    false,
			},
			bidResponse: &openrtb2.BidResponse{
				SeatBid: []openrtb2.SeatBid{
					{Seat: "pubmatic"},
				},
			},
			expected:    true,
			description: "Nil Bid slice on first seat is treated as empty",
		},
		{
			name: "APS endpoint first seat empty bids second seat ignored should return true",
			rctx: models.RequestCtx{
				Endpoint: models.EndpointAPS,
				Debug:    false,
			},
			bidResponse: &openrtb2.BidResponse{
				SeatBid: []openrtb2.SeatBid{
					{Bid: []openrtb2.Bid{}},
					{Bid: []openrtb2.Bid{{ID: "bid-on-second-seat"}}},
				},
			},
			expected:    true,
			description: "Only first seat is inspected for bids",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SetAPSResponseReject(tt.rctx, tt.bidResponse)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}
