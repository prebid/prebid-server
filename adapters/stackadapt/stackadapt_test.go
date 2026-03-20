package stackadapt

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
)

const testsDir = "stackadapttest"
const testsBidderEndpoint = "http://localhost/br?publisher_id={{.PublisherID}}&supply_id={{.SupplyId}}"

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderStackAdapt,
		config.Adapter{Endpoint: testsBidderEndpoint},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, testsDir, bidder)
}

func TestGetMediaTypeForBid(t *testing.T) {
	tests := []struct {
		name         string
		mtype        openrtb2.MarkupType
		expectedType openrtb_ext.BidType
		wantErr      bool
	}{
		{name: "banner", mtype: 1, expectedType: openrtb_ext.BidTypeBanner},
		{name: "video", mtype: 2, expectedType: openrtb_ext.BidTypeVideo},
		{name: "audio", mtype: 3, expectedType: openrtb_ext.BidTypeAudio},
		{name: "native", mtype: 4, expectedType: openrtb_ext.BidTypeNative},
		{name: "zero", mtype: 0, expectedType: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bidType, err := getMediaTypeForBid(openrtb2.Bid{MType: tt.mtype})
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, bidType)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedType, bidType)
			}
		})
	}
}

func TestResolveMacros(t *testing.T) {
	tests := []struct {
		name         string
		bid          *openrtb2.Bid
		expectedAdM  string
		expectedNURL string
		expectedBURL string
	}{
		{
			name: "nil_bid",
			bid:  nil,
		},
		{
			name: "replaces_in_all_fields",
			bid: &openrtb2.Bid{
				Price: 2.50,
				AdM:   "price=${AUCTION_PRICE}",
				NURL:  "http://win?p=${AUCTION_PRICE}",
				BURL:  "http://bill?p=${AUCTION_PRICE}",
			},
			expectedAdM:  "price=2.5",
			expectedNURL: "http://win?p=2.5",
			expectedBURL: "http://bill?p=2.5",
		},
		{
			name: "multiple_macros_same_field",
			bid: &openrtb2.Bid{
				Price: 1.00,
				AdM:   "${AUCTION_PRICE}+${AUCTION_PRICE}",
			},
			expectedAdM: "1+1",
		},
		{
			name: "no_macros",
			bid: &openrtb2.Bid{
				Price: 5.00,
				AdM:   "<div>ad</div>",
				NURL:  "http://example.com",
			},
			expectedAdM:  "<div>ad</div>",
			expectedNURL: "http://example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolveMacros(tt.bid)
			if tt.bid == nil {
				return
			}
			assert.Equal(t, tt.expectedAdM, tt.bid.AdM)
			assert.Equal(t, tt.expectedNURL, tt.bid.NURL)
			assert.Equal(t, tt.expectedBURL, tt.bid.BURL)
		})
	}
}

func TestGetNativeAdm(t *testing.T) {
	tests := []struct {
		name        string
		adm         string
		expectedAdm string
		wantErr     bool
	}{
		{
			name:        "unwraps_native_envelope",
			adm:         `{"native":{"link":{"url":"https://example.com"},"assets":[{"id":1,"title":{"text":"Title"}}]}}`,
			expectedAdm: `{"link":{"url":"https://example.com"},"assets":[{"id":1,"title":{"text":"Title"}}]}`,
		},
		{
			name:        "already_unwrapped",
			adm:         `{"link":{"url":"https://example.com"},"assets":[{"id":1,"title":{"text":"Title"}}]}`,
			expectedAdm: `{"link":{"url":"https://example.com"},"assets":[{"id":1,"title":{"text":"Title"}}]}`,
		},
		{
			name:    "invalid_json",
			adm:     `not json`,
			wantErr: true,
		},
		{
			name:    "native_key_not_object",
			adm:     `{"native":"string_value"}`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getNativeAdm(tt.adm)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.JSONEq(t, tt.expectedAdm, result)
			}
		})
	}
}

func TestSetPublisherID(t *testing.T) {
	t.Run("site_without_publisher", func(t *testing.T) {
		req := &openrtb2.BidRequest{Site: &openrtb2.Site{Page: "https://example.com"}}
		setPublisherID(req, "pub-1")
		assert.Equal(t, "pub-1", req.Site.Publisher.ID)
	})

	t.Run("site_with_existing_publisher", func(t *testing.T) {
		req := &openrtb2.BidRequest{Site: &openrtb2.Site{
			Page:      "https://example.com",
			Publisher: &openrtb2.Publisher{ID: "old", Name: "OldPub"},
		}}
		setPublisherID(req, "pub-2")
		assert.Equal(t, "pub-2", req.Site.Publisher.ID)
		assert.Equal(t, "OldPub", req.Site.Publisher.Name)
	})

	t.Run("app_without_publisher", func(t *testing.T) {
		req := &openrtb2.BidRequest{App: &openrtb2.App{Bundle: "com.example"}}
		setPublisherID(req, "app-pub-1")
		assert.Equal(t, "app-pub-1", req.App.Publisher.ID)
	})

	t.Run("app_with_existing_publisher", func(t *testing.T) {
		req := &openrtb2.BidRequest{App: &openrtb2.App{
			Bundle:    "com.example",
			Publisher: &openrtb2.Publisher{ID: "old", Name: "OldAppPub"},
		}}
		setPublisherID(req, "app-pub-2")
		assert.Equal(t, "app-pub-2", req.App.Publisher.ID)
		assert.Equal(t, "OldAppPub", req.App.Publisher.Name)
	})

	t.Run("site_takes_precedence_over_app", func(t *testing.T) {
		req := &openrtb2.BidRequest{
			Site: &openrtb2.Site{Page: "https://example.com"},
			App:  &openrtb2.App{Bundle: "com.example"},
		}
		setPublisherID(req, "pub-site")
		assert.Equal(t, "pub-site", req.Site.Publisher.ID)
		assert.Nil(t, req.App.Publisher)
	})

	t.Run("no_site_or_app", func(t *testing.T) {
		req := &openrtb2.BidRequest{}
		setPublisherID(req, "pub-1")
		assert.Nil(t, req.Site)
		assert.Nil(t, req.App)
	})
}

func TestSetImpsAndGetEndpointParams(t *testing.T) {
	t.Run("valid_with_all_params", func(t *testing.T) {
		req := &openrtb2.BidRequest{
			Imp: []openrtb2.Imp{{
				Banner: &openrtb2.Banner{},
				Ext:    json.RawMessage(`{"bidder":{"publisherId":"pub-1","supplyId":"ssp-1","placementId":"pl-1","bidfloor":1.5,"banner":{"expdir":[1,3]}}}`),
			}},
		}
		pubID, supplyID, err := setImpsAndGetEndpointParams(req)
		assert.NoError(t, err)
		assert.Equal(t, "pub-1", pubID)
		assert.Equal(t, "ssp-1", supplyID)
		assert.Equal(t, "pl-1", req.Imp[0].TagID)
		assert.Equal(t, 1.5, req.Imp[0].BidFloor)
		assert.Equal(t, "USD", req.Imp[0].BidFloorCur)
		assert.Len(t, req.Imp[0].Banner.ExpDir, 2)
	})

	t.Run("publisher_id_and_supply_id", func(t *testing.T) {
		req := &openrtb2.BidRequest{
			Imp: []openrtb2.Imp{{
				Ext: json.RawMessage(`{"bidder":{"publisherId":"pub-2","supplyId":"ssp-2"}}`),
			}},
		}
		pubID, supplyID, err := setImpsAndGetEndpointParams(req)
		assert.NoError(t, err)
		assert.Equal(t, "pub-2", pubID)
		assert.Equal(t, "ssp-2", supplyID)
		assert.Empty(t, req.Imp[0].TagID)
		assert.Zero(t, req.Imp[0].BidFloor)
	})

	t.Run("missing_publisher_id", func(t *testing.T) {
		req := &openrtb2.BidRequest{
			Imp: []openrtb2.Imp{{
				Ext: json.RawMessage(`{"bidder":{"supplyId":"ssp-1"}}`),
			}},
		}
		_, _, err := setImpsAndGetEndpointParams(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "publisherId is required")
	})

	t.Run("missing_supply_id", func(t *testing.T) {
		req := &openrtb2.BidRequest{
			Imp: []openrtb2.Imp{{
				Ext: json.RawMessage(`{"bidder":{"publisherId":"pub-1"}}`),
			}},
		}
		_, _, err := setImpsAndGetEndpointParams(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "supplyId is required")
	})

	t.Run("invalid_ext_json", func(t *testing.T) {
		req := &openrtb2.BidRequest{
			Imp: []openrtb2.Imp{{
				Ext: json.RawMessage(`{invalid`),
			}},
		}
		_, _, err := setImpsAndGetEndpointParams(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unable to unmarshal ext")
	})

	t.Run("invalid_bidder_ext_json", func(t *testing.T) {
		req := &openrtb2.BidRequest{
			Imp: []openrtb2.Imp{{
				Ext: json.RawMessage(`{"bidder":"not-an-object"}`),
			}},
		}
		_, _, err := setImpsAndGetEndpointParams(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unable to unmarshal bidder ext")
	})

	t.Run("multi_imp_uses_first_publisher_id_and_supply_id", func(t *testing.T) {
		req := &openrtb2.BidRequest{
			Imp: []openrtb2.Imp{
				{Ext: json.RawMessage(`{"bidder":{"publisherId":"first","supplyId":"ssp-first"}}`)},
				{Ext: json.RawMessage(`{"bidder":{"publisherId":"second","supplyId":"ssp-second"}}`)},
			},
		}
		pubID, supplyID, err := setImpsAndGetEndpointParams(req)
		assert.NoError(t, err)
		assert.Equal(t, "first", pubID)
		assert.Equal(t, "ssp-first", supplyID)
	})

	t.Run("zero_bidfloor_not_set", func(t *testing.T) {
		req := &openrtb2.BidRequest{
			Imp: []openrtb2.Imp{{
				Ext: json.RawMessage(`{"bidder":{"publisherId":"pub-1","supplyId":"ssp-1","bidfloor":0}}`),
			}},
		}
		_, _, err := setImpsAndGetEndpointParams(req)
		assert.NoError(t, err)
		assert.Zero(t, req.Imp[0].BidFloor)
		assert.Empty(t, req.Imp[0].BidFloorCur)
	})

	t.Run("expdir_ignored_without_banner", func(t *testing.T) {
		req := &openrtb2.BidRequest{
			Imp: []openrtb2.Imp{{
				Ext: json.RawMessage(`{"bidder":{"publisherId":"pub-1","supplyId":"ssp-1","banner":{"expdir":[1,2]}}}`),
			}},
		}
		_, _, err := setImpsAndGetEndpointParams(req)
		assert.NoError(t, err)
		assert.Nil(t, req.Imp[0].Banner)
	})
}

func TestMakeRequests(t *testing.T) {
	a, err := Builder(openrtb_ext.BidderStackAdapt, config.Adapter{Endpoint: testsBidderEndpoint}, config.Server{})
	assert.NoError(t, err)

	t.Run("site_request", func(t *testing.T) {
		req := &openrtb2.BidRequest{
			Imp:  []openrtb2.Imp{{Ext: json.RawMessage(`{"bidder":{"publisherId":"pub-1","supplyId":"ssp-1"}}`)}},
			Site: &openrtb2.Site{Page: "https://example.com"},
		}
		data, errs := a.MakeRequests(req, nil)
		assert.Empty(t, errs)
		assert.Len(t, data, 1)
		assert.Equal(t, http.MethodPost, data[0].Method)
		assert.Equal(t, "http://localhost/br?publisher_id=pub-1&supply_id=ssp-1", data[0].Uri)
	})

	t.Run("app_request", func(t *testing.T) {
		req := &openrtb2.BidRequest{
			Imp: []openrtb2.Imp{{Ext: json.RawMessage(`{"bidder":{"publisherId":"pub-1","supplyId":"ssp-1"}}`)}},
			App: &openrtb2.App{Bundle: "com.example"},
		}
		data, errs := a.MakeRequests(req, nil)
		assert.Empty(t, errs)
		assert.Len(t, data, 1)

		var body openrtb2.BidRequest
		assert.NoError(t, json.Unmarshal(data[0].Body, &body))
		assert.Equal(t, "pub-1", body.App.Publisher.ID)
		assert.Equal(t, "http://localhost/br?publisher_id=pub-1&supply_id=ssp-1", data[0].Uri)
	})

	t.Run("missing_publisher_id_returns_error", func(t *testing.T) {
		req := &openrtb2.BidRequest{
			Imp:  []openrtb2.Imp{{Ext: json.RawMessage(`{"bidder":{"supplyId":"ssp-1"}}`)}},
			Site: &openrtb2.Site{},
		}
		data, errs := a.MakeRequests(req, nil)
		assert.Nil(t, data)
		assert.Len(t, errs, 1)
	})

	t.Run("missing_supply_id_returns_error", func(t *testing.T) {
		req := &openrtb2.BidRequest{
			Imp:  []openrtb2.Imp{{Ext: json.RawMessage(`{"bidder":{"publisherId":"pub-1"}}`)}},
			Site: &openrtb2.Site{},
		}
		data, errs := a.MakeRequests(req, nil)
		assert.Nil(t, data)
		assert.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "supplyId is required")
	})
}

func TestBuildEndpointURL(t *testing.T) {
	endpointTemplate := "http://localhost/br?publisher_id={{.PublisherID}}&supply_id={{.SupplyId}}"
	a, err := Builder(openrtb_ext.BidderStackAdapt, config.Adapter{Endpoint: endpointTemplate}, config.Server{})
	assert.NoError(t, err)

	req := &openrtb2.BidRequest{
		Imp:  []openrtb2.Imp{{Ext: json.RawMessage(`{"bidder":{"publisherId":"test-pub","supplyId":"test-ssp"}}`)}},
		Site: &openrtb2.Site{},
	}
	data, errs := a.MakeRequests(req, nil)
	assert.Empty(t, errs)
	assert.Len(t, data, 1)
	assert.Equal(t, "http://localhost/br?publisher_id=test-pub&supply_id=test-ssp", data[0].Uri)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderStackAdapt, config.Adapter{
		Endpoint: "{{Malformed}}",
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	assert.Error(t, buildErr)
}

func TestMakeBids(t *testing.T) {
	a, err := Builder(openrtb_ext.BidderStackAdapt,
		config.Adapter{Endpoint: testsBidderEndpoint}, config.Server{})
	assert.NoError(t, err)

	t.Run("204_no_content", func(t *testing.T) {
		resp, errs := a.MakeBids(nil, nil, &adapters.ResponseData{StatusCode: http.StatusNoContent})
		assert.Nil(t, resp)
		assert.Nil(t, errs)
	})

	t.Run("500_server_error", func(t *testing.T) {
		resp, errs := a.MakeBids(nil, nil, &adapters.ResponseData{StatusCode: http.StatusInternalServerError})
		assert.Nil(t, resp)
		assert.Len(t, errs, 1)
	})

	t.Run("valid_banner_bid", func(t *testing.T) {
		resp, errs := a.MakeBids(
			&openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "imp-1"}}},
			nil,
			&adapters.ResponseData{
				StatusCode: http.StatusOK,
				Body:       []byte(`{"cur":"USD","seatbid":[{"bid":[{"id":"bid-1","impid":"imp-1","price":1.5,"adm":"<div>ad</div>","mtype":1}]}]}`),
			},
		)
		assert.Nil(t, errs)
		assert.Len(t, resp.Bids, 1)
		assert.Equal(t, openrtb_ext.BidTypeBanner, resp.Bids[0].BidType)
		assert.Equal(t, "USD", resp.Currency)
	})

	t.Run("unsupported_mtype_skipped", func(t *testing.T) {
		resp, errs := a.MakeBids(
			&openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "imp-1"}}},
			nil,
			&adapters.ResponseData{
				StatusCode: http.StatusOK,
				Body:       []byte(`{"seatbid":[{"bid":[{"id":"bid-1","impid":"imp-1","price":1.0,"mtype":99}]}]}`),
			},
		)
		assert.Len(t, errs, 1)
		assert.Empty(t, resp.Bids)
	})

	t.Run("native_unwrap_and_macro_resolution", func(t *testing.T) {
		nativeAdm := `{"native":{"link":{"url":"https://example.com/click"},"assets":[{"id":1,"title":{"text":"Title"}}],"imptrackers":["https://example.com/imp?p=${AUCTION_PRICE}"]}}`
		body := `{"seatbid":[{"bid":[{"id":"bid-1","impid":"imp-1","price":3.5,"adm":` + mustMarshalString(nativeAdm) + `,"mtype":4}]}]}`

		resp, errs := a.MakeBids(
			&openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "imp-1"}}},
			nil,
			&adapters.ResponseData{StatusCode: http.StatusOK, Body: []byte(body)},
		)
		assert.Nil(t, errs)
		assert.Len(t, resp.Bids, 1)
		assert.Equal(t, openrtb_ext.BidTypeNative, resp.Bids[0].BidType)
		assert.NotContains(t, resp.Bids[0].Bid.AdM, `"native"`)
		assert.NotContains(t, resp.Bids[0].Bid.AdM, "${AUCTION_PRICE}")
	})

	t.Run("invalid_response_body", func(t *testing.T) {
		resp, errs := a.MakeBids(
			&openrtb2.BidRequest{},
			nil,
			&adapters.ResponseData{StatusCode: http.StatusOK, Body: []byte(`not json`)},
		)
		assert.Nil(t, resp)
		assert.Len(t, errs, 1)
	})
}

func mustMarshalString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
