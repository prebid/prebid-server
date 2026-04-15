package matterfull

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeRequestsEmptyImps(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderMatterfull, config.Adapter{
		Endpoint: "https://prebid.matterfull.co/?uqhash={{.PublisherID}}",
	}, config.Server{})
	require.NoError(t, err)

	req := &openrtb2.BidRequest{ID: "req", Imp: nil}
	gotReqs, gotErrs := bidder.MakeRequests(req, &adapters.ExtraRequestInfo{})
	assert.Nil(t, gotReqs, "expected no requests when imp slice is empty")
	assert.Empty(t, gotErrs)
}

func TestMakeRequestsAllImpsInvalid(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderMatterfull, config.Adapter{
		Endpoint: "https://prebid.matterfull.co/?uqhash={{.PublisherID}}",
	}, config.Server{})
	require.NoError(t, err)

	tests := []struct {
		name string
		imp  openrtb2.Imp
	}{
		{"invalid ext: malformed JSON", openrtb2.Imp{
			ID: "1", Banner: &openrtb2.Banner{W: ptr(300), H: ptr(250)},
			Ext: []byte(`{`),
		}},
		{"invalid ext: bidder not matterfull object", openrtb2.Imp{
			ID: "1", Banner: &openrtb2.Banner{W: ptr(300), H: ptr(250)},
			Ext: []byte(`{"bidder": 123}`),
		}},
		{"empty publisher id", openrtb2.Imp{
			ID: "1", Banner: &openrtb2.Banner{W: ptr(300), H: ptr(250)},
			Ext: []byte(`{"bidder": {"pid": ""}}`),
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &openrtb2.BidRequest{ID: "req", Imp: []openrtb2.Imp{tt.imp}}
			gotReqs, gotErrs := bidder.MakeRequests(req, &adapters.ExtraRequestInfo{})
			assert.Nil(t, gotReqs, "expected no requests when all imps invalid")
			assert.NotEmpty(t, gotErrs, "expected at least one error")
		})
	}
}

func TestMakeBidsInvalidJSON(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderMatterfull, config.Adapter{
		Endpoint: "https://prebid.matterfull.co/?uqhash={{.PublisherID}}",
	}, config.Server{})
	require.NoError(t, err)

	internalReq := &openrtb2.BidRequest{ID: "req", Imp: []openrtb2.Imp{{ID: "imp1"}}}
	externalReq := &adapters.RequestData{Method: "POST", Uri: "https://prebid.matterfull.co/?uqhash=pub", Body: nil, ImpIDs: []string{"imp1"}}
	response := &adapters.ResponseData{StatusCode: 200, Body: []byte("not valid json")}

	gotResp, gotErrs := bidder.MakeBids(internalReq, externalReq, response)
	assert.Nil(t, gotResp)
	require.Len(t, gotErrs, 1)
	assert.Contains(t, gotErrs[0].Error(), "Bad server response")
}

func TestMakeRequestsBuildEndpointError(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderMatterfull, config.Adapter{
		Endpoint: "https://prebid.matterfull.co/?uqhash={{slice .PublisherID 100}}",
	}, config.Server{})
	require.NoError(t, err)

	req := &openrtb2.BidRequest{
		ID: "req",
		Imp: []openrtb2.Imp{{
			ID:     "imp1",
			Banner: &openrtb2.Banner{W: ptr(300), H: ptr(250)},
			Ext:    bidderExt("pub"),
		}},
	}

	gotReqs, gotErrs := bidder.MakeRequests(req, &adapters.ExtraRequestInfo{})
	assert.Empty(t, gotReqs)
	require.Len(t, gotErrs, 1)
	assert.Contains(t, gotErrs[0].Error(), "error calling slice")
}

func TestMakeRequestsMarshalError(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderMatterfull, config.Adapter{
		Endpoint: "https://prebid.matterfull.co/?uqhash={{.PublisherID}}",
	}, config.Server{})
	require.NoError(t, err)

	req := &openrtb2.BidRequest{
		ID: "req",
		Imp: []openrtb2.Imp{{
			ID:       "imp1",
			Banner:   &openrtb2.Banner{W: ptr(300), H: ptr(250)},
			BidFloor: math.NaN(),
			Ext:      bidderExt("pub"),
		}},
	}

	gotReqs, gotErrs := bidder.MakeRequests(req, &adapters.ExtraRequestInfo{})
	assert.Empty(t, gotReqs)
	require.Len(t, gotErrs, 1)
	assert.Contains(t, gotErrs[0].Error(), "unsupported value: NaN")
}

func TestCompatBannerImpression(t *testing.T) {
	t.Run("missing size and format", func(t *testing.T) {
		imp := &openrtb2.Imp{Banner: &openrtb2.Banner{}}

		err := compatBannerImpression(imp)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Expected at least one banner.format entry or explicit w/h")
	})

	t.Run("single format", func(t *testing.T) {
		imp := &openrtb2.Imp{Banner: &openrtb2.Banner{
			Format: []openrtb2.Format{{W: 300, H: 250}},
		}}

		err := compatBannerImpression(imp)

		require.NoError(t, err)
		require.NotNil(t, imp.Banner.W)
		require.NotNil(t, imp.Banner.H)
		assert.Equal(t, int64(300), *imp.Banner.W)
		assert.Equal(t, int64(250), *imp.Banner.H)
		assert.Nil(t, imp.Banner.Format)
	})

	t.Run("multiple formats deep copy", func(t *testing.T) {
		originalFormats := []openrtb2.Format{{W: 300, H: 250}, {W: 728, H: 90}}
		imp := &openrtb2.Imp{Banner: &openrtb2.Banner{
			Format: originalFormats,
		}}

		err := compatBannerImpression(imp)

		require.NoError(t, err)
		require.Len(t, imp.Banner.Format, 1)
		assert.Equal(t, openrtb2.Format{W: 728, H: 90}, imp.Banner.Format[0])
		originalFormats[1] = openrtb2.Format{W: 160, H: 600}
		assert.Equal(t, openrtb2.Format{W: 728, H: 90}, imp.Banner.Format[0])
	})
}

func TestCompatImpressionWithoutBanner(t *testing.T) {
	imp := &openrtb2.Imp{
		Ext: []byte(`{"bidder":{"pid":"pub"}}`),
	}

	err := compatImpression(imp)

	require.NoError(t, err)
	assert.Nil(t, imp.Ext)
}

func TestGetMediaTypeForImpID(t *testing.T) {
	imps := []openrtb2.Imp{
		{ID: "banner-imp", Banner: &openrtb2.Banner{}},
		{ID: "video-imp", Video: &openrtb2.Video{}},
	}

	assert.Equal(t, openrtb_ext.BidTypeBanner, getMediaTypeForImpID("banner-imp", imps))
	assert.Equal(t, openrtb_ext.BidTypeVideo, getMediaTypeForImpID("video-imp", imps))
	assert.Equal(t, openrtb_ext.BidTypeBanner, getMediaTypeForImpID("missing", imps))
}

func ptr(i int64) *int64 { return &i }

func bidderExt(publisherID string) []byte {
	ext, err := json.Marshal(map[string]any{
		"bidder": map[string]any{
			"pid": publisherID,
		},
	})
	if err != nil {
		panic(err)
	}
	return ext
}

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderMatterfull, config.Adapter{
		Endpoint: "https://prebid.matterfull.co/?uqhash={{.PublisherID}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	require.NoError(t, buildErr, "Builder returned unexpected error")

	adapterstest.RunJSONBidderTest(t, "matterfulltest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderMatterfull, config.Adapter{
		Endpoint: "{{Malformed}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr, "Expected error due to malformed endpoint template")
}
