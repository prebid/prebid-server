package endpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
)

// Builder creates a new Mile endpoint module instance.
// This module provides a custom HTTP endpoint for Mile adapter requests.
func Builder(rawConfig json.RawMessage, deps moduledeps.ModuleDeps) (interface{}, error) {
	config, err := parseConfig(rawConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse mile endpoint config: %w", err)
	}

	if !config.Enabled {
		return &Module{enabled: false}, nil
	}

	store, err := NewRedisSiteStore(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create redis store: %w", err)
	}

	return &Module{
		enabled:        true,
		config:         config,
		store:          store,
		deps:           deps,
		requestTimeout: time.Duration(config.RequestTimeoutMs) * time.Millisecond,
		maxBody:        config.MaxRequestSize,
	}, nil
}

// Module implements the Mile endpoint as a PBS module.
type Module struct {
	enabled        bool
	config         *Config
	store          SiteStore
	deps           moduledeps.ModuleDeps
	hooks          Hooks
	auctionHandler httprouter.Handle
	requestTimeout time.Duration
	maxBody        int64
}

// EndpointInfo describes an HTTP endpoint provided by this module.
type EndpointInfo struct {
	Method  string
	Path    string
	Handler httprouter.Handle
}

// SetAuctionHandler allows the router to inject the auction handler.
// This must be called before the endpoint is used.
func (m *Module) SetAuctionHandler(handler httprouter.Handle) {
	m.auctionHandler = handler
}

// SetHooks allows setting custom lifecycle hooks.
func (m *Module) SetHooks(hooks Hooks) {
	m.hooks = hooks
}

// GetEndpoints returns the HTTP endpoints provided by this module.
func (m *Module) GetEndpoints() []EndpointInfo {
	if !m.enabled {
		return nil
	}

	path := m.config.Endpoint
	if path == "" {
		path = "/mile/v1/request"
	}

	return []EndpointInfo{
		{
			Method:  "POST",
			Path:    path,
			Handler: m.Handle,
		},
	}
}

// Shutdown releases resources held by the module.
func (m *Module) Shutdown() error {
	if m.store != nil {
		return m.store.Close()
	}
	return nil
}

// HandleEntrypointHook implements a no-op entrypoint hook so the module can be registered
// in the hook repository even though its primary purpose is providing an HTTP endpoint.
func (m *Module) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, payload hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	return hookstage.HookResult[hookstage.EntrypointPayload]{}, nil
}

// Handle processes incoming Mile adapter requests.
func (m *Module) Handle(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if m.auctionHandler == nil {
		writeError(w, http.StatusServiceUnavailable, "auction handler not configured")
		return
	}

	start := time.Now()
	ctx := r.Context()
	if m.requestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, m.requestTimeout)
		defer cancel()
	}

	reqBody, err := io.ReadAll(io.LimitReader(r.Body, m.maxBody))
	if err != nil {
		m.onException(ctx, MileRequest{}, err)
		writeError(w, http.StatusBadRequest, "unable to read request body")
		return
	}
	if len(reqBody) == 0 {
		m.onException(ctx, MileRequest{}, fmt.Errorf("empty body"))
		writeError(w, http.StatusBadRequest, "empty request body")
		return
	}

	// Parse OpenRTB request only
	var ortbReq openrtb2.BidRequest
	if err := json.Unmarshal(reqBody, &ortbReq); err != nil {
		m.onException(ctx, MileRequest{}, fmt.Errorf("invalid JSON payload: %w", err))
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	if ortbReq.Site == nil {
		m.onException(ctx, MileRequest{}, fmt.Errorf("missing site object in OpenRTB request"))
		writeError(w, http.StatusBadRequest, "missing site object in OpenRTB request")
		return
	}
	if ortbReq.Device == nil {
		m.onException(ctx, MileRequest{}, fmt.Errorf("missing device object in OpenRTB request"))
		writeError(w, http.StatusBadRequest, "missing device object in OpenRTB request")
		return
	}
	if ortbReq.User == nil {
		m.onException(ctx, MileRequest{}, fmt.Errorf("missing user object in OpenRTB request"))
		writeError(w, http.StatusBadRequest, "missing user object in OpenRTB request")
		return
	}
	mileReq := convertORTBToMile(&ortbReq)
	mileReq.Raw = reqBody

	var debugRequested bool
	if mileReq.BaseORTB != nil && len(mileReq.BaseORTB.Ext) > 0 {
		var ext map[string]json.RawMessage
		if err := json.Unmarshal(mileReq.BaseORTB.Ext, &ext); err == nil {
			if prebidRaw, ok := ext["prebid"]; ok {
				var prebid struct {
					Debug bool `json:"debug"`
				}
				_ = json.Unmarshal(prebidRaw, &prebid)
				debugRequested = prebid.Debug
			}
		}
	}

	// Auth token validation
	if m.config.AuthToken != "" {
		if token := r.Header.Get("X-Mile-Token"); token != m.config.AuthToken {
			m.onException(ctx, mileReq, fmt.Errorf("unauthorized"))
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
	}

	// Validate required fields
	if err := validateRequest(mileReq); err != nil {
		m.onException(ctx, mileReq, err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Custom validation hook
	if err := m.hooks.applyValidate(ctx, mileReq); err != nil {
		m.onException(ctx, mileReq, err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Redis lookup for all placements
	siteConfigs, err := m.store.GetMulti(ctx, mileReq.SiteID, mileReq.PlacementIDs)
	if err != nil {
		switch err {
		case ErrSiteNotFound:
			m.onException(ctx, mileReq, err)
			writeError(w, http.StatusNotFound, fmt.Sprintf("site not found: %s", mileReq.SiteID))
		default:
			m.onException(ctx, mileReq, err)
			writeError(w, http.StatusBadGateway, fmt.Sprintf("failed to load site configuration: %v", err))
		}
		return
	}

	// Process each placement in parallel
	type auctionResult struct {
		placementID string
		bids        []MileBid
		ext         json.RawMessage
		err         error
	}

	results := make(chan auctionResult, len(mileReq.PlacementIDs))
	var wg sync.WaitGroup

	for _, placementID := range mileReq.PlacementIDs {
		siteCfg, ok := siteConfigs[placementID]
		if !ok {
			glog.Warningf("mile: no config found for placement %s, skipping", placementID)
			continue
		}

		wg.Add(1)
		go func(placementID string, siteCfg *SiteConfig) {
			defer wg.Done()
			bids, ext, err := m.processPlacement(ctx, r, mileReq, placementID, siteCfg)
			results <- auctionResult{placementID: placementID, bids: bids, ext: ext, err: err}
		}(placementID, siteCfg)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Aggregate all bids
	var allBids []MileBid
	var allExt json.RawMessage
	for result := range results {
		if result.err != nil {
			glog.Warningf("mile: auction failed for placement %s: %v", result.placementID, result.err)
			continue
		}
		allBids = append(allBids, result.bids...)
		if allExt == nil {
			allExt = result.ext
		}
	}

	// Build response
	mileResp := MileResponse{Bids: allBids, Ext: allExt}
	if mileResp.Bids == nil {
		mileResp.Bids = []MileBid{}
	}

	if !debugRequested && len(mileResp.Ext) > 0 {
		var ext map[string]json.RawMessage
		if err := json.Unmarshal(mileResp.Ext, &ext); err == nil {
			if _, ok := ext["debug"]; ok {
				delete(ext, "debug")
				mileResp.Ext, _ = json.Marshal(ext)
			}
		}
	}

	mileRespBytes, err := json.Marshal(mileResp)
	if err != nil {
		m.onException(ctx, mileReq, err)
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}

	// Log request duration
	glog.V(2).Infof("mile: request completed in %v for site=%s placements=%s", time.Since(start), mileReq.SiteID, strings.Join(mileReq.PlacementIDs, ","))

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(mileRespBytes); err != nil {
		glog.Warningf("mile write response failed: %v", err)
	}
}

// processPlacement runs an auction for a single placement and returns the bids.
func (m *Module) processPlacement(ctx context.Context, r *http.Request, mileReq MileRequest, placementID string, siteCfg *SiteConfig) ([]MileBid, json.RawMessage, error) {
	// Build OpenRTB request
	ortbReq, err := buildOpenRTBRequest(mileReq, placementID, siteCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("build request failed: %w", err)
	}

	// Before hook
	ortbReq, err = m.hooks.applyBefore(ctx, mileReq, siteCfg, ortbReq)
	if err != nil {
		return nil, nil, fmt.Errorf("before hook failed: %w", err)
	}

	// Marshal for auction
	auctionBody, err := json.Marshal(ortbReq)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal request failed: %w", err)
	}
	glog.V(3).Infof("mile: auction request site=%s placement=%s: %s", mileReq.SiteID, placementID, string(auctionBody))

	// Call auction in-process
	auctionReq := r.Clone(ctx)
	auctionReq.Method = http.MethodPost
	auctionReq.URL = &url.URL{Path: "/openrtb2/auction"}
	auctionReq.Body = io.NopCloser(bytes.NewReader(auctionBody))
	auctionReq.Header = cloneHeaders(r.Header)
	auctionReq.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	m.auctionHandler(recorder, auctionReq, nil)

	resp := recorder.Result()
	defer resp.Body.Close()

	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, nil, fmt.Errorf("read response failed: %w", readErr)
	}

	// After hook
	respBody, _, afterErr := m.hooks.applyAfter(ctx, mileReq, siteCfg, resp.StatusCode, respBody)
	if afterErr != nil {
		glog.Warningf("mile after hook failed for placement %s: %v", placementID, afterErr)
	}

	// Check for error status
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, nil, fmt.Errorf("auction returned status %d", resp.StatusCode)
	}

	// Parse and transform response
	if len(respBody) == 0 {
		return []MileBid{}, nil, nil
	}

	var br openrtb2.BidResponse
	if err := json.Unmarshal(respBody, &br); err != nil {
		glog.Warningf("mile: failed to parse auction response for placement %s: %v", placementID, err)
		return []MileBid{}, nil, nil
	}

	mileResp := transformToMileResponse(&br)
	return mileResp.Bids, mileResp.Ext, nil
}

func (m *Module) onException(ctx context.Context, req MileRequest, err error) {
	if m.hooks.OnException != nil && err != nil {
		m.hooks.OnException(ctx, req, err)
	}
	glog.Warningf("mile: exception for site=%s placements=%v: %v", req.SiteID, req.PlacementIDs, err)
}

func validateRequest(req MileRequest) error {
	switch {
	case req.SiteID == "":
		return fmt.Errorf("siteId is required")
	case len(req.PlacementIDs) == 0:
		return fmt.Errorf("placementIds is required")
	default:
		return nil
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": message})
}

func cloneHeaders(src http.Header) http.Header {
	dst := make(http.Header, len(src))
	for k, v := range src {
		values := make([]string, len(v))
		copy(values, v)
		dst[k] = values
	}
	return dst
}

func convertORTBToMile(ortb *openrtb2.BidRequest) MileRequest {
	if ortb == nil {
		return MileRequest{}
	}
	mileReq := MileRequest{
		BaseORTB: ortb,
		ImpIDMap: make(map[string]string),
	}
	if ortb.Site != nil {
		mileReq.SiteID = ortb.Site.ID
		if ortb.Site.Publisher != nil {
			mileReq.PublisherID = ortb.Site.Publisher.ID
		}
	}
	for _, imp := range ortb.Imp {
		var pID string
		if len(imp.Ext) > 0 {
			var ext struct {
				PlacementID string `json:"placementId"`
			}
			if err := json.Unmarshal(imp.Ext, &ext); err == nil {
				pID = ext.PlacementID
			}
		}
		if pID == "" {
			pID = imp.TagID
		}
		if pID != "" {
			mileReq.PlacementIDs = append(mileReq.PlacementIDs, pID)
			mileReq.ImpIDMap[pID] = imp.ID
		}
	}
	return mileReq
}
