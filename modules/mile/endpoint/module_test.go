package endpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
	"github.com/stretchr/testify/require"
)

type mockStore struct {
	sites map[string]*SiteConfig
	err   error
}

func (m *mockStore) Get(ctx context.Context, siteID, _ string) (*SiteConfig, error) {
	if m.err != nil {
		return nil, m.err
	}
	site, ok := m.sites[siteID]
	if !ok {
		return nil, ErrSiteNotFound
	}
	return site, nil
}

func (m *mockStore) GetMulti(ctx context.Context, siteID string, placementIDs []string) (map[string]*SiteConfig, error) {
	if m.err != nil {
		return nil, m.err
	}
	site, ok := m.sites[siteID]
	if !ok {
		return nil, ErrSiteNotFound
	}
	// Return the same site config for all placements
	result := make(map[string]*SiteConfig, len(placementIDs))
	for _, placementID := range placementIDs {
		result[placementID] = site
	}
	return result, nil
}

func (m *mockStore) Close() error { return nil }

func TestModuleBuilder(t *testing.T) {
	cfg := json.RawMessage(`{
		"enabled": true,
		"endpoint": "/mile/v1/request",
		"request_timeout_ms": 500,
		"redis_timeout_ms": 200,
		"max_request_size": 524288,
		"redis": {
			"addr": "localhost:6379"
		}
	}`)

	module, err := Builder(cfg, moduledeps.ModuleDeps{})
	if err != nil {
		t.Fatalf("Builder failed: %v", err)
	}

	m, ok := module.(*Module)
	if !ok {
		t.Fatal("expected *Module type")
	}

	if !m.enabled {
		t.Error("expected module to be enabled")
	}

	endpoints := m.GetEndpoints()
	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}

	if endpoints[0].Path != "/mile/v1/request" {
		t.Errorf("expected path /mile/v1/request, got %s", endpoints[0].Path)
	}

	if endpoints[0].Method != "POST" {
		t.Errorf("expected method POST, got %s", endpoints[0].Method)
	}
}

func TestModuleBuilderDisabled(t *testing.T) {
	cfg := json.RawMessage(`{"enabled": false}`)

	module, err := Builder(cfg, moduledeps.ModuleDeps{})
	if err != nil {
		t.Fatalf("Builder failed: %v", err)
	}

	m, ok := module.(*Module)
	if !ok {
		t.Fatal("expected *Module type")
	}

	if m.enabled {
		t.Error("expected module to be disabled")
	}

	endpoints := m.GetEndpoints()
	if len(endpoints) != 0 {
		t.Fatalf("expected 0 endpoints for disabled module, got %d", len(endpoints))
	}
}

func TestModuleHandle(t *testing.T) {
	store := &mockStore{
		sites: map[string]*SiteConfig{
			"FKKJK": {
				SiteID:      "FKKJK",
				PublisherID: "12345",
				Placement: PlacementConfig{
					Sizes: [][]int{{300, 250}},
					Floor: 0.1,
					Bidders: []PlacementBidder{
						{Bidder: "appnexus", Params: json.RawMessage(`{"placementId": "123"}`)},
					},
				},
			},
		},
	}

	auctionCalled := false
	auctionHandler := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		auctionCalled = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"cur":"USD","seatbid":[{"seat":"appnexus","bid":[{"impid":"p1","price":0.1,"adm":"<ad>","w":300,"h":250,"crid":"cr","ext":{"prebid":{"targeting":{"hb_bidder":"appnexus"}}}}]}]}`))
	}

	m := &Module{
		enabled:        true,
		config:         &Config{Endpoint: "/mile/v1/request", MaxRequestSize: 512 * 1024},
		store:          store,
		requestTimeout: 0,
		maxBody:        512 * 1024,
	}
	m.SetAuctionHandler(auctionHandler)

	body := []byte(`{"siteId":"FKKJK","publisherId":"12345","placementIds":["p1"],"customData":[{"targeting":{"k":"v"}}]}`)
	req := httptest.NewRequest(http.MethodPost, "/mile/v1/request", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	m.Handle(rec, req, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if !auctionCalled {
		t.Fatal("expected auction to be called")
	}

	var mileResp MileResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &mileResp))
	require.Len(t, mileResp.Bids, 1)
	require.Equal(t, "p1", mileResp.Bids[0].RequestID)
	require.InDelta(t, 0.1, mileResp.Bids[0].CPM, 1e-6)
	require.Equal(t, "appnexus", mileResp.Bids[0].Bidder)
}

func TestModuleHandleSiteNotFound(t *testing.T) {
	store := &mockStore{sites: map[string]*SiteConfig{}}
	auctionHandler := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.WriteHeader(http.StatusOK)
	}

	m := &Module{
		enabled:        true,
		config:         &Config{Endpoint: "/mile/v1/request", MaxRequestSize: 512 * 1024},
		store:          store,
		requestTimeout: 0,
		maxBody:        512 * 1024,
	}
	m.SetAuctionHandler(auctionHandler)

	body := []byte(`{"siteId":"missing","publisherId":"12345","placementIds":["p1"]}`)
	req := httptest.NewRequest(http.MethodPost, "/mile/v1/request", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	m.Handle(rec, req, nil)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestModuleHandleHooks(t *testing.T) {
	store := &mockStore{
		sites: map[string]*SiteConfig{
			"FKKJK": {
				SiteID:      "FKKJK",
				PublisherID: "12345",
				Placement: PlacementConfig{
					Sizes: [][]int{{300, 250}},
					Floor: 0.1,
					Bidders: []PlacementBidder{
						{Bidder: "appnexus", Params: json.RawMessage(`{}`)},
					},
				},
			},
		},
	}

	auctionHandler := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"cur":"USD","seatbid":[{"seat":"appnexus","bid":[{"impid":"p1","price":0.2,"adm":"<ad2>","w":300,"h":250,"crid":"cr2"}]}]}`))
	}

	beforeCalled := false
	afterCalled := false

	m := &Module{
		enabled:        true,
		config:         &Config{Endpoint: "/mile/v1/request", MaxRequestSize: 512 * 1024},
		store:          store,
		requestTimeout: 0,
		maxBody:        512 * 1024,
		hooks: Hooks{
			Before: func(ctx context.Context, req MileRequest, site *SiteConfig, ortb *openrtb2.BidRequest) (*openrtb2.BidRequest, error) {
				beforeCalled = true
				ortb.Test = int8(1)
				return ortb, nil
			},
			After: func(ctx context.Context, req MileRequest, site *SiteConfig, status int, body []byte) ([]byte, int, error) {
				afterCalled = true
				// Override auction response with a different bid
				return []byte(`{"cur":"USD","seatbid":[{"seat":"override","bid":[{"impid":"p1","price":0.5,"adm":"<hooked>","w":320,"h":50,"crid":"cr-hook","ext":{"prebid":{"targeting":{"hb_bidder":"override"}}}}]}]}`), status, nil
			},
		},
	}
	m.SetAuctionHandler(auctionHandler)

	body := []byte(`{"siteId":"FKKJK","publisherId":"12345","placementIds":["p1"]}`)
	req := httptest.NewRequest(http.MethodPost, "/mile/v1/request", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	m.Handle(rec, req, nil)

	if !beforeCalled || !afterCalled {
		t.Fatal("expected hooks to be called")
	}

	var mileResp MileResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &mileResp))
	require.Len(t, mileResp.Bids, 1)
	require.Equal(t, "p1", mileResp.Bids[0].RequestID)
	require.InDelta(t, 0.5, mileResp.Bids[0].CPM, 1e-6)
	require.Equal(t, "override", mileResp.Bids[0].Bidder)
}

func TestTransformToMileResponse_NoBids(t *testing.T) {
	resp := transformToMileResponse(nil)
	require.Len(t, resp.Bids, 0)

	resp = transformToMileResponse(&openrtb2.BidResponse{SeatBid: []openrtb2.SeatBid{}})
	require.Len(t, resp.Bids, 0)
}

func TestTransformToMileResponse_PicksHighestPerImp(t *testing.T) {
	br := &openrtb2.BidResponse{
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidderA",
				Bid: []openrtb2.Bid{
					{ID: "b1", ImpID: "imp1", Price: 1.2, W: 300, H: 250, AdM: "<a1>", CrID: "cr1"},
					{ID: "b2", ImpID: "imp2", Price: 0.5, W: 320, H: 50, AdM: "<a2>", CrID: "cr2", Ext: json.RawMessage(`{"prebid":{"targeting":{"hb_bidder":"bidderA"}}}`)},
				},
			},
			{
				Seat: "bidderB",
				Bid: []openrtb2.Bid{
					{ID: "b3", ImpID: "imp1", Price: 1.5, W: 300, H: 250, AdM: "<b1>", CrID: "cr3", Ext: json.RawMessage(`{"prebid":{"targeting":{"hb_bidder":"bidderB"}}}`)},
				},
			},
		},
	}

	resp := transformToMileResponse(br)
	require.Len(t, resp.Bids, 2)

	// imp1 should pick bidderB with higher CPM
	var imp1 MileBid
	var imp2 MileBid
	for _, b := range resp.Bids {
		if b.RequestID == "imp1" {
			imp1 = b
		}
		if b.RequestID == "imp2" {
			imp2 = b
		}
	}

	require.Equal(t, "bidderB", imp1.Bidder)
	require.InDelta(t, 1.5, imp1.CPM, 1e-6)
	require.Equal(t, "<b1>", imp1.Ad)
	require.Equal(t, int64(300), imp1.Width)
	require.Equal(t, int64(250), imp1.Height)

	require.Equal(t, "bidderA", imp2.Bidder)
	require.InDelta(t, 0.5, imp2.CPM, 1e-6)
	require.Equal(t, "<a2>", imp2.Ad)
}

func TestTransformToMileResponse_InferBannerDefault(t *testing.T) {
	br := &openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder",
				Bid: []openrtb2.Bid{
					{ID: "b1", ImpID: "imp1", Price: 1.0, W: 0, H: 0, AdM: "<ad>", CrID: "cr", Ext: json.RawMessage(`{"prebid":{"targeting":{"hb_bidder":"bidder"}}}`)},
				},
			},
		},
	}
	resp := transformToMileResponse(br)
	require.Len(t, resp.Bids, 1)
	require.Equal(t, "banner", resp.Bids[0].MediaType)
}

func TestModuleHandleNoAuctionHandler(t *testing.T) {
	m := &Module{
		enabled:        true,
		config:         &Config{Endpoint: "/mile/v1/request"},
		requestTimeout: 0,
		maxBody:        512 * 1024,
	}
	// Note: SetAuctionHandler not called

	body := []byte(`{"siteId":"FKKJK","publisherId":"12345","placementIds":["p1"]}`)
	req := httptest.NewRequest(http.MethodPost, "/mile/v1/request", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	m.Handle(rec, req, nil)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestModuleHandleEmptyBody(t *testing.T) {
	m := &Module{
		enabled:        true,
		config:         &Config{Endpoint: "/mile/v1/request"},
		requestTimeout: 0,
		maxBody:        512 * 1024,
		auctionHandler: func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {},
	}

	req := httptest.NewRequest(http.MethodPost, "/mile/v1/request", bytes.NewReader([]byte{}))
	rec := httptest.NewRecorder()

	m.Handle(rec, req, nil)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestModuleHandleInvalidJSON(t *testing.T) {
	m := &Module{
		enabled:        true,
		config:         &Config{Endpoint: "/mile/v1/request"},
		requestTimeout: 0,
		maxBody:        512 * 1024,
		auctionHandler: func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {},
	}

	req := httptest.NewRequest(http.MethodPost, "/mile/v1/request", bytes.NewReader([]byte(`{invalid`)))
	rec := httptest.NewRecorder()

	m.Handle(rec, req, nil)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestModuleHandleMissingRequiredFields(t *testing.T) {
	m := &Module{
		enabled:        true,
		config:         &Config{Endpoint: "/mile/v1/request"},
		requestTimeout: 0,
		maxBody:        512 * 1024,
		auctionHandler: func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {},
	}

	// Missing siteId
	req := httptest.NewRequest(http.MethodPost, "/mile/v1/request", bytes.NewReader([]byte(`{"publisherId":"123","placementIds":["p1"]}`)))
	rec := httptest.NewRecorder()

	m.Handle(rec, req, nil)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing siteId, got %d", rec.Code)
	}
}

func TestModuleHandleAuthToken(t *testing.T) {
	store := &mockStore{
		sites: map[string]*SiteConfig{
			"FKKJK": {
				SiteID:      "FKKJK",
				PublisherID: "12345",
				Placement: PlacementConfig{
					Sizes: [][]int{{300, 250}},
					Bidders: []PlacementBidder{
						{Bidder: "appnexus", Params: json.RawMessage(`{}`)},
					},
				},
			},
		},
	}

	m := &Module{
		enabled:        true,
		config:         &Config{Endpoint: "/mile/v1/request", AuthToken: "secret123", MaxRequestSize: 512 * 1024},
		store:          store,
		requestTimeout: 0,
		maxBody:        512 * 1024,
		auctionHandler: func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		},
	}

	// Without token - should fail
	body := []byte(`{"siteId":"FKKJK","publisherId":"12345","placementIds":["p1"]}`)
	req := httptest.NewRequest(http.MethodPost, "/mile/v1/request", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	m.Handle(rec, req, nil)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}

	// With correct token - should succeed
	req = httptest.NewRequest(http.MethodPost, "/mile/v1/request", bytes.NewReader(body))
	req.Header.Set("X-Mile-Token", "secret123")
	rec = httptest.NewRecorder()

	m.Handle(rec, req, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with correct token, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestModuleHandleMissingPublisherID(t *testing.T) {
	store := &mockStore{
		sites: map[string]*SiteConfig{
			"FKKJK": {
				SiteID:      "FKKJK",
				PublisherID: "12345",
				Placement: PlacementConfig{
					Sizes: [][]int{{300, 250}},
					Bidders: []PlacementBidder{
						{Bidder: "appnexus", Params: json.RawMessage(`{}`)},
					},
				},
			},
		},
	}

	m := &Module{
		enabled:        true,
		config:         &Config{Endpoint: "/mile/v1/request", MaxRequestSize: 512 * 1024},
		store:          store,
		requestTimeout: 0,
		maxBody:        512 * 1024,
		auctionHandler: func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		},
	}

	// Request without publisherId - should succeed now
	body := []byte(`{"siteId":"FKKJK","placementIds":["p1"]}`)
	req := httptest.NewRequest(http.MethodPost, "/mile/v1/request", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	m.Handle(rec, req, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 without publisherId, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestModuleHandleMultiplePlacements(t *testing.T) {
	store := &mockStore{
		sites: map[string]*SiteConfig{
			"FKKJK": {
				SiteID:      "FKKJK",
				PublisherID: "12345",
				Placement: PlacementConfig{
					Sizes: [][]int{{300, 250}},
					Floor: 0.1,
					Bidders: []PlacementBidder{
						{Bidder: "appnexus", Params: json.RawMessage(`{"placementId": "123"}`)},
					},
				},
			},
		},
	}

	auctionCount := 0
	auctionHandler := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		auctionCount++
		// Parse request to get placement ID for unique response
		var req openrtb2.BidRequest
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)
		impID := "unknown"
		if len(req.Imp) > 0 {
			impID = req.Imp[0].ID
		}
		w.WriteHeader(http.StatusOK)
		resp := `{"cur":"USD","seatbid":[{"seat":"appnexus","bid":[{"impid":"` + impID + `","price":0.1,"adm":"<ad>","w":300,"h":250,"crid":"cr","ext":{"prebid":{"targeting":{"hb_bidder":"appnexus"}}}}]}]}`
		_, _ = w.Write([]byte(resp))
	}

	m := &Module{
		enabled:        true,
		config:         &Config{Endpoint: "/mile/v1/request", MaxRequestSize: 512 * 1024},
		store:          store,
		requestTimeout: 0,
		maxBody:        512 * 1024,
	}
	m.SetAuctionHandler(auctionHandler)

	// Request with multiple placement IDs
	body := []byte(`{"siteId":"FKKJK","publisherId":"12345","placementIds":["p1","p2","p3"]}`)
	req := httptest.NewRequest(http.MethodPost, "/mile/v1/request", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	m.Handle(rec, req, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify auction was called for each placement
	if auctionCount != 3 {
		t.Fatalf("expected 3 auction calls, got %d", auctionCount)
	}

	var mileResp MileResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &mileResp))
	require.Len(t, mileResp.Bids, 3)

	// Verify we have bids for all placements
	placementIDs := make(map[string]bool)
	for _, bid := range mileResp.Bids {
		placementIDs[bid.RequestID] = true
	}
	require.True(t, placementIDs["p1"], "expected bid for p1")
	require.True(t, placementIDs["p2"], "expected bid for p2")
	require.True(t, placementIDs["p3"], "expected bid for p3")
}

func TestModuleHandleORTB(t *testing.T) {
	store := &mockStore{
		sites: map[string]*SiteConfig{
			"ViXOj3": {
				SiteID:      "ViXOj3",
				PublisherID: "590",
				Placement: PlacementConfig{
					Sizes: [][]int{{300, 250}},
					Floor: 0.1,
					Bidders: []PlacementBidder{
						{Bidder: "appnexus", Params: json.RawMessage(`{"placementId": "123"}`)},
					},
				},
			},
		},
	}

	auctionCalled := false
	var capturedReq openrtb2.BidRequest
	auctionHandler := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		auctionCalled = true
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedReq)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"cur":"USD","seatbid":[{"seat":"appnexus","bid":[{"impid":"22670","price":0.1,"adm":"<ad>","w":300,"h":250,"crid":"cr","ext":{"prebid":{"targeting":{"hb_bidder":"appnexus"}}}}]}]}`))
	}

	m := &Module{
		enabled:        true,
		config:         &Config{Endpoint: "/mile/v1/request", MaxRequestSize: 512 * 1024},
		store:          store,
		requestTimeout: 0,
		maxBody:        512 * 1024,
	}
	m.SetAuctionHandler(auctionHandler)

	body := []byte(`{"id":"ff74969e-094c-44bb-8cb4-99b4f8c72421","imp":[{"id":"588f527f-9aaf-483b-ac4e-d29ba763b531","tagid":"22670","secure":1,"banner":{"format":[{"w":300,"h":250}]},"ext":{"placementId":"22670"}}],"site":{"id":"ViXOj3","publisher":{"id":"590"}}}`)
	req := httptest.NewRequest(http.MethodPost, "/mile/v1/request", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	m.Handle(rec, req, nil)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, auctionCalled)
	require.Equal(t, "ff74969e-094c-44bb-8cb4-99b4f8c72421", capturedReq.ID)
	require.Equal(t, "ViXOj3", capturedReq.Site.ID)
	require.Equal(t, "590", capturedReq.Site.Publisher.ID)
	require.Len(t, capturedReq.Imp, 1)
	require.Equal(t, "22670", capturedReq.Imp[0].ID)

	var mileResp MileResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &mileResp))
	require.Len(t, mileResp.Bids, 1)
	require.Equal(t, "22670", mileResp.Bids[0].RequestID)
}

func TestConvertORTBToMile(t *testing.T) {
	raw := []byte(`{
    "id": "ff74969e-094c-44bb-8cb4-99b4f8c72421",
    "imp": [
        {
            "id": "588f527f-9aaf-483b-ac4e-d29ba763b531",
            "tagid": "22670",
            "secure": 1,
            "banner": {
                "format": [
                    {
                        "w": 300,
                        "h": 250
                    }
                ]
            },
            "ext": {
                "placementId": "22670"
            }
        }
    ],
    "site": {
        "id": "ViXOj3",
        "publisher": {
            "id": "590"
        }
    }
}`)
	var ortb openrtb2.BidRequest
	require.NoError(t, json.Unmarshal(raw, &ortb))
	require.NotNil(t, ortb.Site)
	require.Equal(t, "ViXOj3", ortb.Site.ID)

	mileReq := convertORTBToMile(&ortb)
	require.Equal(t, "ViXOj3", mileReq.SiteID)
	require.Equal(t, "590", mileReq.PublisherID)
	require.Equal(t, []string{"22670"}, mileReq.PlacementIDs)
}
