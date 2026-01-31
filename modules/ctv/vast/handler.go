package vast

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	
)

// VastHandler handles HTTP requests for CTV VAST responses
type VastHandler struct {
	// TODO: Add auction handler/dependencies when integrating with main server
}

// NewVastHandler creates a new VAST HTTP handler
func NewVastHandler() *VastHandler {
	return &VastHandler{}
}

// ServeHTTP handles GET requests for VAST XML
// Query parameters are converted to OpenRTB request (TODO: implement query parsing)
// The request is sent through the auction pipeline (TODO: integrate with auction)
// The bid response is converted to VAST XML using BuildVastFromBidResponse
func (h *VastHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Only support GET for now
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Parse query parameters and build OpenRTB request
	// Example: ?imp=video1&w=640&h=480&duration=30&position=1
	req := buildOpenRTBRequestFromQuery(r)

	// TODO: Call auction pipeline to get bid response
	resp, err := callAuctionPipeline(ctx, req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Auction failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Build VAST from bid response
	cfg := getReceiverConfig(r)
	vastResult, err := BuildVastFromBidResponse(ctx, req, resp, cfg)
	if err != nil {
		http.Error(w, fmt.Sprintf("VAST generation failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Write VAST XML response
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	w.Write(vastResult.VastXML)
}

// buildOpenRTBRequestFromQuery converts query parameters to OpenRTB request
// TODO: Implement full query parameter parsing
func buildOpenRTBRequestFromQuery(r *http.Request) *openrtb2.BidRequest {
	// Placeholder implementation
	w := int64(640)
	h := int64(480)
	return &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{
			{
				ID: "imp1",
				Video: &openrtb2.Video{
					W:           &w,
					H:           &h,
					MIMEs:       []string{"video/mp4"},
					MinDuration: 15,
					MaxDuration: 30,
				},
			},
		},
	}
}

// callAuctionPipeline calls the main Prebid Server auction logic
// TODO: Integrate with actual auction handler
func callAuctionPipeline(ctx context.Context, req *openrtb2.BidRequest) (*openrtb2.BidResponse, error) {
	// Placeholder - this would call the real auction handler
	return &openrtb2.BidResponse{
		ID: req.ID,
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "testbidder",
				Bid: []openrtb2.Bid{
					{
						ID:    "bid1",
						ImpID: "imp1",
						Price: 5.0,
						AdM:   `<VAST version="3.0"><Ad id="test"><InLine><AdSystem>Test</AdSystem><AdTitle>Test Ad</AdTitle></InLine></Ad></VAST>`,
					},
				},
			},
		},
	}, nil
}

// getReceiverConfig extracts receiver configuration from request
// TODO: Support multiple receivers, config from query params
func getReceiverConfig(r *http.Request) ReceiverConfig {
	return ReceiverConfig{
		Receiver:           "GAM_SSU",
		VastVersionDefault: "3.0",
		DefaultCurrency:    "USD",
		MaxAdsInPod:        1,
		SelectionStrategy:  "SINGLE",
		AllowSkeletonVast:  true,
		CollisionPolicy:    CollisionPolicyVastWins,
		PlacementRules: PlacementRules{
			PricingPlacement:    PlacementInline,
			AdvertiserPlacement: PlacementInline,
			CategoriesPlacement: PlacementExtensions,
			DebugPlacement:      PlacementExtensions,
		},
	}
}

// VastHandlerForTesting is a test helper that allows injecting a bid response
type VastHandlerForTesting struct {
	MockBidResponse *openrtb2.BidResponse
	MockConfig      ReceiverConfig
}

// ServeHTTP handles test requests with mocked bid response
func (h *VastHandlerForTesting) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Use mock request
	req := &openrtb2.BidRequest{
		ID: "test-request",
		Imp: []openrtb2.Imp{
			{ID: "imp1"},
		},
	}

	// Use injected mock response
	resp := h.MockBidResponse
	if resp == nil {
		http.Error(w, "No mock response configured", http.StatusInternalServerError)
		return
	}

	// Build VAST from bid response
	cfg := h.MockConfig
	vastResult, err := BuildVastFromBidResponse(ctx, req, resp, cfg)
	if err != nil {
		http.Error(w, fmt.Sprintf("VAST generation failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Write VAST XML response
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	w.Write(vastResult.VastXML)
}

// DebugHandler returns JSON debug info about the VAST result
func (h *VastHandlerForTesting) DebugHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req := &openrtb2.BidRequest{
		ID: "test-request",
		Imp: []openrtb2.Imp{
			{ID: "imp1"},
		},
	}

	resp := h.MockBidResponse
	if resp == nil {
		http.Error(w, "No mock response configured", http.StatusInternalServerError)
		return
	}

	cfg := h.MockConfig
	vastResult, err := BuildVastFromBidResponse(ctx, req, resp, cfg)
	if err != nil {
		http.Error(w, fmt.Sprintf("VAST generation failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Return JSON with debug info
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"no_ad":           vastResult.NoAd,
		"warnings":        vastResult.Warnings,
		"selected_count":  len(vastResult.Selected),
		"vast_xml_length": len(vastResult.VastXML),
		"vast_xml":        string(vastResult.VastXML),
	})
}
