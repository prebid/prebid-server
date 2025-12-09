package rulesengine

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/prebid/prebid-server/v3/hooks"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/prebid/rulesengine/config"
	"github.com/prebid/prebid-server/v3/rules"
	"github.com/stretchr/testify/assert"
)

func TestNewCacheEntry(t *testing.T) {
	testCases := []struct {
		name                 string
		inCfg                *config.PbRulesEngine
		inCfgRaw             *json.RawMessage
		expectedRulesetCount int
		expectedErr          error
	}{
		{
			name:                 "nil-ruleset-config",
			expectedRulesetCount: 0,
			expectedErr:          errors.New("no rules engine configuration provided"),
		},
		{
			name:                 "nil-cfgRaw",
			inCfg:                &config.PbRulesEngine{},
			expectedRulesetCount: 0,
			expectedErr:          errors.New("Can't create identifier hash from empty raw json configuration"),
		},
		{
			name:                 "nil-ruleset-array",
			inCfg:                &config.PbRulesEngine{},
			inCfgRaw:             getValidJsonConfig(),
			expectedRulesetCount: 0,
			expectedErr:          nil,
		},
		{
			name: "empty-ruleset-array",
			inCfg: &config.PbRulesEngine{
				RuleSets: []config.RuleSet{},
			},
			inCfgRaw:             getValidJsonConfig(),
			expectedRulesetCount: 0,
			expectedErr:          nil,
		},
		{
			name: "static-ruleset-with-wrong-stage",
			inCfg: &config.PbRulesEngine{
				RuleSets: []config.RuleSet{
					{Stage: "wrong-stage"},
				},
			},
			inCfgRaw:             getValidJsonConfig(),
			expectedRulesetCount: 0,
		},
		{
			name: "createCacheRuleSet-throws-error",
			inCfg: &config.PbRulesEngine{
				RuleSets: []config.RuleSet{
					{
						Stage: hooks.StageProcessedAuctionRequest,
						ModelGroups: []config.ModelGroup{
							{
								Default: []config.Result{
									{
										Func: "unknownResultFunction",
									},
								},
							},
						},
					},
				},
			},
			inCfgRaw:             getValidJsonConfig(),
			expectedRulesetCount: 0,
		},
		{
			name: "dynamic-ruleset",
			inCfg: &config.PbRulesEngine{
				GenerateRulesFromBidderConfig: true,
				RuleSets:                      []config.RuleSet{},
			},
			inCfgRaw:             getValidJsonConfig(),
			expectedRulesetCount: 1,
		},
		{
			name: "single-static-ruleset",
			inCfg: &config.PbRulesEngine{
				RuleSets: []config.RuleSet{
					{
						Stage: hooks.StageProcessedAuctionRequest,
						ModelGroups: []config.ModelGroup{
							{
								Default: []config.Result{
									{
										Func: ExcludeBiddersName,
										Args: json.RawMessage(`{"bidders": ["bidderA"], "seatNonBid": 111}`),
									},
								},
							},
						},
					},
				},
			},
			inCfgRaw:             getValidJsonConfig(),
			expectedRulesetCount: 1,
		},
		{
			name: "single-static-ruleset-with-dynamic-ruleset",
			inCfg: &config.PbRulesEngine{
				GenerateRulesFromBidderConfig: true,
				RuleSets: []config.RuleSet{
					{
						Stage: hooks.StageProcessedAuctionRequest,
						ModelGroups: []config.ModelGroup{
							{
								Default: []config.Result{
									{
										Func: ExcludeBiddersName,
										Args: json.RawMessage(`{"bidders": ["bidderA"], "seatNonBid": 111}`),
									},
								},
							},
						},
					},
				},
			},
			inCfgRaw:             getValidJsonConfig(),
			expectedRulesetCount: 2,
		},
		{
			name: "multiple-static-rulesets-some-with-the-wrong-stage",
			inCfg: &config.PbRulesEngine{
				RuleSets: []config.RuleSet{
					{Stage: "wrong-stage"},
					{
						Stage: hooks.StageProcessedAuctionRequest,
						ModelGroups: []config.ModelGroup{
							{
								Default: []config.Result{
									{
										Func: ExcludeBiddersName,
										Args: json.RawMessage(`{"bidders": ["bidderA"], "seatNonBid": 111}`),
									},
								},
							},
						},
					},
				},
			},
			inCfgRaw:             getValidJsonConfig(),
			expectedRulesetCount: 1,
		},
		{
			name: "Multiple-entries-with-supported-rulesets",
			inCfg: &config.PbRulesEngine{
				RuleSets: []config.RuleSet{
					{
						Stage: hooks.StageProcessedAuctionRequest,
						ModelGroups: []config.ModelGroup{
							{
								Default: []config.Result{
									{
										Func: IncludeBiddersName,
										Args: json.RawMessage(`{"bidders": ["bidderFoo"], "seatNonBid": 505}`),
									},
								},
							},
						},
					},
					{
						Stage: hooks.StageProcessedAuctionRequest,
						ModelGroups: []config.ModelGroup{
							{
								Default: []config.Result{
									{
										Func: ExcludeBiddersName,
										Args: json.RawMessage(`{"bidders": ["bidderBar"], "seatNonBid": 111}`),
									},
								},
							},
						},
					},
				},
			},
			inCfgRaw:             getValidJsonConfig(),
			expectedRulesetCount: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cacheEntry, err := NewCacheEntry(tc.inCfg, tc.inCfgRaw, map[string][]string{})

			assert.Len(t, cacheEntry.ruleSetsForProcessedAuctionRequestStage, tc.expectedRulesetCount)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

func TestCreateCacheRuleSet(t *testing.T) {
	testCases := []struct {
		name            string
		in              *config.RuleSet
		expectedRuleSet cacheRuleSet[hookstage.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]
		expectedErr     error
	}{
		{
			name:            "nil-ruleset-config",
			in:              nil,
			expectedRuleSet: cacheRuleSet[hookstage.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{},
			expectedErr:     errors.New("no rules engine configuration provided"),
		},
		{
			name: "nil-modelgroup-array",
			in: &config.RuleSet{
				ModelGroups: nil,
			},
			expectedRuleSet: cacheRuleSet[hookstage.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{modelGroups: []cacheModelGroup[hookstage.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{}},
			expectedErr:     nil,
		},
		{
			name: "modelgroup-array-empty",
			in: &config.RuleSet{
				ModelGroups: []config.ModelGroup{},
			},
			expectedRuleSet: cacheRuleSet[hookstage.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{modelGroups: []cacheModelGroup[hookstage.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{}},
			expectedErr:     nil,
		},
		{
			name: "invalid-model-group",
			in: &config.RuleSet{
				ModelGroups: []config.ModelGroup{
					{
						Default: []config.Result{
							{
								Func: "unknownResultFunction",
							},
						},
					},
				},
			},
			expectedRuleSet: cacheRuleSet[hookstage.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{modelGroups: []cacheModelGroup[hookstage.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{}},
			expectedErr:     errors.New("result function unknownResultFunction was not created"),
		},
		{
			name: "one-valid-modelgroup",
			in: &config.RuleSet{
				ModelGroups: []config.ModelGroup{
					{
						Default: []config.Result{
							{
								Func: ExcludeBiddersName,
								Args: json.RawMessage(`{"bidders": ["bidderA"], "seatNonBid": 111}`),
							},
						},
					},
				},
			},
			expectedRuleSet: cacheRuleSet[hookstage.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{
				modelGroups: []cacheModelGroup[hookstage.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{
					{
						tree: rules.Tree[hookstage.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{
							Root: &rules.Node[hookstage.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{},
							DefaultFunctions: []rules.ResultFunction[hookstage.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{
								&ExcludeBidders{
									Args: config.ResultFuncParams{
										Bidders:    []string{"bidderA"},
										SeatNonBid: 111,
									},
								},
							},
						},
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "valid-and-invalid-model-groups",
			in: &config.RuleSet{
				ModelGroups: []config.ModelGroup{
					{
						Default: []config.Result{
							{Func: "unknownResultFunction"},
						},
					},
					{
						Default: []config.Result{
							{
								Func: ExcludeBiddersName,
								Args: json.RawMessage(`{"bidders": ["bidderA"], "seatNonBid": 111}`),
							},
						},
					},
				},
			},
			expectedRuleSet: cacheRuleSet[hookstage.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{modelGroups: []cacheModelGroup[hookstage.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{}},
			expectedErr:     errors.New("result function unknownResultFunction was not created"),
		},
		{
			name: "multiple-valid-model-groups",
			in: &config.RuleSet{
				ModelGroups: []config.ModelGroup{
					{
						Default: []config.Result{
							{
								Func: ExcludeBiddersName,
								Args: json.RawMessage(`{"bidders": ["bidderFoo"], "seatNonBid": 111}`),
							},
						},
					},
					{
						Default: []config.Result{
							{
								Func: IncludeBiddersName,
								Args: json.RawMessage(`{"bidders": ["bidderBar"], "seatNonBid": 222}`),
							},
						},
					},
				},
			},
			expectedRuleSet: cacheRuleSet[hookstage.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{
				modelGroups: []cacheModelGroup[hookstage.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{
					{
						tree: rules.Tree[hookstage.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{
							Root: &rules.Node[hookstage.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{},
							DefaultFunctions: []rules.ResultFunction[hookstage.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{
								&ExcludeBidders{
									Args: config.ResultFuncParams{
										Bidders:    []string{"bidderFoo"},
										SeatNonBid: 111,
									},
								},
							},
						},
					},
					{
						tree: rules.Tree[hookstage.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{
							Root: &rules.Node[hookstage.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{},
							DefaultFunctions: []rules.ResultFunction[hookstage.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{
								&IncludeBidders{
									Args: config.ResultFuncParams{
										Bidders:    []string{"bidderBar"},
										SeatNonBid: 222,
									},
								},
							},
						},
					},
				},
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ruleset, err := createCacheRuleSet(tc.in)

			assert.Equal(t, tc.expectedRuleSet, ruleset)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

func TestHashConfig(t *testing.T) {
	emptyJSON := json.RawMessage(``)
	testCases := []struct {
		name            string
		in              *json.RawMessage
		expectEmptyHash bool
	}{
		{
			name:            "Nil-input",
			in:              nil,
			expectEmptyHash: true,
		},
		{
			name:            "Empty-input",
			in:              &emptyJSON,
			expectEmptyHash: true,
		},
		{
			name:            "Success",
			in:              getValidJsonConfig(),
			expectEmptyHash: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out := hashConfig(tc.in)

			if tc.expectEmptyHash {
				assert.Empty(t, out)
			} else {
				assert.NotEmpty(t, out)
			}
		})
	}
}

func getValidJsonConfig() *json.RawMessage {
	rv := json.RawMessage(`
  {
    "enabled": true,
    "generate_rules_from_bidderconfig": true,
    "timestamp": "20250131 00:00:00",
    "set_definitions": {
      "country_groups": {
        "EEA": ["FRA", "DEU"]
      }
    },
    "rulesets": [
      {
        "stage": "processed_auction_request",
        "name": "exclude-in-jpn",
        "version": "1234",
        "modelgroups": [
          {
            "weight": 100,
            "analyticsKey": "experiment-name",
            "version": "4567",
            "schema": [
              {
                "function": "deviceCountryIn",
                "args": {"countries": ["USA"]}
              },
              {
                "function": "dataCenterIn",
                "args": {"datacenters": ["us-east", "us-west"]}
              },
              {
                "function": "channel"
              }
            ],
            "default": [],
            "rules": [
              {
                "conditions": [
                  "true",
                  "true",
                  "amp"
                ],
                "results": [
                  {
                    "function": "excludeBidders",
                    "args": {"bidders": ["bidderA"], "seatNonBid": 111}
                  }
                ]
              },
              {
                "conditions": [
                  "true",
                  "false",
                  "web"
                ],
                "results": [
                  {
                    "function": "excludeBidders",
                    "args": {"bidders": ["bidderB"], "seatNonBid": 222}
                  }
                ]
              },
              {
                "conditions": [
                  "false",
                  "false",
                  "*"
                ],
                "results": [
                  {
                    "function": "includeBidders",
                    "args": {"bidders": ["bidderC"], "seatNonBid": 333}
                  }
                ]
              }
            ]
          },
          {
            "weight": 1,
            "analyticsKey": "experiment-name",
            "version": "3.0",
            "schema": [{"function": "channel"}],
            "rules": [
              {
                "conditions": ["*"],
                "results": [{"function": "includeBidders", "args": {"bidders": ["bidderC"], "seatNonBid": 333}}]
              }
            ]
          }
        ]
      }
    ]
  }
`)
	return &rv
}
