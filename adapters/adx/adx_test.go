package adx

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderAdx,
		config.Adapter{
			Endpoint: "http://adx.example.com/ortb/{{.SupplyId}}",
		},
		config.Server{
			ExternalUrl: "http://hosturl.com",
			GvlID:       0,
			DataCenter:  "2",
		},
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "adxtest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	bidder, err := Builder(
		openrtb_ext.BidderAdx,
		config.Adapter{Endpoint: "http://adx.example.com/ortb/{{.SupplyId"},
		config.Server{},
	)

	assert.Nil(t, bidder)
	assert.ErrorContains(t, err, "unable to parse endpoint url template")
}

// validRequest is a request that MakeRequests accepts, for tests that need to
// break exactly one thing in it.
func validRequest() *openrtb2.BidRequest {
	return &openrtb2.BidRequest{
		ID:  "test-req",
		App: &openrtb2.App{Bundle: "com.example.game"},
		Imp: []openrtb2.Imp{{
			ID:     "imp-1",
			Banner: &openrtb2.Banner{},
			Ext:    json.RawMessage(`{"bidder":{"supply_id":"testkey"}}`),
		}},
	}
}

// An endpoint template that parses but cannot be executed against the macro
// params only fails when a request is built.
func TestMakeRequestsEndpointTemplateExecutionError(t *testing.T) {
	bidder, err := Builder(
		openrtb_ext.BidderAdx,
		config.Adapter{Endpoint: "http://adx.example.com/ortb/{{.NoSuchMacro}}"},
		config.Server{},
	)
	assert.NoError(t, err)

	requests, errs := bidder.MakeRequests(validRequest(), &adapters.ExtraRequestInfo{})

	assert.Empty(t, requests)
	if assert.Len(t, errs, 2) {
		assert.ErrorContains(t, errs[0], "NoSuchMacro")
		assert.Equal(t, &errortypes.BadInput{Message: "adx: no valid impressions in request"}, errs[1])
	}
}
