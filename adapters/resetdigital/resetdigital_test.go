package resetdigital

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
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderResetDigital, config.Adapter{
		Endpoint: "https://test.com"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "resetdigitaltest", bidder)
}

func TestGetBidType(t *testing.T) {
	cases := []struct {
		name    string
		bid     openrtb2.Bid
		request *openrtb2.BidRequest
		want    openrtb_ext.BidType
		wantErr bool
	}{
		{
			name: "banner",
			bid:  openrtb2.Bid{ImpID: "imp-banner"},
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-banner", Banner: &openrtb2.Banner{}},
				},
			},
			want:    openrtb_ext.BidTypeBanner,
			wantErr: false,
		},
		{
			name: "video",
			bid:  openrtb2.Bid{ImpID: "imp-video"},
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-video", Video: &openrtb2.Video{}},
				},
			},
			want:    openrtb_ext.BidTypeVideo,
			wantErr: false,
		},
		{
			name: "audio",
			bid:  openrtb2.Bid{ImpID: "imp-audio"},
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-audio", Audio: &openrtb2.Audio{}},
				},
			},
			want:    openrtb_ext.BidTypeAudio,
			wantErr: false,
		},
		{
			name: "native",
			bid:  openrtb2.Bid{ImpID: "imp-native"},
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-native", Native: &openrtb2.Native{}},
				},
			},
			want:    openrtb_ext.BidTypeNative,
			wantErr: false,
		},
		{
			name: "no matching imp",
			bid:  openrtb2.Bid{ImpID: "no-match"},
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-1", Banner: &openrtb2.Banner{}},
				},
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := getBidType(c.bid, c.request)
			if (err != nil) != c.wantErr {
				t.Errorf("getBidType() error = %v, wantErr %v", err, c.wantErr)
				return
			}
			if got != c.want {
				t.Errorf("getBidType() got = %v, want %v", got, c.want)
			}
		})
	}
}

func TestGetMediaType(t *testing.T) {
	cases := []struct {
		name string
		imp  openrtb2.Imp
		want openrtb_ext.BidType
	}{
		{
			name: "banner",
			imp:  openrtb2.Imp{Banner: &openrtb2.Banner{}},
			want: openrtb_ext.BidTypeBanner,
		},
		{
			name: "video",
			imp:  openrtb2.Imp{Video: &openrtb2.Video{}},
			want: openrtb_ext.BidTypeVideo,
		},
		{
			name: "audio",
			imp:  openrtb2.Imp{Audio: &openrtb2.Audio{}},
			want: openrtb_ext.BidTypeAudio,
		},
		{
			name: "native",
			imp:  openrtb2.Imp{Native: &openrtb2.Native{}},
			want: openrtb_ext.BidTypeNative,
		},
		{
			name: "default to banner",
			imp:  openrtb2.Imp{},
			want: openrtb_ext.BidTypeBanner,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := getMediaType(c.imp)
			if got != c.want {
				t.Errorf("getMediaType() got = %v, want %v", got, c.want)
			}
		})
	}
}

func TestMakeBidsOpenRTB(t *testing.T) {

    bidRequest := &openrtb2.BidRequest{
        ID: "12345",
        Imp: []openrtb2.Imp{
            {
                ID:     "001",
                Banner: &openrtb2.Banner{},
            },
        },
    }

    bidResponseJSON := `{
        "bids": [{
            "bid_id": "bid1",
            "imp_id": "001",
            "cpm": 2.0,
            "cid": "test-cid",
            "crid": "test-crid",
            "adid": "test-adid",
            "w": "300",
            "h": "250",
            "seat": "resetdigital",
            "html": "<html><body>Ad</body></html>"
        }]
    }`

    responseData := &adapters.ResponseData{
        StatusCode: http.StatusOK,
        Body:       []byte(bidResponseJSON),
    }

    a := adapter{endpoint: "https://test.com"}

    bidderResponse, errs := a.MakeBids(bidRequest, &adapters.RequestData{}, responseData)

    assert.Empty(t, errs)
    assert.NotNil(t, bidderResponse)
    assert.Equal(t, "USD", bidderResponse.Currency)
    assert.Len(t, bidderResponse.Bids, 1)
    
    assert.Equal(t, "bid1", bidderResponse.Bids[0].Bid.ID)
    assert.Equal(t, "001", bidderResponse.Bids[0].Bid.ImpID)
    assert.Equal(t, 2.0, bidderResponse.Bids[0].Bid.Price)
}

func TestMakeBidsNoMatchingImp(t *testing.T) {
	bidRequest := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{
			{
				ID:     "test-imp-1",
				Banner: &openrtb2.Banner{},
			},
		},
	}

	bidResponseJSON := `{
		"id": "test-request-id",
		"seatbid": [{
			"bid": [{
				"id": "bid1",
				"impid": "non-matching-imp",
				"price": 2.0
			}]
		}],
		"cur": "USD"
	}`

	responseData := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(bidResponseJSON),
	}

	a := adapter{endpoint: "https://test.com"}

	bidderResponse, errs := a.MakeBids(bidRequest, &adapters.RequestData{}, responseData)

	assert.Empty(t, errs)
	assert.NotNil(t, bidderResponse)
	assert.Empty(t, bidderResponse.Bids)
}

func TestMakeRequestsHeaders(t *testing.T) {
	bidRequest := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{
			{
				ID:     "test-imp-1",
				Banner: &openrtb2.Banner{},
				Ext:    json.RawMessage(`{"bidder":{"placement_id":"42"}}`),
			},
		},
		Device: &openrtb2.Device{
			UA: "test-user-agent",
		},
		Site: &openrtb2.Site{
			Page: "https://example.com/page",
		},
	}

	a := adapter{endpoint: "https://test.endpoint.com"}

	requests, errs := a.MakeRequests(bidRequest, nil)

	assert.Empty(t, errs)
	assert.Len(t, requests, 1)
	
	assert.Equal(t, "application/json", requests[0].Headers.Get("Content-Type"))
	assert.Equal(t, "application/json", requests[0].Headers.Get("Accept"))
	assert.Equal(t, "2.6", requests[0].Headers.Get("X-OpenRTB-Version"))
}

func TestMakeRequestsProduction(t *testing.T) {
	bidRequest := &openrtb2.BidRequest{
		ID: "test-production-request",
		Imp: []openrtb2.Imp{
			{
				ID:     "test-imp-1",
				Banner: &openrtb2.Banner{},
				Ext:    json.RawMessage(`{"bidder":{"placement_id":"42"}}`),
			},
		},
		Site: &openrtb2.Site{
			Domain: "example.com",
		},
	}

	a := adapter{endpoint: "https://test.endpoint.com"}

	requests, errs := a.MakeRequests(bidRequest, nil)

	assert.Empty(t, errs)
	assert.Len(t, requests, 1)
	assert.Equal(t, "https://test.endpoint.com?pid=42", requests[0].Uri)
	assert.Equal(t, "POST", requests[0].Method)

	var requestObj openrtb2.BidRequest
	err := json.Unmarshal(requests[0].Body, &requestObj)
	assert.NoError(t, err)
	assert.Equal(t, "test-production-request", requestObj.ID)
	assert.Len(t, requestObj.Imp, 1)
	assert.Equal(t, "test-imp-1", requestObj.Imp[0].ID)
}

func TestMakeRequestsMissingPlacementID(t *testing.T) {
	bidRequest := &openrtb2.BidRequest{
		ID: "test-production-request",
		Imp: []openrtb2.Imp{
			{
				ID:     "test-imp-1",
				Banner: &openrtb2.Banner{},
				Ext:    json.RawMessage(`{"bidder":{"placement_id":""}}`),
			},
		},
	}

	a := adapter{endpoint: "https://test.endpoint.com"}

	requests, errs := a.MakeRequests(bidRequest, nil)

	assert.Empty(t, errs)
	assert.Len(t, requests, 1)
}

func TestMakeRequestsMultipleImps(t *testing.T) {
	bidRequest := &openrtb2.BidRequest{
		ID: "test-multiple-imps",
		Imp: []openrtb2.Imp{
			{
				ID:     "imp-1",
				Banner: &openrtb2.Banner{},
				Ext:    json.RawMessage(`{"bidder":{"placement_id":"42"}}`),
			},
			{
				ID:    "imp-2",
				Video: &openrtb2.Video{},
				Ext:   json.RawMessage(`{"bidder":{"placement_id":"43"}}`),
			},
			{
				ID:    "imp-3",
				Audio: &openrtb2.Audio{},
				Ext:   json.RawMessage(`{"bidder":{"placement_id":"44"}}`),
			},
		},
		Site: &openrtb2.Site{
			Domain: "example.com",
		},
	}

	a := adapter{endpoint: "https://test.endpoint.com"}

	requests, errs := a.MakeRequests(bidRequest, nil)

	assert.Empty(t, errs)
	assert.Len(t, requests, 3)
	
	for i, request := range requests {
		assert.Contains(t, request.Uri, "https://test.endpoint.com?pid=4")
		assert.Equal(t, "POST", request.Method)
		assert.Len(t, request.ImpIDs, 1)
		assert.Equal(t, bidRequest.Imp[i].ID, request.ImpIDs[0])
	}
}

func TestMakeBidsZeroPrice(t *testing.T) {
	bidRequest := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{
			{
				ID:     "test-imp-1",
				Banner: &openrtb2.Banner{},
			},
		},
	}

	bidResponseJSON := `{
		"id": "test-request-id",
		"seatbid": [{
			"bid": [{
				"id": "bid1",
				"impid": "test-imp-1",
				"price": 0.0
			}]
		}],
		"cur": "USD"
	}`

	responseData := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(bidResponseJSON),
	}

	a := adapter{endpoint: "https://test.com"}

	bidderResponse, errs := a.MakeBids(bidRequest, &adapters.RequestData{}, responseData)

	assert.Empty(t, errs)
	assert.NotNil(t, bidderResponse)
	assert.Empty(t, bidderResponse.Bids)
}

func TestMakeBidsMissingCurrency(t *testing.T) {
	bidRequest := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{
			{
				ID:     "test-imp-1",
				Banner: &openrtb2.Banner{},
			},
		},
	}

	bidResponseJSON := `{
		"id": "test-request-id",
		"seatbid": [{
			"bid": [{
				"id": "bid1",
				"impid": "test-imp-1",
				"price": 1.0,
				"adm": "<html><body>Ad</body></html>"
			}]
		}]
	}`

	responseData := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(bidResponseJSON),
	}

	a := adapter{endpoint: "https://test.com"}

	bidderResponse, errs := a.MakeBids(bidRequest, &adapters.RequestData{}, responseData)

	assert.Empty(t, errs)
	assert.NotNil(t, bidderResponse)
	assert.Equal(t, "USD", bidderResponse.Currency)
	
	if len(bidderResponse.Bids) > 0 {
		assert.True(t, bidderResponse.Bids[0].Bid.Price > 0)
	} else {
		t.Log("No se encontraron pujas en la respuesta, pero este comportamiento es aceptable")
	}
}

func TestMakeRequestsInvalidJson(t *testing.T) {
	bidRequest := &openrtb2.BidRequest{
		ID: "test-invalid-json",
		Imp: []openrtb2.Imp{
			{
				ID:     "test-imp-1",
				Banner: &openrtb2.Banner{},
				Ext:    json.RawMessage(`{"bidder":{"placement_id":"42"`),
			},
		},
	}

	a := adapter{endpoint: "https://test.endpoint.com"}

	requests, errs := a.MakeRequests(bidRequest, nil)

	assert.Len(t, errs, 1)
	assert.Empty(t, requests)
	assert.Contains(t, errs[0].Error(), "unexpected end of JSON input")
}

func TestMakeBidsServerError(t *testing.T) {
	bidRequest := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{
			{
				ID:     "test-imp-1",
				Banner: &openrtb2.Banner{},
			},
		},
	}

	responseData := &adapters.ResponseData{
		StatusCode: http.StatusBadRequest,
		Body:       []byte(`{"error":"Invalid request"}`),
	}

	a := adapter{endpoint: "https://test.com"}

	bidderResponse, errs := a.MakeBids(bidRequest, &adapters.RequestData{}, responseData)

	assert.Len(t, errs, 1)
	assert.Nil(t, bidderResponse)
	assert.Contains(t, errs[0].Error(), "Unexpected status code")
}