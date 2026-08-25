package tunnl

import (
	"errors"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
	"github.com/stretchr/testify/assert"
)

const testEndpoint = "https://us1.rapidtag.net/api/v1/bid?sid=tunnlus{{.MediaType}}"

func buildTestBidder(t *testing.T) adapters.Bidder {
	t.Helper()

	bidder, buildErr := Builder(openrtb_ext.BidderTunnl, config.Adapter{
		Endpoint: testEndpoint},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}
	return bidder
}

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "tunnltest", buildTestBidder(t))
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderTunnl, config.Adapter{
		Endpoint: "{{Malformed}}"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
}

// Tunnl takes no publisher parameters, so imp.ext is never read: an imp with no
// ext at all still produces a request against the host configured endpoint.
func TestMakeRequestsIgnoresImpExt(t *testing.T) {
	bidder := buildTestBidder(t)

	requests, errs := bidder.MakeRequests(&openrtb2.BidRequest{
		ID: "req-1",
		Imp: []openrtb2.Imp{{
			ID:     "imp-1",
			Banner: &openrtb2.Banner{},
		}},
	}, &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs)
	assert.Len(t, requests, 1)
	assert.Equal(t, "https://us1.rapidtag.net/api/v1/bid?sid=tunnlusban", requests[0].Uri)
}

// The only MakeRequests failure left is an imp with no supported media type,
// since there are no parameters to reject.
func TestMakeRequestsUnsupportedMediaType(t *testing.T) {
	bidder := buildTestBidder(t)

	requests, errs := bidder.MakeRequests(&openrtb2.BidRequest{
		ID: "req-1",
		Imp: []openrtb2.Imp{{
			ID:  "imp-1",
			Ext: []byte(`{"bidder":{}}`),
		}},
	}, &adapters.ExtraRequestInfo{})

	assert.Empty(t, requests)
	assert.Len(t, errs, 1)

	var badInput *errortypes.BadInput
	assert.True(t, errors.As(errs[0], &badInput), "expected BadInput, got %T", errs[0])
}

// One bad imp must not prevent the remaining imps from being dispatched.
func TestMakeRequestsPartialFailure(t *testing.T) {
	bidder := buildTestBidder(t)

	requests, errs := bidder.MakeRequests(&openrtb2.BidRequest{
		ID: "req-1",
		Imp: []openrtb2.Imp{
			{ID: "good", Banner: &openrtb2.Banner{}, Ext: []byte(`{"bidder":{}}`)},
			{ID: "bad", Ext: []byte(`{"bidder":{}}`)},
		},
	}, &adapters.ExtraRequestInfo{})

	assert.Len(t, requests, 1)
	assert.Len(t, errs, 1)
	assert.Equal(t, []string{"good"}, requests[0].ImpIDs)
}

// The JSON fixtures compare error messages only, so the error TYPE contract is
// pinned here instead.
func TestMakeBidsErrorTypes(t *testing.T) {
	bidder := buildTestBidder(t)
	request := &openrtb2.BidRequest{ID: "req-1", Imp: []openrtb2.Imp{{ID: "imp-1"}}}
	requestData := &adapters.RequestData{Uri: "https://us1.rapidtag.net/api/v1/bid?sid=tunnlusban"}

	t.Run("400 is bad input", func(t *testing.T) {
		_, errs := bidder.MakeBids(request, requestData, &adapters.ResponseData{StatusCode: 400})

		assert.Len(t, errs, 1)
		var badInput *errortypes.BadInput
		assert.True(t, errors.As(errs[0], &badInput), "expected BadInput, got %T", errs[0])
	})

	t.Run("500 is bad server response", func(t *testing.T) {
		_, errs := bidder.MakeBids(request, requestData, &adapters.ResponseData{StatusCode: 500})

		assert.Len(t, errs, 1)
		var badServerResponse *errortypes.BadServerResponse
		assert.True(t, errors.As(errs[0], &badServerResponse), "expected BadServerResponse, got %T", errs[0])
	})

	t.Run("204 is a silent no-bid", func(t *testing.T) {
		response, errs := bidder.MakeBids(request, requestData, &adapters.ResponseData{StatusCode: 204})

		assert.Nil(t, response)
		assert.Nil(t, errs)
	})
}

// When the bid omits mtype and the request URI cannot be attributed to a format,
// the bid must be rejected rather than guessed.
func TestMakeBidsUndeterminableMediaType(t *testing.T) {
	bidder := buildTestBidder(t)

	_, errs := bidder.MakeBids(
		&openrtb2.BidRequest{ID: "req-1", Imp: []openrtb2.Imp{{ID: "imp-1"}}},
		&adapters.RequestData{Uri: "https://us1.rapidtag.net/api/v1/bid?sid=urban"},
		&adapters.ResponseData{
			StatusCode: 200,
			Body:       []byte(`{"id":"req-1","seatbid":[{"bid":[{"id":"b1","impid":"imp-1","price":1}]}]}`),
		})

	assert.Len(t, errs, 1)
	var badServerResponse *errortypes.BadServerResponse
	assert.True(t, errors.As(errs[0], &badServerResponse), "expected BadServerResponse, got %T", errs[0])
}

// An audio bid can never match a Tunnl request, since audio imps are dropped in
// MakeRequests. It must be rejected rather than relabelled as the format the
// sub-request happened to be built for.
func TestMakeBidsRejectsAudioBid(t *testing.T) {
	bidder := buildTestBidder(t)

	response, errs := bidder.MakeBids(
		&openrtb2.BidRequest{ID: "req-1", Imp: []openrtb2.Imp{{ID: "imp-1"}}},
		&adapters.RequestData{Uri: "https://us1.rapidtag.net/api/v1/bid?sid=tunnlusban"},
		&adapters.ResponseData{
			StatusCode: 200,
			Body:       []byte(`{"id":"req-1","seatbid":[{"bid":[{"id":"b1","impid":"imp-1","price":1,"mtype":3}]}]}`),
		})

	assert.Empty(t, response.Bids)
	assert.Len(t, errs, 1)

	var badServerResponse *errortypes.BadServerResponse
	assert.True(t, errors.As(errs[0], &badServerResponse), "expected BadServerResponse, got %T", errs[0])
	assert.Contains(t, errs[0].Error(), "audio")
}

// The region lives entirely in the host configured endpoint. A host serving
// another datacenter overrides it, and both the host and the sid must follow.
func TestHostRegionOverride(t *testing.T) {
	testCases := []struct {
		name     string
		endpoint string
		expected string
	}{
		{
			name:     "us is the open source default",
			endpoint: testEndpoint,
			expected: "https://us1.rapidtag.net/api/v1/bid?sid=tunnlusvid",
		},
		{
			name:     "eu",
			endpoint: "https://eu1.rapidtag.net/api/v1/bid?sid=tunnleu{{.MediaType}}",
			expected: "https://eu1.rapidtag.net/api/v1/bid?sid=tunnleuvid",
		},
		{
			name:     "ap",
			endpoint: "https://ap1.rapidtag.net/api/v1/bid?sid=tunnlap{{.MediaType}}",
			expected: "https://ap1.rapidtag.net/api/v1/bid?sid=tunnlapvid",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			bidder, buildErr := Builder(openrtb_ext.BidderTunnl,
				config.Adapter{Endpoint: test.endpoint},
				config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
			if buildErr != nil {
				t.Fatalf("Builder returned unexpected error %v", buildErr)
			}

			requests, errs := bidder.MakeRequests(&openrtb2.BidRequest{
				ID:  "req-1",
				Imp: []openrtb2.Imp{{ID: "imp-1", Video: &openrtb2.Video{}}},
			}, &adapters.ExtraRequestInfo{})

			assert.Empty(t, errs)
			assert.Len(t, requests, 1)
			assert.Equal(t, test.expected, requests[0].Uri)
		})
	}
}

// An imp that declares every supported media type must be split into one
// single-format request each, with no other media type object left behind.
func TestMakeRequestsSplitsAllFormats(t *testing.T) {
	bidder := buildTestBidder(t)

	requests, errs := bidder.MakeRequests(&openrtb2.BidRequest{
		ID: "req-1",
		Imp: []openrtb2.Imp{{
			ID:     "imp-1",
			Banner: &openrtb2.Banner{},
			Video:  &openrtb2.Video{},
			Native: &openrtb2.Native{},
			Audio:  &openrtb2.Audio{MIMEs: []string{"audio/mp4"}},
		}},
	}, &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs)
	assert.Len(t, requests, 3)

	uris := make([]string, 0, len(requests))
	for _, request := range requests {
		uris = append(uris, request.Uri)

		var body openrtb2.BidRequest
		assert.NoError(t, jsonutil.Unmarshal(request.Body, &body))
		assert.Len(t, body.Imp, 1)

		// Audio is not a declared capability and must never be forwarded.
		assert.Nil(t, body.Imp[0].Audio)

		declared := 0
		for _, present := range []bool{
			body.Imp[0].Banner != nil, body.Imp[0].Video != nil, body.Imp[0].Native != nil,
		} {
			if present {
				declared++
			}
		}
		assert.Equal(t, 1, declared, "each split request carries exactly one media type")
	}

	assert.ElementsMatch(t, []string{
		"https://us1.rapidtag.net/api/v1/bid?sid=tunnlusban",
		"https://us1.rapidtag.net/api/v1/bid?sid=tunnlusvid",
		"https://us1.rapidtag.net/api/v1/bid?sid=tunnlusnat",
	}, uris)
}

// The media type of a bid is recovered from the URI the adapter itself built,
// so it must survive an endpoint whose shape the adapter knows nothing about.
// Anything not emitted by this adapter must not be attributed to a format.
func TestURIFormatMapping(t *testing.T) {
	testCases := []struct {
		name     string
		endpoint string
		expected map[string]string
	}{
		{
			name:     "default sid based endpoint",
			endpoint: testEndpoint,
			expected: map[string]string{
				"https://us1.rapidtag.net/api/v1/bid?sid=tunnlusban": formatBanner,
				"https://us1.rapidtag.net/api/v1/bid?sid=tunnlusvid": formatVideo,
				"https://us1.rapidtag.net/api/v1/bid?sid=tunnlusnat": formatNative,
			},
		},
		{
			name:     "host proxy that carries no sid at all",
			endpoint: "https://proxy.internal/tunnl?fmt={{.MediaType}}",
			expected: map[string]string{
				"https://proxy.internal/tunnl?fmt=ban": formatBanner,
				"https://proxy.internal/tunnl?fmt=vid": formatVideo,
				"https://proxy.internal/tunnl?fmt=nat": formatNative,
			},
		},
		{
			name:     "media type in the path rather than the query",
			endpoint: "https://proxy.internal/tunnl/{{.MediaType}}/bid",
			expected: map[string]string{
				"https://proxy.internal/tunnl/ban/bid": formatBanner,
				"https://proxy.internal/tunnl/vid/bid": formatVideo,
				"https://proxy.internal/tunnl/nat/bid": formatNative,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			bidder, buildErr := Builder(openrtb_ext.BidderTunnl,
				config.Adapter{Endpoint: test.endpoint},
				config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
			if buildErr != nil {
				t.Fatalf("Builder returned unexpected error %v", buildErr)
			}

			assert.Equal(t, test.expected, bidder.(*adapter).uriFormats)

			// A URI this adapter never emits stays unattributed, so a bid arriving
			// against it is rejected rather than guessed.
			assert.Empty(t, bidder.(*adapter).uriFormats["https://proxy.internal/bid?sid=urban"])
		})
	}
}

// An endpoint that does not vary by media type would send every format to the
// same URI and make responses unattributable, so it is rejected at build time
// rather than silently dropping every bid at auction time.
func TestBuilderRejectsFormatInvariantEndpoint(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderTunnl,
		config.Adapter{Endpoint: "https://us1.rapidtag.net/api/v1/bid?sid=tunnlus"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
	assert.Contains(t, buildErr.Error(), "must vary by media type")
}
