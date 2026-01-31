package vast

import (
"github.com/prebid/prebid-server/v3/modules/ctv/vast/model"
)

// VastEnricher implements the Enricher interface
type VastEnricher struct{}

// NewEnricher creates a new VAST enricher
func NewEnricher() *VastEnricher {
return &VastEnricher{}
}

// Enrich implements Enricher.Enrich
func (e *VastEnricher) Enrich(ad *model.VastAd, meta CanonicalMeta, cfg ReceiverConfig) ([]string, error) {
// ad is already the correct type
_ = ad

// TODO: Implement enrichment logic
// - Add pricing based on PlacementRules
// - Add advertiser domain based on PlacementRules
// - Add categories based on PlacementRules
// - Add debug info if enabled
// - Apply collision policy
// - Handle inline vs extensions placement
return nil, nil
}
