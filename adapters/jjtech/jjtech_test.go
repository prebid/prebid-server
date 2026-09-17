package jjtech

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
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

func TestMakeRequestsBatchesImpsIntoOneCall(t *testing.T) {
	bidder := buildTestAdapter(t)
	request := &openrtb2.BidRequest{
		ID:   "test-request-id",
		Imp:  []openrtb2.Imp{bannerImp("imp-1", "test-placement-1"), bannerImp("imp-2", "test-placement-2")},
		Site: &openrtb2.Site{Page: "https://example.com/article"},
	}

	requests, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs)
	require.Len(t, requests, 1, "all imps must be batched into a single call")
	assert.Equal(t, http.MethodPost, requests[0].Method)
	assert.Equal(t, testEndpoint, requests[0].Uri)
	assert.Equal(t, []string{"imp-1", "imp-2"}, requests[0].ImpIDs)
	assert.Equal(t, "application/json;charset=utf-8", requests[0].Headers.Get("Content-Type"))
}

func TestMakeRequestsRewritesImpExtToJJTech(t *testing.T) {
	bidder := buildTestAdapter(t)
	request := &openrtb2.BidRequest{
		ID:   "test-request-id",
		Imp:  []openrtb2.Imp{bannerImp("imp-1", "test-placement-1"), bannerImp("imp-2", "test-placement-2")},
		Site: &openrtb2.Site{Page: "https://example.com/article"},
	}

	requests, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})
	require.Empty(t, errs)
	require.Len(t, requests, 1)

	var sent openrtb2.BidRequest
	require.NoError(t, jsonutil.Unmarshal(requests[0].Body, &sent))
	require.Len(t, sent.Imp, 2)
	assert.JSONEq(t, `{"jjtech":{"placementId":"test-placement-1"}}`, string(sent.Imp[0].Ext))
	assert.JSONEq(t, `{"jjtech":{"placementId":"test-placement-2"}}`, string(sent.Imp[1].Ext))
}

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

func TestMakeRequestsRejectsMissingPlacementID(t *testing.T) {
	bidder := buildTestAdapter(t)
	bad := openrtb2.Imp{
		ID:     "imp-bad",
		Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
		Ext:    json.RawMessage(`{"bidder":{}}`),
	}
	request := &openrtb2.BidRequest{
		ID:   "test-request-id",
		Imp:  []openrtb2.Imp{bad, bannerImp("imp-good", "test-placement-1")},
		Site: &openrtb2.Site{Page: "https://example.com/article"},
	}

	requests, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "imp-bad")
	require.Len(t, requests, 1, "the valid imp must still be sent")
	assert.Equal(t, []string{"imp-good"}, requests[0].ImpIDs)
}

func TestMakeRequestsNoValidImpsReturnsNoRequest(t *testing.T) {
	bidder := buildTestAdapter(t)
	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID:     "imp-bad",
			Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
			Ext:    json.RawMessage(`{"bidder":{"placementId":""}}`),
		}},
		Site: &openrtb2.Site{Page: "https://example.com/article"},
	}

	requests, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	assert.Empty(t, requests)
	require.Len(t, errs, 1)
}
