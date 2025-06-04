package rulesengine

import (
	"encoding/json"
	"errors"
	"testing"

	hs "github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/prebid/rulesengine/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/rules"
	"github.com/stretchr/testify/assert"
)

func TestNewCacheEntry(t *testing.T) {
	type testInput struct {
		cfg    *config.PbRulesEngine
		cfgRaw *json.RawMessage
	}

	type testOutput struct {
		emptyRulesetArray bool
		err               error
	}

	testCases := []struct {
		name     string
		in       testInput
		expected testOutput
	}{
		{
			name: "nil ruleset config",
			in:   testInput{},
			expected: testOutput{
				emptyRulesetArray: true,
				err:               errors.New("no rules engine configuration provided"),
			},
		},
		{
			name: "nil cfgRaw",
			in: testInput{
				cfg: &config.PbRulesEngine{},
			},
			expected: testOutput{
				emptyRulesetArray: true,
				err:               errors.New("Can't create identifier hash from empty raw json configuration"),
			},
		},
		{
			name: "empty ruleset array",
			in: testInput{
				cfg:    &config.PbRulesEngine{},
				cfgRaw: getValidJsonConfig(),
			},
			expected: testOutput{
				emptyRulesetArray: true,
				err:               nil,
			},
		},
		{
			name: "ruleset with wrong stage",
			in: testInput{
				cfg: &config.PbRulesEngine{
					RuleSets: []config.RuleSet{
						{Stage: "wrong-stage"},
					},
				},
				cfgRaw: getValidJsonConfig(),
			},
			expected: testOutput{
				emptyRulesetArray: true,
			},
		},
		{
			name: "createCacheRuleSet() throws error",
			in: testInput{
				cfg: &config.PbRulesEngine{
					RuleSets: []config.RuleSet{
						{
							Stage: "processed_auction",
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
				cfgRaw: getValidJsonConfig(),
			},
			expected: testOutput{
				emptyRulesetArray: true,
			},
		},
		{
			name: "Success",
			in: testInput{
				cfg: &config.PbRulesEngine{
					RuleSets: []config.RuleSet{
						{
							Stage: "processed_auction",
							ModelGroups: []config.ModelGroup{
								{
									Default: []config.Result{
										{
											Func: ExcludeBiddersName,
											Args: json.RawMessage(`[{"bidders": ["bidderA"], "seatNonBid": 111}]`),
										},
									},
								},
							},
						},
					},
				},
				cfgRaw: getValidJsonConfig(),
			},
			expected: testOutput{
				emptyRulesetArray: false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cacheEntry, err := NewCacheEntry(tc.in.cfg, tc.in.cfgRaw)

			if tc.expected.emptyRulesetArray {
				assert.Empty(t, cacheEntry.ruleSetsForProcessedAuctionRequestStage)
			} else {
				assert.NotEmpty(t, cacheEntry.ruleSetsForProcessedAuctionRequestStage)
			}

			assert.Equal(t, tc.expected.err, err)
		})
	}
}

func TestCreateCacheRuleSet(t *testing.T) {
	type testOutput struct {
		ruleset cacheRuleSet[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]
		err     error
	}

	testCases := []struct {
		name     string
		in       *config.RuleSet
		expected testOutput
	}{
		{
			name: "nil ruleset config",
			in:   nil,
			expected: testOutput{
				ruleset: cacheRuleSet[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{},
				err:     errors.New("no rules engine configuration provided"),
			},
		},
		{
			name: "modelgroup array empty",
			in: &config.RuleSet{
				ModelGroups: []config.ModelGroup{},
			},
			expected: testOutput{
				ruleset: cacheRuleSet[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{modelGroups: []cacheModelGroup[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{}},
				err:     nil,
			},
		},
		{
			name: "NewTree() throws error",
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
			expected: testOutput{
				ruleset: cacheRuleSet[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{modelGroups: []cacheModelGroup[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{}},
				err:     errors.New("result function unknownResultFunction was not created"),
			},
		},
		{
			name: "Success",
			in: &config.RuleSet{
				ModelGroups: []config.ModelGroup{
					{
						Default: []config.Result{
							{
								Func: ExcludeBiddersName,
								Args: json.RawMessage(`[{"bidders": ["bidderA"], "seatNonBid": 111}]`),
							},
						},
					},
				},
			},
			expected: testOutput{
				ruleset: cacheRuleSet[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
					modelGroups: []cacheModelGroup[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
						{
							tree: rules.Tree[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
								Root: &rules.Node[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{},
								DefaultFunctions: []rules.ResultFunction[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
									&ExcludeBidders{
										Args: []ResultFuncParams{
											{
												Bidders:    []string{"bidderA"},
												SeatNonBid: 111,
											},
										},
									},
								},
							},
						},
					},
				},
				err: nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ruleset, err := createCacheRuleSet(tc.in)

			assert.Equal(t, tc.expected.ruleset, ruleset)
			assert.Equal(t, tc.expected.err, err)
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
			name:            "Nil input",
			in:              nil,
			expectEmptyHash: true,
		},
		{
			name:            "Empty input",
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
    "ruleSets": [
      {
        "stage": "processed-auction-request",
        "name": "exclude-in-jpn",
        "version": "1234",
        "modelGroups": [
          {
            "weight": 100,
            "analyticsKey": "experiment-name",
            "version": "4567",
            "schema": [
              {
                "function": "deviceCountry",
                "args": ["USA"]
              },
              {
                "function": "dataCenters",
                "args": ["us-east", "us-west"]
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
                    "args": [{"bidders": ["bidderA"], "seatNonBid": 111}]
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
                    "args": [{"bidders": ["bidderB"], "seatNonBid": 222}]
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
                    "args": [{"bidders": ["bidderC"], "seatNonBid": 333}]
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
                "results": [{"function": "includeBidders", "args": [{"bidders": ["bidderC"], "seatNonBid": 333}]}]
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
