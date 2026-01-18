package scalibur

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAdapter() adapters.Bidder {
	adapter, _ := Builder(
		openrtb_ext.BidderScalibur,
		config.Adapter{Endpoint: "https://srv.scalibur.io/adserver/ortb?type=prebid-server"},
		config.Server{},
	)
	return adapter
}

//
// ------------------------------------------------------------------------------------------
// MAKE REQUESTS TESTS
// ------------------------------------------------------------------------------------------
//

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "scaliburtest", newTestAdapter())
}

func TestMakeRequests_SuccessBanner(t *testing.T) {
	bidder := newTestAdapter()

	ext, _ := json.Marshal(adapters.ExtImpBidder{
		Bidder: json.RawMessage(`{
			"placementId": "p123",
			"bidfloor": 1.25,
			"bidfloorcur": "EUR"
		}`),
	})

	req := &openrtb2.BidRequest{
		ID: "req1",
		Imp: []openrtb2.Imp{
			{
				ID:  "imp1",
				Ext: ext,
				Banner: &openrtb2.Banner{
					W: ptrInt64(300), H: ptrInt64(250),
				},
			},
		},
		Site: &openrtb2.Site{Page: "https://test.com"},
	}

	requests, errs := bidder.MakeRequests(req, &adapters.ExtraRequestInfo{
		CurrencyConversions: &mockConversions{},
	})

	require.Len(t, errs, 0)
	require.Len(t, requests, 1)

	r := requests[0]
	assert.Equal(t, "https://srv.scalibur.io/adserver/ortb?type=prebid-server", r.Uri)
	assert.Equal(t, "POST", r.Method)
	assert.Contains(t, r.Headers.Get("Content-Type"), "application/json")

	// Ensure body contains rewritten ext fields
	var out openrtb2.BidRequest
	require.NoError(t, json.Unmarshal(r.Body, &out))
	require.Len(t, out.Imp, 1)

	imp := out.Imp[0]

	assert.Equal(t, float64(1.25), imp.BidFloor)
	assert.Equal(t, "USD", imp.BidFloorCur)

	var outExt map[string]interface{}
	require.NoError(t, json.Unmarshal(imp.Ext, &outExt))
	assert.Equal(t, "p123", outExt["placementId"])
}

func TestMakeRequests_InvalidExt(t *testing.T) {
	bidder := newTestAdapter()

	// Missing placementId
	badExt, _ := json.Marshal(adapters.ExtImpBidder{
		Bidder: json.RawMessage(`{"bidfloor": 1.5}`),
	})

	req := &openrtb2.BidRequest{
		ID: "req2",
		Imp: []openrtb2.Imp{
			{
				ID:  "imp1",
				Ext: badExt,
				Banner: &openrtb2.Banner{
					W: ptrInt64(300), H: ptrInt64(250),
				},
			},
		},
	}

	requests, errs := bidder.MakeRequests(req, &adapters.ExtraRequestInfo{})

	require.Len(t, requests, 0)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "placementId is required")
}

func TestMakeRequests_VideoDefaultsApplied(t *testing.T) {
	bidder := newTestAdapter()

	ext, _ := json.Marshal(adapters.ExtImpBidder{
		Bidder: json.RawMessage(`{"placementId": "p123"}`),
	})

	req := &openrtb2.BidRequest{
		ID: "req-video",
		Imp: []openrtb2.Imp{
			{
				ID:    "v1",
				Ext:   ext,
				Video: &openrtb2.Video{
					// Intentionally empty → should fill defaults
				},
			},
		},
	}

	requests, errs := bidder.MakeRequests(req, &adapters.ExtraRequestInfo{
		CurrencyConversions: &mockConversions{},
	})

	require.Len(t, errs, 0)
	require.Len(t, requests, 1)

	var out openrtb2.BidRequest
	require.NoError(t, json.Unmarshal(requests[0].Body, &out))

	v := out.Imp[0].Video
	require.NotNil(t, v)

	assert.NotEmpty(t, v.MIMEs)
	assert.NotZero(t, v.MinDuration)
	assert.NotZero(t, v.MaxDuration)
	assert.NotZero(t, v.MaxBitRate)
	assert.NotEmpty(t, v.Protocols)
	assert.NotNil(t, v.W)
	assert.NotNil(t, v.H)
	assert.NotZero(t, v.Placement)
	assert.NotZero(t, v.Linearity)
}

type mockConversions struct{}

func (m *mockConversions) GetRate(from string, to string) (float64, error) {
	return 1.0, nil
}

func (m *mockConversions) GetRates() *map[string]map[string]float64 {
	return nil
}

//
// ------------------------------------------------------------------------------------------
// MAKE BIDS TESTS
// ------------------------------------------------------------------------------------------
//

func TestMakeBids_SuccessBanner(t *testing.T) {
	bidder := newTestAdapter()

	mockReq := &openrtb2.BidRequest{
		ID: "req1",
		Imp: []openrtb2.Imp{
			{
				ID:     "1",
				Banner: &openrtb2.Banner{W: ptrInt64(300), H: ptrInt64(250)},
			},
		},
	}

	mockReqData := &adapters.RequestData{
		Body: func() []byte {
			b, _ := json.Marshal(mockReq)
			return b
		}(),
	}

	mockResp := &openrtb2.BidResponse{
		ID:  "resp1",
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Bid: []openrtb2.Bid{
					{
						ID:    "b1",
						ImpID: "1",
						Price: 2.5,
						AdM:   "<div>ad markup</div>",
						W:     300,
						H:     250,
					},
				},
			},
		},
	}

	respData := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body: func() []byte {
			b, _ := json.Marshal(mockResp)
			return b
		}(),
	}

	bidderResp, errs := bidder.MakeBids(mockReq, mockReqData, respData)

	require.Len(t, errs, 0)
	require.NotNil(t, bidderResp)
	require.Len(t, bidderResp.Bids, 1)

	b := bidderResp.Bids[0]
	assert.Equal(t, openrtb_ext.BidTypeBanner, b.BidType)
	assert.Equal(t, float64(2.5), b.Bid.Price)
	assert.Equal(t, "<div>ad markup</div>", b.Bid.AdM)
}

func TestMakeBids_EmptySeatBid(t *testing.T) {
	bidder := newTestAdapter()

	mockReq := &openrtb2.BidRequest{
		ID:  "req2",
		Imp: []openrtb2.Imp{},
	}

	mockReqData := &adapters.RequestData{
		Body: []byte(`{"id":"req2","imp":[]}`),
	}

	respData := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"id":"resp","seatbid":[]}`),
	}

	bidderResp, errs := bidder.MakeBids(mockReq, mockReqData, respData)

	require.Len(t, errs, 0)
	assert.Nil(t, bidderResp)
}

func TestMakeBids_Status204(t *testing.T) {
	bidder := newTestAdapter()

	respData := &adapters.ResponseData{
		StatusCode: http.StatusNoContent,
	}

	resp, errs := bidder.MakeBids(&openrtb2.BidRequest{}, &adapters.RequestData{}, respData)

	assert.Nil(t, resp)
	assert.Nil(t, errs)
}

func TestMakeBids_InvalidJSON(t *testing.T) {
	bidder := newTestAdapter()

	respData := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`not-json`),
	}

	_, errs := bidder.MakeBids(&openrtb2.BidRequest{}, &adapters.RequestData{}, respData)
	require.Len(t, errs, 1)
}

func TestMakeBids_ImpNotFound(t *testing.T) {
	bidder := newTestAdapter()

	mockReq := &openrtb2.BidRequest{
		ID: "req1",
		Imp: []openrtb2.Imp{
			{ID: "1"},
		},
	}

	mockReqData := &adapters.RequestData{
		Body: []byte(`{"id":"req1","imp":[{"id":"1"}]}`),
	}

	mockResp := &openrtb2.BidResponse{
		ID: "resp1",
		SeatBid: []openrtb2.SeatBid{
			{
				Bid: []openrtb2.Bid{
					{
						ID:    "b1",
						ImpID: "non-existent",
					},
				},
			},
		},
	}

	respData := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body: func() []byte {
			b, _ := json.Marshal(mockResp)
			return b
		}(),
	}

	_, errs := bidder.MakeBids(mockReq, mockReqData, respData)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "Invalid bid imp ID non-existent")
}

func TestGetBidMediaType_Error(t *testing.T) {
	bid := openrtb2.Bid{
		ID:    "bid1",
		ImpID: "imp1",
	}
	imp := &openrtb2.Imp{
		ID: "imp1",
		// No banner or video
	}

	_, err := getBidMediaType(bid, imp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported or ambiguous media type")
}

func TestParseScaliburExt_ErrorCases(t *testing.T) {
	// Case 1: Invalid JSON for imp.ext
	_, err := parseScaliburExt(json.RawMessage(`{invalid`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to parse imp.ext")

	// Case 2: Missing placementId
	_, err = parseScaliburExt(json.RawMessage(`{"bidder": {"bidfloor": 1.0}}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "placementId is required")
}

func ptrInt64(x int64) *int64 { return &x }
