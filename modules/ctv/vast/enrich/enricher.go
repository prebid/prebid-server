package enrich

import (
	"github.com/prebid/prebid-server/v3/modules/ctv/vast"
	"github.com/prebid/prebid-server/v3/modules/ctv/vast/model"
)

// Enricher implements the Enricher interface
type Enricher struct{}

// NewEnricher creates a new VAST enricher
func NewEnricher() vast.Enricher {
	return &Enricher{}
}

// Enrich implements Enricher.Enrich
func (e *Enricher) Enrich(v *model.VastAd, meta vast.CanonicalMeta, cfg vast.ReceiverConfig) ([]string, error) {
	// TODO: Implement enrichment logic
	// - Add pricing based on PlacementRules
	// - Add advertiser domain based on PlacementRules
	// - Add categories based on PlacementRules
	// - Add debug info if enabled
	// - Apply collision policy
	// - Handle inline vs extensions placement
	return nil, nil
}
