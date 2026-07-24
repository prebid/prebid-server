package hypelab

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
	"github.com/prebid/prebid-server/v4/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderHypeLab, config.Adapter{
		Endpoint: "https://api.hypelab.com/v1/rtb_requests",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "hypelabtest", bidder)
}

func TestMakeRequestsNoValidImps(t *testing.T) {
	bidder := &adapter{endpoint: "http://example.com/openrtb2"}

	requests, errs := bidder.MakeRequests(&openrtb2.BidRequest{
		ID: "request",
		Imp: []openrtb2.Imp{
			{
				ID:     "invalid-ext",
				Banner: &openrtb2.Banner{},
				Ext:    json.RawMessage(`"bad"`),
			},
			{
				ID:     "invalid-bidder-ext",
				Banner: &openrtb2.Banner{},
				Ext:    json.RawMessage(`{"bidder":"bad"}`),
			},
			{
				ID:     "missing-placement",
				Banner: &openrtb2.Banner{},
				Ext:    json.RawMessage(`{"bidder":{"property_slug":"property"}}`),
			},
		},
	}, &adapters.ExtraRequestInfo{})

	assert.Nil(t, requests)
	require.Len(t, errs, 3)
	assert.EqualError(t, errs[0], "imp invalid-ext: unable to unmarshal ext")
	assert.EqualError(t, errs[1], "imp invalid-bidder-ext: unable to unmarshal ext.bidder")
	assert.EqualError(t, errs[2], "imp missing-placement: property_slug and placement_slug are required")
}

func TestMakeRequestsSkipsInvalidImps(t *testing.T) {
	bidder := &adapter{endpoint: "http://example.com/openrtb2"}

	requests, errs := bidder.MakeRequests(&openrtb2.BidRequest{
		ID:  "request",
		Ext: json.RawMessage(`{"existing":true}`),
		Imp: []openrtb2.Imp{
			{
				ID:     "missing-property",
				Banner: &openrtb2.Banner{},
				Ext:    json.RawMessage(`{"bidder":{"placement_slug":"placement"}}`),
			},
			{
				ID:     "valid",
				Banner: &openrtb2.Banner{},
				Ext: json.RawMessage(`{"bidder":{
					"property_slug":"property",
					"placement_slug":"placement"
				}}`),
			},
		},
	}, &adapters.ExtraRequestInfo{})

	require.Len(t, errs, 1)
	require.Len(t, requests, 1)
	assert.ElementsMatch(t, []string{"valid"}, requests[0].ImpIDs)

	var outgoingRequest openrtb2.BidRequest
	require.NoError(t, jsonutil.Unmarshal(requests[0].Body, &outgoingRequest))
	require.Len(t, outgoingRequest.Imp, 1)
	assert.Equal(t, "placement", outgoingRequest.Imp[0].TagID)
	assert.Equal(t, displayManager, outgoingRequest.Imp[0].DisplayManager)

	var requestExt map[string]json.RawMessage
	require.NoError(t, jsonutil.Unmarshal(outgoingRequest.Ext, &requestExt))
	assert.JSONEq(t, `true`, string(requestExt["existing"]))
	assert.JSONEq(t, `"prebid-server"`, string(requestExt["source"]))
	assert.JSONEq(t, `"prebid-server@unknown"`, string(requestExt["provider_version"]))

	var impExt adapters.ExtImpBidder
	require.NoError(t, jsonutil.Unmarshal(outgoingRequest.Imp[0].Ext, &impExt))

	var params openrtb_ext.ExtImpHypeLab
	require.NoError(t, jsonutil.Unmarshal(impExt.Bidder, &params))
	assert.Equal(t, openrtb_ext.ExtImpHypeLab{
		PropertySlug:  "property",
		PlacementSlug: "placement",
	}, params)
}

func TestMakeBidsInvalidResponse(t *testing.T) {
	bidder := &adapter{}

	bidderResponse, errs := bidder.MakeBids(&openrtb2.BidRequest{}, nil, &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`not-json`),
	})

	assert.Nil(t, bidderResponse)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "expect")
}

func TestGetBidMediaType(t *testing.T) {
	impLookup := map[string]openrtb2.Imp{
		"banner": {ID: "banner", Banner: &openrtb2.Banner{}},
		"video":  {ID: "video", Video: &openrtb2.Video{}},
		"native": {ID: "native", Native: &openrtb2.Native{}},
		"multi":  {ID: "multi", Banner: &openrtb2.Banner{}, Video: &openrtb2.Video{}},
		"empty":  {ID: "empty"},
	}

	testCases := []struct {
		name          string
		bid           openrtb2.Bid
		expected      openrtb_ext.BidType
		expectedError string
	}{
		{
			name:     "mtype banner",
			bid:      openrtb2.Bid{ID: "bid", ImpID: "banner", MType: openrtb2.MarkupBanner},
			expected: openrtb_ext.BidTypeBanner,
		},
		{
			name:     "mtype video",
			bid:      openrtb2.Bid{ID: "bid", ImpID: "video", MType: openrtb2.MarkupVideo},
			expected: openrtb_ext.BidTypeVideo,
		},
		{
			name:     "mtype native",
			bid:      openrtb2.Bid{ID: "bid", ImpID: "native", MType: openrtb2.MarkupNative},
			expected: openrtb_ext.BidTypeNative,
		},
		{
			name:          "unsupported mtype",
			bid:           openrtb2.Bid{ID: "bid", ImpID: "banner", MType: openrtb2.MarkupAudio},
			expectedError: "bid bid uses unsupported mtype 3",
		},
		{
			name:     "display creative type",
			bid:      openrtb2.Bid{ID: "bid", ImpID: "banner", Ext: json.RawMessage(`{"hypelab":{"creative_type":"display"}}`)},
			expected: openrtb_ext.BidTypeBanner,
		},
		{
			name:     "video creative type",
			bid:      openrtb2.Bid{ID: "bid", ImpID: "video", Ext: json.RawMessage(`{"hypelab":{"creative_type":"video"}}`)},
			expected: openrtb_ext.BidTypeVideo,
		},
		{
			name:     "native creative type",
			bid:      openrtb2.Bid{ID: "bid", ImpID: "native", Ext: json.RawMessage(`{"hypelab":{"creative_type":"native"}}`)},
			expected: openrtb_ext.BidTypeNative,
		},
		{
			name:          "unsupported creative type",
			bid:           openrtb2.Bid{ID: "bid", ImpID: "banner", Ext: json.RawMessage(`{"hypelab":{"creative_type":"audio"}}`)},
			expectedError: "bid bid has unsupported creative_type audio",
		},
		{
			name:          "invalid ext",
			bid:           openrtb2.Bid{ID: "bid", ImpID: "banner", Ext: json.RawMessage(`"bad"`)},
			expectedError: "bid bid has invalid ext",
		},
		{
			name:     "vast markup",
			bid:      openrtb2.Bid{ID: "bid", ImpID: "video", AdM: "  <VAST version=\"4.3\"></VAST>"},
			expected: openrtb_ext.BidTypeVideo,
		},
		{
			name:     "single media type fallback",
			bid:      openrtb2.Bid{ID: "bid", ImpID: "banner"},
			expected: openrtb_ext.BidTypeBanner,
		},
		{
			name:     "ext without hypelab falls back to imp",
			bid:      openrtb2.Bid{ID: "bid", ImpID: "banner", Ext: json.RawMessage(`{"other":{}}`)},
			expected: openrtb_ext.BidTypeBanner,
		},
		{
			name:          "ambiguous multiformat fallback",
			bid:           openrtb2.Bid{ID: "bid", ImpID: "multi"},
			expectedError: "unable to determine media type for bid bid on imp multi",
		},
		{
			name:          "unknown imp",
			bid:           openrtb2.Bid{ID: "bid", ImpID: "unknown", MType: openrtb2.MarkupBanner},
			expectedError: "bid bid references unknown imp unknown",
		},
		{
			name:          "no media type fallback",
			bid:           openrtb2.Bid{ID: "bid", ImpID: "empty"},
			expectedError: "unable to determine media type for bid bid on imp empty",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := getBidMediaType(&test.bid, impLookup)

			if test.expectedError != "" {
				require.Error(t, err)
				assert.EqualError(t, err, test.expectedError)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestPrebidServerVersion(t *testing.T) {
	originalVersion := version.Ver
	t.Cleanup(func() {
		version.Ver = originalVersion
	})

	version.Ver = "test-version"
	assert.Equal(t, "test-version", prebidServerVersion())

	version.Ver = ""
	assert.Equal(t, version.VerUnknown, prebidServerVersion())
}
