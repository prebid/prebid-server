package optimizationmodule

import "encoding/json"

func GetConf() json.RawMessage {

	return json.RawMessage(`
{
    "schema": [
    {
      "function": "deviceCountry",
      "args": ["JPN"]
    },
    {
      "function": "dataCenters",
      "args": ["us-east", "us-west"]
    },
    {
      "function": "channel"
    }
  ],
  "rules": [
    {
      "conditions": ["true", "true", "amp"],
      "results": [
        {
          "function": "excludeBidders",
          "args": [
            {
              "bidders": ["bidderA"]
            }
          ]
        }
      ]
    },
    {
      "conditions": ["true", "false","web"],
      "results": [
        {
          "function": "excludeBidders",
          "args": [
            {
              "bidders": ["bidderB"]
            }
          ]
        }
      ]
    },
    {
      "conditions": ["false", "false", "*"],
      "results": [
        {
          "function": "setDeviceIP",
          "args": [
            {
              "bidders": ["bidderC"]
            }
          ]
        }
      ]
    }
  ]
}`)

}
