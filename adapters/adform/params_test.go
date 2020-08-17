package adform

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

// This file actually intends to test static/bidder-params/adform.json
//
// These also validate the format of the external API: request.imp[i].ext.adform

// TestValidParams makes sure that the adform schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAdform, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected adform params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the adform schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAdform, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"mid":123}`,
	`{"mid":"123"}`,
	`{"mid":123,"priceType":"gross"}`,
	`{"mid":"123","priceType":"net"}`,
	`{"mid":"123","mkv":" color :blue , length : 350"}`,
	`{"mid":"123","mkv":"color:"}`,
	`{"mid":"123","mkw":"green,male"}`,
	`{"mid":"123","mkv":" ","mkw":" "}`,
	`{"mid":"123","cdims":"500x300,400x200","mkw":" "}`,
	`{"mid":"123","cdims":"500x300","mkv":" ","mkw":" "}`,
	`{"mid":"123","minp":2.1}`,
	`{"mid":"123","url":"https://adform.com/page"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"notmid":"123"}`,
	`{"mid":"123","priceType":"ne"}`,
	`{"mid":"123","mkv":"color:blue,:350"}`,
	`{"mid":"123","mkv":"color:blue;length:350"}`,
	`{"mid":"123","mkv":"color"}`,
	`{"mid":"123","mkv":"color:blue,l&ngth:350"}`,
	`{"mid":"123","mkv":"color::blue"}`,
	`{"mid":"123","mkw":"fem&le"}`,
	`{"mid":"123","minp":"2.1"}`,
	`{"mid":"123","cdims":"500x300:400:200","mkw":" "}`,
	`{"mid":"123","cdims":"500x300,400:200","mkv":" ","mkw":" "}`,
	`{"mid":"123","url":10}`,
}
