package rulesengine

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/prebid/prebid-server/v3/hooks"
	hs "github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/prebid/rulesengine/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/rules"
	"github.com/stretchr/testify/assert"
)

func TestNewCacheEntry(t *testing.T) {
	testCases := []struct {
		name                    string
		inCfg                   *config.PbRulesEngine
		inCfgRaw                *json.RawMessage
		expectEmptyRulesetArray bool
		expectedErr             error
	}{
		{
			name:                    "nil-ruleset-config",
			expectEmptyRulesetArray: true,
			expectedErr:             errors.New("no rules engine configuration provided"),
		},
		{
			name:                    "nil-cfgRaw",
			inCfg:                   &config.PbRulesEngine{},
			expectEmptyRulesetArray: true,
			expectedErr:             errors.New("Can't create identifier hash from empty raw json configuration"),
		},
		{
			name:                    "nil-ruleset-array",
			inCfg:                   &config.PbRulesEngine{},
			inCfgRaw:                getValidJsonConfig(),
			expectEmptyRulesetArray: true,
			expectedErr:             nil,
		},
		{
			name: "empty-ruleset-array",
			inCfg: &config.PbRulesEngine{
				RuleSets: []config.RuleSet{},
			},
			inCfgRaw:                getValidJsonConfig(),
			expectEmptyRulesetArray: true,
			expectedErr:             nil,
		},
		{
			name: "ruleset-with-wrong-stage",
			inCfg: &config.PbRulesEngine{
				RuleSets: []config.RuleSet{
					{Stage: "wrong-stage"},
				},
			},
			inCfgRaw:                getValidJsonConfig(),
			expectEmptyRulesetArray: true,
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
			inCfgRaw:                getValidJsonConfig(),
			expectEmptyRulesetArray: true,
		},
		{
			name: "single-ruleset-entry-right-stage",
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
			inCfgRaw:                getValidJsonConfig(),
			expectEmptyRulesetArray: false,
		},
		{
			name: "Multiple-ruleset-entries-some-with-the-wrong-stage",
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
			inCfgRaw:                getValidJsonConfig(),
			expectEmptyRulesetArray: false,
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
						Stage: "processed_auction",
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
			inCfgRaw:                getValidJsonConfig(),
			expectEmptyRulesetArray: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cacheEntry, err := NewCacheEntry(tc.inCfg, tc.inCfgRaw)

			if tc.expectEmptyRulesetArray {
				assert.Empty(t, cacheEntry.ruleSetsForProcessedAuctionRequestStage)
			} else {
				assert.NotEmpty(t, cacheEntry.ruleSetsForProcessedAuctionRequestStage)
			}

			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

func TestCreateCacheRuleSet(t *testing.T) {
	testCases := []struct {
		name            string
		in              *config.RuleSet
		expectedRuleSet cacheRuleSet[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]
		expectedErr     error
	}{
		{
			name:            "nil-ruleset-config",
			in:              nil,
			expectedRuleSet: cacheRuleSet[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{},
			expectedErr:     errors.New("no rules engine configuration provided"),
		},
		{
			name: "nil-modelgroup-array",
			in: &config.RuleSet{
				ModelGroups: nil,
			},
			expectedRuleSet: cacheRuleSet[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{modelGroups: []cacheModelGroup[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{}},
			expectedErr:     nil,
		},
		{
			name: "modelgroup-array-empty",
			in: &config.RuleSet{
				ModelGroups: []config.ModelGroup{},
			},
			expectedRuleSet: cacheRuleSet[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{modelGroups: []cacheModelGroup[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{}},
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
			expectedRuleSet: cacheRuleSet[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{modelGroups: []cacheModelGroup[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{}},
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
			expectedRuleSet: cacheRuleSet[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
				modelGroups: []cacheModelGroup[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
					{
						tree: rules.Tree[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
							Root: &rules.Node[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{},
							DefaultFunctions: []rules.ResultFunction[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
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
			expectedRuleSet: cacheRuleSet[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{modelGroups: []cacheModelGroup[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{}},
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
			expectedRuleSet: cacheRuleSet[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
				modelGroups: []cacheModelGroup[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
					{
						tree: rules.Tree[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
							Root: &rules.Node[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{},
							DefaultFunctions: []rules.ResultFunction[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
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
						tree: rules.Tree[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
							Root: &rules.Node[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{},
							DefaultFunctions: []rules.ResultFunction[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
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
    "generateRulesFromBidderConfig": true,
    "timestamp": "20250131 00:00:00",
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
