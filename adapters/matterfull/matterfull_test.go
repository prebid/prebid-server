package matterfull

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
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

func ptr(i int64) *int64 { return &i }

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
