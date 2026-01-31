package format

import (
	"encoding/xml"
	"fmt"

	"github.com/prebid/prebid-server/v3/modules/ctv/vast/core"
	"github.com/prebid/prebid-server/v3/modules/ctv/vast/model"
)

// Formatter implements the Formatter interface for GAM_SSU receiver
type Formatter struct{}

// NewFormatter creates a new VAST formatter
func NewFormatter() core.Formatter {
	return &Formatter{}
}

// Format implements Formatter.Format for GAM_SSU receiver
// Takes a list of enriched VAST ads and combines them into a single VAST XML document.
// For each ad:
// - Creates one <Ad> element
// - Sets Ad.id from the ad's existing ID (should be set during enrichment to meta.AdID or meta.BidID)
// - Sets sequence attribute if multiple ads are present
// - Preserves the enriched InLine subtree and any tracking InnerXML
func (f *Formatter) Format(ads []*model.VastAd, cfg core.ReceiverConfig) ([]byte, []string, error) {
	if len(ads) == 0 {
		// Return empty VAST with no ads
		return model.BuildNoAdVast(cfg.VastVersionDefault), nil, nil
	}

	// Determine VAST version - use config default or "3.0" if not specified
	version := cfg.VastVersionDefault
	if version == "" {
		version = "3.0"
	}

	// Build the root VAST structure
	vast := &model.Vast{
		Version: version,
		Ads:     make([]model.Ad, 0, len(ads)),
	}

	var warnings []string

	// Add each ad to the VAST response
	for i, ad := range ads {
		if ad == nil {
			warnings = append(warnings, fmt.Sprintf("skipping nil ad at position %d", i))
			continue
		}

		// Create a copy of the ad to avoid modifying the input
		adCopy := *ad

		// Set sequence attribute if we have multiple ads (1-indexed)
		if len(ads) > 1 {
			// If sequence is already set (non-zero), preserve it; otherwise use position
			if adCopy.Sequence == 0 {
				adCopy.Sequence = i + 1
			}
		} else {
			// Single ad - no sequence needed
			adCopy.Sequence = 0
		}

		// Validate that the ad has InLine content
		if adCopy.InLine == nil {
			warnings = append(warnings, fmt.Sprintf("ad %s has no InLine content", adCopy.ID))
		}

		vast.Ads = append(vast.Ads, adCopy)
	}

	// Marshal to XML with indentation
	xmlBytes, err := xml.MarshalIndent(vast, "", "  ")
	if err != nil {
		return nil, warnings, fmt.Errorf("failed to marshal VAST XML: %w", err)
	}

	// Prepend XML header
	result := append([]byte(xml.Header), xmlBytes...)

	return result, warnings, nil
}
