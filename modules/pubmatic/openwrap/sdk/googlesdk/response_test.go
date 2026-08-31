package googlesdk

import (
	"encoding/json"
	"strings"
	"testing"

	nativeResponse "github.com/prebid/openrtb/v20/native1/response"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/openrtb/v20/openrtb3"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models/nbr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetGoogleSDKResponseReject(t *testing.T) {
	tests := []struct {
		name        string
		rctx        models.RequestCtx
		bidResponse *openrtb2.BidResponse
		want        bool
	}{
		{
			name: "NBR present with debug false",
			rctx: models.RequestCtx{Debug: false, Endpoint: models.EndpointGoogleSDK},
			bidResponse: &openrtb2.BidResponse{
				NBR: openrtb3.NoBidUnknownError.Ptr(),
			},
			want: true,
		},
		{
			name: "NBR present with debug true",
			rctx: models.RequestCtx{Debug: true, Endpoint: models.EndpointGoogleSDK},
			bidResponse: &openrtb2.BidResponse{
				NBR: openrtb3.NoBidUnknownError.Ptr(),
			},
			want: false,
		},
		{
			name:        "Empty bid response",
			rctx:        models.RequestCtx{Debug: false, Endpoint: models.EndpointGoogleSDK},
			bidResponse: &openrtb2.BidResponse{},
			want:        true,
		},
		{
			name: "Valid bid response",
			rctx: models.RequestCtx{Debug: false, Endpoint: models.EndpointGoogleSDK},
			bidResponse: &openrtb2.BidResponse{
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{{ID: "1"}},
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SetGoogleSDKResponseReject(tt.rctx, tt.bidResponse)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetDeclaredAd(t *testing.T) {
	tests := []struct {
		name string
		rctx models.RequestCtx
		bid  openrtb2.Bid
		want models.DeclaredAd
	}{
		{
			name: "Banner ad",
			rctx: models.RequestCtx{
				Trackers: map[string]models.OWTracker{
					"1": {BidType: "banner"},
				},
			},
			bid: openrtb2.Bid{
				ID:  "1",
				AdM: "<a href='http://example.com'>Click</a>",
			},
			want: models.DeclaredAd{
				HTMLSnippet:     "<a href='http://example.com'>Click</a>",
				ClickThroughURL: []string{"http://example.com"},
			},
		},
		{
			name: "Banner ad with click_urls array",
			rctx: models.RequestCtx{
				Trackers: map[string]models.OWTracker{
					"2": {BidType: "banner"},
				},
			},
			bid: openrtb2.Bid{
				ID:  "2",
				AdM: `{"click_urls":["http://array-url.com"]}`,
			},
			want: models.DeclaredAd{
				HTMLSnippet:     `{"click_urls":["http://array-url.com"]}`,
				ClickThroughURL: []string{"http://array-url.com"},
			},
		},
		{
			name: "Video ad",
			rctx: models.RequestCtx{
				Trackers: map[string]models.OWTracker{
					"1": {BidType: "video"},
				},
			},
			bid: openrtb2.Bid{
				ID:  "1",
				AdM: "<VAST><Ad><InLine><Creatives><Creative><Linear><VideoClicks><ClickThrough>http://example.com</ClickThrough></VideoClicks></Linear></Creative></Creatives></InLine></Ad></VAST>",
			},
			want: models.DeclaredAd{
				VideoVastXML:    "<VAST><Ad><InLine><Creatives><Creative><Linear><VideoClicks><ClickThrough>http://example.com</ClickThrough></VideoClicks></Linear></Creative></Creatives></InLine></Ad></VAST>",
				ClickThroughURL: []string{"http://example.com"},
			},
		},
		{
			name: "Native ad",
			rctx: models.RequestCtx{
				Trackers: map[string]models.OWTracker{
					"1": {BidType: "native"},
				},
			},
			bid: openrtb2.Bid{
				ID:  "1",
				AdM: `{"link":{"url":"http://example.com"}}`,
			},
			want: models.DeclaredAd{
				NativeResponse: &nativeResponse.Response{
					Link: nativeResponse.Link{
						URL: "http://example.com",
					},
				},
				ClickThroughURL: []string{"http://example.com"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getDeclaredAd(tt.rctx, tt.bid)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSetSDKRenderedAdID(t *testing.T) {
	tests := []struct {
		name     string
		app      *openrtb2.App
		endpoint string
		want     string
	}{
		{
			name:     "Non GoogleSDK endpoint",
			app:      &openrtb2.App{},
			endpoint: "other",
			want:     "",
		},
		{
			name: "Valid SDK ID in installed_sdk.id",
			app: &openrtb2.App{
				Ext: json.RawMessage(`{"installed_sdk":{"id":"test-sdk-id"}}`),
			},
			endpoint: models.EndpointGoogleSDK,
			want:     "test-sdk-id",
		},
		{
			name: "Valid SDK ID in installed_sdk array",
			app: &openrtb2.App{
				Ext: json.RawMessage(`{"installed_sdk":[{"id":"test-sdk-id"}]}`),
			},
			endpoint: models.EndpointGoogleSDK,
			want:     "test-sdk-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SetSDKRenderedAdID(tt.app, tt.endpoint)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetBannerClickThroughURL(t *testing.T) {
	tests := []struct {
		name string
		bid  openrtb2.Bid
		want []string
	}{
		{
			name: "Empty creative",
			bid:  openrtb2.Bid{AdM: ""},
			want: []string{},
		},
		{
			name: "JSON creative with click_urls array",
			bid:  openrtb2.Bid{AdM: `{"click_urls":["http://example.com"]}`},
			want: []string{"http://example.com"},
		},
		{
			name: "HTML creative with anchor tag",
			bid:  openrtb2.Bid{AdM: `<a href="http://example.com">Click</a>`},
			want: []string{"http://example.com"},
		},
		{
			name: "Creative with ADomain",
			bid:  openrtb2.Bid{AdM: `<script url="http://example.com">Click</script>`, ADomain: []string{"http://example.com"}},
			want: []string{"http://example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getBannerClickThroughURL(tt.bid)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractClickURLFromJSON(t *testing.T) {
	tests := []struct {
		name     string
		creative string
		want     string
	}{
		{
			name:     "No click_urls",
			creative: `{"other":"value"}`,
			want:     "",
		},
		{
			name:     "Click URLs array",
			creative: `{"click_urls":["http://example.com"]}`,
			want:     "http://example.com",
		},
		{
			name:     "Click URL string",
			creative: `{"click_urls":"http://example.com"}`,
			want:     "http://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractClickURLFromJSON(tt.creative)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractClickURLFromHTML(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "Valid anchor tag",
			html: `<a href="http://example.com">Click</a>`,
			want: "http://example.com",
		},
		{
			name: "No anchor tag",
			html: `<div>No link here</div>`,
			want: "",
		},
		{
			name: "Invalid HTML",
			html: "Not HTML",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractClickURLFromHTML(tt.html)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestApplyGoogleSDKResponse(t *testing.T) {
	tests := []struct {
		name     string
		rctx     models.RequestCtx
		bidResp  *openrtb2.BidResponse
		wantResp *openrtb2.BidResponse
		wantNBR  bool
	}{
		{
			name:     "Non GoogleSDK endpoint returns input",
			rctx:     models.RequestCtx{Endpoint: "other"},
			bidResp:  &openrtb2.BidResponse{ID: "test-non-gsdk"},
			wantResp: &openrtb2.BidResponse{ID: "test-non-gsdk"},
			wantNBR:  false,
		},
		{
			name:     "GoogleSDK endpoint, debug true, empty SeatBid",
			rctx:     models.RequestCtx{Endpoint: models.EndpointGoogleSDK, Debug: true},
			bidResp:  &openrtb2.BidResponse{ID: "test-debug-empty"},
			wantResp: &openrtb2.BidResponse{ID: "test-debug-empty"},
			wantNBR:  false,
		},
		{
			name:     "GoogleSDK endpoint, debug true, NBR present",
			rctx:     models.RequestCtx{Endpoint: models.EndpointGoogleSDK, Debug: true},
			bidResp:  &openrtb2.BidResponse{ID: "test-debug-nbr", NBR: openrtb3.NoBidUnknownError.Ptr()},
			wantResp: &openrtb2.BidResponse{ID: "test-debug-nbr", NBR: openrtb3.NoBidUnknownError.Ptr()},
			wantNBR:  true,
		},
		{
			name:     "GoogleSDK endpoint, reject true sets NBR",
			rctx:     models.RequestCtx{Endpoint: models.EndpointGoogleSDK, GoogleSDK: models.GoogleSDK{Reject: true}, StartTime: 0},
			bidResp:  &openrtb2.BidResponse{ID: "test-reject", NBR: openrtb3.NoBidUnknownError.Ptr()},
			wantResp: &openrtb2.BidResponse{ID: "test-reject", NBR: openrtb3.NoBidUnknownError.Ptr()},
			wantNBR:  true,
		},
		{
			name: "GoogleSDK endpoint, reject missing clickthrough URL",
			rctx: models.RequestCtx{
				Endpoint:  models.EndpointGoogleSDK,
				StartTime: 1234567890,
				Trackers: map[string]models.OWTracker{
					"bid1": {
						BidType: models.Banner,
					},
				},
			},
			bidResp: &openrtb2.BidResponse{
				ID:  "test-reject-clickthrough",
				Cur: "USD",
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{
							{
								ID:  "bid1",
								AdM: "<html><body>No clickthrough URL</body></html>",
							},
						},
					},
				},
			},
			wantResp: &openrtb2.BidResponse{
				ID:  "test-reject-clickthrough",
				Cur: "USD",
				SeatBid: []openrtb2.SeatBid{{
					Bid: []openrtb2.Bid{{
						ID:  "bid1",
						AdM: "<html><body>No clickthrough URL</body></html>",
					}},
				}},
				NBR: nbr.ResponseRejectedMissingParam.Ptr(),
			},
			wantNBR: true,
		},
		{
			name: "GoogleSDK endpoint, customizeBid path",
			rctx: models.RequestCtx{
				Endpoint: models.EndpointGoogleSDK,
				Trackers: map[string]models.OWTracker{
					"bid1": {
						BidType: models.Banner,
					},
				},
			},
			bidResp: &openrtb2.BidResponse{
				ID:  "test-customok",
				Cur: "USD",
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{
							{
								ID:  "bid1",
								AdM: "<html><body><a href=\"http://example.com/click\">Click here</a></body></html>",
							},
						},
					},
				},
			},
			wantResp: &openrtb2.BidResponse{
				ID:    "test-customok",
				BidID: "bid1",
				Cur:   "USD",
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{
							{
								ID:  "bid1",
								Ext: json.RawMessage(`{"sdk_rendered_ad":{"rendering_data":"{\"id\":\"test-customok\",\"seatbid\":[{\"bid\":[{\"id\":\"bid1\",\"impid\":\"\",\"price\":0,\"adm\":\"\\u003chtml\\u003e\\u003cbody\\u003e\\u003ca href=\\\"http://example.com/click\\\"\\u003eClick here\\u003c/a\\u003e\\u003c/body\\u003e\\u003c/html\\u003e\"}]}],\"cur\":\"USD\"}","declared_ad":{"click_through_url":["http://example.com/click"],"html_snippet":"\u003chtml\u003e\u003cbody\u003e\u003ca href=\"http://example.com/click\"\u003eClick here\u003c/a\u003e\u003c/body\u003e\u003c/html\u003e"}}}`),
							},
						},
					},
				},
			},
			wantNBR: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyGoogleSDKResponse(tt.rctx, tt.bidResp)
			// For cases with processing time, only compare NBR and ID
			if got.Ext != nil && strings.Contains(string(got.Ext), "processing_time_ms") {
				assert.Equal(t, tt.wantResp.NBR, got.NBR)
				assert.Equal(t, tt.wantResp.ID, got.ID)
				assert.Contains(t, string(got.Ext), "processing_time_ms")
			} else {
				assert.Equal(t, tt.wantResp, got)
			}
		})
	}
}

func TestGetVideoClickThroughURL(t *testing.T) {
	tests := []struct {
		name     string
		bid      openrtb2.Bid
		expected []string
	}{
		{
			name: "Valid VAST with ClickThrough",
			bid: openrtb2.Bid{
				AdM: `<?xml version="1.0"?>
				<VAST version="2.0">
					<Ad>
						<InLine>
							<Creatives>
								<Creative>
									<Linear>
										<VideoClicks>
											<ClickThrough>http://example.com/click</ClickThrough>
										</VideoClicks>
									</Linear>
								</Creative>
							</Creatives>
						</InLine>
					</Ad>
				</VAST>`,
				ADomain: []string{"fallback.com"},
			},
			expected: []string{"http://example.com/click"},
		},
		{
			name: "Invalid XML",
			bid: openrtb2.Bid{
				AdM:     "<invalid>xml",
				ADomain: []string{"fallback.com"},
			},
			expected: []string{"fallback.com"},
		},
		{
			name: "Valid XML but no ClickThrough",
			bid: openrtb2.Bid{
				AdM: `<?xml version="1.0"?>
				<VAST version="2.0">
					<Ad>
						<InLine>
							<Creatives>
								<Creative>
									<Linear>
										<VideoClicks>
										</VideoClicks>
									</Linear>
								</Creative>
							</Creatives>
						</InLine>
					</Ad>
				</VAST>`,
				ADomain: []string{"fallback.com"},
			},
			expected: []string{"fallback.com"},
		},
		{
			name: "Empty AdM",
			bid: openrtb2.Bid{
				AdM:     "",
				ADomain: []string{"fallback.com"},
			},
			expected: []string{"fallback.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getVideoClickThroughURL(tt.bid)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCustomizeBid(t *testing.T) {
	type want struct {
		bids   []openrtb2.Bid
		wantOK bool
	}

	tests := []struct {
		name        string
		rctx        models.RequestCtx
		bidResponse *openrtb2.BidResponse
		want        want
	}{
		{
			name:        "marshal error returns nil,false",
			rctx:        models.RequestCtx{},
			bidResponse: &openrtb2.BidResponse{}, // Empty SeatBid triggers error path
			want: want{
				bids:   nil,
				wantOK: false,
			},
		},
		{
			name: "empty bid returns nil,false",
			rctx: models.RequestCtx{},
			bidResponse: &openrtb2.BidResponse{
				ID: "id",
				SeatBid: []openrtb2.SeatBid{
					{Bid: []openrtb2.Bid{}},
				},
			},
			want: want{
				bids:   nil,
				wantOK: false,
			},
		},
		{
			name: "empty seatbid returns nil,false",
			rctx: models.RequestCtx{},
			bidResponse: &openrtb2.BidResponse{
				ID: "id",
			},
			want: want{
				bids:   nil,
				wantOK: false,
			},
		},
		{
			name: "happy path",
			rctx: models.RequestCtx{
				GoogleSDK: models.GoogleSDK{
					SDKRenderedAdID: "sdkrenderedaid",
				},
				StartTime: 1234567890,
				Endpoint:  models.EndpointGoogleSDK,
				Trackers: map[string]models.OWTracker{
					"bidid": {
						BidType: models.Banner,
					},
				},
			},
			bidResponse: &openrtb2.BidResponse{
				ID: "id",
				SeatBid: []openrtb2.SeatBid{
					{Bid: []openrtb2.Bid{
						{
							ID:  "bidid",
							AdM: `<html><body><a href="http://example.com/click">Click here</a></body></html>`,
						},
					}},
				},
			},
			want: want{
				bids: []openrtb2.Bid{
					{
						ID:  "bidid",
						Ext: json.RawMessage(`{"sdk_rendered_ad":{"id":"sdkrenderedaid","rendering_data":"{\"id\":\"id\",\"seatbid\":[{\"bid\":[{\"id\":\"bidid\",\"impid\":\"\",\"price\":0,\"adm\":\"\\u003chtml\\u003e\\u003cbody\\u003e\\u003ca href=\\\"http://example.com/click\\\"\\u003eClick here\\u003c/a\\u003e\\u003c/body\\u003e\\u003c/html\\u003e\"}]}]}","declared_ad":{"click_through_url":["http://example.com/click"],"html_snippet":"\u003chtml\u003e\u003cbody\u003e\u003ca href=\"http://example.com/click\"\u003eClick here\u003c/a\u003e\u003c/body\u003e\u003c/html\u003e"}}}`),
						AdM: "",
					},
				},
				wantOK: true,
			},
		},
		{
			name: "reject_empty_clickthrough_url",
			rctx: models.RequestCtx{
				StartTime: 1234567890,
				Endpoint:  models.EndpointGoogleSDK,
				Trackers: map[string]models.OWTracker{
					"bidid": {
						BidType: models.Banner,
					},
				},
			},
			bidResponse: &openrtb2.BidResponse{
				ID: "responseid",
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{
							{
								ID:  "bidid",
								AdM: "<html><body>No clickthrough URL</body></html>",
							},
						},
					},
				},
			},
			want: want{
				bids:   nil,
				wantOK: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bids, ok := customizeBid(tt.rctx, tt.bidResponse)
			assert.Equal(t, tt.want.bids, bids)
			assert.Equal(t, tt.want.wantOK, ok)
		})
	}
}

func TestApplyGoogleSDKResponse_PreservesTrackersInRenderingData(t *testing.T) {
	trackersExt := json.RawMessage(`{"trackers":[{"event":"ad_attribute","url":"https://t.pubmatic.com?bidid=789&ad_attribute={ADATTRIBUTE}"}]}`)
	br := &openrtb2.BidResponse{
		ID:  "test-customok",
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{{
			Bid: []openrtb2.Bid{{
				ID:  "bid1",
				AdM: `<html><body><a href="http://example.com/click">Click here</a></body></html>`,
				Ext: trackersExt,
			}},
		}},
	}
	rctx := models.RequestCtx{
		Endpoint: models.EndpointGoogleSDK,
		Trackers: map[string]models.OWTracker{
			"bid1": {BidType: models.Banner},
		},
	}

	out := ApplyGoogleSDKResponse(rctx, br)
	require.Len(t, out.SeatBid, 1)
	require.Len(t, out.SeatBid[0].Bid, 1)

	var bidExt models.GoogleSDKBidExt
	require.NoError(t, json.Unmarshal(out.SeatBid[0].Bid[0].Ext, &bidExt))
	require.NotEmpty(t, bidExt.SDKRenderedAd.RenderingData)

	var embedded openrtb2.BidResponse
	require.NoError(t, json.Unmarshal([]byte(bidExt.SDKRenderedAd.RenderingData), &embedded))
	require.Len(t, embedded.SeatBid, 1)
	require.Len(t, embedded.SeatBid[0].Bid, 1)
	assert.JSONEq(t, string(trackersExt), string(embedded.SeatBid[0].Bid[0].Ext))
}
