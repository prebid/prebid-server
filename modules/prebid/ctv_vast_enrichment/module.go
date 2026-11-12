package ctv_vast_enrichment

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
	"github.com/prebid/prebid-server/v4/modules/prebid/ctv_vast_enrichment/model"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
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
		enricher:   EnricherFactory(),
	}, nil
}

// Module implements the CTV VAST enrichment functionality as a PBS hook module.
// It processes raw bidder responses to enrich VAST XML with additional metadata
// such as pricing, categories, and advertiser information.
type Module struct {
	hostConfig CTVVastConfig
	// enricher is the VAST enricher used by the hook. It is set in Builder via
	// the EnricherFactory variable to avoid an import cycle with the enrich subpackage.
	enricher Enricher
}

// EnricherFactory is a package-level variable that provides the default Enricher
// implementation. It is overridden by the enrich subpackage via init() to avoid
// an import cycle (parent → enrich → parent).
// NOTE: Architectural TODO — move shared types (CanonicalMeta, ReceiverConfig) to a
// types subpackage so parent and enrich can both import it without a cycle.
var EnricherFactory func() Enricher = func() Enricher {
	return &hookEnricher{}
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

	// Check if module is enabled; nil is treated as disabled (explicit opt-in required)
	if !mergedCfg.IsEnabled() {
		return result, nil
	}

	// No bids to process
	if payload.BidderResponse == nil || len(payload.BidderResponse.Bids) == 0 {
		return result, nil
	}

	// Convert config to ReceiverConfig (canonical method in config.go)
	receiverCfg := mergedCfg.ReceiverConfig()

	// Use injected enricher (set via EnricherFactory in Builder).
	// Fall back to hookEnricher for tests that construct Module{} directly.
	enricher := m.enricher
	if enricher == nil {
		enricher = &hookEnricher{}
	}

	// modifiedBids is allocated lazily — only when the first enrichment actually happens.
	// Until then we track the index so we can back-fill originals if needed.
	var modifiedBids []*adapters.TypedBid

	for i, typedBid := range payload.BidderResponse.Bids {
		if typedBid == nil || typedBid.Bid == nil {
			if modifiedBids != nil {
				modifiedBids = append(modifiedBids, typedBid)
			}
			continue
		}

		bid := typedBid.Bid

		// Skip non-video bids — explicit type check prevents banner/native from being parsed as VAST
		if typedBid.BidType != openrtb_ext.BidTypeVideo || bid.AdM == "" {
			if modifiedBids != nil {
				modifiedBids = append(modifiedBids, typedBid)
			}
			continue
		}

		// Try to parse as VAST
		vastDoc, err := model.ParseVastAdm(bid.AdM)
		if err != nil {
			// Not valid VAST, skip enrichment
			if modifiedBids != nil {
				modifiedBids = append(modifiedBids, typedBid)
			}
			continue
		}

		// Resolve the actual DSP currency — BidderResponse.Currency is the authoritative source
		bidCurrency := payload.BidderResponse.Currency
		if bidCurrency == "" {
			bidCurrency = receiverCfg.DefaultCurrency
		}
		if bidCurrency == "" {
			bidCurrency = "USD"
		}

		// Build bid context for enrichment
		bidContext := CanonicalMeta{
			BidID:    bid.ID,
			Price:    bid.Price,
			Currency: bidCurrency,
			Adomain:  primaryDomain(bid.ADomain),
			Cats:     bid.Cat,
			Seat:     payload.Bidder,
		}

		// Extract the first Ad element and delegate enrichment to the injected Enricher.
		// The enricher (set via EnricherFactory) handles Duration, Categories,
		// AdvertiserPlacement, PricingPlacement, and debug extensions — BUG 3 fix.
		ad := model.ExtractFirstAd(vastDoc)
		if ad == nil {
			if modifiedBids != nil {
				modifiedBids = append(modifiedBids, typedBid)
			}
			continue
		}

		if _, enrichErr := enricher.Enrich(ad, bidContext, receiverCfg); enrichErr != nil {
			// Enrichment failed — keep original bid
			if modifiedBids != nil {
				modifiedBids = append(modifiedBids, typedBid)
			}
			continue
		}

		// Format back to XML
		xmlBytes, err := vastDoc.Marshal()
		if err != nil {
			// Keep original bid on format error
			if modifiedBids != nil {
				modifiedBids = append(modifiedBids, typedBid)
			}
			continue
		}

		// First enrichment: lazily allocate and back-fill all preceding original bids
		if modifiedBids == nil {
			modifiedBids = make([]*adapters.TypedBid, i, len(payload.BidderResponse.Bids))
			copy(modifiedBids, payload.BidderResponse.Bids[:i])
		}

		// Create new bid with enriched VAST
		enrichedBid := &openrtb2.Bid{}
		*enrichedBid = *bid
		enrichedBid.AdM = string(xmlBytes)

		// Create new TypedBid with enriched bid — preserve BidMeta for analytics/targeting
		enrichedTypedBid := &adapters.TypedBid{
			Bid:          enrichedBid,
			BidType:      typedBid.BidType,
			BidVideo:     typedBid.BidVideo,
			BidMeta:      typedBid.BidMeta,
			DealPriority: typedBid.DealPriority,
			Seat:         typedBid.Seat,
		}
		modifiedBids = append(modifiedBids, enrichedTypedBid)
	}

	// If we made changes, set mutation via ChangeSet
	if modifiedBids != nil {
		changeSet := hookstage.ChangeSet[hookstage.RawBidderResponsePayload]{}
		changeSet.RawBidderResponse().Bids().UpdateBids(modifiedBids)
		result.ChangeSet = changeSet
	}

	return result, nil
}

// primaryDomain returns the first domain from a slice, or empty string if none.
// VAST <Advertiser> is a single human-readable string — joining multiple domains is non-standard.
func primaryDomain(domains []string) string {
	if len(domains) == 0 {
		return ""
	}
	return domains[0]
}

// hookEnricher is a minimal fallback enricher used when EnricherFactory is not overridden.
// In production the enrich subpackage registers its VastEnricher via init().
// This fallback handles only Pricing and Advertiser to remain backward-compatible.
type hookEnricher struct{}

func (h *hookEnricher) Enrich(ad *model.Ad, meta CanonicalMeta, cfg ReceiverConfig) ([]string, error) {
	if ad == nil || ad.InLine == nil {
		return nil, nil
	}
	inline := ad.InLine

	// Pricing
	if inline.Pricing == nil && meta.Price > 0 {
		currency := meta.Currency
		if currency == "" {
			currency = cfg.DefaultCurrency
		}
		if currency == "" {
			currency = "USD"
		}
		inline.Pricing = &model.Pricing{
			Value:    model.FormatPrice(meta.Price),
			Model:    "CPM",
			Currency: currency,
		}
	}

	// Advertiser
	if strings.TrimSpace(inline.Advertiser) == "" && meta.Adomain != "" {
		inline.Advertiser = meta.Adomain
	}

	return nil, nil
}
