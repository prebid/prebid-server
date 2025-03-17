package pixfuture

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// int64Ptr is a helper function to create a pointer to an int64 value
func int64Ptr(i int64) *int64 {
	return &i
}

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderPixfuture, config.Adapter{
		Endpoint: "http://any.url",
	}, config.Server{})
	require.NoError(t, buildErr, "Builder returned unexpected error")

	dirs := []string{"pixfuturetest/exemplary", "pixfuturetest/supplemental"}

	for _, dir := range dirs {
		t.Run(dir, func(t *testing.T) {
			files, err := filepath.Glob(filepath.Join(dir, "*.json"))
			require.NoErrorf(t, err, "Failed to glob JSON files in %s", dir)
			t.Logf("Found %d JSON files in %s", len(files), dir)

			for _, file := range files {
				t.Run(filepath.Base(file), func(t *testing.T) {
					tmpDir, err := ioutil.TempDir("", "pixfuture_test_")
					require.NoError(t, err, "Failed to create temp dir")
					defer os.RemoveAll(tmpDir)

					src := file
					dst := filepath.Join(tmpDir, filepath.Base(file))
					input, err := ioutil.ReadFile(src)
					require.NoErrorf(t, err, "Failed to read %s", src)
					err = ioutil.WriteFile(dst, input, 0644)
					require.NoErrorf(t, err, "Failed to write %s", dst)

					t.Logf("Testing JSON file: %s", file)
					adapterstest.RunJSONBidderTest(t, tmpDir, bidder)
				})
			}
		})
	}
}

func TestMakeRequests(t *testing.T) {
	adapter := &adapter{endpoint: "http://test.url"}

	tests := []struct {
		name           string
		bidRequest     *openrtb2.BidRequest
		wantReqs       int
		wantErrs       int
		wantErrMessage string
	}{
		{
			name: "Valid Native",
			bidRequest: &openrtb2.BidRequest{
				ID: "test-request-id",
				Imp: []openrtb2.Imp{
					{
						ID:     "imp1",
						Native: &openrtb2.Native{Request: "{\"ver\":\"1.2\"}", Ver: "1.2"},
						Ext:    json.RawMessage(`{"bidder":{"pix_id":"55463"}}`),
					},
				},
			},
			wantReqs: 1,
			wantErrs: 0,
		},
		{
			name:           "Nil Request",
			bidRequest:     nil,
			wantReqs:       0,
			wantErrs:       1,
			wantErrMessage: "No impressions in bid request",
		},
		{
			name:           "Empty Impressions",
			bidRequest:     &openrtb2.BidRequest{ID: "test-request-id", Imp: []openrtb2.Imp{}},
			wantReqs:       0,
			wantErrs:       1,
			wantErrMessage: "No impressions in bid request",
		},
		{
			name: "Invalid Ext",
			bidRequest: &openrtb2.BidRequest{
				ID: "test-request-id",
				Imp: []openrtb2.Imp{
					{ID: "imp1", Banner: &openrtb2.Banner{W: int64Ptr(300), H: int64Ptr(250)}, Ext: json.RawMessage(`{"bidder":"invalid"}`)},
				},
			},
			wantReqs:       0,
			wantErrs:       1,
			wantErrMessage: "Invalid impression extension",
		},
		{
			name: "Missing PixID",
			bidRequest: &openrtb2.BidRequest{
				ID: "test-request-id",
				Imp: []openrtb2.Imp{
					{ID: "imp1", Video: &openrtb2.Video{W: int64Ptr(640), H: int64Ptr(360)}, Ext: json.RawMessage(`{"bidder":{"pix_id":""}}`)},
				},
			},
			wantReqs:       0,
			wantErrs:       1,
			wantErrMessage: "Missing pix_id",
		},
		{
			name: "Short PixID",
			bidRequest: &openrtb2.BidRequest{
				ID: "test-request-id",
				Imp: []openrtb2.Imp{
					{ID: "imp1", Banner: &openrtb2.Banner{W: int64Ptr(300), H: int64Ptr(250)}, Ext: json.RawMessage(`{"bidder":{"pix_id":"12"}}`)},
				},
			},
			wantReqs:       0,
			wantErrs:       1,
			wantErrMessage: "pix_id must be at least 3 characters long",
		},
		{
			name: "No Supported Type",
			bidRequest: &openrtb2.BidRequest{
				ID: "test-request-id",
				Imp: []openrtb2.Imp{
					{ID: "imp1", Ext: json.RawMessage(`{"bidder":{"pix_id":"55463"}}`)},
				},
			},
			wantReqs:       0,
			wantErrs:       1,
			wantErrMessage: "Banner, Native, or Video impression required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqs, errs := adapter.MakeRequests(tt.bidRequest, nil)
			assert.Len(t, reqs, tt.wantReqs, "Request count mismatch")
			assert.Len(t, errs, tt.wantErrs, "Error count mismatch")
			if tt.wantErrs > 0 {
				assert.Contains(t, errs[0].Error(), tt.wantErrMessage, "Error message mismatch")
			}
			if tt.wantReqs > 0 {
				assert.Equal(t, "http://test.url", reqs[0].Uri)
				assert.Equal(t, "POST", reqs[0].Method)
				assert.Equal(t, "application/json", reqs[0].Headers.Get("Content-Type"))
			}
		})
	}
}

func TestMakeBids(t *testing.T) {
	adapter := &adapter{endpoint: "http://test.url"}
	internalReq := &openrtb2.BidRequest{ID: "test-request-id", Imp: []openrtb2.Imp{{ID: "imp1"}}}
	externalReq := &adapters.RequestData{Body: []byte(`{"id":"test-request-id"}`)}

	tests := []struct {
		name           string
		respData       *adapters.ResponseData
		wantBids       int
		wantErrs       int
		wantErrMessage string
	}{
		{
			name: "Valid Banner",
			respData: &adapters.ResponseData{
				StatusCode: http.StatusOK,
				Body: []byte(`{
					"id": "test-response-id",
					"cur": "USD",
					"seatbid": [{"bid": [{"id": "bid1", "impid": "imp1", "price": 1.23, "adm": "<div>Banner Ad</div>", "ext": {"prebid": {"type": "banner"}}}]}]
				}`),
			},
			wantBids: 1,
			wantErrs: 0,
		},
		{
			name:     "No Content",
			respData: &adapters.ResponseData{StatusCode: http.StatusNoContent},
			wantBids: 0,
			wantErrs: 0,
		},
		{
			name:           "Bad Status",
			respData:       &adapters.ResponseData{StatusCode: http.StatusBadRequest},
			wantBids:       0,
			wantErrs:       1,
			wantErrMessage: "Unexpected status code: 400",
		},
		{
			name:           "Invalid Response",
			respData:       &adapters.ResponseData{StatusCode: http.StatusOK, Body: []byte(`{invalid json}`)},
			wantBids:       0,
			wantErrs:       1,
			wantErrMessage: "Invalid response format",
		},
		{
			name: "No Valid Bids",
			respData: &adapters.ResponseData{
				StatusCode: http.StatusOK,
				Body:       []byte(`{"id": "test-response-id", "seatbid": [{"bid": [{"id": "bid1", "impid": "imp1", "ext": {"prebid": {"type": "invalid"}}}]}]}`),
			},
			wantBids: 0,
			wantErrs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bidResp, errs := adapter.MakeBids(internalReq, externalReq, tt.respData)
			if tt.wantBids > 0 {
				assert.NotNil(t, bidResp, "Bid response should not be nil")
				assert.Equal(t, "USD", bidResp.Currency)
				assert.Len(t, bidResp.Bids, tt.wantBids, "Bid count mismatch")
			} else {
				assert.Nil(t, bidResp, "Bid response should be nil")
			}
			assert.Len(t, errs, tt.wantErrs, "Error count mismatch")
			if tt.wantErrs > 0 {
				assert.Contains(t, errs[0].Error(), tt.wantErrMessage, "Error message mismatch")
			}
		})
	}
}

func TestGetMediaTypeForBid(t *testing.T) {
	tests := []struct {
		name      string
		bidExt    string
		wantType  openrtb_ext.BidType
		wantError bool
	}{
		{
			name:     "Banner",
			bidExt:   `{"prebid":{"type":"banner"}}`,
			wantType: openrtb_ext.BidTypeBanner,
		},
		{
			name:     "Video",
			bidExt:   `{"prebid":{"type":"video"}}`,
			wantType: openrtb_ext.BidTypeVideo,
		},
		{
			name:     "Native",
			bidExt:   `{"prebid":{"type":"native"}}`,
			wantType: openrtb_ext.BidTypeNative,
		},
		{
			name:      "Invalid Type",
			bidExt:    `{"prebid":{"type":"invalid"}}`,
			wantType:  "",
			wantError: true,
		},
		{
			name:      "Invalid JSON",
			bidExt:    `{"prebid":"invalid"}`,
			wantType:  "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bid := openrtb2.Bid{Ext: json.RawMessage(tt.bidExt)}
			bidType, err := getMediaTypeForBid(bid)
			if tt.wantError {
				assert.Error(t, err)
				assert.Empty(t, bidType)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantType, bidType)
			}
		})
	}
}
