package endpoint

import (
	"bytes"
	"context"
	"encoding/json"
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

func (m *mockStore) Get(ctx context.Context, siteID string) (*SiteConfig, error) {
	if m.err != nil {
		return nil, m.err
	}
	site, ok := m.sites[siteID]
	if !ok {
		return nil, ErrSiteNotFound
	}
	return site, nil
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
				Bidders:     []string{"appnexus"},
				Placements: map[string]PlacementConfig{
					"p1": {PlacementID: "p1", Sizes: [][]int{{300, 250}}, Floor: 0.1},
				},
			},
		},
	}

	auctionCalled := false
	auctionHandler := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		auctionCalled = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"cur":"USD","seatbid":[{"seat":"appnexus","bid":[{"impid":"p1","price":0.1,"adm":"<ad>","w":300,"h":250,"crid":"cr"}]}]}`))
	}

	m := &Module{
		enabled:        true,
		config:         &Config{Endpoint: "/mile/v1/request", MaxRequestSize: 512 * 1024},
		store:          store,
		requestTimeout: 0,
		maxBody:        512 * 1024,
	}
	m.SetAuctionHandler(auctionHandler)

	body := []byte(`{"siteId":"FKKJK","publisherId":"12345","placementId":"p1","customData":[{"targeting":{"k":"v"}}]}`)
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

	body := []byte(`{"siteId":"missing","publisherId":"12345","placementId":"p1"}`)
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
				Bidders:     []string{"appnexus"},
				Placements: map[string]PlacementConfig{
					"p1": {PlacementID: "p1", Sizes: [][]int{{300, 250}}, Floor: 0.1},
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
				return []byte(`{"cur":"USD","seatbid":[{"seat":"override","bid":[{"impid":"p1","price":0.5,"adm":"<hooked>","w":320,"h":50,"crid":"cr-hook"}]}]}`), status, nil
			},
		},
	}
	m.SetAuctionHandler(auctionHandler)

	body := []byte(`{"siteId":"FKKJK","publisherId":"12345","placementId":"p1"}`)
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
					{ID: "b2", ImpID: "imp2", Price: 0.5, W: 320, H: 50, AdM: "<a2>", CrID: "cr2"},
				},
			},
			{
				Seat: "bidderB",
				Bid: []openrtb2.Bid{
					{ID: "b3", ImpID: "imp1", Price: 1.5, W: 300, H: 250, AdM: "<b1>", CrID: "cr3"},
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
					{ID: "b1", ImpID: "imp1", Price: 1.0, W: 0, H: 0, AdM: "<ad>", CrID: "cr"},
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

	body := []byte(`{"siteId":"FKKJK","publisherId":"12345","placementId":"p1"}`)
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
	req := httptest.NewRequest(http.MethodPost, "/mile/v1/request", bytes.NewReader([]byte(`{"publisherId":"123","placementId":"p1"}`)))
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
				Bidders:     []string{"appnexus"},
				Placements: map[string]PlacementConfig{
					"p1": {PlacementID: "p1", Sizes: [][]int{{300, 250}}},
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
	body := []byte(`{"siteId":"FKKJK","publisherId":"12345","placementId":"p1"}`)
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
