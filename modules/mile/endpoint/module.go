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

	var mileReq MileRequest
	if err := json.Unmarshal(reqBody, &mileReq); err != nil {
		m.onException(ctx, mileReq, err)
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	mileReq.Raw = reqBody

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

	// Redis lookup
	siteCfg, err := m.store.Get(ctx, mileReq.SiteID, mileReq.PlacementID)
	if err != nil {
		switch err {
		case ErrSiteNotFound:
			m.onException(ctx, mileReq, err)
			writeError(w, http.StatusNotFound, fmt.Sprintf("site not found: %s", mileReq.SiteID))
		default:
			m.onException(ctx, mileReq, err)
			writeError(w, http.StatusBadGateway, "failed to load site configuration")
		}
		return
	}

	// Build OpenRTB request
	ortbReq, err := buildOpenRTBRequest(mileReq, siteCfg)
	if err != nil {
		m.onException(ctx, mileReq, err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Before hook
	ortbReq, err = m.hooks.applyBefore(ctx, mileReq, siteCfg, ortbReq)
	if err != nil {
		m.onException(ctx, mileReq, err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Marshal for auction
	auctionBody, err := json.Marshal(ortbReq)
	if err != nil {
		m.onException(ctx, mileReq, err)
		writeError(w, http.StatusInternalServerError, "failed to encode auction request")
		return
	}
	glog.V(3).Infof("mile: auction request site=%s placement=%s: %s", mileReq.SiteID, mileReq.PlacementID, string(auctionBody))

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
		glog.Warningf("mile: failed to read auction response: %v", readErr)
	}

	// After hook
	respBody, status, afterErr := m.hooks.applyAfter(ctx, mileReq, siteCfg, resp.StatusCode, respBody)
	if afterErr != nil {
		glog.Warningf("mile after hook failed: %v", afterErr)
	}

	// Transform auction response into bidder-style response
	var mileRespBytes []byte
	if resp.StatusCode >= http.StatusBadRequest {
		// Propagate error body/status as-is
		mileRespBytes = respBody
	} else {
		mileRespBytes = m.toMileResponse(respBody)
		status = http.StatusOK
	}

	// Log request duration
	glog.V(2).Infof("mile: request completed in %v for site=%s placement=%s", time.Since(start), mileReq.SiteID, mileReq.PlacementID)

	// Return response
	copyHeaders(w, resp.Header)
	w.WriteHeader(status)
	if _, err := w.Write(mileRespBytes); err != nil {
		glog.Warningf("mile write response failed: %v", err)
	}
}

// toMileResponse converts the auction response JSON into the MileResponse format.
func (m *Module) toMileResponse(body []byte) []byte {
	if len(body) == 0 {
		empty := MileResponse{Bids: []MileBid{}}
		raw, _ := json.Marshal(empty)
		return raw
	}

	var br openrtb2.BidResponse
	if err := json.Unmarshal(body, &br); err != nil {
		// If parsing fails, return empty bids to avoid surfacing raw errors to the adapter
		glog.Warningf("mile: failed to parse auction response: %v", err)
		empty := MileResponse{Bids: []MileBid{}}
		raw, _ := json.Marshal(empty)
		return raw
	}

	resp := transformToMileResponse(&br)
	raw, err := json.Marshal(resp)
	if err != nil {
		glog.Warningf("mile: failed to marshal mile response: %v", err)
		empty := MileResponse{Bids: []MileBid{}}
		raw, _ := json.Marshal(empty)
		return raw
	}
	return raw
}

func (m *Module) onException(ctx context.Context, req MileRequest, err error) {
	if m.hooks.OnException != nil && err != nil {
		m.hooks.OnException(ctx, req, err)
	}
	glog.Warningf("mile: exception for site=%s placement=%s: %v", req.SiteID, req.PlacementID, err)
}

func validateRequest(req MileRequest) error {
	switch {
	case req.SiteID == "":
		return fmt.Errorf("siteId is required")
	case req.PublisherID == "":
		return fmt.Errorf("publisherId is required")
	case req.PlacementID == "":
		return fmt.Errorf("placementId is required")
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

func copyHeaders(dst http.ResponseWriter, src http.Header) {
	for k, v := range src {
		for _, val := range v {
			dst.Header().Add(k, val)
		}
	}
}
