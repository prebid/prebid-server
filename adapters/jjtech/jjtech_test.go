package jjtech

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

const testEndpoint = "https://ind-apac.jambojar.com/jjt-prebid/rtb-apac"

func buildTestAdapter(t *testing.T) adapters.Bidder {
	t.Helper()
	bidder, err := Builder(openrtb_ext.BidderJJTech, config.Adapter{Endpoint: testEndpoint}, config.Server{})
	require.NoError(t, err)
	return bidder
}

func bannerImp(id, placementID string) openrtb2.Imp {
	return openrtb2.Imp{
		ID:     id,
		Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
		Ext:    json.RawMessage(`{"bidder":{"placementId":"` + placementID + `"}}`),
	}
}

func TestBuilder(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderJJTech, config.Adapter{Endpoint: testEndpoint}, config.Server{})
	assert.NoError(t, err)
	assert.NotNil(t, bidder)
}

// Kept as a Go test because the JSON harness only inspects the outgoing request,
// so it cannot assert that the caller's own impression was left untouched.
func TestMakeRequestsDoesNotMutateIncomingRequest(t *testing.T) {
	bidder := buildTestAdapter(t)
	request := &openrtb2.BidRequest{
		ID:   "test-request-id",
		Imp:  []openrtb2.Imp{bannerImp("imp-1", "test-placement-1")},
		Site: &openrtb2.Site{Page: "https://example.com/article"},
	}

	_, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	require.Empty(t, errs)
	assert.JSONEq(t, `{"bidder":{"placementId":"test-placement-1"}}`, string(request.Imp[0].Ext),
		"the caller's request must be left untouched")
}

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderJJTech, config.Adapter{Endpoint: testEndpoint}, config.Server{})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "jjtechtest", bidder)
}
