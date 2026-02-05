// Package format provides VAST XML formatting capabilities.
package format

import (
	"encoding/xml"

	"github.com/prebid/prebid-server/v3/modules/prebid/ctv_vast_enrichment"
	"github.com/prebid/prebid-server/v3/modules/prebid/ctv_vast_enrichment/model"
)

// VastFormatter implements the Formatter interface for GAM_SSU and other receivers.
type VastFormatter struct{}

// NewFormatter creates a new VastFormatter instance.
func NewFormatter() *VastFormatter {
	return &VastFormatter{}
}

// Format converts enriched VAST ads into XML output.
// It implements the vast.Formatter interface.
//
// For each EnrichedAd, it creates one <Ad> element with:
// - id attribute from meta.AdID if available, else meta.BidID
// - sequence attribute from EnrichedAd.Sequence (if multiple ads)
// - The enriched InLine subtree from the ad
func (f *VastFormatter) Format(ads []vast.EnrichedAd, cfg vast.ReceiverConfig) ([]byte, []string, error) {
	var warnings []string

	// Determine VAST version
	version := cfg.VastVersionDefault
	if version == "" {
		version = "4.0"
	}

	// Handle no-ad case
	if len(ads) == 0 {
		noAdXML := model.BuildNoAdVast(version)
		return noAdXML, warnings, nil
	}

	// Build the VAST document
	vastDoc := model.Vast{
		Version: version,
		Ads:     make([]model.Ad, 0, len(ads)),
	}

	isPod := len(ads) > 1

	for _, enriched := range ads {
		if enriched.Ad == nil {
			warnings = append(warnings, "skipping nil ad in format")
			continue
		}

		// Create a copy of the ad to avoid modifying the original
		ad := copyAd(enriched.Ad)

		// Set Ad.ID from meta (prefer AdID if tracked, else BidID)
		ad.ID = deriveAdID(enriched.Meta)

		// Set sequence attribute for pods (multiple ads)
		if isPod && enriched.Sequence > 0 {
			ad.Sequence = enriched.Sequence
		} else if !isPod {
			ad.Sequence = 0 // Don't set sequence for single ad
		}

		vastDoc.Ads = append(vastDoc.Ads, *ad)
	}

	// Handle case where all ads were nil
	if len(vastDoc.Ads) == 0 {
		noAdXML := model.BuildNoAdVast(version)
		warnings = append(warnings, "all ads were nil, returning no-ad VAST")
		return noAdXML, warnings, nil
	}

	// Marshal with indentation
	xmlBytes, err := xml.MarshalIndent(vastDoc, "", "  ")
	if err != nil {
		return nil, warnings, err
	}

	// Add XML declaration
	output := append([]byte(xml.Header), xmlBytes...)

	return output, warnings, nil
}

// deriveAdID determines the Ad ID from metadata.
// Uses BidID as the identifier (AdID is not currently tracked in CanonicalMeta).
func deriveAdID(meta vast.CanonicalMeta) string {
	// BidID is the primary identifier
	if meta.BidID != "" {
		return meta.BidID
	}
	// Fallback to ImpID if BidID is empty
	if meta.ImpID != "" {
		return "imp-" + meta.ImpID
	}
	return ""
}

// copyAd creates a shallow copy of an Ad to avoid modifying the original.
func copyAd(src *model.Ad) *model.Ad {
	if src == nil {
		return nil
	}
	ad := *src
	return &ad
}

// Ensure VastFormatter implements Formatter interface.
var _ vast.Formatter = (*VastFormatter)(nil)
