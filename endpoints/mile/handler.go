package mile

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
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/metrics"
)

// Handler wires the Mile endpoint into PBS.
type Handler struct {
	store          SiteStore
	auction        httprouter.Handle
	metrics        metrics.MetricsEngine
	cfg            config.Mile
	hooks          Hooks
	requestTimeout time.Duration
	maxBody        int64
}

// NewHandler constructs the Mile endpoint and a shutdown callback.
func NewHandler(cfg *config.Configuration, store SiteStore, auction httprouter.Handle, metricsEngine metrics.MetricsEngine, hooks Hooks) (httprouter.Handle, func(), error) {
	if cfg == nil {
		return nil, nil, fmt.Errorf("config required")
	}
	if store == nil {
		return nil, nil, fmt.Errorf("site store required")
	}
	if auction == nil {
		return nil, nil, fmt.Errorf("auction handler required")
	}

	maxBody := cfg.Mile.MaxRequestSize
	if maxBody == 0 {
		maxBody = cfg.MaxRequestSize
	}
	if maxBody == 0 {
		maxBody = 512 * 1024
	}

	h := &Handler{
		store:          store,
		auction:        auction,
		metrics:        metricsEngine,
		cfg:            cfg.Mile,
		hooks:          hooks,
		requestTimeout: time.Duration(cfg.Mile.RequestTimeoutMs) * time.Millisecond,
		maxBody:        maxBody,
	}

	shutdown := func() {}
	if store != nil {
		shutdown = func() {
			_ = store.Close()
		}
	}

	return h.handle, shutdown, nil
}

func (h *Handler) handle(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	start := time.Now()
	ctx := r.Context()
	if h.requestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.requestTimeout)
		defer cancel()
	}

	labels := metrics.Labels{
		RType:         metrics.ReqTypeORTB2Web,
		RequestStatus: metrics.RequestStatusOK,
	}

	defer func() {
		if h.metrics != nil {
			h.metrics.RecordRequest(labels)
			h.metrics.RecordRequestTime(labels, time.Since(start))
		}
	}()

	reqBody, err := io.ReadAll(io.LimitReader(r.Body, h.maxBody))
	if err != nil {
		labels.RequestStatus = metrics.RequestStatusBadInput
		h.onException(ctx, MileRequest{}, err)
		writeError(w, http.StatusBadRequest, "unable to read request body")
		return
	}
	if len(reqBody) == 0 {
		labels.RequestStatus = metrics.RequestStatusBadInput
		h.onException(ctx, MileRequest{}, fmt.Errorf("empty body"))
		writeError(w, http.StatusBadRequest, "empty request body")
		return
	}

	var mileReq MileRequest
	if err := json.Unmarshal(reqBody, &mileReq); err != nil {
		labels.RequestStatus = metrics.RequestStatusBadInput
		h.onException(ctx, mileReq, err)
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	mileReq.Raw = reqBody
	labels.RequestSize = len(reqBody)

	if h.cfg.AuthToken != "" {
		if token := r.Header.Get("X-Mile-Token"); token != h.cfg.AuthToken {
			labels.RequestStatus = metrics.RequestStatusBadInput
			h.onException(ctx, mileReq, fmt.Errorf("unauthorized"))
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
	}

	if err := validateRequest(mileReq); err != nil {
		labels.RequestStatus = metrics.RequestStatusBadInput
		h.onException(ctx, mileReq, err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.hooks.applyValidate(ctx, mileReq); err != nil {
		labels.RequestStatus = metrics.RequestStatusBadInput
		h.onException(ctx, mileReq, err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	siteCfg, err := h.store.Get(ctx, mileReq.SiteID)
	if err != nil {
		switch err {
		case ErrSiteNotFound:
			labels.RequestStatus = metrics.RequestStatusBadInput
			h.onException(ctx, mileReq, err)
			writeError(w, http.StatusNotFound, fmt.Sprintf("site not found: %s", mileReq.SiteID))
		default:
			labels.RequestStatus = metrics.RequestStatusNetworkErr
			h.onException(ctx, mileReq, err)
			writeError(w, http.StatusBadGateway, "failed to load site configuration")
		}
		return
	}

	ortbReq, err := buildOpenRTBRequest(mileReq, siteCfg)
	if err != nil {
		labels.RequestStatus = metrics.RequestStatusBadInput
		h.onException(ctx, mileReq, err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ortbReq, err = h.hooks.applyBefore(ctx, mileReq, siteCfg, ortbReq)
	if err != nil {
		labels.RequestStatus = metrics.RequestStatusBadInput
		h.onException(ctx, mileReq, err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	auctionBody, err := json.Marshal(ortbReq)
	if err != nil {
		labels.RequestStatus = metrics.RequestStatusErr
		h.onException(ctx, mileReq, err)
		writeError(w, http.StatusInternalServerError, "failed to encode auction request")
		return
	}

	auctionReq := r.Clone(ctx)
	auctionReq.Method = http.MethodPost
	auctionReq.URL = &url.URL{Path: "/openrtb2/auction"}
	auctionReq.Body = io.NopCloser(bytes.NewReader(auctionBody))
	auctionReq.Header = cloneHeaders(r.Header)
	auctionReq.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	h.auction(recorder, auctionReq, nil)

	resp := recorder.Result()
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	respBody, status, afterErr := h.hooks.applyAfter(ctx, mileReq, siteCfg, resp.StatusCode, respBody)
	if afterErr != nil {
		glog.Warningf("mile after hook failed: %v", afterErr)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		labels.RequestStatus = metrics.RequestStatusErr
	}

	copyHeaders(w, resp.Header)
	w.WriteHeader(status)
	if _, err := w.Write(respBody); err != nil {
		glog.Warningf("mile write response failed: %v", err)
	}
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

func (h *Handler) onException(ctx context.Context, req MileRequest, err error) {
	if h.hooks.OnException != nil && err != nil {
		h.hooks.OnException(ctx, req, err)
	}
}
