package rulesengine

import (
	"github.com/prebid/prebid-server/v3/rules"
)

// buildBidderConfigRuleSet builds a dynamic ruleset based on the geoscope annotations in the
// static bidder-info bidder YAML files
func buildBidderConfigRuleSet(geoscopes map[string][]string) ([]cacheRuleSet[RequestWrapper, ProcessedAuctionHookResult], error) {
	crs := cacheRuleSet[RequestWrapper, ProcessedAuctionHookResult]{
		name: "Dynamic ruleset from geoscopes",
		modelGroups: []cacheModelGroup[RequestWrapper, ProcessedAuctionHookResult]{
			{
				weight:       100,
				version:      "1.0",
				analyticsKey: "bidderConfig",
			},
		},
	}

	builder := NewBidderConfigRuleSetBuilder[RequestWrapper, ProcessedAuctionHookResult](geoscopes)

	tree, err := rules.NewTree[RequestWrapper, ProcessedAuctionHookResult](builder)
	if err != nil {
		return nil, err
	}
	crs.modelGroups[0].tree = *tree

	return []cacheRuleSet[RequestWrapper, ProcessedAuctionHookResult]{crs}, nil
}
