// Package vast provides CTV VAST processing capabilities for Prebid Server.
//
// This module handles the complete VAST workflow for Connected TV (CTV) ad serving:
//   - Bid selection from OpenRTB auction responses
//   - VAST ad enrichment with tracking and metadata
//   - VAST XML formatting for various downstream receivers
//
// The package is organized into sub-packages:
//   - model: VAST data structures
//   - select: Bid selection logic
//   - enrich: VAST ad enrichment
//   - format: VAST XML formatting
//
// Example usage:
//
//	cfg := vast.ReceiverConfig{
//		Receiver:           vast.ReceiverGAMSSU,
//		DefaultCurrency:    "USD",
//		VastVersionDefault: "4.0",
//		MaxAdsInPod:        5,
//		SelectionStrategy:  vast.SelectionMaxRevenue,
//		CollisionPolicy:    vast.CollisionReject,
//	}
//
//	processor := vast.NewProcessor(cfg, selector, enricher, formatter)
//	result := processor.Process(bidRequest, bidResponse)
package vast

import (
	"context"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/ctv/vast/model"
)

// BuildVastFromBidResponse orchestrates the complete VAST processing pipeline.
// It selects bids, parses/creates VAST, enriches ads, and formats final XML.
//
// Steps:
//  1. Select bids from response using configured strategy
//  2. Parse VAST from each bid's AdM (or create skeleton if allowed)
//  3. Enrich each ad with metadata (pricing, categories, etc.)
//  4. Format all ads into final VAST XML
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - req: OpenRTB bid request
//   - resp: OpenRTB bid response from auction
//   - cfg: Receiver configuration
//   - selector: Bid selection implementation
//   - enricher: VAST enrichment implementation
//   - formatter: VAST formatting implementation
//
// Returns VastResult containing XML output, warnings, and selected bids.
func BuildVastFromBidResponse(
	ctx context.Context,
	req *openrtb2.BidRequest,
	resp *openrtb2.BidResponse,
	cfg ReceiverConfig,
	selector BidSelector,
	enricher Enricher,
	formatter Formatter,
) (VastResult, error) {
	result := VastResult{
		Warnings: make([]string, 0),
		Errors:   make([]error, 0),
	}

	// Step 1: Select bids
	selected, selectWarnings, err := selector.Select(req, resp, cfg)
	if err != nil {
		result.Errors = append(result.Errors, err)
		result.NoAd = true
		result.VastXML = model.BuildNoAdVast(cfg.VastVersionDefault)
		return result, err
	}
	result.Warnings = append(result.Warnings, selectWarnings...)
	result.Selected = selected

	// Step 2: Handle no bids case
	if len(selected) == 0 {
		result.NoAd = true
		result.VastXML = model.BuildNoAdVast(cfg.VastVersionDefault)
		return result, nil
	}

	// Step 3: Parse and enrich each selected bid's VAST
	enrichedAds := make([]EnrichedAd, 0, len(selected))

	parserCfg := model.ParserConfig{
		AllowSkeletonVast:  cfg.AllowSkeletonVast,
		VastVersionDefault: cfg.VastVersionDefault,
	}

	for _, sb := range selected {
		// Parse VAST from AdM (or create skeleton)
		parsedVast, parseWarnings, parseErr := model.ParseVastOrSkeleton(sb.Bid.AdM, parserCfg)
		result.Warnings = append(result.Warnings, parseWarnings...)

		if parseErr != nil {
			result.Warnings = append(result.Warnings, "failed to parse VAST for bid "+sb.Bid.ID+": "+parseErr.Error())
			continue
		}

		// Extract the first Ad from parsed VAST
		ad := model.ExtractFirstAd(parsedVast)
		if ad == nil {
			result.Warnings = append(result.Warnings, "no ad found in VAST for bid "+sb.Bid.ID)
			continue
		}

		// Enrich the ad with metadata
		enrichWarnings, enrichErr := enricher.Enrich(ad, sb.Meta, cfg)
		result.Warnings = append(result.Warnings, enrichWarnings...)
		if enrichErr != nil {
			result.Warnings = append(result.Warnings, "enrichment failed for bid "+sb.Bid.ID+": "+enrichErr.Error())
			// Continue with unenriched ad
		}

		// Store enriched ad
		enrichedAds = append(enrichedAds, EnrichedAd{
			Ad:       ad,
			Meta:     sb.Meta,
			Sequence: sb.Sequence,
		})
	}

	// Step 4: Handle case where all bids failed parsing
	if len(enrichedAds) == 0 {
		result.NoAd = true
		result.VastXML = model.BuildNoAdVast(cfg.VastVersionDefault)
		result.Warnings = append(result.Warnings, "all selected bids failed VAST parsing")
		return result, nil
	}

	// Step 5: Format the final VAST XML
	xmlBytes, formatWarnings, formatErr := formatter.Format(enrichedAds, cfg)
	result.Warnings = append(result.Warnings, formatWarnings...)

	if formatErr != nil {
		result.Errors = append(result.Errors, formatErr)
		result.NoAd = true
		result.VastXML = model.BuildNoAdVast(cfg.VastVersionDefault)
		return result, formatErr
	}

	result.VastXML = xmlBytes
	result.NoAd = false

	return result, nil
}

// Processor orchestrates the VAST processing workflow.
type Processor struct {
	selector  BidSelector
	enricher  Enricher
	formatter Formatter
	config    ReceiverConfig
}

// NewProcessor creates a new Processor with the given configuration.
func NewProcessor(cfg ReceiverConfig, selector BidSelector, enricher Enricher, formatter Formatter) *Processor {
	return &Processor{
		selector:  selector,
		enricher:  enricher,
		formatter: formatter,
		config:    cfg,
	}
}

// Process executes the complete VAST processing workflow.
func (p *Processor) Process(ctx context.Context, req *openrtb2.BidRequest, resp *openrtb2.BidResponse) VastResult {
	result, _ := BuildVastFromBidResponse(ctx, req, resp, p.config, p.selector, p.enricher, p.formatter)
	return result
}

// DefaultConfig returns a default ReceiverConfig for GAM SSU.
func DefaultConfig() ReceiverConfig {
	return ReceiverConfig{
		Receiver:           ReceiverGAMSSU,
		DefaultCurrency:    "USD",
		VastVersionDefault: "4.0",
		MaxAdsInPod:        5,
		SelectionStrategy:  SelectionMaxRevenue,
		CollisionPolicy:    CollisionReject,
		Placement: PlacementRules{
			Pricing: PricingRules{
				FloorCPM:   0,
				CeilingCPM: 0,
				Currency:   "USD",
			},
			Advertiser: AdvertiserRules{
				BlockedDomains: []string{},
				AllowedDomains: []string{},
			},
			Categories: CategoryRules{
				BlockedCategories: []string{},
				AllowedCategories: []string{},
			},
			Debug: false,
		},
		Debug: false,
	}
}
