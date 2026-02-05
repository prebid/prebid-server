package vast

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
	"github.com/prebid/prebid-server/v3/modules/prebid/ctv_vast_enrichment/model"
)

// Builder creates a new CTV VAST enrichment module instance.
// It parses the host-level configuration and initializes the module
// with default selector, enricher, and formatter implementations.
func Builder(cfg json.RawMessage, deps moduledeps.ModuleDeps) (interface{}, error) {
	var hostCfg CTVVastConfig
	if len(cfg) > 0 {
		if err := json.Unmarshal(cfg, &hostCfg); err != nil {
			return nil, err
		}
	}

	return Module{
		hostConfig: hostCfg,
	}, nil
}

// Module implements the CTV VAST enrichment functionality as a PBS hook module.
// It processes raw bidder responses to enrich VAST XML with additional metadata
// such as pricing, categories, and advertiser information.
type Module struct {
	hostConfig CTVVastConfig
}

// HandleRawBidderResponseHook processes bidder responses to enrich VAST XML.
// For each bid containing VAST (video bids), the hook:
//   - Parses the VAST XML from the bid's AdM field
//   - Enriches the VAST with pricing, category, and advertiser metadata
//   - Updates the bid's AdM with the enriched VAST XML
//
// The enrichment is controlled by the module configuration at host, account,
// and request levels. If enrichment is disabled, the response passes through unchanged.
func (m Module) HandleRawBidderResponseHook(
	ctx context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.RawBidderResponsePayload,
) (hookstage.HookResult[hookstage.RawBidderResponsePayload], error) {
	result := hookstage.HookResult[hookstage.RawBidderResponsePayload]{}

	// Parse account-level config if present
	var accountCfg *CTVVastConfig
	if len(miCtx.AccountConfig) > 0 {
		var cfg CTVVastConfig
		if err := json.Unmarshal(miCtx.AccountConfig, &cfg); err != nil {
			return result, err
		}
		accountCfg = &cfg
	}

	// Merge configurations: host < account
	mergedCfg := MergeCTVVastConfig(&m.hostConfig, accountCfg, nil)

	// Check if module is enabled
	if mergedCfg.Enabled != nil && !*mergedCfg.Enabled {
		return result, nil
	}

	// No bids to process
	if payload.BidderResponse == nil || len(payload.BidderResponse.Bids) == 0 {
		return result, nil
	}

	// Convert config to ReceiverConfig
	receiverCfg := configToReceiverConfig(mergedCfg)

	// Process each bid
	changesMade := false
	for i := range payload.BidderResponse.Bids {
		typedBid := payload.BidderResponse.Bids[i]
		if typedBid == nil || typedBid.Bid == nil {
			continue
		}

		bid := typedBid.Bid

		// Skip non-video bids (no AdM or not VAST)
		if bid.AdM == "" {
			continue
		}

		// Try to parse as VAST
		vastDoc, err := model.ParseVastAdm(bid.AdM)
		if err != nil {
			// Not valid VAST, skip enrichment
			continue
		}

		// Build bid context for enrichment
		bidContext := CanonicalMeta{
			BidID:    bid.ID,
			Price:    bid.Price,
			Currency: receiverCfg.DefaultCurrency,
			Adomain:  strings.Join(bid.ADomain, ","),
			Cats:     bid.Cat,
			Seat:     payload.Bidder,
		}

		// Enrich the VAST document inline
		enrichedVast := enrichVastDocument(vastDoc, bidContext, receiverCfg)

		// Format back to XML
		xmlBytes, err := enrichedVast.Marshal()
		if err != nil {
			// Keep original AdM on format error
			continue
		}

		// Update bid with enriched VAST
		bid.AdM = string(xmlBytes)
		changesMade = true
	}

	// If we made changes, set mutation
	if changesMade {
		result.ChangeSet.AddMutation(
			func(payload hookstage.RawBidderResponsePayload) (hookstage.RawBidderResponsePayload, error) {
				return payload, nil
			},
			hookstage.MutationUpdate,
			"ctv-vast-enrichment",
		)
	}

	return result, nil
}

// configToReceiverConfig converts CTVVastConfig to ReceiverConfig
func configToReceiverConfig(cfg CTVVastConfig) ReceiverConfig {
	rc := DefaultConfig()

	if cfg.Receiver != "" {
		switch cfg.Receiver {
		case "GAM_SSU":
			rc.Receiver = ReceiverGAMSSU
		case "GENERIC":
			rc.Receiver = ReceiverGeneric
		}
	}

	if cfg.DefaultCurrency != "" {
		rc.DefaultCurrency = cfg.DefaultCurrency
	}

	if cfg.VastVersionDefault != "" {
		rc.VastVersionDefault = cfg.VastVersionDefault
	}

	if cfg.MaxAdsInPod > 0 {
		rc.MaxAdsInPod = cfg.MaxAdsInPod
	}

	if cfg.SelectionStrategy != "" {
		switch cfg.SelectionStrategy {
		case "max_revenue", "MAX_REVENUE":
			rc.SelectionStrategy = SelectionMaxRevenue
		case "top_n", "TOP_N":
			rc.SelectionStrategy = SelectionTopN
		case "single", "SINGLE":
			rc.SelectionStrategy = SelectionSingle
		}
	}

	if cfg.CollisionPolicy != "" {
		switch cfg.CollisionPolicy {
		case "reject", "REJECT":
			rc.CollisionPolicy = CollisionReject
		case "warn", "WARN":
			rc.CollisionPolicy = CollisionWarn
		case "ignore", "IGNORE":
			rc.CollisionPolicy = CollisionIgnore
		}
	}

	if cfg.AllowSkeletonVast != nil {
		rc.AllowSkeletonVast = *cfg.AllowSkeletonVast
	}

	if cfg.Placement != nil {
		if cfg.Placement.PricingPlacement != "" {
			rc.Placement.PricingPlacement = cfg.Placement.PricingPlacement
		}
	}

	return rc
}

// enrichVastDocument enriches a VAST document with bid metadata.
// It adds pricing and advertiser information to the VAST.
func enrichVastDocument(vast *model.Vast, meta CanonicalMeta, cfg ReceiverConfig) *model.Vast {
	if vast == nil {
		return vast
	}

	// Process each ad
	for i := range vast.Ads {
		ad := &vast.Ads[i]
		if ad.InLine == nil {
			continue
		}
		inline := ad.InLine

		// Add pricing if not present
		if inline.Pricing == nil && meta.Price > 0 {
			currency := cfg.DefaultCurrency
			if currency == "" {
				currency = "USD"
			}
			inline.Pricing = &model.Pricing{
				Value:    fmt.Sprintf("%.6f", meta.Price),
				Model:    "CPM",
				Currency: currency,
			}
		}

		// Add advertiser if not present
		if inline.Advertiser == "" && meta.Adomain != "" {
			inline.Advertiser = meta.Adomain
		}
	}

	return vast
}
