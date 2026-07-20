package ferio

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderFerio, config.Adapter{
		Endpoint: "https://ferio.bid/bidder",
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "feriotest", bidder)
}

func TestMakeRequestsSiteMultiImp(t *testing.T) {
	bidder := &adapter{endpoint: "https://ferio.bid/bidder"}
	req := &openrtb2.BidRequest{
		ID: "auction-id",
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{ID: "original-publisher"},
		},
		Imp: []openrtb2.Imp{
			{
				ID:     "imp-1",
				Banner: &openrtb2.Banner{},
				Ext:    json.RawMessage(`{"tid":"transaction-1","prebid":{"storedrequest":{"id":"old-stored-request"},"bidder":{"ferio":{"publisherId":"pub-1","adUnitId":"legacy-plc","tenantId":"legacy-tenant"},"otherBidder":{"foo":"bar"}}},"bidder":{"publisherId":"pub-1","adUnitId":"plc-1","tenantId":"tenant-1"}}`),
			},
			{
				ID:     "imp-2",
				Banner: &openrtb2.Banner{},
				Ext:    json.RawMessage(`{"bidder":{"publisherId":"pub-1","adUnitId":"plc-2","tenantId":"tenant-1"}}`),
			},
		},
	}

	requests, errs := bidder.MakeRequests(req, nil)
	require.Empty(t, errs)
	require.Len(t, requests, 1)

	assert.Equal(t, http.MethodPost, requests[0].Method)
	assert.Equal(t, "https://ferio.bid/bidder", requests[0].Uri)
	assert.Equal(t, []string{"imp-1", "imp-2"}, requests[0].ImpIDs)
	assert.Equal(t, "application/json;charset=utf-8", requests[0].Headers.Get("Content-Type"))
	assert.Equal(t, "application/json", requests[0].Headers.Get("Accept"))

	var outgoing openrtb2.BidRequest
	require.NoError(t, json.Unmarshal(requests[0].Body, &outgoing))
	require.NotNil(t, outgoing.Site)
	require.NotNil(t, outgoing.Site.Publisher)
	assert.Equal(t, "pub-1", outgoing.Site.Publisher.ID)

	require.Len(t, outgoing.Imp, 2)
	assert.Equal(t, "plc-1", outgoing.Imp[0].TagID)
	assert.Equal(t, "plc-2", outgoing.Imp[1].TagID)
	assert.JSONEq(t, `{"tid":"transaction-1","prebid":{"storedrequest":{"id":"old-stored-request"}},"bidder":{"publisherId":"pub-1","adUnitId":"plc-1","tenantId":"tenant-1"}}`, string(outgoing.Imp[0].Ext))
	assert.JSONEq(t, `{"bidder":{"publisherId":"pub-1","adUnitId":"plc-2","tenantId":"tenant-1"}}`, string(outgoing.Imp[1].Ext))

	assert.Equal(t, "original-publisher", req.Site.Publisher.ID)
	assert.Empty(t, req.Imp[0].TagID)
	assert.JSONEq(t, `{"tid":"transaction-1","prebid":{"storedrequest":{"id":"old-stored-request"},"bidder":{"ferio":{"publisherId":"pub-1","adUnitId":"legacy-plc","tenantId":"legacy-tenant"},"otherBidder":{"foo":"bar"}}},"bidder":{"publisherId":"pub-1","adUnitId":"plc-1","tenantId":"tenant-1"}}`, string(req.Imp[0].Ext))
}

func TestMakeRequestsAppPublisher(t *testing.T) {
	bidder := &adapter{endpoint: "https://ferio.bid/bidder"}
	req := &openrtb2.BidRequest{
		ID:  "auction-id",
		App: &openrtb2.App{},
		Imp: []openrtb2.Imp{{
			ID:    "imp-1",
			Video: &openrtb2.Video{},
			Ext:   json.RawMessage(`{"bidder":{"publisherId":"app-pub","adUnitId":"app-plc","tenantId":"tenant-1"}}`),
		}},
	}

	requests, errs := bidder.MakeRequests(req, nil)
	require.Empty(t, errs)
	require.Len(t, requests, 1)
	assert.Equal(t, "https://ferio.bid/bidder", requests[0].Uri)

	var outgoing openrtb2.BidRequest
	require.NoError(t, json.Unmarshal(requests[0].Body, &outgoing))
	require.NotNil(t, outgoing.App)
	require.NotNil(t, outgoing.App.Publisher)
	assert.Equal(t, "app-pub", outgoing.App.Publisher.ID)
	assert.Equal(t, "app-plc", outgoing.Imp[0].TagID)
	assert.JSONEq(t, `{"bidder":{"publisherId":"app-pub","adUnitId":"app-plc","tenantId":"tenant-1"}}`, string(outgoing.Imp[0].Ext))
}

func TestMakeRequestsRejectsMixedPublisherIDs(t *testing.T) {
	bidder := &adapter{endpoint: "https://ferio.bid/bidder"}
	req := &openrtb2.BidRequest{
		ID:   "auction-id",
		Site: &openrtb2.Site{},
		Imp: []openrtb2.Imp{
			{
				ID:     "imp-1",
				Banner: &openrtb2.Banner{},
				Ext:    json.RawMessage(`{"bidder":{"publisherId":"pub-1","adUnitId":"plc-1","tenantId":"tenant-1"}}`),
			},
			{
				ID:     "imp-2",
				Banner: &openrtb2.Banner{},
				Ext:    json.RawMessage(`{"bidder":{"publisherId":"pub-2","adUnitId":"plc-2","tenantId":"tenant-1"}}`),
			},
		},
	}

	requests, errs := bidder.MakeRequests(req, nil)
	assert.Nil(t, requests)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "publisherId")
}

func TestMakeRequestsAllowsMixedTenantIDs(t *testing.T) {
	bidder := &adapter{endpoint: "https://ferio.bid/bidder"}
	req := &openrtb2.BidRequest{
		ID:   "auction-id",
		Site: &openrtb2.Site{},
		Imp: []openrtb2.Imp{
			{
				ID:     "imp-1",
				Banner: &openrtb2.Banner{},
				Ext:    json.RawMessage(`{"bidder":{"publisherId":"pub-1","adUnitId":"plc-1","tenantId":"tenant-1"}}`),
			},
			{
				ID:     "imp-2",
				Banner: &openrtb2.Banner{},
				Ext:    json.RawMessage(`{"bidder":{"publisherId":"pub-1","adUnitId":"plc-2","tenantId":"tenant-2"}}`),
			},
		},
	}

	requests, errs := bidder.MakeRequests(req, nil)
	require.Empty(t, errs)
	require.Len(t, requests, 1)

	var outgoing openrtb2.BidRequest
	require.NoError(t, json.Unmarshal(requests[0].Body, &outgoing))
	require.Len(t, outgoing.Imp, 2)
	assert.JSONEq(t, `{"bidder":{"publisherId":"pub-1","adUnitId":"plc-1","tenantId":"tenant-1"}}`, string(outgoing.Imp[0].Ext))
	assert.JSONEq(t, `{"bidder":{"publisherId":"pub-1","adUnitId":"plc-2","tenantId":"tenant-2"}}`, string(outgoing.Imp[1].Ext))
}

func TestMakeRequestsPreservesConfiguredEndpointQueryParams(t *testing.T) {
	bidder := &adapter{endpoint: "https://ferio.bid/bidder?foo=bar"}
	req := &openrtb2.BidRequest{
		ID:   "auction-id",
		Site: &openrtb2.Site{},
		Imp: []openrtb2.Imp{{
			ID:     "imp-1",
			Banner: &openrtb2.Banner{},
			Ext:    json.RawMessage(`{"bidder":{"publisherId":"pub-1","adUnitId":"plc-1","tenantId":"tenant-1"}}`),
		}},
	}

	requests, errs := bidder.MakeRequests(req, nil)
	require.Empty(t, errs)
	require.Len(t, requests, 1)
	assert.Equal(t, "https://ferio.bid/bidder?foo=bar", requests[0].Uri)
}

func TestMakeRequestsInvalidImpExt(t *testing.T) {
	testCases := []struct {
		name        string
		impExt      json.RawMessage
		expectedErr string
	}{
		{
			name:        "invalid imp ext",
			impExt:      json.RawMessage(`not-json`),
			expectedErr: "invalid imp.ext",
		},
		{
			name:        "invalid bidder ext",
			impExt:      json.RawMessage(`{"bidder":"not-an-object"}`),
			expectedErr: "invalid imp.ext.bidder",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			bidder := &adapter{endpoint: "https://ferio.bid/bidder"}
			req := &openrtb2.BidRequest{
				ID:   "auction-id",
				Site: &openrtb2.Site{},
				Imp: []openrtb2.Imp{{
					ID:     "imp-1",
					Banner: &openrtb2.Banner{},
					Ext:    test.impExt,
				}},
			}

			requests, errs := bidder.MakeRequests(req, nil)
			assert.Nil(t, requests)
			require.Len(t, errs, 2)
			assert.Contains(t, errs[0].Error(), test.expectedErr)
			assert.EqualError(t, errs[1], "found no valid impressions")
		})
	}
}

func TestMakeBidsMediaTypes(t *testing.T) {
	bidder := &adapter{}
	resp := openrtb2.BidResponse{
		Cur: "EUR",
		SeatBid: []openrtb2.SeatBid{{
			Bid: []openrtb2.Bid{
				{ID: "bid-1", ImpID: "imp-1", MType: openrtb2.MarkupBanner},
				{ID: "bid-2", ImpID: "imp-2", MType: openrtb2.MarkupVideo},
				{ID: "bid-3", ImpID: "imp-3", MType: openrtb2.MarkupNative},
			},
		}},
	}

	bidResponse, errs := bidder.MakeBids(&openrtb2.BidRequest{}, nil, responseData(t, http.StatusOK, resp))
	require.Empty(t, errs)
	require.NotNil(t, bidResponse)
	assert.Equal(t, "EUR", bidResponse.Currency)
	require.Len(t, bidResponse.Bids, 3)
	assert.Equal(t, openrtb_ext.BidTypeBanner, bidResponse.Bids[0].BidType)
	assert.Equal(t, openrtb_ext.BidTypeVideo, bidResponse.Bids[1].BidType)
	assert.Equal(t, openrtb_ext.BidTypeNative, bidResponse.Bids[2].BidType)
}

func TestMakeBidsMissingMType(t *testing.T) {
	bidder := &adapter{}
	resp := openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{{
			Bid: []openrtb2.Bid{{
				ID:    "bid-1",
				ImpID: "imp-1",
				Ext:   json.RawMessage(`{"prebid":{"type":"video"}}`),
			}},
		}},
	}

	bidResponse, errs := bidder.MakeBids(&openrtb2.BidRequest{}, nil, responseData(t, http.StatusOK, resp))
	require.NotNil(t, bidResponse)
	assert.Empty(t, bidResponse.Bids)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "missing mtype for imp imp-1")
}

func TestMakeBidsNoContent(t *testing.T) {
	bidder := &adapter{}

	bidResponse, errs := bidder.MakeBids(&openrtb2.BidRequest{}, nil, &adapters.ResponseData{StatusCode: http.StatusNoContent})
	assert.Nil(t, bidResponse)
	assert.Nil(t, errs)
}

func TestMakeBidsErrors(t *testing.T) {
	bidder := &adapter{}

	bidResponse, errs := bidder.MakeBids(&openrtb2.BidRequest{}, nil, &adapters.ResponseData{StatusCode: http.StatusInternalServerError})
	assert.Nil(t, bidResponse)
	require.Len(t, errs, 1)

	bidResponse, errs = bidder.MakeBids(&openrtb2.BidRequest{}, nil, &adapters.ResponseData{StatusCode: http.StatusOK, Body: []byte(`not-json`)})
	assert.Nil(t, bidResponse)
	require.Len(t, errs, 1)

	resp := openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{{
			Bid: []openrtb2.Bid{{ID: "bid-1", ImpID: "imp-1", MType: openrtb2.MarkupAudio}},
		}},
	}
	bidResponse, errs = bidder.MakeBids(&openrtb2.BidRequest{}, nil, responseData(t, http.StatusOK, resp))
	require.NotNil(t, bidResponse)
	assert.Empty(t, bidResponse.Bids)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "unsupported")
}

func responseData(t *testing.T, statusCode int, response openrtb2.BidResponse) *adapters.ResponseData {
	t.Helper()

	body, err := json.Marshal(response)
	require.NoError(t, err)

	return &adapters.ResponseData{
		StatusCode: statusCode,
		Body:       body,
	}
}
