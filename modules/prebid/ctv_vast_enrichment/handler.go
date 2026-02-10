package ctv_vast_enrichment

import (
	"context"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
)

// Handler provides HTTP handling for CTV VAST requests.
type Handler struct {
	// Config contains the default receiver configuration.
	Config ReceiverConfig
	// Selector selects bids from auction response.
	Selector BidSelector
	// Enricher enriches VAST ads with metadata.
	Enricher Enricher
	// Formatter formats enriched ads as VAST XML.
	Formatter Formatter
	// AuctionFunc is called to run the auction pipeline.
	// This should be injected with the actual auction implementation.
	AuctionFunc func(ctx context.Context, req *openrtb2.BidRequest) (*openrtb2.BidResponse, error)
}

// NewHandler creates a new VAST HTTP handler with default configuration.
// Note: Selector, Enricher, and Formatter must be set via With* methods
// before the handler can process requests.
func NewHandler() *Handler {
	return &Handler{
		Config: DefaultConfig(),
	}
}

// ServeHTTP handles GET requests for CTV VAST ads.
// Query parameters (TODO: implement full parsing):
//   - pod_id: Pod identifier
//   - duration: Requested pod duration
//   - max_ads: Maximum ads in pod
//
// Response:
//   - 200 OK with Content-Type: application/xml on success
//   - 204 No Content if no ads available
//   - 400 Bad Request for invalid parameters
//   - 500 Internal Server Error for processing failures
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Only accept GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate required dependencies
	if h.Selector == nil || h.Enricher == nil || h.Formatter == nil {
		http.Error(w, "Handler not properly configured", http.StatusInternalServerError)
		return
	}

	// TODO: Parse query parameters and build OpenRTB request
	// This is a placeholder for the actual implementation:
	// - Parse pod_id, duration, max_ads from query string
	// - Build openrtb2.BidRequest with Video imp
	// - Apply site/app context from query or headers
	bidRequest := h.buildBidRequest(r)

	// TODO: Call auction pipeline
	// This is a placeholder - actual implementation would:
	// - Call the Prebid Server auction endpoint
	// - Get BidResponse from exchange
	var bidResponse *openrtb2.BidResponse
	var err error

	if h.AuctionFunc != nil {
		bidResponse, err = h.AuctionFunc(ctx, bidRequest)
		if err != nil {
			http.Error(w, "Auction failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// No auction function configured - return no-ad
		bidResponse = &openrtb2.BidResponse{}
	}

	// Build VAST from bid response
	result, err := BuildVastFromBidResponse(ctx, bidRequest, bidResponse, h.Config, h.Selector, h.Enricher, h.Formatter)
	if err != nil {
		// Log error but still try to return valid VAST
		// result.VastXML should contain no-ad VAST
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	// Handle no-ad case
	if result.NoAd {
		w.WriteHeader(http.StatusOK) // Still 200 per VAST spec
	}

	// Write VAST XML
	w.Write(result.VastXML)
}

// buildBidRequest creates an OpenRTB BidRequest from the HTTP request.
// TODO: Implement full parsing of query parameters.
func (h *Handler) buildBidRequest(r *http.Request) *openrtb2.BidRequest {
	// Placeholder implementation
	// TODO: Parse these from query string:
	// - pod_id -> BidRequest.ID
	// - duration -> Video.MaxDuration
	// - max_ads -> Video.MaxAds (via pod extension)
	// - slot_count -> multiple Imp objects

	query := r.URL.Query()
	podID := query.Get("pod_id")
	if podID == "" {
		podID = "ctv-pod-1"
	}

	return &openrtb2.BidRequest{
		ID: podID,
		Imp: []openrtb2.Imp{
			{
				ID: "imp-1",
				Video: &openrtb2.Video{
					MIMEs:       []string{"video/mp4"},
					MinDuration: 5,
					MaxDuration: 30,
				},
			},
		},
		Site: &openrtb2.Site{
			Page: r.Header.Get("Referer"),
		},
	}
}

// WithConfig sets the receiver configuration.
func (h *Handler) WithConfig(cfg ReceiverConfig) *Handler {
	h.Config = cfg
	return h
}

// WithSelector sets the bid selector.
func (h *Handler) WithSelector(s BidSelector) *Handler {
	h.Selector = s
	return h
}

// WithEnricher sets the VAST enricher.
func (h *Handler) WithEnricher(e Enricher) *Handler {
	h.Enricher = e
	return h
}

// WithFormatter sets the VAST formatter.
func (h *Handler) WithFormatter(f Formatter) *Handler {
	h.Formatter = f
	return h
}

// WithAuctionFunc sets the auction function.
func (h *Handler) WithAuctionFunc(fn func(ctx context.Context, req *openrtb2.BidRequest) (*openrtb2.BidResponse, error)) *Handler {
	h.AuctionFunc = fn
	return h
}
