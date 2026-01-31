package vast

import (
	"context"
	"fmt"

	"github.com/prebid/openrtb/v20/openrtb2"
	
	
	"github.com/prebid/prebid-server/v3/modules/ctv/vast/model"
	
)

// BuildVastFromBidResponse is the main entrypoint function that orchestrates the complete pipeline
// It takes an OpenRTB bid request/response and produces a VAST XML response
func BuildVastFromBidResponse(ctx context.Context, req *openrtb2.BidRequest, resp *openrtb2.BidResponse, cfg ReceiverConfig) (VastResult, error) {
	// Step 1: Create selector and select bids
	bidSelector := NewSelector(cfg.SelectionStrategy)
	selected, selectorWarnings, err := bidSelector.Select(req, resp, cfg)
	if err != nil {
		return VastResult{
			NoAd:     true,
			Warnings: selectorWarnings,
			Errors:   []error{fmt.Errorf("selector failed: %w", err)},
		}, err
	}

	// Collect all warnings
	allWarnings := make([]string, 0, len(selectorWarnings))
	allWarnings = append(allWarnings, selectorWarnings...)

	// Step 3: If no bids selected, return no-ad VAST
	if len(selected) == 0 {
		return VastResult{
			VastXML:  model.BuildNoAdVast(cfg.VastVersionDefault),
			NoAd:     true,
			Warnings: allWarnings,
			Selected: selected,
		}, nil
	}

	// Step 4: For each selected bid, parse/create VAST and enrich
	vasts := make([]*model.VastAd, 0, len(selected))
	enricher := &stubEnricher{} // Use stub enricher for now

	for i := range selected {
		// Parse existing VAST from bid.adm or create skeleton
		vast, parseWarnings, err := model.ParseVastOrSkeleton(selected[i].Bid.AdM, cfg)
		if err != nil {
			return VastResult{
				NoAd:     true,
				Warnings: append(allWarnings, fmt.Sprintf("failed to parse VAST for bid %s: %v", selected[i].Bid.ID, err)),
				Errors:   []error{fmt.Errorf("parse failed for bid %s: %w", selected[i].Bid.ID, err)},
				Selected: selected,
			}, err
		}
		allWarnings = append(allWarnings, parseWarnings...)

		// Get the first (and should be only) ad from the parsed VAST
		var ad *model.VastAd
		if len(vast.Ads) > 0 {
			ad = &vast.Ads[0]
		} else {
			// No ads in parsed VAST, create a basic structure
			ad = &model.Ad{
				ID: selected[i].Meta.BidID,
				InLine: &model.InLine{
					AdSystem: selected[i].Meta.Seat,
					AdTitle:  "Ad",
				},
			}
		}

		// Set ad ID from metadata (prefer AdID, fall back to BidID)
		// Note: AdID would come from bid.ext if available, otherwise use BidID
		if selected[i].Meta.BidID != "" {
			ad.ID = selected[i].Meta.BidID
		}

		// Set sequence from SelectedBid
		ad.Sequence = selected[i].Sequence

		// Enrich the ad with OpenRTB metadata
		enrichWarnings, err := enricher.Enrich(ad, selected[i].Meta, cfg)
		if err != nil {
			allWarnings = append(allWarnings, fmt.Sprintf("enrichment failed for bid %s: %v", selected[i].Bid.ID, err))
			// Don't fail the entire request on enrichment error, just warn
		}
		allWarnings = append(allWarnings, enrichWarnings...)

		vasts = append(vasts, ad)
	}

	// Step 5: Create formatter
	formatter := NewFormatter()

	// Step 6: Format all ads into final VAST XML
	xml, formatWarnings, err := formatter.Format(vasts, cfg)
	if err != nil {
		return VastResult{
			NoAd:     true,
			Warnings: append(allWarnings, formatWarnings...),
			Errors:   []error{fmt.Errorf("formatter failed: %w", err)},
			Selected: selected,
		}, err
	}
	allWarnings = append(allWarnings, formatWarnings...)

	// Step 7: Return successful result
	return VastResult{
		VastXML:  xml,
		NoAd:     false,
		Warnings: allWarnings,
		Selected: selected,
	}, nil
}

// stubEnricher is a minimal enricher implementation to avoid import cycles
// The real enricher will be in the enrich package
type stubEnricher struct{}

func (e *stubEnricher) Enrich(v *model.VastAd, meta CanonicalMeta, cfg ReceiverConfig) ([]string, error) {
	// Minimal enrichment - just ensure basic structure exists
	// The enrich package will have the full implementation
	return nil, nil
}
