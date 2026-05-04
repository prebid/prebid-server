package adsmartx

import (
	"strings"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderAdsmartx,
		config.Adapter{
			Endpoint: "https://ads.adsmartx.com/ads/rtb/prebid/server",
		},
		config.Server{
			ExternalUrl: "http://hosturl.com",
			GvlID:       0,
			DataCenter:  "2",
		},
	)

	require.NoError(t, buildErr, "Builder returned unexpected error")
	adapterstest.RunJSONBidderTest(t, "adsmartxtest", bidder)
}

func TestParseImpExt(t *testing.T) {
	tests := []struct {
		name    string
		ext     jsonutil.RawMessage
		wantErr bool
	}{
		{"Valid ext", jsonutil.RawMessage(`{"bidder":{"bidfloor":0.5}}`), false},
		{"Valid ext with sspId", jsonutil.RawMessage(`{"bidder":{"sspId":"ssp-123","siteId":"site-456"}}`), false},
		{"Invalid JSON", jsonutil.RawMessage(`not-json`), true},
		{"Not an object", jsonutil.RawMessage(`"string"`), true},
		{"Bidder not object", jsonutil.RawMessage(`{"bidder":"not-an-object"}`), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseImpExt(tt.ext)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestGetBidType(t *testing.T) {
	tests := []struct {
		name      string
		mtype     openrtb2.MarkupType
		wantErr   bool
		wantBidTy openrtb_ext.BidType
	}{
		{"Banner", openrtb2.MarkupBanner, false, openrtb_ext.BidTypeBanner},
		{"Video", openrtb2.MarkupVideo, false, openrtb_ext.BidTypeVideo},
		{"Unknown", 99, true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bidType, err := getBidType(tt.mtype)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantBidTy, bidType)
		})
	}
}

func TestMakeRequestsErrors(t *testing.T) {
	a := &adapter{endpoint: "http://test-endpoint"}
	tests := []struct {
		name    string
		imps    []openrtb2.Imp
		wantErr string
	}{
		{"Invalid ext", []openrtb2.Imp{{ID: "1", Ext: jsonutil.RawMessage(`not-json`)}}, "impID 1:"},
		{"No valid imps", []openrtb2.Imp{}, "no valid impressions"},
		{"No banner or video", []openrtb2.Imp{{ID: "1", Ext: jsonutil.RawMessage(`{"bidder":{"bidfloor": 0.5}}`)}}, "no banner or video object specified"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &openrtb2.BidRequest{Imp: tt.imps}
			_, errs := a.MakeRequests(req, nil)
			require.NotEmpty(t, errs, "expected error, got none")
			found := false
			for _, err := range errs {
				if err != nil && (tt.wantErr == "" || strings.Contains(err.Error(), tt.wantErr)) {
					found = true
					break
				}
			}
			assert.True(t, found, "expected error containing %q, got %v", tt.wantErr, errs)
		})
	}
}

func TestMakeBidsErrors(t *testing.T) {
	a := &adapter{endpoint: "http://test-endpoint"}
	validReq := &openrtb2.BidRequest{ID: "1"}
	validReqData := &adapters.RequestData{}
	tests := []struct {
		name     string
		respData *adapters.ResponseData
		wantErr  string
	}{
		{"Non-200/204 response", &adapters.ResponseData{StatusCode: 500, Body: []byte(`{}`)}, "Unexpected status code"},
		{"Invalid JSON", &adapters.ResponseData{StatusCode: 200, Body: []byte(`not-json`)}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errs := a.MakeBids(validReq, validReqData, tt.respData)
			require.NotEmpty(t, errs, "expected error, got none")
			found := false
			for _, err := range errs {
				if err != nil && strings.Contains(err.Error(), tt.wantErr) {
					found = true
					break
				}
			}
			assert.True(t, found, "expected error containing %q, got %v", tt.wantErr, errs)
		})
	}
}

func TestMakeBidsSkipsBadBidType(t *testing.T) {
	a := &adapter{endpoint: "http://test-endpoint"}
	validReq := &openrtb2.BidRequest{ID: "1"}
	validReqData := &adapters.RequestData{}

	respBody := `{
		"id": "1",
		"seatbid": [{
			"bid": [
				{"id": "good-bid", "impid": "1", "price": 1.0, "adm": "<div>ad</div>", "mtype": 1},
				{"id": "bad-bid", "impid": "2", "price": 2.0, "adm": "<div>ad2</div>", "mtype": 99},
				{"id": "good-bid-2", "impid": "3", "price": 3.0, "adm": "<div>ad3</div>", "mtype": 2}
			]
		}],
		"cur": "USD"
	}`

	resp := &adapters.ResponseData{StatusCode: 200, Body: []byte(respBody)}
	bidderResp, errs := a.MakeBids(validReq, validReqData, resp)

	require.NotNil(t, bidderResp, "expected bid response, got nil")
	assert.Len(t, bidderResp.Bids, 2, "expected 2 valid bids (bad bid should be skipped)")
	assert.Len(t, errs, 1, "expected 1 error for the bad bid type")
	assert.Contains(t, errs[0].Error(), "unknown bid type")
}
