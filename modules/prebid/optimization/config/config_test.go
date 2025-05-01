package structs

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	expectedConf := getValidConfig()

	actualConf, err := NewConfig(getJsonRawConfig())

	assert.NoError(t, err)
	assert.Equal(t, expectedConf, actualConf)
}

func TestValidateConfig(t *testing.T) {
	err := ValidateConfig(getJsonRawConfig())
	assert.NoError(t, err)
}

// Test utils
func getJsonRawConfig() json.RawMessage {
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
          }
        ]
      }
    ]
  }
`)

}

func getValidConfig() PbRulesEngine {
	return PbRulesEngine{
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
				},
			},
		},
	}
}
