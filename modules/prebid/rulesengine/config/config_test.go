package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	validator, err := CreateSchemaValidator(RulesEngineSchemaFile)
	assert.NoError(t, err, fmt.Sprintf("could not create schema validator using file %s", RulesEngineSchemaFile))

	testCases := []struct {
		name         string
		inCfg        json.RawMessage
		expectedConf *PbRulesEngine
		expectedErr  error
	}{
		{
			name:        "nil-input-config,-expect-error",
			inCfg:       nil,
			expectedErr: errors.New("JSON schema validation: EOF"),
		},
		{
			name:        "malformed-input-config,-expect-error",
			inCfg:       json.RawMessage(`malformed`),
			expectedErr: errors.New("JSON schema validation: invalid character 'm' looking for beginning of value"),
		},
		{
			name:        "valid-input-config-fails-schema-validation",
			inCfg:       json.RawMessage(`{}`),
			expectedErr: errors.New("JSON schema validation: [(root): enabled is required] [(root): rulesets is required] "),
		},
		{
			name:        "valid-input-config-fails-rule-set-validation",
			inCfg:       getInvalidRuleSetConfig(),
			expectedErr: errors.New("Ruleset no 0 is invalid: ModelGroup 0 number of schema functions differ from number of conditions of rule 0"),
		},
		{
			name:         "success",
			inCfg:        getValidJsonConfig(),
			expectedConf: getValidConfig(),
			expectedErr:  nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualConf, err := NewConfig(tc.inCfg, validator)

			assert.Equal(t, tc.expectedConf, actualConf)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

func TestCreateSchemaValidator(t *testing.T) {
	testCases := []struct {
		desc         string
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
			desc:         "success",
			inSchemaFile: RulesEngineSchemaFile,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			_, err := CreateSchemaValidator(tc.inSchemaFile)
			if len(tc.outErrMsg) > 0 {
				assert.Contains(t, err.Error(), tc.outErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	validator, err := CreateSchemaValidator(RulesEngineSchemaFile)
	assert.NoError(t, err, fmt.Sprintf("could not create schema validator using file %s", RulesEngineSchemaFile))

	type testInput struct {
		inConfig  json.RawMessage
		outErrMsg string
	}

	testGroups := []struct {
		desc  string
		tests []testInput
	}{
		{
			"nil rules engine config",
			[]testInput{{nil, "EOF"}},
		},
		{
			"malformed rules engine config",
			[]testInput{{json.RawMessage(`malformed`), "invalid character 'm' looking for beginning of value"}},
		},
		{
			"Well formed config fails schema validation",
			[]testInput{
				{ //0
					json.RawMessage(`{}`),
					"[(root): enabled is required] [(root): rulesets is required] ",
				},
				{ //1
					json.RawMessage(`{"enabled": true}`),
					"[(root): rulesets is required] ",
				},
				{ //2
					json.RawMessage(`{"enabled": true, "rulesets": []}`),
					"[rulesets: Array must have at least 1 items] ",
				},
				{ //3
					json.RawMessage(`{"enabled": true, "rulesets": [{}]}`),
					"[rulesets.0: stage is required] [rulesets.0: name is required] [rulesets.0: modelgroups is required] ",
				},
				{ //4
					json.RawMessage(`{"enabled": true, "rulesets": [{"stage":"a"}]}`),
					"[rulesets.0: name is required] [rulesets.0: modelgroups is required] [rulesets.0.stage: rulesets.0.stage must be one of the following: \"entrypoint\", \"raw_auction_request\", \"processed_auction_request\", \"bidder_request\", \"raw_bidder_response\", \"all_processed_bid_responses\", \"auction_response\"] ",
				},
				{ //5
					json.RawMessage(`{"enabled": true, "rulesets": [{"stage":"entrypoint"}]}`),
					"[rulesets.0: name is required] [rulesets.0: modelgroups is required] ",
				},
				{ //6
					json.RawMessage(`{"enabled": true, "rulesets": [{"stage":"entrypoint","name":"n"}]}`),
					"[rulesets.0: modelgroups is required] ",
				},
				{ //7
					json.RawMessage(`{"enabled": true, "rulesets": [{"stage":"entrypoint","name":"n","modelgroups":[]}]}`),
					"[rulesets.0.modelgroups: Array must have at least 1 items] ",
				},
				{ //8
					json.RawMessage(`
                    {
                      "enabled": true,
                      "rulesets": [
                        {
                          "stage": "entrypoint",
                          "name": "n",
                          "modelgroups": [
                            {
                              "weight": 101,
							  "schema": [{"function":"channel"}],
                              "rules": [
							    {
								  "conditions": ["cond"],
								  "results": [{"function": "excludeBidders"}]
								}
							  ]
                            }
                          ]
                        }
                      ]
                    }
					`),
					"[rulesets.0.modelgroups.0.weight: Must be less than or equal to 100] ",
				},
				{ //9
					json.RawMessage(`
                    {
                      "enabled": true,
                      "rulesets": [
                        {
                          "stage": "entrypoint",
                          "name": "n",
                          "modelgroups": [
                            {
                              "weight": -1,
							  "schema": [{"function":"channel"}],
                              "rules": [
							    {
								  "conditions": ["cond"],
								  "results": [{"function": "excludeBidders"}]
								}
							  ]
                            }
                          ]
                        }
                      ]
                    }
					`),
					"[rulesets.0.modelgroups.0.weight: Must be greater than or equal to 1] ",
				},
				{ //10
					json.RawMessage(`
                    {
                      "enabled": true,
                      "rulesets": [
                        {
                          "stage": "entrypoint",
                          "name": "n",
                          "modelgroups": [
                            {
                              "schema": [],
                              "rules": [
							    {
								  "conditions": ["cond"],
								  "results": [{"function": "excludeBidders"}]
								}
							  ]
                            }
                          ]
                        }
                      ]
                    }
					`),
					"",
				},
				{ //11
					json.RawMessage(`
                    {
                      "enabled": true,
                      "rulesets": [
                        {
                          "stage": "entrypoint",
                          "name": "n",
                          "modelgroups": [
                            {
							  "schema": [{"function":"channel"}],
                              "rules": []
                            }
                          ]
                        }
                      ]
                    }
					`),
					"",
				},
				{ //12
					json.RawMessage(`
                    {
                      "enabled": true,
                      "rulesets": [
                        {
                          "stage": "entrypoint",
                          "name": "someName",
                          "modelgroups": [
                            {
							  "schema": [{"function":"foo"}],
                              "rules": [
							    {
								  "conditions": ["cond"],
								  "results": [{"function": "excludeBidders"}]
								}
							  ]
                            }
                          ]
                        }
                      ]
                    }
					`),
					"[rulesets.0.modelgroups.0.schema.0.function: rulesets.0.modelgroups.0.schema.0.function must be one of the following: \"channel\", \"dataCenter\", \"dataCenterIn\", \"deviceCountry\", \"deviceCountryIn\", \"eidAvailable\", \"eidIn\", \"fpdAvailable\", \"gppSidAvailable\", \"gppSidIn\", \"percent\", \"tcfInScope\", \"userFpdAvailable\"] ",
				},
				{ //13
					json.RawMessage(`
                    {
                      "enabled": true,
                      "rulesets": [
                        {
                          "stage": "entrypoint",
                          "name": "someName",
                          "modelgroups": [
                            {
							  "schema": [{"function":"channel"}],
                              "rules": [
							    {
								  "conditions": [],
								  "results": [{"function": "excludeBidders"}]
								}
							  ]
                            }
                          ]
                        }
                      ]
                    }
					`),
					"[rulesets.0.modelgroups.0.rules.0.conditions: Array must have at least 1 items] ",
				},
				{ //14
					json.RawMessage(`
                    {
                      "enabled": true,
                      "rulesets": [
                        {
                          "stage": "entrypoint",
                          "name": "someName",
                          "modelgroups": [
                            {
							  "schema": [{"function":"channel"}],
                              "rules": [
							    {
								  "conditions": ["cond"],
								  "results": []
								}
							  ]
                            }
                          ]
                        }
                      ]
                    }
					`),
					"",
				},
				{ //15
					json.RawMessage(`
                    {
                      "enabled": true,
                      "rulesets": [
                        {
                          "stage": "entrypoint",
                          "name": "someName",
                          "modelgroups": [
                            {
							  "schema": [{"function":"channel"}],
                              "rules": [
							    {
								  "conditions": ["cond"],
								  "results": [{"function": "foobar"}]
								}
							  ]
                            }
                          ]
                        }
                      ]
                    }
					`),
					"[rulesets.0.modelgroups.0.rules.0.results.0.function: rulesets.0.modelgroups.0.rules.0.results.0.function must be one of the following: \"excludeBidders\", \"includeBidders\", \"logATag\"] ",
				},
			},
		},
		{
			"successful rules engine schema validation",
			[]testInput{{getValidJsonConfig(), ""}},
		},
	}

	for _, tg := range testGroups {
		for i, tc := range tg.tests {
			t.Run(fmt.Sprintf("%s test %d", tg.desc, i), func(t *testing.T) {
				actualError := validateConfig(tc.inConfig, validator)

				if len(tc.outErrMsg) > 0 {
					assert.EqualError(t, actualError, tc.outErrMsg)
				} else {
					assert.NoError(t, actualError)
				}
			})
		}
	}
}

func TestValidateRuleSet(t *testing.T) {
	testCases := []struct {
		desc        string
		ruleSet     *RuleSet
		expectedErr error
	}{
		{
			desc: "no-schema-functions-and-no-rules",
			ruleSet: &RuleSet{
				ModelGroups: []ModelGroup{
					{
						Schema: []Schema{},
						Rules:  []Rule{},
					},
				},
			},
			expectedErr: nil,
		},
		{
			desc: "Schema-functions-but-no-rules",
			ruleSet: &RuleSet{
				ModelGroups: []ModelGroup{
					{
						Schema: []Schema{{Func: "channel"}},
						Rules:  []Rule{},
					},
				},
			},
			expectedErr: errors.New("ModelGroup 0 has schema functions but no rules"),
		},
		{
			desc: "Schema-functions-but-no-rule-conditions",
			ruleSet: &RuleSet{
				ModelGroups: []ModelGroup{
					{
						Schema: []Schema{{Func: "channel"}},
						Rules:  []Rule{{Conditions: []string{}}},
					},
				},
			},
			expectedErr: errors.New("ModelGroup 0 number of schema functions differ from number of conditions of rule 0"),
		},
		{
			desc: "no-schema-functions-and-at-least-one-rule",
			ruleSet: &RuleSet{
				ModelGroups: []ModelGroup{
					{
						Schema: []Schema{},
						Rules:  []Rule{{Conditions: []string{}}},
					},
				},
			},
			expectedErr: errors.New("ModelGroup 0 has no schema functions to test its rules against"),
		},
		{
			desc: "More-rule-conditions-than-schema-functions",
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
			desc: "More-schema-functions-than-rule-conditions",
			ruleSet: &RuleSet{
				ModelGroups: []ModelGroup{
					{
						Schema: []Schema{
							{Func: "channel"},
							{Func: "deviceCountry"},
						},
						Rules: []Rule{{Conditions: []string{"web"}}},
					},
				},
			},
			expectedErr: errors.New("ModelGroup 0 number of schema functions differ from number of conditions of rule 0"),
		},
		{
			desc: "equal-number-of-schema-functions-and-result-functions",
			ruleSet: &RuleSet{
				ModelGroups: []ModelGroup{
					{
						Schema: []Schema{
							{Func: "deviceCountryIn", Args: json.RawMessage(`{"countries": ["USA"]}`)},
							{Func: "channel"},
						},
						Rules: []Rule{{Conditions: []string{"true", "web"}}},
					},
				},
			},
			expectedErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := validateRuleSet(tc.ruleSet)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

// Test utils
func getValidJsonConfig() json.RawMessage {
	return json.RawMessage(`
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

}

func getInvalidRuleSetConfig() json.RawMessage {
	return json.RawMessage(`
  {
    "enabled": true,
    "rulesets": [
      {
        "stage": "processed_auction_request",
        "name": "exclude-in-jpn",
        "modelgroups": [
          {
            "weight": 98,
            "schema": [{"function": "channel"}],
            "rules": [
              {
                "conditions": ["true", "amp"],
                "results": [
                  {
                    "function": "excludeBidders",
                    "args": {"bidders": ["bidderA"], "seatNonBid": 111}
                  }
                ]
              },
              {
                "conditions": ["web"],
                "results": [
                  {
                    "function": "excludeBidders",
                    "args": {"bidders": ["bidderB"], "seatNonBid": 222}
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
                "results": [{"function": "includeBidders", "args": {"bidders": ["bidderC"], "seatNonBid": 333}}]
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
		Enabled:   true,
		Timestamp: "20250131 00:00:00",
		RuleSets: []RuleSet{
			{
				Stage:   "processed_auction_request",
				Name:    "exclude-in-jpn",
				Version: "1234",
				ModelGroups: []ModelGroup{
					{
						Weight:       100,
						AnalyticsKey: "experiment-name",
						Version:      "4567",
						Schema: []Schema{
							{Func: "deviceCountryIn", Args: json.RawMessage(`{"countries": ["USA"]}`)},
							{Func: "dataCenterIn", Args: json.RawMessage(`{"datacenters": ["us-east", "us-west"]}`)},
							{Func: "channel"},
						},
						Default: []Result{},
						Rules: []Rule{
							{
								Conditions: []string{"true", "true", "amp"},
								Results: []Result{
									{
										Func: "excludeBidders",
										Args: json.RawMessage(`{"bidders": ["bidderA"], "seatNonBid": 111}`),
									},
								},
							},
							{
								Conditions: []string{"true", "false", "web"},
								Results: []Result{
									{
										Func: "excludeBidders",
										Args: json.RawMessage(`{"bidders": ["bidderB"], "seatNonBid": 222}`),
									},
								},
							},
							{
								Conditions: []string{"false", "false", "*"},
								Results: []Result{
									{
										Func: "includeBidders",
										Args: json.RawMessage(`{"bidders": ["bidderC"], "seatNonBid": 333}`),
									},
								},
							},
						},
					},
					{
						Weight:       1,
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
										Args: json.RawMessage(`{"bidders": ["bidderC"], "seatNonBid": 333}`),
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
