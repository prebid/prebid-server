package structs

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
			expectedErr: errors.New("JSON schema validation: [(root): enabled is required] [(root): ruleSets is required] "),
		},
		{
			desc:        "valid input config fails rule set validation",
			inCfg:       getInvalidRuleSetConfig(),
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
					"[(root): enabled is required] [(root): ruleSets is required] ",
				},
				{ //1
					json.RawMessage(`{"enabled": true}`),
					"[(root): ruleSets is required] ",
				},
				{ //2
					json.RawMessage(`{"enabled": true, "ruleSets": []}`),
					"[ruleSets: Array must have at least 1 items] ",
				},
				{ //3
					json.RawMessage(`{"enabled": true, "ruleSets": [{}]}`),
					"[ruleSets.0: stage is required] [ruleSets.0: name is required] [ruleSets.0: modelGroups is required] ",
				},
				{ //4
					json.RawMessage(`{"enabled": true, "ruleSets": [{"stage":"a"}]}`),
					"[ruleSets.0: name is required] [ruleSets.0: modelGroups is required] [ruleSets.0.stage: ruleSets.0.stage must be one of the following: \"entrypoint\", \"raw-auction\", \"processed-auction-request\", \"bidder-request\", \"raw-bidder-response\", \"all-processed-bid-responses\", \"auction-response\"] ",
				},
				{ //5
					json.RawMessage(`{"enabled": true, "ruleSets": [{"stage":"entrypoint"}]}`),
					"[ruleSets.0: name is required] [ruleSets.0: modelGroups is required] ",
				},
				{ //6
					json.RawMessage(`{"enabled": true, "ruleSets": [{"stage":"entrypoint","name":"n"}]}`),
					"[ruleSets.0: modelGroups is required] ",
				},
				{ //7
					json.RawMessage(`{"enabled": true, "ruleSets": [{"stage":"entrypoint","name":"n","modelGroups":[]}]}`),
					"[ruleSets.0.modelGroups: Array must have at least 1 items] ",
				},
				{ //8
					json.RawMessage(`
                    {
                      "enabled": true,
                      "ruleSets": [
                        {
                          "stage": "entrypoint",
                          "name": "n",
                          "modelGroups": [
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
					"[ruleSets.0.modelGroups.0.weight: Must be less than or equal to 100] ",
				},
				{ //9
					json.RawMessage(`
                    {
                      "enabled": true,
                      "ruleSets": [
                        {
                          "stage": "entrypoint",
                          "name": "n",
                          "modelGroups": [
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
					"[ruleSets.0.modelGroups.0.weight: Must be greater than or equal to 1] ",
				},
				{ //10
					json.RawMessage(`
                    {
                      "enabled": true,
                      "ruleSets": [
                        {
                          "stage": "entrypoint",
                          "name": "n",
                          "modelGroups": [
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
					"[ruleSets.0.modelGroups.0.schema: Array must have at least 1 items] ",
				},
				{ //11
					json.RawMessage(`
                    {
                      "enabled": true,
                      "ruleSets": [
                        {
                          "stage": "entrypoint",
                          "name": "n",
                          "modelGroups": [
                            {
							  "schema": [{"function":"channel"}],
                              "rules": []
                            }
                          ]
                        }
                      ]
                    }
					`),
					"[ruleSets.0.modelGroups.0.rules: Array must have at least 1 items] ",
				},
				{ //12
					json.RawMessage(`
                    {
                      "enabled": true,
                      "ruleSets": [
                        {
                          "stage": "entrypoint",
                          "name": "someName",
                          "modelGroups": [
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
					"[ruleSets.0.modelGroups.0.schema.0.function: ruleSets.0.modelGroups.0.schema.0.function must be one of the following: \"deviceCountry\", \"dataCenters\", \"channel\", \"eidAvailable\", \"userFpdAvailable\", \"fpdAvail\", \"gppSid\", \"tcfInScope\", \"percent\", \"prebidKey\", \"domain\", \"bundle\", \"deviceType\"] ",
				},
				{ //13
					json.RawMessage(`
                    {
                      "enabled": true,
                      "ruleSets": [
                        {
                          "stage": "entrypoint",
                          "name": "someName",
                          "modelGroups": [
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
					"[ruleSets.0.modelGroups.0.rules.0.conditions: Array must have at least 1 items] ",
				},
				{ //14
					json.RawMessage(`
                    {
                      "enabled": true,
                      "ruleSets": [
                        {
                          "stage": "entrypoint",
                          "name": "someName",
                          "modelGroups": [
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
					"[ruleSets.0.modelGroups.0.rules.0.results: Array must have at least 1 items] ",
				},
				{ //15
					json.RawMessage(`
                    {
                      "enabled": true,
                      "ruleSets": [
                        {
                          "stage": "entrypoint",
                          "name": "someName",
                          "modelGroups": [
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
					"[ruleSets.0.modelGroups.0.rules.0.results.0.function: ruleSets.0.modelGroups.0.rules.0.results.0.function must be one of the following: \"excludeBidders\", \"includeBidders\", \"logATag\"] ",
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
					assert.Equal(t, tc.outErrMsg, actualError.Error())
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
			desc: "Error is expected. Unequal number of schema and result functions",
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
			desc: "Success. Equal number of schema functions and result functions",
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
			desc: "Success. Weights add up to 100",
			ruleSet: &RuleSet{
				ModelGroups: []ModelGroup{
					{Weight: 98},
					{Weight: 2},
				},
			},
			expectedErr: nil,
		},
		{
			desc: "Success. Weights don't add to 100",
			ruleSet: &RuleSet{
				ModelGroups: []ModelGroup{
					{Weight: 50},
					{},
				},
			},
			expectedErr: nil,
		},
		{
			desc: "Success. All weights are 0",
			ruleSet: &RuleSet{
				ModelGroups: []ModelGroup{
					{Weight: 0},
					{Weight: 0},
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

}

func getInvalidRuleSetConfig() json.RawMessage {
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
		Enabled:   true,
		Timestamp: "20250131 00:00:00",
		RuleSets: []RuleSet{
			{
				Stage:   "processed-auction-request",
				Name:    "exclude-in-jpn",
				Version: "1234",
				ModelGroups: []ModelGroup{
					{
						Weight:       100,
						AnalyticsKey: "experiment-name",
						Version:      "4567",
						Schema: []Schema{
							{Func: "deviceCountry", Args: json.RawMessage(`["USA"]`)},
							{Func: "dataCenters", Args: json.RawMessage(`["us-east", "us-west"]`)},
							{Func: "channel"},
						},
						Default: []Result{},
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
