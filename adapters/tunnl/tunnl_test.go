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

const (
	testEndpoint = "https://us1.rapidtag.net/api/v1/bid?sid={{.SourceId}}"
	testSid      = "tunnl_x_use_g"
	testImpExt   = `{"bidder":{"sid":"tunnl_x_use_g"}}`
	testURI      = "https://us1.rapidtag.net/api/v1/bid?sid=tunnl_x_use_g"
)

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

// The sid is the one publisher parameter, and it ends up in the endpoint URL.
func TestMakeRequestsPlacesSidInEndpoint(t *testing.T) {
	bidder := buildTestBidder(t)

	requests, errs := bidder.MakeRequests(&openrtb2.BidRequest{
		ID: "req-1",
		Imp: []openrtb2.Imp{{
			ID:     "imp-1",
			Banner: &openrtb2.Banner{},
			Ext:    []byte(testImpExt),
		}},
	}, &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs)
	assert.Len(t, requests, 1)
	assert.Equal(t, testURI, requests[0].Uri)
}

// A missing or unusable sid is a publisher misconfiguration, so the imp is
// rejected as bad input rather than sent without identification.
func TestMakeRequestsRejectsMissingSid(t *testing.T) {
	testCases := []struct {
		name string
		ext  []byte
	}{
		{name: "no sid field", ext: []byte(`{"bidder":{}}`)},
		{name: "empty sid", ext: []byte(`{"bidder":{"sid":""}}`)},
		{name: "sid of the wrong type", ext: []byte(`{"bidder":{"sid":123}}`)},
		{name: "malformed imp ext", ext: []byte(`{"bidder":`)},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			bidder := buildTestBidder(t)

			requests, errs := bidder.MakeRequests(&openrtb2.BidRequest{
				ID:  "req-1",
				Imp: []openrtb2.Imp{{ID: "imp-1", Banner: &openrtb2.Banner{}, Ext: test.ext}},
			}, &adapters.ExtraRequestInfo{})

			assert.Empty(t, requests)
			assert.Len(t, errs, 1)

			var badInput *errortypes.BadInput
			assert.True(t, errors.As(errs[0], &badInput), "expected BadInput, got %T", errs[0])
		})
	}
}

// The sid carries the region and the endpoint host carries it again, so a
// mismatch would bid against the wrong region and be attributed to the wrong
// place. It is rejected instead.
func TestMakeRequestsValidatesSidRegion(t *testing.T) {
	testCases := []struct {
		name        string
		endpoint    string
		sid         string
		expectError bool
	}{
		{name: "us east on the us host", endpoint: testEndpoint, sid: "tunnl_x_use_g"},
		{name: "us west on the us host", endpoint: testEndpoint, sid: "tunnl_x_usw_g"},
		{
			name:     "eu on the eu host",
			endpoint: "https://eu1.rapidtag.net/api/v1/bid?sid={{.SourceId}}",
			sid:      "tunnl_x_eu_g",
		},
		{
			name:     "ap on the ap host",
			endpoint: "https://ap1.rapidtag.net/api/v1/bid?sid={{.SourceId}}",
			sid:      "tunnl_x_ap_g",
		},
		{
			name:     "partner name containing underscores",
			endpoint: testEndpoint,
			sid:      "tunnl_big_partner_co_use_g",
		},
		{name: "eu sid on the us host", endpoint: testEndpoint, sid: "tunnl_x_eu_g", expectError: true},
		{
			name:        "us sid on the eu host",
			endpoint:    "https://eu1.rapidtag.net/api/v1/bid?sid={{.SourceId}}",
			sid:         "tunnl_x_use_g",
			expectError: true,
		},
		{name: "unknown region", endpoint: testEndpoint, sid: "tunnl_x_zz_g", expectError: true},
		{name: "too few segments to carry a region", endpoint: testEndpoint, sid: "tunnl_use", expectError: true},
		{
			name:     "unrecognised host skips the check",
			endpoint: "https://proxy.internal/tunnl?sid={{.SourceId}}",
			sid:      "anything-at-all",
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
				ID: "req-1",
				Imp: []openrtb2.Imp{{
					ID:     "imp-1",
					Banner: &openrtb2.Banner{},
					Ext:    []byte(`{"bidder":{"sid":"` + test.sid + `"}}`),
				}},
			}, &adapters.ExtraRequestInfo{})

			if test.expectError {
				assert.Empty(t, requests)
				assert.Len(t, errs, 1)

				var badInput *errortypes.BadInput
				assert.True(t, errors.As(errs[0], &badInput), "expected BadInput, got %T", errs[0])
				return
			}

			assert.Empty(t, errs)
			assert.Len(t, requests, 1)
		})
	}
}

// The sid travels in the URL, so imps can only share a request when they share
// a sid. The common case, one sid across every ad unit, stays a single call.
func TestMakeRequestsGroupsImpsBySid(t *testing.T) {
	bidder := buildTestBidder(t)

	requests, errs := bidder.MakeRequests(&openrtb2.BidRequest{
		ID: "req-1",
		Imp: []openrtb2.Imp{
			{ID: "imp-1", Banner: &openrtb2.Banner{}, Ext: []byte(`{"bidder":{"sid":"tunnl_x_use_g"}}`)},
			{ID: "imp-2", Video: &openrtb2.Video{}, Ext: []byte(`{"bidder":{"sid":"tunnl_x_usw_g"}}`)},
			{ID: "imp-3", Native: &openrtb2.Native{}, Ext: []byte(`{"bidder":{"sid":"tunnl_x_use_g"}}`)},
		},
	}, &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs)
	assert.Len(t, requests, 2, "one request per distinct sid")

	// Grouping preserves the order the sids first appeared in, so the mapping
	// from request to sid is stable.
	assert.Equal(t, "https://us1.rapidtag.net/api/v1/bid?sid=tunnl_x_use_g", requests[0].Uri)
	assert.Equal(t, []string{"imp-1", "imp-3"}, requests[0].ImpIDs)

	assert.Equal(t, "https://us1.rapidtag.net/api/v1/bid?sid=tunnl_x_usw_g", requests[1].Uri)
	assert.Equal(t, []string{"imp-2"}, requests[1].ImpIDs)

	// Every imp in a group must reach the endpoint in one body.
	var body openrtb2.BidRequest
	assert.NoError(t, jsonutil.Unmarshal(requests[0].Body, &body))
	assert.Len(t, body.Imp, 2)
}

// One bad imp must not prevent the remaining imps from being dispatched.
func TestMakeRequestsPartialFailure(t *testing.T) {
	bidder := buildTestBidder(t)

	requests, errs := bidder.MakeRequests(&openrtb2.BidRequest{
		ID: "req-1",
		Imp: []openrtb2.Imp{
			{ID: "good", Banner: &openrtb2.Banner{}, Ext: []byte(testImpExt)},
			{ID: "bad", Banner: &openrtb2.Banner{}, Ext: []byte(`{"bidder":{}}`)},
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
	requestData := &adapters.RequestData{Uri: testURI}

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

// mtype only exists from ORTB 2.6, so a 2.5 response omits it and the media
// type has to come from the impression the bid was made against.
func TestMakeBidsFallsBackToImpMediaType(t *testing.T) {
	testCases := []struct {
		name     string
		imp      openrtb2.Imp
		expected openrtb_ext.BidType
	}{
		{
			name:     "banner only imp",
			imp:      openrtb2.Imp{ID: "imp-1", Banner: &openrtb2.Banner{}},
			expected: openrtb_ext.BidTypeBanner,
		},
		{
			name:     "video only imp",
			imp:      openrtb2.Imp{ID: "imp-1", Video: &openrtb2.Video{}},
			expected: openrtb_ext.BidTypeVideo,
		},
		{
			name:     "native only imp",
			imp:      openrtb2.Imp{ID: "imp-1", Native: &openrtb2.Native{}},
			expected: openrtb_ext.BidTypeNative,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			bidder := buildTestBidder(t)

			response, errs := bidder.MakeBids(
				&openrtb2.BidRequest{ID: "req-1", Imp: []openrtb2.Imp{test.imp}},
				&adapters.RequestData{Uri: testURI},
				&adapters.ResponseData{
					StatusCode: 200,
					Body:       []byte(`{"id":"req-1","seatbid":[{"bid":[{"id":"b1","impid":"imp-1","price":1}]}]}`),
				})

			assert.Empty(t, errs)
			assert.Len(t, response.Bids, 1)
			assert.Equal(t, test.expected, response.Bids[0].BidType)
		})
	}
}

// mtype takes precedence over the impression, so a multi format imp is still
// attributed correctly when the response says what it bid on.
func TestMakeBidsPrefersMtype(t *testing.T) {
	bidder := buildTestBidder(t)

	response, errs := bidder.MakeBids(
		&openrtb2.BidRequest{ID: "req-1", Imp: []openrtb2.Imp{{
			ID:     "imp-1",
			Banner: &openrtb2.Banner{},
			Video:  &openrtb2.Video{},
		}}},
		&adapters.RequestData{Uri: testURI},
		&adapters.ResponseData{
			StatusCode: 200,
			Body:       []byte(`{"id":"req-1","seatbid":[{"bid":[{"id":"b1","impid":"imp-1","price":1,"mtype":2}]}]}`),
		})

	assert.Empty(t, errs)
	assert.Len(t, response.Bids, 1)
	assert.Equal(t, openrtb_ext.BidTypeVideo, response.Bids[0].BidType)
}

// A multi format imp gives no single answer without mtype, so guessing would
// mislabel the creative. Such a bid is rejected.
func TestMakeBidsRejectsAmbiguousMediaType(t *testing.T) {
	bidder := buildTestBidder(t)

	response, errs := bidder.MakeBids(
		&openrtb2.BidRequest{ID: "req-1", Imp: []openrtb2.Imp{{
			ID:     "imp-1",
			Banner: &openrtb2.Banner{},
			Video:  &openrtb2.Video{},
		}}},
		&adapters.RequestData{Uri: testURI},
		&adapters.ResponseData{
			StatusCode: 200,
			Body:       []byte(`{"id":"req-1","seatbid":[{"bid":[{"id":"b1","impid":"imp-1","price":1}]}]}`),
		})

	assert.Empty(t, response.Bids)
	assert.Len(t, errs, 1)

	var badServerResponse *errortypes.BadServerResponse
	assert.True(t, errors.As(errs[0], &badServerResponse), "expected BadServerResponse, got %T", errs[0])
	assert.Contains(t, errs[0].Error(), "mtype")
}

// A bid against an imp that was never sent cannot be attributed at all.
func TestMakeBidsRejectsUnknownImp(t *testing.T) {
	bidder := buildTestBidder(t)

	_, errs := bidder.MakeBids(
		&openrtb2.BidRequest{ID: "req-1", Imp: []openrtb2.Imp{{ID: "imp-1", Banner: &openrtb2.Banner{}}}},
		&adapters.RequestData{Uri: testURI},
		&adapters.ResponseData{
			StatusCode: 200,
			Body:       []byte(`{"id":"req-1","seatbid":[{"bid":[{"id":"b1","impid":"nonexistent","price":1}]}]}`),
		})

	assert.Len(t, errs, 1)
	var badServerResponse *errortypes.BadServerResponse
	assert.True(t, errors.As(errs[0], &badServerResponse), "expected BadServerResponse, got %T", errs[0])
}

// Tunnl reads device signals off the HTTP headers for geo and device detection,
// which is the only source available for app traffic.
func TestMakeRequestsForwardsDeviceHeaders(t *testing.T) {
	testCases := []struct {
		name     string
		device   *openrtb2.Device
		expected map[string][]string
	}{
		{
			name:     "no device",
			device:   nil,
			expected: map[string][]string{},
		},
		{
			name:   "ua and ipv4",
			device: &openrtb2.Device{UA: "test-agent", IP: "1.2.3.4"},
			expected: map[string][]string{
				"User-Agent":      {"test-agent"},
				"X-Forwarded-For": {"1.2.3.4"},
			},
		},
		{
			name:   "ipv6 and ipv4 are both forwarded",
			device: &openrtb2.Device{IPv6: "2001:db8::1", IP: "1.2.3.4"},
			expected: map[string][]string{
				"X-Forwarded-For": {"1.2.3.4", "2001:db8::1"},
			},
		},
		{
			name:     "empty device fields are omitted",
			device:   &openrtb2.Device{UA: "", IP: ""},
			expected: map[string][]string{},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			bidder := buildTestBidder(t)

			requests, errs := bidder.MakeRequests(&openrtb2.BidRequest{
				ID:     "req-1",
				Device: test.device,
				Imp:    []openrtb2.Imp{{ID: "imp-1", Banner: &openrtb2.Banner{}, Ext: []byte(testImpExt)}},
			}, &adapters.ExtraRequestInfo{})

			assert.Empty(t, errs)
			assert.Len(t, requests, 1)

			// The static headers are always present regardless of device.
			assert.Equal(t, "application/json;charset=utf-8", requests[0].Headers.Get("Content-Type"))
			assert.Equal(t, "2.6", requests[0].Headers.Get("X-OpenRTB-Version"))

			for _, header := range []string{"User-Agent", "X-Forwarded-For"} {
				assert.Equal(t, test.expected[header], requests[0].Headers.Values(header), header)
			}
		})
	}
}

// Audio is not a declared capability, so an audio bid can never match what was
// requested. It must be rejected rather than relabelled.
func TestMakeBidsRejectsAudioBid(t *testing.T) {
	bidder := buildTestBidder(t)

	response, errs := bidder.MakeBids(
		&openrtb2.BidRequest{ID: "req-1", Imp: []openrtb2.Imp{{ID: "imp-1", Banner: &openrtb2.Banner{}}}},
		&adapters.RequestData{Uri: testURI},
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

// The region lives in the host configured endpoint. A host serving another
// datacenter overrides it, and the sid its publishers use must follow.
func TestHostRegionOverride(t *testing.T) {
	testCases := []struct {
		name     string
		endpoint string
		sid      string
		expected string
	}{
		{
			name:     "us is the open source default",
			endpoint: testEndpoint,
			sid:      "tunnl_x_use_g",
			expected: "https://us1.rapidtag.net/api/v1/bid?sid=tunnl_x_use_g",
		},
		{
			name:     "eu",
			endpoint: "https://eu1.rapidtag.net/api/v1/bid?sid={{.SourceId}}",
			sid:      "tunnl_x_eu_g",
			expected: "https://eu1.rapidtag.net/api/v1/bid?sid=tunnl_x_eu_g",
		},
		{
			name:     "ap",
			endpoint: "https://ap1.rapidtag.net/api/v1/bid?sid={{.SourceId}}",
			sid:      "tunnl_x_ap_g",
			expected: "https://ap1.rapidtag.net/api/v1/bid?sid=tunnl_x_ap_g",
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
				ID: "req-1",
				Imp: []openrtb2.Imp{{
					ID:    "imp-1",
					Video: &openrtb2.Video{},
					Ext:   []byte(`{"bidder":{"sid":"` + test.sid + `"}}`),
				}},
			}, &adapters.ExtraRequestInfo{})

			assert.Empty(t, errs)
			assert.Len(t, requests, 1)
			assert.Equal(t, test.expected, requests[0].Uri)
		})
	}
}

// A sid is opaque apart from its region, so anything a URL would otherwise
// reinterpret has to survive into the endpoint intact.
func TestMakeRequestsEscapesSid(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderTunnl,
		config.Adapter{Endpoint: "https://proxy.internal/tunnl?sid={{.SourceId}}"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	requests, errs := bidder.MakeRequests(&openrtb2.BidRequest{
		ID: "req-1",
		Imp: []openrtb2.Imp{{
			ID:     "imp-1",
			Banner: &openrtb2.Banner{},
			Ext:    []byte(`{"bidder":{"sid":"tunnl_x&evil=1_use_g"}}`),
		}},
	}, &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs)
	assert.Len(t, requests, 1)
	assert.Equal(t, "https://proxy.internal/tunnl?sid=tunnl_x%26evil%3D1_use_g", requests[0].Uri)
}
