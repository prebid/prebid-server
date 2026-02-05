// Package enrich provides VAST ad enrichment capabilities.
package enrich

import (
	"fmt"
	"strings"

	"github.com/prebid/prebid-server/v3/modules/prebid/ctv_vast_enrichment"
	"github.com/prebid/prebid-server/v3/modules/prebid/ctv_vast_enrichment/model"
)

// VastEnricher implements the Enricher interface.
// It uses CollisionPolicy "VAST_WINS" - existing VAST values are not overwritten.
type VastEnricher struct{}

// NewEnricher creates a new VastEnricher instance.
func NewEnricher() *VastEnricher {
	return &VastEnricher{}
}

// Enrich adds tracking, extensions, and other data to a VAST ad.
// It implements the vast.Enricher interface.
// CollisionPolicy "VAST_WINS": existing values in VAST are preserved.
func (e *VastEnricher) Enrich(ad *model.Ad, meta vast.CanonicalMeta, cfg vast.ReceiverConfig) ([]string, error) {
	var warnings []string

	if ad == nil {
		return warnings, nil
	}

	// Only enrich InLine ads, not Wrapper ads
	if ad.InLine == nil {
		warnings = append(warnings, "skipping enrichment: ad is not InLine")
		return warnings, nil
	}

	inline := ad.InLine

	// Ensure Extensions exists for adding extension-based enrichments
	if inline.Extensions == nil {
		inline.Extensions = &model.Extensions{}
	}

	// Enrich Pricing
	pricingWarnings := e.enrichPricing(inline, meta, cfg)
	warnings = append(warnings, pricingWarnings...)

	// Enrich Advertiser
	advertiserWarnings := e.enrichAdvertiser(inline, meta, cfg)
	warnings = append(warnings, advertiserWarnings...)

	// Enrich Duration
	durationWarnings := e.enrichDuration(inline, meta)
	warnings = append(warnings, durationWarnings...)

	// Enrich Categories (always as extension)
	categoryWarnings := e.enrichCategories(inline, meta)
	warnings = append(warnings, categoryWarnings...)

	// Add debug extension if enabled
	if cfg.Debug || cfg.Placement.Debug {
		e.addDebugExtension(inline, meta)
	}

	return warnings, nil
}

// enrichPricing adds pricing information if not present.
// VAST_WINS: only adds if InLine.Pricing is nil or empty.
func (e *VastEnricher) enrichPricing(inline *model.InLine, meta vast.CanonicalMeta, cfg vast.ReceiverConfig) []string {
	var warnings []string

	// Skip if no price to add
	if meta.Price <= 0 {
		return warnings
	}

	// Check collision policy - VAST_WINS means don't overwrite existing
	if inline.Pricing != nil && inline.Pricing.Value != "" {
		warnings = append(warnings, "pricing: VAST_WINS - keeping existing pricing")
		return warnings
	}

	// Format the price value
	priceStr := formatPrice(meta.Price)
	currency := meta.Currency
	if currency == "" {
		currency = cfg.DefaultCurrency
	}
	if currency == "" {
		currency = "USD"
	}

	// Determine placement location
	placement := cfg.Placement.PricingPlacement
	if placement == "" {
		placement = vast.PlacementVastPricing
	}

	switch placement {
	case vast.PlacementVastPricing:
		inline.Pricing = &model.Pricing{
			Model:    "CPM",
			Currency: currency,
			Value:    priceStr,
		}
	case vast.PlacementExtension:
		ext := model.ExtensionXML{
			Type:     "pricing",
			InnerXML: fmt.Sprintf("<Price model=\"CPM\" currency=\"%s\">%s</Price>", currency, priceStr),
		}
		inline.Extensions.Extension = append(inline.Extensions.Extension, ext)
	default:
		// Default to VAST_PRICING
		inline.Pricing = &model.Pricing{
			Model:    "CPM",
			Currency: currency,
			Value:    priceStr,
		}
	}

	return warnings
}

// enrichAdvertiser adds advertiser information if not present.
// VAST_WINS: only adds if InLine.Advertiser is empty.
func (e *VastEnricher) enrichAdvertiser(inline *model.InLine, meta vast.CanonicalMeta, cfg vast.ReceiverConfig) []string {
	var warnings []string

	// Skip if no advertiser to add
	if meta.Adomain == "" {
		return warnings
	}

	// Check collision policy - VAST_WINS means don't overwrite existing
	if strings.TrimSpace(inline.Advertiser) != "" {
		warnings = append(warnings, "advertiser: VAST_WINS - keeping existing advertiser")
		return warnings
	}

	// Determine placement location
	placement := cfg.Placement.AdvertiserPlacement
	if placement == "" {
		placement = vast.PlacementAdvertiserTag
	}

	switch placement {
	case vast.PlacementAdvertiserTag:
		inline.Advertiser = meta.Adomain
	case vast.PlacementExtension:
		ext := model.ExtensionXML{
			Type:     "advertiser",
			InnerXML: fmt.Sprintf("<Advertiser>%s</Advertiser>", escapeXML(meta.Adomain)),
		}
		inline.Extensions.Extension = append(inline.Extensions.Extension, ext)
	default:
		// Default to ADVERTISER_TAG
		inline.Advertiser = meta.Adomain
	}

	return warnings
}

// enrichDuration adds duration to Linear creative if not present.
// VAST_WINS: only adds if Linear.Duration is empty.
func (e *VastEnricher) enrichDuration(inline *model.InLine, meta vast.CanonicalMeta) []string {
	var warnings []string

	// Skip if no duration to add
	if meta.DurSec <= 0 {
		return warnings
	}

	// Find the Linear creative
	if inline.Creatives == nil || len(inline.Creatives.Creative) == 0 {
		return warnings
	}

	for i := range inline.Creatives.Creative {
		creative := &inline.Creatives.Creative[i]
		if creative.Linear == nil {
			continue
		}

		// Check collision policy - VAST_WINS means don't overwrite existing
		if strings.TrimSpace(creative.Linear.Duration) != "" {
			warnings = append(warnings, "duration: VAST_WINS - keeping existing duration")
			continue
		}

		// Set duration in HH:MM:SS format
		creative.Linear.Duration = model.SecToHHMMSS(meta.DurSec)
	}

	return warnings
}

// enrichCategories adds IAB categories as an extension.
func (e *VastEnricher) enrichCategories(inline *model.InLine, meta vast.CanonicalMeta) []string {
	var warnings []string

	// Skip if no categories to add
	if len(meta.Cats) == 0 {
		return warnings
	}

	// Build category extension XML
	var categoryXML strings.Builder
	for _, cat := range meta.Cats {
		categoryXML.WriteString(fmt.Sprintf("<Category>%s</Category>", escapeXML(cat)))
	}

	ext := model.ExtensionXML{
		Type:     "iab_category",
		InnerXML: categoryXML.String(),
	}
	inline.Extensions.Extension = append(inline.Extensions.Extension, ext)

	return warnings
}

// addDebugExtension adds OpenRTB debug information as an extension.
func (e *VastEnricher) addDebugExtension(inline *model.InLine, meta vast.CanonicalMeta) {
	var debugXML strings.Builder
	debugXML.WriteString(fmt.Sprintf("<BidID>%s</BidID>", escapeXML(meta.BidID)))
	debugXML.WriteString(fmt.Sprintf("<ImpID>%s</ImpID>", escapeXML(meta.ImpID)))
	if meta.DealID != "" {
		debugXML.WriteString(fmt.Sprintf("<DealID>%s</DealID>", escapeXML(meta.DealID)))
	}
	debugXML.WriteString(fmt.Sprintf("<Seat>%s</Seat>", escapeXML(meta.Seat)))
	debugXML.WriteString(fmt.Sprintf("<Price>%s</Price>", formatPrice(meta.Price)))
	debugXML.WriteString(fmt.Sprintf("<Currency>%s</Currency>", escapeXML(meta.Currency)))

	ext := model.ExtensionXML{
		Type:     "openrtb",
		InnerXML: debugXML.String(),
	}
	inline.Extensions.Extension = append(inline.Extensions.Extension, ext)
}

// formatPrice formats a price value with appropriate precision.
func formatPrice(price float64) string {
	// Use up to 4 decimal places, trimming trailing zeros
	s := fmt.Sprintf("%.4f", price)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" {
		return "0"
	}
	return s
}

// escapeXML escapes special characters for XML content.
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// Ensure VastEnricher implements Enricher interface.
var _ vast.Enricher = (*VastEnricher)(nil)
