package ctv

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/endpoints/ctv/vast/enricher"
	"github.com/prebid/prebid-server/v3/endpoints/ctv/vast/formatter"
	"github.com/prebid/prebid-server/v3/endpoints/ctv/vast/model"
	"github.com/prebid/prebid-server/v3/endpoints/ctv/vast/selector"
	"github.com/prebid/prebid-server/v3/exchange"
	"github.com/prebid/prebid-server/v3/logger"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/ortb"
	"github.com/prebid/prebid-server/v3/stored_requests"
	"github.com/prebid/prebid-server/v3/util/uuidutil"
	"github.com/prebid/prebid-server/v3/version"
)

// CTVEndpointDeps holds dependencies for the CTV VAST endpoint
type CTVEndpointDeps struct {
	uuidGenerator    uuidutil.UUIDGenerator
	ex               exchange.Exchange
	requestValidator ortb.RequestValidator
	storedReqFetcher stored_requests.Fetcher
	accounts         stored_requests.AccountFetcher
	cfg              *config.Configuration
	metricsEngine    metrics.MetricsEngine
	logger           logger.Logger
}

// NewCTVVastEndpoint creates a new CTV VAST endpoint handler
func NewCTVVastEndpoint(
	uuidGenerator uuidutil.UUIDGenerator,
	ex exchange.Exchange,
	requestValidator ortb.RequestValidator,
	storedReqFetcher stored_requests.Fetcher,
	accounts stored_requests.AccountFetcher,
	cfg *config.Configuration,
	metricsEngine metrics.MetricsEngine,
) httprouter.Handle {
	deps := &CTVEndpointDeps{
		uuidGenerator:    uuidGenerator,
		ex:               ex,
		requestValidator: requestValidator,
		storedReqFetcher: storedReqFetcher,
		accounts:         accounts,
		cfg:              cfg,
		metricsEngine:    metricsEngine,
		logger:           logger.NewGlogLogger(),
	}

	return deps.HandleVast
}

// HandleVast handles CTV VAST GET requests
func (deps *CTVEndpointDeps) HandleVast(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	start := time.Now()

	// Set headers
	w.Header().Set("X-Prebid", version.BuildXPrebidHeader(version.Ver))
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")

	// Record metrics
	labels := metrics.Labels{
		Source:        metrics.DemandCTV,
		RType:         metrics.ReqTypeCTV,
		PubID:         metrics.PublisherUnknown,
		CookieFlag:    metrics.CookieFlagUnknown,
		RequestStatus: metrics.RequestStatusOK,
	}
	defer func() {
		deps.metricsEngine.RecordRequest(labels)
		deps.metricsEngine.RecordRequestTime(labels, time.Since(start))
	}()

	ctx := r.Context()

	// Parse query parameters
	queryParams, err := deps.parseQueryParams(r)
	if err != nil {
		logger.Warnf("CTV VAST: Error parsing query params: %v", err)
		deps.writeEmptyVAST(w, config.CTVVastDefaults())
		return
	}

	// Get configuration
	vastConfig := deps.getVastConfig(ctx, queryParams.PublisherID)
	if !vastConfig.Enabled {
		deps.logger.Infof("CTV VAST: Endpoint disabled")
		deps.writeEmptyVAST(w, vastConfig)
		return
	}

	// Build OpenRTB request
	bidRequest, err := deps.buildBidRequest(ctx, queryParams, vastConfig)
	if err != nil {
		deps.logger.Warnf("CTV VAST: Error building bid request: %v", err)
		labels.RequestStatus = metrics.RequestStatusBadInput
		deps.writeEmptyVAST(w, vastConfig)
		return
	}

	// Execute auction
	auctionRequest := &exchange.AuctionRequest{
		BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: bidRequest},
		Account:           config.Account{ID: queryParams.PublisherID},
		StartTime:         start,
	}

	debugLog := &exchange.DebugLog{}
	auctionResponse, err := deps.ex.HoldAuction(ctx, auctionRequest, debugLog)
	if err != nil {
		deps.logger.Warnf("CTV VAST: Auction error: %v", err)
		labels.RequestStatus = metrics.RequestStatusErr
		deps.writeEmptyVAST(w, vastConfig)
		return
	}

	// Check for bids
	if auctionResponse == nil || auctionResponse.BidResponse == nil || len(auctionResponse.BidResponse.SeatBid) == 0 {
		deps.logger.Infof("CTV VAST: No bids returned")
		labels.RequestStatus = metrics.RequestStatusOK
		deps.writeEmptyVAST(w, vastConfig)
		return
	}

	// Process auction response into VAST
	vastXML, err := deps.processAuctionResponse(auctionResponse.BidResponse, vastConfig)
	if err != nil {
		deps.logger.Warnf("CTV VAST: Error processing auction response: %v", err)
		labels.RequestStatus = metrics.RequestStatusErr
		deps.writeEmptyVAST(w, vastConfig)
		return
	}

	// Write VAST response
	w.WriteHeader(http.StatusOK)
	w.Write(vastXML)
	
	labels.PubID = queryParams.PublisherID
	labels.RequestStatus = metrics.RequestStatusOK
}

// CTVQueryParams holds parsed query parameters
type CTVQueryParams struct {
	PublisherID      string
	StoredRequestID  string
	Width            int
	Height           int
	MinDuration      int
	MaxDuration      int
	Macros           map[string]string
	DebugEnabled     bool
}

// parseQueryParams extracts and validates query parameters
func (deps *CTVEndpointDeps) parseQueryParams(r *http.Request) (*CTVQueryParams, error) {
	query := r.URL.Query()

	params := &CTVQueryParams{
		PublisherID:      query.Get("publisher_id"),
		StoredRequestID:  query.Get("stored_request_id"),
		Macros:           make(map[string]string),
		DebugEnabled:     query.Get("debug") == "1",
	}

	// Parse dimensions
	if w := query.Get("width"); w != "" {
		if width, err := strconv.Atoi(w); err == nil {
			params.Width = width
		}
	}
	if h := query.Get("height"); h != "" {
		if height, err := strconv.Atoi(h); err == nil {
			params.Height = height
		}
	}

	// Parse duration constraints
	if minDur := query.Get("min_duration"); minDur != "" {
		if min, err := strconv.Atoi(minDur); err == nil {
			params.MinDuration = min
		}
	}
	if maxDur := query.Get("max_duration"); maxDur != "" {
		if max, err := strconv.Atoi(maxDur); err == nil {
			params.MaxDuration = max
		}
	}

	// Collect all other query params as potential macros
	for key, values := range query {
		if len(values) > 0 && !isReservedParam(key) {
			params.Macros[key] = values[0]
		}
	}

	return params, nil
}

// isReservedParam checks if a parameter is reserved
func isReservedParam(key string) bool {
	reserved := map[string]bool{
		"publisher_id":      true,
		"stored_request_id": true,
		"width":             true,
		"height":            true,
		"min_duration":      true,
		"max_duration":      true,
		"debug":             true,
	}
	return reserved[key]
}

// getVastConfig retrieves merged VAST configuration
func (deps *CTVEndpointDeps) getVastConfig(ctx context.Context, publisherID string) config.CTVVast {
	// For MVP, use host defaults
	// In production, would fetch account config from deps.accounts
	return config.CTVVastDefaults()
}

// buildBidRequest constructs an OpenRTB BidRequest from query params
func (deps *CTVEndpointDeps) buildBidRequest(ctx context.Context, params *CTVQueryParams, vastConfig config.CTVVast) (*openrtb2.BidRequest, error) {
	// Generate request ID
	requestID, err := deps.uuidGenerator.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate request ID: %w", err)
	}

	// Create video impression
	imp := openrtb2.Imp{
		ID: "1",
		Video: &openrtb2.Video{
			MIMEs:     []string{"video/mp4", "video/webm"},
			Protocols: []adcom1.MediaCreativeSubtype{adcom1.CreativeVAST20, adcom1.CreativeVAST30, adcom1.CreativeVAST40},
		},
	}

	// Set dimensions if provided
	if params.Width > 0 {
		imp.Video.W = openrtb2.Int64Ptr(int64(params.Width))
	}
	if params.Height > 0 {
		imp.Video.H = openrtb2.Int64Ptr(int64(params.Height))
	}

	// Set duration constraints
	if params.MinDuration > 0 {
		imp.Video.MinDuration = int64(params.MinDuration)
	}
	if params.MaxDuration > 0 {
		imp.Video.MaxDuration = int64(params.MaxDuration)
	}

	// Build request
	bidRequest := &openrtb2.BidRequest{
		ID: requestID,
		Imp: []openrtb2.Imp{imp},
		Device: &openrtb2.Device{
			UA: "", // Could be populated from headers
		},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				ID: params.PublisherID,
			},
		},
		TMax: int64(deps.cfg.TmaxDefault),
		Ext:  nil,
	}

	// Add stored request data if provided
	if vastConfig.StoredRequestsEnabled && params.StoredRequestID != "" {
		// For MVP, skip stored request fetching
		// In production, would fetch and merge stored request data
	}

	return bidRequest, nil
}

// processAuctionResponse converts auction response to VAST XML
func (deps *CTVEndpointDeps) processAuctionResponse(bidResponse *openrtb2.BidResponse, vastConfig config.CTVVast) ([]byte, error) {
	// Select winning bids
	selectionConfig := selector.Config{
		Strategy:    selector.Strategy(vastConfig.SelectionStrategy),
		MaxAdsInPod: vastConfig.MaxAdsInPod,
	}

	sel := selector.NewSelector()
	selectionResult, err := sel.Select(bidResponse, selectionConfig)
	if err != nil {
		return nil, fmt.Errorf("bid selection failed: %w", err)
	}

	if len(selectionResult.Bids) == 0 {
		// No bids selected, return empty VAST
		return deps.formatEmptyVAST(vastConfig)
	}

	// Create VAST
	vast := model.NewEmptyVAST(vastConfig.VastVersionDefault)

	// Enrich VAST with bid data
	enrichConfig := deps.buildEnricherConfig(vastConfig)
	enrich := enricher.NewEnricher(enrichConfig)

	for i, bidWithSeat := range selectionResult.Bids {
		sequence := selector.GetSequence(bidWithSeat.Bid, i)
		err := enrich.Enrich(vast, bidWithSeat.Bid, bidWithSeat.Seat, bidResponse, sequence)
		if err != nil {
			logger.Warnf("CTV VAST: Failed to enrich ad %d: %v", i, err)
			continue
		}
	}

	// Format VAST for receiver
	formatterConfig := formatter.Config{
		Profile:        formatter.ReceiverProfile(vastConfig.Receiver),
		DefaultVersion: vastConfig.VastVersionDefault,
	}

	factory := formatter.NewFormatterFactory()
	format := factory.CreateFormatter(formatterConfig)

	return format.Format(vast)
}

// buildEnricherConfig creates enricher config from VAST config
func (deps *CTVEndpointDeps) buildEnricherConfig(vastConfig config.CTVVast) enricher.Config {
	return enricher.Config{
		CollisionPolicy: enricher.CollisionPolicy(vastConfig.CollisionPolicy),
		PlacementRules: enricher.PlacementRules{
			Price:      enricher.Placement(vastConfig.PlacementRules.Price),
			Currency:   enricher.Placement(vastConfig.PlacementRules.Currency),
			Advertiser: enricher.Placement(vastConfig.PlacementRules.Advertiser),
			Categories: enricher.Placement(vastConfig.PlacementRules.Categories),
			Duration:   enricher.Placement(vastConfig.PlacementRules.Duration),
			IDs:        enricher.Placement(vastConfig.PlacementRules.IDs),
			DealID:     enricher.Placement(vastConfig.PlacementRules.DealID),
		},
		DefaultCurrency: vastConfig.DefaultCurrency,
		IncludeDebugIDs: vastConfig.IncludeDebugIDs,
	}
}

// writeEmptyVAST writes an empty VAST response
func (deps *CTVEndpointDeps) writeEmptyVAST(w http.ResponseWriter, vastConfig config.CTVVast) {
	emptyVAST, err := deps.formatEmptyVAST(vastConfig)
	if err != nil {
		logger.Warnf("CTV VAST: Failed to format empty VAST: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(emptyVAST)
}

// formatEmptyVAST formats an empty VAST response
func (deps *CTVEndpointDeps) formatEmptyVAST(vastConfig config.CTVVast) ([]byte, error) {
	formatterConfig := formatter.Config{
		Profile:        formatter.ReceiverProfile(vastConfig.Receiver),
		DefaultVersion: vastConfig.VastVersionDefault,
	}
	return formatter.FormatEmptyVAST(formatterConfig)
}
