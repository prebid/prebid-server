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

func singleImpRequest() *openrtb2.BidRequest {
	return &openrtb2.BidRequest{
		ID:   "test-request-id",
		Imp:  []openrtb2.Imp{bannerImp("imp-1", "test-placement-1")},
		Site: &openrtb2.Site{Page: "https://example.com/article"},
	}
}

func TestMakeBidsMapsBannerBid(t *testing.T) {
	bidder := buildTestAdapter(t)
	body := []byte(`{
		"id": "test-request-id",
		"cur": "USD",
		"seatbid": [{
			"seat": "jjtech",
			"bid": [{
				"id": "jjt-bid-1",
				"impid": "imp-1",
				"price": 1.5,
				"adm": "<div>jjt</div>",
				"crid": "jjt-creative-1",
				"adomain": ["jambojar-tech.com"],
				"w": 300,
				"h": 250,
				"mtype": 1
			}]
		}]
	}`)

	response, errs := bidder.MakeBids(singleImpRequest(), &adapters.RequestData{}, &adapters.ResponseData{StatusCode: http.StatusOK, Body: body})

	assert.Empty(t, errs)
	require.NotNil(t, response)
	require.Len(t, response.Bids, 1)
	assert.Equal(t, "USD", response.Currency)
	assert.Equal(t, openrtb_ext.BidTypeBanner, response.Bids[0].BidType)
	assert.Equal(t, "imp-1", response.Bids[0].Bid.ImpID)
	assert.Equal(t, 1.5, response.Bids[0].Bid.Price)
	assert.Equal(t, "jjt-creative-1", response.Bids[0].Bid.CrID)
}

func TestMakeBidsNoContentReturnsNoBids(t *testing.T) {
	bidder := buildTestAdapter(t)

	response, errs := bidder.MakeBids(singleImpRequest(), &adapters.RequestData{}, &adapters.ResponseData{StatusCode: http.StatusNoContent})

	assert.Nil(t, response)
	assert.Nil(t, errs)
}

func TestMakeBidsServerErrorReturnsError(t *testing.T) {
	bidder := buildTestAdapter(t)

	response, errs := bidder.MakeBids(singleImpRequest(), &adapters.RequestData{}, &adapters.ResponseData{StatusCode: http.StatusInternalServerError, Body: []byte(`{}`)})

	assert.Nil(t, response)
	require.Len(t, errs, 1)
}

func TestMakeBidsUnparseableBodyReturnsError(t *testing.T) {
	bidder := buildTestAdapter(t)

	response, errs := bidder.MakeBids(singleImpRequest(), &adapters.RequestData{}, &adapters.ResponseData{StatusCode: http.StatusOK, Body: []byte(`not json`)})

	assert.Nil(t, response)
	require.Len(t, errs, 1)
}

func TestMakeBidsRejectsUnsupportedMType(t *testing.T) {
	bidder := buildTestAdapter(t)
	body := []byte(`{
		"id": "test-request-id",
		"cur": "USD",
		"seatbid": [{
			"seat": "jjtech",
			"bid": [{"id": "jjt-bid-video", "impid": "imp-1", "price": 1.5, "mtype": 2}]
		}]
	}`)

	response, errs := bidder.MakeBids(singleImpRequest(), &adapters.RequestData{}, &adapters.ResponseData{StatusCode: http.StatusOK, Body: body})

	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "unsupported mtype")
	require.NotNil(t, response)
	assert.Empty(t, response.Bids, "a non-banner bid must be dropped")
}

func TestMakeBidsDefaultsCurrencyWhenAbsent(t *testing.T) {
	bidder := buildTestAdapter(t)
	body := []byte(`{
		"id": "test-request-id",
		"seatbid": [{
			"seat": "jjtech",
			"bid": [{"id": "jjt-bid-1", "impid": "imp-1", "price": 1.5, "adm": "<div>jjt</div>", "mtype": 1}]
		}]
	}`)

	response, errs := bidder.MakeBids(singleImpRequest(), &adapters.RequestData{}, &adapters.ResponseData{StatusCode: http.StatusOK, Body: body})

	assert.Empty(t, errs)
	require.NotNil(t, response)
	assert.Equal(t, "USD", response.Currency, "PBS defaults to USD when the response omits cur")
}

func TestMakeBidsCopiesNonUSDCurrency(t *testing.T) {
	bidder := buildTestAdapter(t)
	body := []byte(`{
		"id": "test-request-id",
		"cur": "EUR",
		"seatbid": [{
			"seat": "jjtech",
			"bid": [{"id": "jjt-bid-1", "impid": "imp-1", "price": 1.5, "adm": "<div>jjt</div>", "mtype": 1}]
		}]
	}`)

	response, errs := bidder.MakeBids(singleImpRequest(), &adapters.RequestData{}, &adapters.ResponseData{StatusCode: http.StatusOK, Body: body})

	assert.Empty(t, errs)
	require.NotNil(t, response)
	assert.Equal(t, "EUR", response.Currency, "a non-USD cur in the response must be copied, not dropped")
}

func TestMakeBidsRejectedBidDoesNotAffectOthers(t *testing.T) {
	bidder := buildTestAdapter(t)
	body := []byte(`{
		"id": "test-request-id",
		"cur": "USD",
		"seatbid": [{
			"seat": "jjtech",
			"bid": [
				{"id": "jjt-bid-banner", "impid": "imp-1", "price": 1.5, "adm": "<div>jjt</div>", "mtype": 1},
				{"id": "jjt-bid-video", "impid": "imp-1", "price": 2.0, "mtype": 2}
			]
		}]
	}`)

	response, errs := bidder.MakeBids(singleImpRequest(), &adapters.RequestData{}, &adapters.ResponseData{StatusCode: http.StatusOK, Body: body})

	require.Len(t, errs, 1)
	require.NotNil(t, response)
	require.Len(t, response.Bids, 1, "the valid banner bid must still be returned despite the rejected video bid")
	assert.Equal(t, "jjt-bid-banner", response.Bids[0].Bid.ID)
	assert.Equal(t, openrtb_ext.BidTypeBanner, response.Bids[0].BidType)
}
