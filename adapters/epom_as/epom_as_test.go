package epom_as

import (
	"encoding/json"
	"fmt"
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

const testEndpoint = "https://{{.Host}}/hb/bid"

// End-to-end MakeRequests/MakeBids behaviour is exercised by the JSON fixtures
// under epom_astest/exemplary and epom_astest/supplemental. The Go tests below
// cover only what fixtures cannot: the error-TYPE contract (BadInput vs
// BadServerResponse, which fixtures compare by message only), the promise that
// the caller's request is never mutated, and the determinism of the custom-param
// merge, which a single fixture run cannot distinguish from luck.

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderEpomAs,
		config.Adapter{Endpoint: testEndpoint},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 849, DataCenter: "2"},
	)
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "epom_astest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(
		openrtb_ext.BidderEpomAs,
		config.Adapter{Endpoint: "{{Malformed}}"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 849, DataCenter: "2"},
	)

	assert.Error(t, buildErr)
}

func TestMakeRequestsErrorsAreBadInput(t *testing.T) {
	testCases := []struct {
		name    string
		impExt  json.RawMessage
		message string
	}{
		{
			name:    "imp.ext is not an object",
			impExt:  json.RawMessage(`"not-an-object"`),
			message: "imp test-imp-id: missing bidder ext",
		},
		{
			name:    "imp.ext.bidder is not an object",
			impExt:  json.RawMessage(`{"bidder":"not-an-object"}`),
			message: "imp test-imp-id: cannot resolve host or placementKey",
		},
		{
			name:    "host would rewrite the outbound url",
			impExt:  json.RawMessage(`{"bidder":{"host":"ads.example.com/collect","placementKey":"a4f21c9e7b"}}`),
			message: "imp test-imp-id: invalid host",
		},
		{
			name:    "host is absent",
			impExt:  json.RawMessage(`{"bidder":{"placementKey":"a4f21c9e7b"}}`),
			message: "imp test-imp-id: invalid host",
		},
		{
			name:    "placementKey is absent",
			impExt:  json.RawMessage(`{"bidder":{"host":"ads.example.com"}}`),
			message: "imp test-imp-id: missing placementKey",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requests, errs := newAdapter(t).MakeRequests(givenRequest(givenImp("test-imp-id", tc.impExt)), &adapters.ExtraRequestInfo{})

			assert.Empty(t, requests)
			require.Len(t, errs, 1)
			assert.IsType(t, &errortypes.BadInput{}, errs[0])
			assert.Contains(t, errs[0].Error(), tc.message)
		})
	}
}

func TestMakeBidsErrorTypes(t *testing.T) {
	request := givenRequest(givenImp("test-imp-id", json.RawMessage(`{"bidder":{"host":"ads.example.com","placementKey":"a4f21c9e7b"}}`)))

	testCases := []struct {
		name     string
		status   int
		body     string
		wantType error
	}{
		{
			name:     "4xx is the publisher's problem",
			status:   http.StatusBadRequest,
			body:     `{}`,
			wantType: &errortypes.BadInput{},
		},
		{
			name:     "5xx is the exchange's problem",
			status:   http.StatusInternalServerError,
			body:     `{}`,
			wantType: &errortypes.BadServerResponse{},
		},
		{
			name:     "unparseable body",
			status:   http.StatusOK,
			body:     `not json at all`,
			wantType: &errortypes.BadServerResponse{},
		},
		{
			name:     "unsupported mtype",
			status:   http.StatusOK,
			body:     `{"id":"test-request-id","cur":"USD","seatbid":[{"bid":[{"id":"b1","impid":"test-imp-id","price":1,"mtype":3}]}]}`,
			wantType: &errortypes.BadServerResponse{},
		},
		{
			name:     "no mtype and no matching imp",
			status:   http.StatusOK,
			body:     `{"id":"test-request-id","cur":"USD","seatbid":[{"bid":[{"id":"b1","impid":"no-such-imp","price":1}]}]}`,
			wantType: &errortypes.BadServerResponse{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			response := &adapters.ResponseData{StatusCode: tc.status, Body: []byte(tc.body)}

			_, errs := newAdapter(t).MakeBids(request, &adapters.RequestData{}, response)

			require.Len(t, errs, 1)
			assert.IsType(t, tc.wantType, errs[0])
		})
	}
}

// TestCallerRequestNotMutated pins the copy-on-write surface: MakeRequests
// rewrites tagid, the floor and imp.ext, and the exchange reuses the same
// request object for every bidder in the auction.
func TestCallerRequestNotMutated(t *testing.T) {
	impExt := json.RawMessage(`{"bidder":{"host":"ads.example.com","placementKey":"a4f21c9e7b","channel":"sports-uk","customParams":{"section":"sport"},"bidFloor":2.5,"bidFloorCur":"EUR"}}`)
	request := givenRequest(givenImp("test-imp-id", impExt))

	requests, errs := newAdapter(t).MakeRequests(request, &adapters.ExtraRequestInfo{})
	assert.Empty(t, errs)
	require.Len(t, requests, 1)

	assert.Empty(t, request.Imp[0].TagID, "caller's imp[0].TagID must not be mutated")
	assert.EqualValues(t, 0, request.Imp[0].BidFloor, "caller's imp[0].BidFloor must not be mutated")
	assert.Empty(t, request.Imp[0].BidFloorCur, "caller's imp[0].BidFloorCur must not be mutated")
	assert.JSONEq(t, string(impExt), string(request.Imp[0].Ext), "caller's imp[0].Ext must not be mutated")
	assert.Len(t, request.Imp, 1, "caller's imp slice must not be re-sliced")
}

func TestGetMediaTypeForBid(t *testing.T) {
	bannerImp := &openrtb2.Imp{ID: "banner-imp", Banner: &openrtb2.Banner{}}
	videoImp := &openrtb2.Imp{ID: "video-imp", Video: &openrtb2.Video{}}
	nativeImp := &openrtb2.Imp{ID: "native-imp", Native: &openrtb2.Native{}}
	mixedImp := &openrtb2.Imp{ID: "mixed-imp", Banner: &openrtb2.Banner{}, Video: &openrtb2.Video{}}
	imps := map[string]*openrtb2.Imp{
		bannerImp.ID: bannerImp, videoImp.ID: videoImp, nativeImp.ID: nativeImp, mixedImp.ID: mixedImp,
	}

	testCases := []struct {
		name     string
		bid      openrtb2.Bid
		wantType openrtb_ext.BidType
		wantErr  string
	}{
		{
			name:     "mtype banner",
			bid:      openrtb2.Bid{ID: "b1", ImpID: "banner-imp", MType: openrtb2.MarkupBanner},
			wantType: openrtb_ext.BidTypeBanner,
		},
		{
			name:     "no mtype resolves from a banner imp",
			bid:      openrtb2.Bid{ID: "b2", ImpID: "banner-imp"},
			wantType: openrtb_ext.BidTypeBanner,
		},
		{
			name:     "no mtype resolves from a video imp",
			bid:      openrtb2.Bid{ID: "b3", ImpID: "video-imp"},
			wantType: openrtb_ext.BidTypeVideo,
		},
		{
			name:    "no mtype and no matching imp",
			bid:     openrtb2.Bid{ID: "b4", ImpID: "no-such-imp"},
			wantErr: "unresolved mtype for bid b4: no imp no-such-imp",
		},
		{
			name:     "mtype video",
			bid:      openrtb2.Bid{ID: "b5", ImpID: "video-imp", MType: openrtb2.MarkupVideo},
			wantType: openrtb_ext.BidTypeVideo,
		},
		{
			name:     "mtype native",
			bid:      openrtb2.Bid{ID: "b6", ImpID: "native-imp", MType: openrtb2.MarkupNative},
			wantType: openrtb_ext.BidTypeNative,
		},
		{
			name:    "mtype audio, which this ad server does not serve",
			bid:     openrtb2.Bid{ID: "b7", ImpID: "banner-imp", MType: openrtb2.MarkupAudio},
			wantErr: "unsupported mtype 3 for bid b7",
		},
		{
			// An imp offering two formats says nothing about which one a bid without
			// mtype filled, and guessing renders the wrong creative into the slot.
			name:    "no mtype on an imp offering two formats",
			bid:     openrtb2.Bid{ID: "b8", ImpID: "mixed-imp"},
			wantErr: "unresolved mtype for bid b8: imp mixed-imp offers 2 formats",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bidType, err := getMediaTypeForBid(tc.bid, imps)

			if tc.wantErr != "" {
				require.Error(t, err)
				assert.IsType(t, &errortypes.BadServerResponse{}, err)
				assert.Equal(t, tc.wantErr, err.Error())
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.wantType, bidType)
		})
	}
}

// TestCustomParamsMergeIsDeterministic guards the reason the adapter caps
// nothing: with no entry dropped, Go's randomised map iteration order cannot
// change which keys reach the wire, so the marshalled body is stable across
// runs. A single fixture pass would not tell a stable result from a lucky one.
func TestCustomParamsMergeIsDeterministic(t *testing.T) {
	params := map[string]interface{}{}
	for i := 0; i < 40; i++ {
		params[fmt.Sprintf("key%02d", i)] = fmt.Sprintf("value-%02d", i)
	}

	first, err := mergeCustomParams(nil, params)
	require.NoError(t, err)
	require.NotNil(t, first)

	for i := 0; i < 50; i++ {
		again, err := mergeCustomParams(nil, params)
		require.NoError(t, err)
		assert.Equal(t, string(first), string(again))
	}

	var decoded map[string]string
	require.NoError(t, json.Unmarshal(first, &decoded))
	assert.Len(t, decoded, 40, "no custom param may be dropped")
}

func newAdapter(t *testing.T) adapters.Bidder {
	t.Helper()
	bidder, err := Builder(
		openrtb_ext.BidderEpomAs,
		config.Adapter{Endpoint: testEndpoint},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 849, DataCenter: "2"},
	)
	require.NoError(t, err)
	return bidder
}

func givenImp(id string, ext json.RawMessage) openrtb2.Imp {
	return openrtb2.Imp{
		ID:     id,
		Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
		Ext:    ext,
	}
}

func givenRequest(imps ...openrtb2.Imp) *openrtb2.BidRequest {
	return &openrtb2.BidRequest{
		ID:   "test-request-id",
		Imp:  imps,
		Site: &openrtb2.Site{Page: "https://publisher.example.com/article"},
	}
}
