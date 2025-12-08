package mile

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
	"github.com/prebid/prebid-server/v3/config"
	metricsConf "github.com/prebid/prebid-server/v3/metrics/config"
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

func TestHandleSuccessInvokesAuction(t *testing.T) {
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

	var auctionBody []byte
	auctionCalled := false
	auction := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		auctionCalled = true
		var err error
		auctionBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read auction body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}

	cfg := &config.Configuration{
		MaxRequestSize: 512 * 1024,
		Mile: config.Mile{
			MaxRequestSize: 512 * 1024,
		},
	}

	handler, _, err := NewHandler(cfg, store, auction, &metricsConf.NilMetricsEngine{}, Hooks{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := []byte(`{"siteId":"FKKJK","publisherId":"12345","placementId":"p1","customData":[{"targeting":{"k":"v"}}]}`)
	req := httptest.NewRequest(http.MethodPost, "/mile/v1/request", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler(rec, req, nil)

	if status := rec.Code; status != http.StatusOK {
		t.Fatalf("expected 200 got %d", status)
	}
	if !auctionCalled {
		t.Fatalf("expected auction to be invoked")
	}

	var ortb map[string]any
	if err := json.Unmarshal(auctionBody, &ortb); err != nil {
		t.Fatalf("auction body not JSON: %v", err)
	}
	if ortb["site"] == nil {
		t.Fatalf("expected site in auction body")
	}
}

func TestHandleSiteNotFound(t *testing.T) {
	store := &mockStore{sites: map[string]*SiteConfig{}}
	auction := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.WriteHeader(http.StatusOK)
	}
	cfg := &config.Configuration{
		MaxRequestSize: 512 * 1024,
		Mile:           config.Mile{MaxRequestSize: 512 * 1024},
	}
	handler, _, err := NewHandler(cfg, store, auction, &metricsConf.NilMetricsEngine{}, Hooks{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := []byte(`{"siteId":"missing","publisherId":"12345","placementId":"p1"}`)
	req := httptest.NewRequest(http.MethodPost, "/mile/v1/request", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler(rec, req, nil)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleHooksApplied(t *testing.T) {
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

	auction := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}

	cfg := &config.Configuration{
		MaxRequestSize: 512 * 1024,
		Mile:           config.Mile{MaxRequestSize: 512 * 1024},
	}

	beforeCalled := false
	afterCalled := false
	hooks := Hooks{
		Before: func(ctx context.Context, req MileRequest, site *SiteConfig, ortb *openrtb2.BidRequest) (*openrtb2.BidRequest, error) {
			beforeCalled = true
			ortb.Test = int8(1)
			return ortb, nil
		},
		After: func(ctx context.Context, req MileRequest, site *SiteConfig, status int, body []byte) ([]byte, int, error) {
			afterCalled = true
			return []byte(`{"status":"hooked"}`), status, nil
		},
	}

	handler, _, err := NewHandler(cfg, store, auction, &metricsConf.NilMetricsEngine{}, hooks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := []byte(`{"siteId":"FKKJK","publisherId":"12345","placementId":"p1"}`)
	req := httptest.NewRequest(http.MethodPost, "/mile/v1/request", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler(rec, req, nil)

	if !beforeCalled || !afterCalled {
		t.Fatalf("expected hooks to be called")
	}
	if got := rec.Body.String(); got != `{"status":"hooked"}` {
		t.Fatalf("unexpected response %s", got)
	}
}
