package rulesengine

import (
	"github.com/prebid/prebid-server/v4/rules"
)

// buildBidderConfigRuleSet builds a dynamic ruleset based on the geoscope annotations in the
// static bidder-info bidder YAML files
func buildBidderConfigRuleSet(geoscopes map[string][]string, setDefinitions map[string][]string) ([]cacheRuleSet[RequestWrapper, ProcessedAuctionHookResult], error) {
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

	builder := NewBidderConfigRuleSetBuilder[RequestWrapper, ProcessedAuctionHookResult](geoscopes, setDefinitions)

	tree, err := rules.NewTree[RequestWrapper, ProcessedAuctionHookResult](builder)
	if err != nil {
		return nil, err
	}
	crs.modelGroups[0].tree = *tree
	// Propagate the analytics key and model version onto the tree so they are available in the
	// ResultFunctionMeta at execution time (e.g. for surfacing them in exclusion warnings).
	crs.modelGroups[0].tree.AnalyticsKey = crs.modelGroups[0].analyticsKey
	crs.modelGroups[0].tree.ModelVersion = crs.modelGroups[0].version

	return []cacheRuleSet[RequestWrapper, ProcessedAuctionHookResult]{crs}, nil
}
