package ocm

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testEndpoint = "https://pbam.test.invalid/openrtb2/auction"

// End-to-end MakeRequests/MakeBids behavior lives in the JSON fixtures under
// ocmtest/. The Go tests below supplement them with error-type and shared-object
// mutation contracts that JSON fixtures cannot express.

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderOcm,
		config.Adapter{Endpoint: testEndpoint},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1148, DataCenter: "2"})
	require.NoError(t, buildErr, "Builder returned unexpected error")

	adapterstest.RunJSONBidderTest(t, "ocmtest", bidder)
}

func TestBuilderStoresEndpoint(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderOcm, config.Adapter{Endpoint: testEndpoint}, config.Server{})
	require.NoError(t, err)
	assert.Equal(t, testEndpoint, bidder.(*adapter).endpoint)
}

func TestMakeRequestsNoImps(t *testing.T) {
	requests, errs := newBidder(t).MakeRequests(&openrtb2.BidRequest{ID: "no-imps"}, &adapters.ExtraRequestInfo{})
	assert.Nil(t, requests)
	assert.Nil(t, errs)
}

// TestMakeRequestsErrorTypes pins every MakeRequests rejection to *errortypes.BadInput,
// which is what makes prebid-server attribute the failure to the publisher's request
// rather than to the OCM exchange.
func TestMakeRequestsErrorTypes(t *testing.T) {
	tests := []struct {
		name        string
		imps        []openrtb2.Imp
		wantMessage string
	}{
		{
			name:        "imp.ext is not an object",
			imps:        []openrtb2.Imp{bannerImp("imp-1", json.RawMessage(`[]`))},
			wantMessage: "imp[0]: unable to unmarshal ext",
		},
		{
			name:        "imp.ext.bidder is not an object",
			imps:        []openrtb2.Imp{bannerImp("imp-1", json.RawMessage(`{"bidder":[]}`))},
			wantMessage: "imp[0]: unable to unmarshal ext.bidder",
		},
		{
			name:        "publisherId missing",
			imps:        []openrtb2.Imp{bannerImp("imp-1", json.RawMessage(`{"bidder":{"placementId":"p"}}`))},
			wantMessage: "imp[0]: publisherId is required and must not be blank",
		},
		{
			name:        "publisherId blank",
			imps:        []openrtb2.Imp{bannerImp("imp-1", json.RawMessage(`{"bidder":{"publisherId":"   ","placementId":"p"}}`))},
			wantMessage: "imp[0]: publisherId is required and must not be blank",
		},
		{
			name:        "placementId missing",
			imps:        []openrtb2.Imp{bannerImp("imp-1", json.RawMessage(`{"bidder":{"publisherId":"pub"}}`))},
			wantMessage: "imp[0]: placementId is required and must not be blank",
		},
		{
			name:        "placementId blank",
			imps:        []openrtb2.Imp{bannerImp("imp-1", json.RawMessage(`{"bidder":{"publisherId":"pub","placementId":" "}}`))},
			wantMessage: "imp[0]: placementId is required and must not be blank",
		},
		{
			name: "publisher ids diverge across imps",
			imps: []openrtb2.Imp{
				bannerImp("imp-1", json.RawMessage(`{"bidder":{"publisherId":"pub-a","placementId":"p1"}}`)),
				bannerImp("imp-2", json.RawMessage(`{"bidder":{"publisherId":"pub-b","placementId":"p2"}}`)),
			},
			wantMessage: `imp[1]: all impressions must share the same publisherId, found "pub-b" and "pub-a"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := &openrtb2.BidRequest{
				ID:   "req",
				Imp:  test.imps,
				Site: &openrtb2.Site{Domain: "example.gr"},
			}

			requests, errs := newBidder(t).MakeRequests(request, &adapters.ExtraRequestInfo{})
			assert.Nil(t, requests)
			require.Len(t, errs, 1)
			assert.IsType(t, &errortypes.BadInput{}, errs[0])
			assert.Contains(t, errs[0].Error(), test.wantMessage)
		})
	}
}

// TestMakeRequestsSetsAppPublisher covers the app branch of setPublisherID; the
// site branch is exercised by the JSON fixtures.
func TestMakeRequestsSetsAppPublisher(t *testing.T) {
	request := &openrtb2.BidRequest{
		ID:  "app-req",
		Imp: []openrtb2.Imp{bannerImp("imp-1", json.RawMessage(`{"bidder":{"publisherId":"pub-a","placementId":"p1"}}`))},
		App: &openrtb2.App{Bundle: "com.orangeclickmedia.demo"},
	}

	requests, errs := newBidder(t).MakeRequests(request, &adapters.ExtraRequestInfo{})
	require.Empty(t, errs)
	require.Len(t, requests, 1)

	var sent openrtb2.BidRequest
	require.NoError(t, json.Unmarshal(requests[0].Body, &sent))
	require.NotNil(t, sent.App)
	require.NotNil(t, sent.App.Publisher)
	assert.Equal(t, "pub-a", sent.App.Publisher.ID)
	assert.Nil(t, sent.Site)
}

func TestMakeRequestsSanitizesForwardedContext(t *testing.T) {
	tests := []struct {
		name         string
		setPublisher func(*openrtb2.BidRequest, *openrtb2.Publisher)
		getPublisher func(*openrtb2.BidRequest) *openrtb2.Publisher
	}{
		{
			name: "site",
			setPublisher: func(request *openrtb2.BidRequest, publisher *openrtb2.Publisher) {
				request.Site = &openrtb2.Site{Domain: "example.test", Publisher: publisher}
			},
			getPublisher: func(request *openrtb2.BidRequest) *openrtb2.Publisher {
				return request.Site.Publisher
			},
		},
		{
			name: "app",
			setPublisher: func(request *openrtb2.BidRequest, publisher *openrtb2.Publisher) {
				request.App = &openrtb2.App{Bundle: "com.example.app", Publisher: publisher}
			},
			getPublisher: func(request *openrtb2.BidRequest) *openrtb2.Publisher {
				return request.App.Publisher
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			publisherExt := json.RawMessage(`{
				"prebid":{"parentAccount":"origin-host","unrelatedPrebid":"keep"},
				"unrelatedPublisher":{"value":1}
			}`)
			sharedPublisher := &openrtb2.Publisher{
				ID:   "origin-publisher",
				Name: "shared publisher",
				Ext:  publisherExt,
			}
			request := &openrtb2.BidRequest{
				ID: "req",
				Imp: []openrtb2.Imp{bannerImp("imp-1", json.RawMessage(`{
					"bidder":{"publisherId":"ocm-publisher","placementId":"placement-1"},
					"gpid":"/1234/example#slot"
				}`))},
				Ext: json.RawMessage(`{
					"prebid":{
						"aliases":{"publisher_ocm":"ocm"},
						"unrelatedPrebid":"keep"
					},
					"unrelatedRequest":{"value":2}
				}`),
			}
			test.setPublisher(request, sharedPublisher)

			requests, errs := newBidder(t).MakeRequests(request, &adapters.ExtraRequestInfo{})
			require.Empty(t, errs)
			require.Len(t, requests, 1)

			var sent openrtb2.BidRequest
			require.NoError(t, json.Unmarshal(requests[0].Body, &sent))
			sentPublisher := test.getPublisher(&sent)
			require.NotNil(t, sentPublisher)
			assert.Equal(t, "ocm-publisher", sentPublisher.ID)
			assert.Equal(t, "shared publisher", sentPublisher.Name)
			assert.JSONEq(t, `{
				"prebid":{"unrelatedPrebid":"keep"},
				"unrelatedPublisher":{"value":1}
			}`, string(sentPublisher.Ext))
			assert.JSONEq(t, `{
				"prebid":{
					"unrelatedPrebid":"keep"
				},
				"unrelatedRequest":{"value":2}
			}`, string(sent.Ext))
			require.Len(t, sent.Imp, 1)
			assert.JSONEq(t, `{
				"gpid":"/1234/example#slot",
				"prebid":{"storedrequest":{"id":"placement-1"}}
			}`, string(sent.Imp[0].Ext))

			// The publisher is shared by adapter requests. Sanitizing the OCM copy
			// must not rewrite the object retained by the originating request.
			assert.Equal(t, "origin-publisher", sharedPublisher.ID)
			assert.Equal(t, publisherExt, sharedPublisher.Ext)
			assert.NotSame(t, sharedPublisher, test.getPublisher(request))
		})
	}
}

func TestMakeBidsPreservesVideoMetadata(t *testing.T) {
	request := &openrtb2.BidRequest{ID: "req", Imp: []openrtb2.Imp{{ID: "imp-1"}}}
	response := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body: []byte(`{
			"id":"req",
			"seatbid":[{"bid":[{
				"id":"bid-1",
				"impid":"imp-1",
				"price":1.2,
				"mtype":2,
				"ext":{"prebid":{"type":"video","video":{"duration":30,"primary_category":"IAB1-2"}}}
			}]}]
		}`),
	}

	bidResponse, errs := newBidder(t).MakeBids(request, nil, response)
	require.Empty(t, errs)
	require.Len(t, bidResponse.Bids, 1)
	assert.Equal(t, &openrtb_ext.ExtBidPrebidVideo{
		Duration:        30,
		PrimaryCategory: "IAB1-2",
	}, bidResponse.Bids[0].BidVideo)
}

func TestMakeBidsIgnoresVideoMetadataForNonVideoBid(t *testing.T) {
	request := &openrtb2.BidRequest{ID: "req", Imp: []openrtb2.Imp{{ID: "imp-1"}}}
	response := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body: []byte(`{
			"id":"req",
			"seatbid":[{"bid":[{
				"id":"bid-1",
				"impid":"imp-1",
				"price":1.2,
				"mtype":1,
				"ext":{"prebid":{"type":"video","video":{"duration":30,"primary_category":"IAB1-2"}}}
			}]}]
		}`),
	}

	bidResponse, errs := newBidder(t).MakeBids(request, nil, response)
	require.Empty(t, errs)
	require.Len(t, bidResponse.Bids, 1)
	assert.Equal(t, openrtb_ext.BidTypeBanner, bidResponse.Bids[0].BidType)
	assert.Nil(t, bidResponse.Bids[0].BidVideo)
}

func TestMakeBidsErrorTypes(t *testing.T) {
	tests := []struct {
		name     string
		response *adapters.ResponseData
		wantType error
	}{
		{
			name:     "400 is bad input",
			response: &adapters.ResponseData{StatusCode: http.StatusBadRequest, Body: []byte(`{}`)},
			wantType: &errortypes.BadInput{},
		},
		{
			name:     "500 is bad server response",
			response: &adapters.ResponseData{StatusCode: http.StatusInternalServerError, Body: []byte(`{}`)},
			wantType: &errortypes.BadServerResponse{},
		},
		{
			name:     "unparseable body is bad server response",
			response: &adapters.ResponseData{StatusCode: http.StatusOK, Body: []byte(`not json`)},
			wantType: &errortypes.BadServerResponse{},
		},
	}

	request := &openrtb2.BidRequest{ID: "req", Imp: []openrtb2.Imp{{ID: "imp-1"}}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bidResponse, errs := newBidder(t).MakeBids(request, nil, test.response)
			assert.Nil(t, bidResponse)
			require.Len(t, errs, 1)
			assert.IsType(t, test.wantType, errs[0])
		})
	}
}

func TestGetMediaTypeForBid(t *testing.T) {
	tests := []struct {
		name  string
		mType openrtb2.MarkupType
		want  openrtb_ext.BidType
	}{
		{name: "banner", mType: openrtb2.MarkupBanner, want: openrtb_ext.BidTypeBanner},
		{name: "video", mType: openrtb2.MarkupVideo, want: openrtb_ext.BidTypeVideo},
		{name: "native", mType: openrtb2.MarkupNative, want: openrtb_ext.BidTypeNative},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bidType, err := getMediaTypeForBid(openrtb2.Bid{ID: "bid-1", MType: test.mType})
			require.NoError(t, err)
			assert.Equal(t, test.want, bidType)
		})
	}

	t.Run("audio is not supported", func(t *testing.T) {
		bidType, err := getMediaTypeForBid(openrtb2.Bid{ID: "bid-1", MType: openrtb2.MarkupAudio})
		assert.Empty(t, bidType)
		require.Error(t, err)
		assert.IsType(t, &errortypes.BadServerResponse{}, err)
		assert.Contains(t, err.Error(), `unsupported mtype 3 for bid "bid-1"`)
	})
}

// TestGetMediaTypeForBidUnsupportedMTypeIgnoresExt guards the precedence rule: a
// set-but-unsupported mtype must never fall through to ext.prebid.type. Allowing it
// would let an ext of "banner" relabel an audio bid, overriding the authoritative
// OpenRTB field and yielding a media type OCM does not declare support for.
func TestGetMediaTypeForBidUnsupportedMTypeIgnoresExt(t *testing.T) {
	tests := []struct {
		name    string
		mType   openrtb2.MarkupType
		bidExt  string
		wantMsg string
	}{
		{
			name:    "audio mtype with banner ext",
			mType:   openrtb2.MarkupAudio,
			bidExt:  `{"prebid":{"type":"banner"}}`,
			wantMsg: `unsupported mtype 3 for bid "bid-1"`,
		},
		{
			name:    "audio mtype with video ext",
			mType:   openrtb2.MarkupAudio,
			bidExt:  `{"prebid":{"type":"video"}}`,
			wantMsg: `unsupported mtype 3 for bid "bid-1"`,
		},
		{
			name:    "unknown mtype with banner ext",
			mType:   openrtb2.MarkupType(99),
			bidExt:  `{"prebid":{"type":"banner"}}`,
			wantMsg: `unsupported mtype 99 for bid "bid-1"`,
		},
		{
			name:    "negative mtype with banner ext",
			mType:   openrtb2.MarkupType(-1),
			bidExt:  `{"prebid":{"type":"banner"}}`,
			wantMsg: `unsupported mtype -1 for bid "bid-1"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bidType, err := getMediaTypeForBid(openrtb2.Bid{
				ID:    "bid-1",
				MType: test.mType,
				Ext:   json.RawMessage(test.bidExt),
			})

			assert.Empty(t, bidType, "an unsupported mtype must not be reclassified from ext")
			require.Error(t, err)
			assert.IsType(t, &errortypes.BadServerResponse{}, err)
			assert.Contains(t, err.Error(), test.wantMsg)
		})
	}
}

// TestGetMediaTypeForBidExtFallback covers the ext.prebid.type fallback, which
// matters when the upstream is itself a Prebid Server: it forwards the winning
// adapter's mtype verbatim and reports the type it resolved in ext.prebid.type,
// so a bid can legitimately arrive with mtype unset.
func TestGetMediaTypeForBidExtFallback(t *testing.T) {
	tests := []struct {
		name    string
		bidExt  string
		want    openrtb_ext.BidType
		wantErr bool
	}{
		{name: "banner from ext", bidExt: `{"prebid":{"type":"banner"}}`, want: openrtb_ext.BidTypeBanner},
		{name: "video from ext", bidExt: `{"prebid":{"type":"video"}}`, want: openrtb_ext.BidTypeVideo},
		{name: "native from ext", bidExt: `{"prebid":{"type":"native"}}`, want: openrtb_ext.BidTypeNative},
		{name: "audio in ext is rejected", bidExt: `{"prebid":{"type":"audio"}}`, wantErr: true},
		{name: "unknown type in ext is rejected", bidExt: `{"prebid":{"type":"banana"}}`, wantErr: true},
		{name: "ext without prebid is rejected", bidExt: `{"origbidcpm":1.5}`, wantErr: true},
		{name: "malformed ext is rejected", bidExt: `[]`, wantErr: true},
		{name: "empty ext is rejected", bidExt: ``, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bid := openrtb2.Bid{ID: "bid-1"}
			if test.bidExt != "" {
				bid.Ext = json.RawMessage(test.bidExt)
			}

			bidType, err := getMediaTypeForBid(bid)
			if test.wantErr {
				assert.Empty(t, bidType)
				require.Error(t, err)
				assert.IsType(t, &errortypes.BadServerResponse{}, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.want, bidType)
		})
	}
}

// TestGetMediaTypeForBidMTypeWinsOverExt pins mtype as authoritative when it names
// a supported type: a Prebid Server upstream can report a different type in ext
// than the mtype it forwards, and OpenRTB makes mtype the normative signal. The
// unsupported-mtype counterpart is TestGetMediaTypeForBidUnsupportedMTypeIgnoresExt.
func TestGetMediaTypeForBidMTypeWinsOverExt(t *testing.T) {
	bidType, err := getMediaTypeForBid(openrtb2.Bid{
		ID:    "bid-1",
		MType: openrtb2.MarkupBanner,
		Ext:   json.RawMessage(`{"prebid":{"type":"video"}}`),
	})

	require.NoError(t, err)
	assert.Equal(t, openrtb_ext.BidTypeBanner, bidType)
}

func newBidder(t *testing.T) adapters.Bidder {
	t.Helper()

	bidder, err := Builder(openrtb_ext.BidderOcm, config.Adapter{Endpoint: testEndpoint}, config.Server{})
	require.NoError(t, err)
	return bidder
}

func bannerImp(id string, ext json.RawMessage) openrtb2.Imp {
	return openrtb2.Imp{
		ID:     id,
		Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
		Ext:    ext,
	}
}

// TestSetStoredRequestID covers the ext rewriting directly, including the
// branches MakeRequests cannot reach because parseImpExt rejects a malformed
// imp.ext before this runs.
func TestSetStoredRequestID(t *testing.T) {
	tests := []struct {
		name    string
		impExt  string
		want    string
		wantErr string
	}{
		{
			name:   "creates prebid and storedrequest",
			impExt: `{"bidder":{"publisherId":"a"}}`,
			want:   `{"prebid":{"storedrequest":{"id":"slot-1"}}}`,
		},
		{
			name:   "merges into existing prebid",
			impExt: `{"prebid":{"is_rewarded_inventory":1}}`,
			want:   `{"prebid":{"is_rewarded_inventory":1,"storedrequest":{"id":"slot-1"}}}`,
		},
		{
			name:   "overwrites an existing storedrequest",
			impExt: `{"prebid":{"storedrequest":{"id":"stale"}}}`,
			want:   `{"prebid":{"storedrequest":{"id":"slot-1"}}}`,
		},
		{
			name:   "tolerates null prebid",
			impExt: `{"prebid":null}`,
			want:   `{"prebid":{"storedrequest":{"id":"slot-1"}}}`,
		},
		{
			name:   "preserves unrelated keys",
			impExt: `{"gpid":"/1/a","tid":"t-1"}`,
			want:   `{"gpid":"/1/a","tid":"t-1","prebid":{"storedrequest":{"id":"slot-1"}}}`,
		},
		{
			name:   "handles absent ext",
			impExt: ``,
			want:   `{"prebid":{"storedrequest":{"id":"slot-1"}}}`,
		},
		{
			name:    "rejects non-object prebid",
			impExt:  `{"prebid":"not-an-object"}`,
			wantErr: "unable to unmarshal ext.prebid",
		},
		{
			name:    "rejects non-object ext",
			impExt:  `["nope"]`,
			wantErr: "unable to unmarshal ext",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			imp := openrtb2.Imp{ID: "imp-1"}
			if test.impExt != "" {
				imp.Ext = json.RawMessage(test.impExt)
			}

			err := setStoredRequestID(&imp, "slot-1")
			if test.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.wantErr)
				return
			}

			require.NoError(t, err)
			assert.JSONEq(t, test.want, string(imp.Ext))
		})
	}
}

// TestMakeRequestsMalformedImpExtPrebid pins the error type for an imp.ext whose
// prebid member is not an object. adapters.ExtImpBidder types that member, so
// parseImpExt rejects it before setStoredRequestID is ever reached; either way the
// publisher gets a BadInput.
func TestMakeRequestsMalformedImpExtPrebid(t *testing.T) {
	imp := bannerImp("imp-1", json.RawMessage(`{"bidder":{"publisherId":"a","placementId":"p"},"prebid":"not-an-object"}`))
	request := &openrtb2.BidRequest{
		ID:   "req",
		Imp:  []openrtb2.Imp{imp},
		Site: &openrtb2.Site{Domain: "example.gr"},
	}

	requests, errs := newBidder(t).MakeRequests(request, &adapters.ExtraRequestInfo{})
	assert.Nil(t, requests)
	require.Len(t, errs, 1)
	assert.IsType(t, &errortypes.BadInput{}, errs[0])
	assert.Contains(t, errs[0].Error(), "imp[0]: unable to unmarshal ext")
}
