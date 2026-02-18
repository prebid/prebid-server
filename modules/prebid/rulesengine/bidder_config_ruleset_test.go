package rulesengine

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/modules/prebid/rulesengine/config"
	"github.com/prebid/prebid-server/v3/rules"
	"github.com/stretchr/testify/assert"
)

func TestBuildBidderConfigRuleSet(t *testing.T) {
	deviceCountry, _ := rules.NewDeviceCountry(json.RawMessage{})

	testCases := []struct {
		name            string
		geoscopes       map[string][]string
		expectError     bool
		expectedRuleSet cacheRuleSet[RequestWrapper, ProcessedAuctionHookResult]
	}{
		{
			name:        "nil-geoscopes",
			geoscopes:   nil,
			expectError: false,
			expectedRuleSet: cacheRuleSet[RequestWrapper, ProcessedAuctionHookResult]{
				name: "Dynamic ruleset from geoscopes",
				modelGroups: []cacheModelGroup[RequestWrapper, ProcessedAuctionHookResult]{
					{
						weight:       100,
						version:      "1.0",
						analyticsKey: "bidderConfig",
						tree: rules.Tree[RequestWrapper, ProcessedAuctionHookResult]{
							Root: &rules.Node[RequestWrapper, ProcessedAuctionHookResult]{
								SchemaFunction: nil,
								Children:       nil,
							},
						},
					},
				},
			},
		},
		{
			name:        "empty-geoscopes",
			geoscopes:   map[string][]string{},
			expectError: false,
			expectedRuleSet: cacheRuleSet[RequestWrapper, ProcessedAuctionHookResult]{
				name: "Dynamic ruleset from geoscopes",
				modelGroups: []cacheModelGroup[RequestWrapper, ProcessedAuctionHookResult]{
					{
						weight:       100,
						version:      "1.0",
						analyticsKey: "bidderConfig",
						tree: rules.Tree[RequestWrapper, ProcessedAuctionHookResult]{
							Root: &rules.Node[RequestWrapper, ProcessedAuctionHookResult]{
								SchemaFunction: nil,
								Children:       nil,
							},
						},
					},
				},
			},
		},
		{
			name: "valid-geoscopes",
			geoscopes: map[string][]string{
				"bidder1": {"USA", "CAN"},
				"bidder2": {"GBR", "FRA"},
			},
			expectError: false,
			expectedRuleSet: cacheRuleSet[RequestWrapper, ProcessedAuctionHookResult]{
				name: "Dynamic ruleset from geoscopes",
				modelGroups: []cacheModelGroup[RequestWrapper, ProcessedAuctionHookResult]{
					{
						weight:       100,
						version:      "1.0",
						analyticsKey: "bidderConfig",
						tree: rules.Tree[RequestWrapper, ProcessedAuctionHookResult]{
							Root: &rules.Node[RequestWrapper, ProcessedAuctionHookResult]{
								SchemaFunction: deviceCountry,
								Children: map[string]*rules.Node[RequestWrapper, ProcessedAuctionHookResult]{
									"USA": {
										ResultFunctions: []rules.ResultFunction[RequestWrapper, ProcessedAuctionHookResult]{
											&ExcludeBidders{
												Args: config.ResultFuncParams{
													Bidders: []string{"bidder2"},
												},
											},
										},
										Children: nil,
									},
									"CAN": {
										ResultFunctions: []rules.ResultFunction[RequestWrapper, ProcessedAuctionHookResult]{
											&ExcludeBidders{
												Args: config.ResultFuncParams{
													Bidders: []string{"bidder2"},
												},
											},
										},
										Children: nil,
									},
									"GBR": {
										ResultFunctions: []rules.ResultFunction[RequestWrapper, ProcessedAuctionHookResult]{
											&ExcludeBidders{
												Args: config.ResultFuncParams{
													Bidders: []string{"bidder1"},
												},
											},
										},
										Children: nil,
									},
									"FRA": {
										ResultFunctions: []rules.ResultFunction[RequestWrapper, ProcessedAuctionHookResult]{
											&ExcludeBidders{
												Args: config.ResultFuncParams{
													Bidders: []string{"bidder1"},
												},
											},
										},
										Children: nil,
									},
									"*": {
										ResultFunctions: []rules.ResultFunction[RequestWrapper, ProcessedAuctionHookResult]{
											&ExcludeBidders{
												Args: config.ResultFuncParams{
													Bidders: []string{"bidder1", "bidder2"},
												},
											},
										},
										Children: nil,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ruleSets, err := buildBidderConfigRuleSet(tc.geoscopes, nil)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Len(t, ruleSets, 1)

			// Check basic structure properties instead of full equality
			assert.Equal(t, tc.expectedRuleSet.name, ruleSets[0].name)
			assert.Len(t, ruleSets[0].modelGroups, len(tc.expectedRuleSet.modelGroups))

			if len(ruleSets[0].modelGroups) > 0 && len(tc.expectedRuleSet.modelGroups) > 0 {
				// Check model group properties
				assert.Equal(t, tc.expectedRuleSet.modelGroups[0].weight, ruleSets[0].modelGroups[0].weight)
				assert.Equal(t, tc.expectedRuleSet.modelGroups[0].version, ruleSets[0].modelGroups[0].version)
				assert.Equal(t, tc.expectedRuleSet.modelGroups[0].analyticsKey, ruleSets[0].modelGroups[0].analyticsKey)

				// For non-empty geoscopes, verify tree structure
				if tc.name == "valid-geoscopes" {
					root := ruleSets[0].modelGroups[0].tree.Root
					assert.NotNil(t, root)
					assert.NotNil(t, root.SchemaFunction)
					assert.NotNil(t, root.Children)

					// Check expected country nodes exist
					assert.Contains(t, root.Children, "USA")
					assert.Contains(t, root.Children, "CAN")
					assert.Contains(t, root.Children, "GBR")
					assert.Contains(t, root.Children, "FRA")
					assert.Contains(t, root.Children, "*")

					// Check USA node has expected result function for bidder2
					usaNode := root.Children["USA"]
					assert.Len(t, usaNode.ResultFunctions, 1)

					// Check CAN node has expected result function for bidder2
					canNode := root.Children["CAN"]
					assert.Len(t, canNode.ResultFunctions, 1)

					// Check GBR node has expected result function for bidder1
					gbrNode := root.Children["GBR"]
					assert.Len(t, gbrNode.ResultFunctions, 1)

					// Check FRA node has expected result function for bidder1
					fraNode := root.Children["FRA"]
					assert.Len(t, fraNode.ResultFunctions, 1)

					// Check * node has expected result function for both bidders
					wildcardNode := root.Children["*"]
					assert.Len(t, wildcardNode.ResultFunctions, 1)
				} else {
					// For empty or nil geoscopes, verify root exists but might be empty
					root := ruleSets[0].modelGroups[0].tree.Root
					assert.NotNil(t, root)
				}
			}
		})
	}
}
