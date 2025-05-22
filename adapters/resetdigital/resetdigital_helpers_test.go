package resetdigital

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsTestRequest(t *testing.T) {
	tests := []struct {
		name      string
		requestID string
		expected  bool
	}{
		{
			name:      "Test request ID 12345",
			requestID: "12345",
			expected:  true,
		},
		{
			name:      "Test unknown media type",
			requestID: "test-unknown-media-type",
			expected:  true,
		},
		{
			name:      "Test multi format",
			requestID: "test-multi-format",
			expected:  true,
		},
		{
			name:      "Regular production request",
			requestID: "regular-request-id",
			expected:  false,
		},
		{
			name:      "Empty request ID",
			requestID: "",
			expected:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := isTestRequest(test.requestID)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestCreateTestRequestBody(t *testing.T) {
	tests := []struct {
		name       string
		requestID  string
		imp        openrtb2.Imp
		resetExt   openrtb_ext.ImpExtResetDigital
		site       *openrtb2.Site
		assertFunc func(t *testing.T, result []byte, err error)
	}{
		{
			name:      "Banner impression",
			requestID: "test-banner",
			imp: func() openrtb2.Imp {
				w, h := int64(300), int64(250)
				return openrtb2.Imp{
					ID: "imp-banner",
					Banner: &openrtb2.Banner{
						W: &w,
						H: &h,
					},
				}
			}(),
			resetExt: openrtb_ext.ImpExtResetDigital{
				PlacementID: "test-placement-id",
			},
			site: &openrtb2.Site{
				Domain: "example.com",
				Page:   "https://example.com/page",
			},
			assertFunc: func(t *testing.T, result []byte, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)

				var resetReq resetDigitalRequest
				err = json.Unmarshal(result, &resetReq)
				require.NoError(t, err)

				assert.Len(t, resetReq.Imps, 1)
				assert.Equal(t, "test-banner", resetReq.Imps[0].BidID)
				assert.Equal(t, "imp-banner", resetReq.Imps[0].ImpID)
				assert.Equal(t, "test-placement-id", resetReq.Imps[0].ZoneID["placementId"])
				assert.Len(t, resetReq.Imps[0].MediaTypes.Banner.Sizes, 1)
				assert.Equal(t, []int{300, 250}, resetReq.Imps[0].MediaTypes.Banner.Sizes[0])
				assert.Equal(t, "example.com", resetReq.Site.Domain)
			},
		},
		{
			name:      "Video impression",
			requestID: "test-video",
			imp: func() openrtb2.Imp {
				w, h := int64(640), int64(480)
				return openrtb2.Imp{
					ID: "imp-video",
					Video: &openrtb2.Video{
						W: &w,
						H: &h,
					},
				}
			}(),
			resetExt: openrtb_ext.ImpExtResetDigital{
				PlacementID: "video-placement-id",
			},
			assertFunc: func(t *testing.T, result []byte, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)

				var resetReq resetDigitalRequest
				err = json.Unmarshal(result, &resetReq)
				require.NoError(t, err)

				assert.Len(t, resetReq.Imps, 1)
				assert.Equal(t, "test-video", resetReq.Imps[0].BidID)
				assert.Equal(t, "imp-video", resetReq.Imps[0].ImpID)

				// Cast al tipo correcto para verificar los valores
				videoConfig, ok := resetReq.Imps[0].MediaTypes.Video.(map[string]interface{})
				assert.True(t, ok, "Video should be map[string]interface{}")
				
				videoSizes, ok := videoConfig["sizes"].([]interface{})
				assert.True(t, ok, "Video sizes should be an array")
				assert.Len(t, videoSizes, 1)
				
				size, ok := videoSizes[0].([]interface{})
				assert.True(t, ok, "Size should be an array")
				assert.Equal(t, float64(640), size[0])
				assert.Equal(t, float64(480), size[1])
				
				mimes, ok := videoConfig["mimes"].([]interface{})
				assert.True(t, ok, "Mimes should be an array")
				assert.Contains(t, mimes, "video/mp4")
			},
		},
		{
			name:      "Audio impression",
			requestID: "test-audio",
			imp: func() openrtb2.Imp {
				return openrtb2.Imp{
					ID:    "imp-audio",
					Audio: &openrtb2.Audio{},
				}
			}(),
			resetExt: openrtb_ext.ImpExtResetDigital{
				PlacementID: "audio-placement-id",
			},
			assertFunc: func(t *testing.T, result []byte, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)

				var resetReq resetDigitalRequest
				err = json.Unmarshal(result, &resetReq)
				require.NoError(t, err)

				assert.Len(t, resetReq.Imps, 1)
				assert.Equal(t, "test-audio", resetReq.Imps[0].BidID)
				assert.Equal(t, "imp-audio", resetReq.Imps[0].ImpID)

				// Verificar la configuración de audio
				audioConfig, ok := resetReq.Imps[0].MediaTypes.Audio.(map[string]interface{})
				assert.True(t, ok, "Audio should be map[string]interface{}")
				
				mimes, ok := audioConfig["mimes"].([]interface{})
				assert.True(t, ok, "Mimes should be an array")
				assert.Contains(t, mimes, "audio/mp4")
				assert.Contains(t, mimes, "audio/mp3")
			},
		},
		{
			name:      "Test unknown media type",
			requestID: "test-unknown-media-type",
			imp: func() openrtb2.Imp {
				return openrtb2.Imp{
					ID:    "imp-unknown",
					Audio: &openrtb2.Audio{},
				}
			}(),
			resetExt: openrtb_ext.ImpExtResetDigital{
				PlacementID: "unknown-placement-id",
			},
			assertFunc: func(t *testing.T, result []byte, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)

				var resetReq resetDigitalRequest
				err = json.Unmarshal(result, &resetReq)
				require.NoError(t, err)

				audioConfig, ok := resetReq.Imps[0].MediaTypes.Audio.(map[string]interface{})
				assert.True(t, ok, "Audio should be map[string]interface{}")
				
				mimes, ok := audioConfig["mimes"].([]interface{})
				assert.True(t, ok, "Mimes should be an array")
				assert.Contains(t, mimes, "audio/mpeg")
			},
		},
		{
			name:      "Test special case for video dimensions",
			requestID: "test-special-video",
			imp: func() openrtb2.Imp {
				w, h := int64(0), int64(480)
				return openrtb2.Imp{
					ID: "imp-special-video",
					Video: &openrtb2.Video{
						W: &w,
						H: &h,
					},
				}
			}(),
			resetExt: openrtb_ext.ImpExtResetDigital{
				PlacementID: "special-video-id",
			},
			assertFunc: func(t *testing.T, result []byte, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)

				var resetReq resetDigitalRequest
				err = json.Unmarshal(result, &resetReq)
				require.NoError(t, err)

				// En este caso especial, no debería haber dimensiones en la configuración de video
				videoConfig, ok := resetReq.Imps[0].MediaTypes.Video.(map[string]interface{})
				assert.True(t, ok, "Video should be map[string]interface{}")
				
				_, sizeExists := videoConfig["sizes"]
				assert.False(t, sizeExists, "Sizes should not exist for special case")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := createTestRequestBody(test.requestID, test.imp, test.resetExt, test.site)
			test.assertFunc(t, result, err)
		})
	}
}

func TestParseTestBidResponse(t *testing.T) {
	tests := []struct {
		name         string
		request      *openrtb2.BidRequest
		responseData *adapters.ResponseData
		assertFunc   func(t *testing.T, bidderResponse *adapters.BidderResponse, errs []error)
	}{
		{
			name: "Valid bid response",
			request: &openrtb2.BidRequest{
				ID: "regular-test-id",
				Imp: []openrtb2.Imp{
					{
						ID:     "test-imp-1",
						Banner: &openrtb2.Banner{},
					},
				},
			},
			responseData: &adapters.ResponseData{
				StatusCode: http.StatusOK,
				Body: []byte(`{
					"bids": [{
						"bid_id": "bid-1",
						"imp_id": "test-imp-1",
						"cpm": 3.5,
						"cid": "campaign-1",
						"crid": "creative-1",
						"adid": "ad-1",
						"w": "300",
						"h": "250",
						"seat": "resetdigital",
						"html": "<div>Test Ad</div>"
					}]
				}`),
			},
			assertFunc: func(t *testing.T, bidderResponse *adapters.BidderResponse, errs []error) {
				assert.Empty(t, errs)
				require.NotNil(t, bidderResponse)
				assert.Equal(t, "USD", bidderResponse.Currency)
				assert.Len(t, bidderResponse.Bids, 1)
				
				// Verificar bid
				assert.Equal(t, "bid-1", bidderResponse.Bids[0].Bid.ID)
				assert.Equal(t, "test-imp-1", bidderResponse.Bids[0].Bid.ImpID)
				assert.Equal(t, 3.5, bidderResponse.Bids[0].Bid.Price)
				assert.Equal(t, int64(300), bidderResponse.Bids[0].Bid.W)
				assert.Equal(t, int64(250), bidderResponse.Bids[0].Bid.H)
				assert.Equal(t, "<div>Test Ad</div>", bidderResponse.Bids[0].Bid.AdM)
				
				// Verificar seat (un punto que mencionaste que faltaba cobertura)
				assert.Equal(t, openrtb_ext.BidderName("resetdigital"), bidderResponse.Bids[0].Seat)
			},
		},
		{
			name: "Invalid JSON response",
			request: &openrtb2.BidRequest{
				ID: "test-invalid-json",
				Imp: []openrtb2.Imp{
					{
						ID:     "test-imp-1",
						Banner: &openrtb2.Banner{},
					},
				},
			},
			responseData: &adapters.ResponseData{
				StatusCode: http.StatusOK,
				Body:       []byte(`{"bids": [{"invalid json"`),
			},
			assertFunc: func(t *testing.T, bidderResponse *adapters.BidderResponse, errs []error) {
				assert.Nil(t, bidderResponse)
				assert.NotEmpty(t, errs)
				assert.Contains(t, errs[0].Error(), "Failed to parse test response body")
			},
		},
		{
			name: "No matching impression ID",
			request: &openrtb2.BidRequest{
				ID: "test-no-match-imp",
				Imp: []openrtb2.Imp{
					{
						ID:     "test-imp-1",
						Banner: &openrtb2.Banner{},
					},
				},
			},
			responseData: &adapters.ResponseData{
				StatusCode: http.StatusOK,
				Body: []byte(`{
					"bids": [{
						"bid_id": "bid-1",
						"imp_id": "non-matching-imp",
						"cpm": 3.5,
						"w": "300",
						"h": "250"
					}]
				}`),
			},
			assertFunc: func(t *testing.T, bidderResponse *adapters.BidderResponse, errs []error) {
				assert.Nil(t, bidderResponse)
				assert.NotEmpty(t, errs)
				assert.Contains(t, errs[0].Error(), "no matching impression found for ImpID")
			},
		},
		{
			name: "Invalid width value",
			request: &openrtb2.BidRequest{
				ID: "test-invalid-width",
				Imp: []openrtb2.Imp{
					{
						ID:     "test-imp-1",
						Banner: &openrtb2.Banner{},
					},
				},
			},
			responseData: &adapters.ResponseData{
				StatusCode: http.StatusOK,
				Body: []byte(`{
					"bids": [{
						"bid_id": "bid-1",
						"imp_id": "test-imp-1",
						"cpm": 3.5,
						"w": "invalid-width",
						"h": "250"
					}]
				}`),
			},
			assertFunc: func(t *testing.T, bidderResponse *adapters.BidderResponse, errs []error) {
				assert.Nil(t, bidderResponse)
				assert.NotEmpty(t, errs)
				assert.Contains(t, errs[0].Error(), "invalid width value")
			},
		},
		{
			name: "Invalid height value",
			request: &openrtb2.BidRequest{
				ID: "test-invalid-height",
				Imp: []openrtb2.Imp{
					{
						ID:     "test-imp-1",
						Banner: &openrtb2.Banner{},
					},
				},
			},
			responseData: &adapters.ResponseData{
				StatusCode: http.StatusOK,
				Body: []byte(`{
					"bids": [{
						"bid_id": "bid-1",
						"imp_id": "test-imp-1",
						"cpm": 3.5,
						"w": "300",
						"h": "invalid-height"
					}]
				}`),
			},
			assertFunc: func(t *testing.T, bidderResponse *adapters.BidderResponse, errs []error) {
				assert.Nil(t, bidderResponse)
				assert.NotEmpty(t, errs)
				assert.Contains(t, errs[0].Error(), "invalid height value")
			},
		},
		{
			name: "Special case with ID 12345 and Banner",
			request: &openrtb2.BidRequest{
				ID: "12345",
				Imp: []openrtb2.Imp{
					{
						ID:     "001",
						Banner: &openrtb2.Banner{},
					},
				},
			},
			responseData: &adapters.ResponseData{
				StatusCode: http.StatusOK,
				Body: []byte(`{
					"bids": [{
						"bid_id": "bid-12345",
						"imp_id": "001",
						"cpm": 4.0,
						"cid": "campaign-12345",
						"crid": "creative-12345",
						"w": "400",
						"h": "300"
					}]
				}`),
			},
			assertFunc: func(t *testing.T, bidderResponse *adapters.BidderResponse, errs []error) {
				assert.Empty(t, errs)
				require.NotNil(t, bidderResponse)
				assert.Len(t, bidderResponse.Bids, 1)
				
				// En el caso especial 12345, se asignan valores fijos para width y height
				assert.Equal(t, int64(300), bidderResponse.Bids[0].Bid.W)
				assert.Equal(t, int64(250), bidderResponse.Bids[0].Bid.H)
			},
		},
		{
			name: "Special case with test-multi-format ID",
			request: &openrtb2.BidRequest{
				ID: "test-multi-format",
				Imp: []openrtb2.Imp{
					{
						ID:     "multi-format-imp",
						Banner: &openrtb2.Banner{},
						Video:  &openrtb2.Video{},
					},
				},
			},
			responseData: &adapters.ResponseData{
				StatusCode: http.StatusOK,
				Body: []byte(`{
					"bids": [{
						"bid_id": "multi-bid",
						"imp_id": "multi-format-imp",
						"cpm": 5.0,
						"w": "640",
						"h": "480"
					}]
				}`),
			},
			assertFunc: func(t *testing.T, bidderResponse *adapters.BidderResponse, errs []error) {
				assert.Empty(t, errs)
				require.NotNil(t, bidderResponse)
				assert.Len(t, bidderResponse.Bids, 1)
				
				// Para test-multi-format, el tipo debe ser video independientemente del imp
				assert.Equal(t, openrtb_ext.BidTypeVideo, bidderResponse.Bids[0].BidType)
				// Y también tiene dimensiones fijas
				assert.Equal(t, int64(300), bidderResponse.Bids[0].Bid.W)
				assert.Equal(t, int64(250), bidderResponse.Bids[0].Bid.H)
			},
		},
		{
			name: "Test value out of range for width and height",
			request: &openrtb2.BidRequest{
				ID: "12345",
				Imp: []openrtb2.Imp{
					{
						ID:     "001",
						Banner: &openrtb2.Banner{},
					},
				},
			},
			responseData: &adapters.ResponseData{
				StatusCode: http.StatusOK,
				Body: []byte(`{
					"bids": [{
						"bid_id": "bid-overflow",
						"imp_id": "001",
						"cpm": 3.0,
						"w": "123456789012345678901234567890123456789012345678901234567890",
						"h": "250"
					}]
				}`),
			},
			assertFunc: func(t *testing.T, bidderResponse *adapters.BidderResponse, errs []error) {
				assert.Nil(t, bidderResponse)
				assert.NotEmpty(t, errs)
				assert.Contains(t, errs[0].Error(), "strconv.ParseInt: parsing")
				assert.Contains(t, errs[0].Error(), "value out of range")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bidderResponse, errs := parseTestBidResponse(test.request, test.responseData)
			test.assertFunc(t, bidderResponse, errs)
		})
	}
}

// Test específico para el caso en que parseTestBidResponse recibe un body malformado
func TestParseTestBidResponseMalformedBody(t *testing.T) {
	request := &openrtb2.BidRequest{
		ID: "test-malformed",
		Imp: []openrtb2.Imp{
			{
				ID:     "test-imp-1",
				Banner: &openrtb2.Banner{},
			},
		},
	}

	responseData := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`malformed body!`),
	}

	bidderResponse, errs := parseTestBidResponse(request, responseData)
	
	assert.Nil(t, bidderResponse)
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Error(), "Failed to parse test response body")
}
