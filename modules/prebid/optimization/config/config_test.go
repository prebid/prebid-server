package structs

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	testCases := []struct {
		desc         string
		inCfg        json.RawMessage
		expectedConf *PbRulesEngine
		expectedErr  error
	}{
		{
			desc:        "nil input config, expect error",
			inCfg:       nil,
			expectedErr: errors.New("JSON schema validation: EOF"),
		},
		{
			desc:        "malformed input config, expect error",
			inCfg:       json.RawMessage(`malformed`),
			expectedErr: errors.New("JSON schema validation: invalid character 'm' looking for beginning of value"),
		},
		{
			desc:        "valid input config fails schema validation",
			inCfg:       json.RawMessage(`{}`),
			expectedErr: errors.New("JSON schema validation: (root): enabled is required | "),
		},
		{
			desc:        "valid input config fails rule set validation",
			inCfg:       getInvalidJsonConfig(),
			expectedErr: errors.New("Ruleset no 0 is invalid: ModelGroup 0 number of schema functions differ from number of conditions of rule 0"),
		},
		{
			desc:         "success",
			inCfg:        getValidJsonConfig(),
			expectedConf: getValidConfig(),
			expectedErr:  nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			actualConf, err := NewConfig(tc.inCfg)

			assert.Equal(t, tc.expectedConf, actualConf)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

func TestValidateConfig(t *testing.T) {
	testCases := []struct {
		desc         string
		inRawCfg     json.RawMessage
		inSchemaFile string
		outErrMsg    string
	}{
		{
			desc:         "wrong json schema file name",
			inSchemaFile: "non-existent-file",
			outErrMsg:    "no such file or directory",
		},
		{
			desc:         "malformed json schema file name",
			inSchemaFile: "sample-schemas/malformed.json",
			outErrMsg:    "invalid character 'm' looking for beginning of value",
		},
		{
			desc:         "nil rules engine config",
			inSchemaFile: "rules-engine-schema.json",
			outErrMsg:    "EOF",
		},
		{
			desc:         "malformed JSON rules engine config",
			inRawCfg:     json.RawMessage(`malformed`),
			inSchemaFile: "rules-engine-schema.json",
			outErrMsg:    "invalid character 'm' looking for beginning of value",
		},
		{
			desc:         "JSON config did not pass schema validation",
			inRawCfg:     json.RawMessage(`{}`),
			inSchemaFile: "rules-engine-schema.json",
			outErrMsg:    "(root): enabled is required",
		},
		{
			desc:         "successful rules engine schema validation",
			inRawCfg:     getValidJsonConfig(),
			inSchemaFile: "rules-engine-schema.json",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := validateConfig(tc.inRawCfg, tc.inSchemaFile)
			if len(tc.outErrMsg) > 0 {
				assert.Contains(t, err.Error(), tc.outErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateRuleSet(t *testing.T) {
	type testInput struct {
		desc        string
		ruleSet     *RuleSet
		expectedErr error
	}
	testGroups := []struct {
		groupDesc string
		tests     []testInput
	}{
		{
			groupDesc: "Error is expected",
			tests: []testInput{
				{
					desc: "Unequal number of schema and result functions",
					ruleSet: &RuleSet{
						ModelGroups: []ModelGroup{
							{
								Schema: []Schema{{Func: "channel"}},
								Rules:  []Rule{{Conditions: []string{"amp", "web"}}},
							},
						},
					},
					expectedErr: errors.New("ModelGroup 0 number of schema functions differ from number of conditions of rule 0"),
				},
				{
					desc: "Weights don't add to 100",
					ruleSet: &RuleSet{
						ModelGroups: []ModelGroup{
							{Weight: 50},
							{Weight: 20},
						},
					},
					expectedErr: errors.New("Model group weights do not add to 100. Sum 70"),
				},
				{
					desc: "One of the weights is 100 but there's more than one modelgroup",
					ruleSet: &RuleSet{
						ModelGroups: []ModelGroup{
							{Weight: 0},
							{Weight: 100},
						},
					},
					expectedErr: errors.New("Weight of model group 1 is 100, leaving no margin for other model group weights"),
				},
			},
		},
		{
			groupDesc: "Success, expect nil error",
			tests: []testInput{
				{
					desc: "Equal number of schema functions and result functions",
					ruleSet: &RuleSet{
						ModelGroups: []ModelGroup{
							{
								Schema: []Schema{
									{Func: "deviceCountry", Args: json.RawMessage(`["USA"]`)},
									{Func: "channel"},
								},
								Rules: []Rule{{Conditions: []string{"true", "web"}}},
							},
						},
					},
					expectedErr: nil,
				},
				{
					desc: "Weights add up to 100",
					ruleSet: &RuleSet{
						ModelGroups: []ModelGroup{
							{Weight: 98},
							{Weight: 2},
						},
					},
					expectedErr: nil,
				},
				{
					desc: "All weights are 0",
					ruleSet: &RuleSet{
						ModelGroups: []ModelGroup{
							{Weight: 0},
							{Weight: 0},
						},
					},
					expectedErr: nil,
				},
			},
		},
	}
	for _, group := range testGroups {
		for _, tc := range group.tests {
			t.Run(group.groupDesc+"-"+tc.desc, func(t *testing.T) {
				err := validateRuleSet(tc.ruleSet)
				assert.Equal(t, tc.expectedErr, err)
			})
		}
	}
}

// Test utils
func getValidJsonConfig() json.RawMessage {
	return json.RawMessage(`
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
            "weight": 98,
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
            "weight": 2,
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

}

func getInvalidJsonConfig() json.RawMessage {
	return json.RawMessage(`
  {
    "enabled": true,
    "ruleSets": [
      {
        "stage": "processed-auction-request",
        "name": "exclude-in-jpn",
        "modelGroups": [
          {
            "weight": 98,
            "schema": [{"function": "channel"}],
            "rules": [
              {
                "conditions": ["true", "amp"],
                "results": [
                  {
                    "function": "excludeBidders",
                    "args": [{"bidders": ["bidderA"], "seatNonBid": 111}]
                  }
                ]
              },
              {
                "conditions": ["web"],
                "results": [
                  {
                    "function": "excludeBidders",
                    "args": [{"bidders": ["bidderB"], "seatNonBid": 222}]
                  }
                ]
              }
            ]
          },
          {
            "weight": 2,
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

}

func getValidConfig() *PbRulesEngine {
	return &PbRulesEngine{
		Enabled:                       true,
		GenerateRulesFromBidderConfig: true,
		Timestamp:                     "20250131 00:00:00",
		RuleSets: []RuleSet{
			{
				Stage:   "processed-auction-request",
				Name:    "exclude-in-jpn",
				Version: "1234",
				ModelGroups: []ModelGroup{
					{
						Weight:       98,
						AnalyticsKey: "experiment-name",
						Version:      "4567",
						Schema: []Schema{
							{Func: "deviceCountry", Args: json.RawMessage(`["USA"]`)},
							{Func: "dataCenters", Args: json.RawMessage(`["us-east", "us-west"]`)},
							{Func: "channel"},
						},
						Rules: []Rule{
							{
								Conditions: []string{"true", "true", "amp"},
								Results: []Result{
									{
										Func: "excludeBidders",
										Args: json.RawMessage(`[{"bidders": ["bidderA"], "seatNonBid": 111}]`),
									},
								},
							},
							{
								Conditions: []string{"true", "false", "web"},
								Results: []Result{
									{
										Func: "excludeBidders",
										Args: json.RawMessage(`[{"bidders": ["bidderB"], "seatNonBid": 222}]`),
									},
								},
							},
							{
								Conditions: []string{"false", "false", "*"},
								Results: []Result{
									{
										Func: "includeBidders",
										Args: json.RawMessage(`[{"bidders": ["bidderC"], "seatNonBid": 333}]`),
									},
								},
							},
						},
					},
					{
						Weight:       2,
						AnalyticsKey: "experiment-name",
						Version:      "3.0",
						Schema: []Schema{
							{Func: "channel"},
						},
						Rules: []Rule{
							{
								Conditions: []string{"*"},
								Results: []Result{
									{
										Func: "includeBidders",
										Args: json.RawMessage(`[{"bidders": ["bidderC"], "seatNonBid": 333}]`),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
