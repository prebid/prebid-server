package risemediatech

import (
	"strings"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderRiseMediaTech,
		config.Adapter{
			Endpoint: "https://dev-ads.risemediatech.com/ads/rtb/prebid/server",
		},
		config.Server{
			ExternalUrl: "http://hosturl.com",
			GvlID:       0,
			DataCenter:  "2",
		},
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error: %v", buildErr)
	}
	require.NoError(t, buildErr, "Builder returned unexpected error")

	adapterstest.RunJSONBidderTest(t, "risemediatechtest", bidder)
}

// Table-driven test for parseImpExt
func TestParseImpExt(t *testing.T) {
	tests := []struct {
		name    string
		ext     jsonutil.RawMessage
		wantErr bool
	}{
		{"Valid ext", jsonutil.RawMessage(`{"bidder":{"placementId":"abc"}}`), false},
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

// Table-driven test for getBidType
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
			bid := &openrtb2.Bid{MType: tt.mtype}
			bidType, err := getBidType(bid)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantBidTy, bidType)
		})
	}
}

// Table-driven test for MakeRequests error branches
func TestMakeRequestsErrors(t *testing.T) {
	a := &adapter{endpoint: "http://test-endpoint"}
	baseImp := openrtb2.Imp{ID: "1", Ext: jsonutil.RawMessage(`{"bidder":{"placementId":"abc"}}`)}
	tests := []struct {
		name    string
		imps    []openrtb2.Imp
		wantErr string
	}{
		{"Invalid ext", []openrtb2.Imp{{ID: "1", Ext: jsonutil.RawMessage(`not-json`)}}, "impID 1:"},
		{"Invalid banner dims", []openrtb2.Imp{{ID: "1", Banner: &openrtb2.Banner{} /* nil w/h */, Ext: baseImp.Ext}}, "invalid banner dimensions"},
		{"Empty video mimes", []openrtb2.Imp{{ID: "1", Video: &openrtb2.Video{W: ptrutil.ToPtr[int64](640), H: ptrutil.ToPtr[int64](480), MIMEs: []string{}}, Ext: baseImp.Ext}}, "missing or empty video.mimes"},
		{"Invalid video dims", []openrtb2.Imp{{ID: "1", Video: &openrtb2.Video{W: ptrutil.ToPtr[int64](0), H: ptrutil.ToPtr[int64](0), MIMEs: []string{"video/mp4"}}, Ext: baseImp.Ext}}, "missing or invalid video width/height"},
		{"No valid imps", []openrtb2.Imp{}, "no valid impressions"},
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

// Table-driven test for MakeBids error branches
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
		{"Unknown mtype", &adapters.ResponseData{StatusCode: 200, Body: []byte(`{"id":"1","seatbid":[{"bid":[{"id":"b1","impid":"1","price":1.0,"adm":"<div>test</div>","mtype":99}]}],"cur":"USD"}`)}, "unknown bid type"},
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
