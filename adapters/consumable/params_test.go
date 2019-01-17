package consumable

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

// This file actually intends to test static/bidder-params/consumable.json
//
// These also validate the format of the external API: request.imp[i].ext.consumable

// TestValidParams makes sure that the 33across schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderConsumable, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected Consumable params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the Consumable schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderConsumable, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected Consumable params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"networkId": 22, "siteId": 1, "unitId": 101, "unitName": "unit-1"}`,
	`{"networkId": 22, "siteId": 1, "unitId": 101, "unitName": "-unit-1"}`, // unitName can start with a dash
	`{"networkId": 22, "siteId": 1, "unitId": 101}`,                        // unitName can be omitted (although prebid.js doesn't allow that)
}

var invalidParams = []string{
	`{"networkId": 22, "siteId": 1, "unitId": 101, "unitName": "--unit-1"}`, // unitName cannot start --
	`{"networkId": 22, "siteId": 1, "unitId": 101, "unitName": "unit 1"}`,   // unitName cannot contain spaces
	`{"networkId": 22, "siteId": 1, "unitId": 101, "unitName": "1unit-1"}`,  // unitName cannot start with a digit
	`{"networkId": "22", "siteId": 1, "unitId": 101, "unitName": 11}`,       // networkId must be a number
	`{"networkId": 22, "siteId": "1", "unitId": 101, "unitName": 11}`,       // siteId must be a number
	`{"networkId": 22, "siteId": 1, "unitId": "101", "unitName": 11}`,       // unitId must be a number
	`{"networkId": 22, "siteId": 1, "unitId": 101, "unitName": 11}`,         // unitName must be a string
	`{"siteId": 1, "unitId": 101, "unitName": 11}`,                          // networkId must be present
	`{"networkId": 22, "unitId": 101, "unitName": 11}`,                      // siteId must be present
	`{"siteId": 1, "networkId": 22, "unitName": 11}`,                        // unitId must be present
}
