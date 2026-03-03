package fwssp

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderFWSSP, json.RawMessage(p)); err != nil {
			t.Errorf("Schema rejected valid params: %s", p)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderFWSSP, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"custom_site_section_id":"ss_12345", "profile_id":"123456:prof_12345", "network_id":"12345"}`,
}

var invalidParams = []string{
	`null`,
	`nil`,
	``,
	`[]`,
	`{}`,
	`{"custom_site_section_id":"ss_12345", "profile_id":"123456:prof_12345", "network_id":12345}`,
	`{"custom_site_section_id":"ss_12345", "network_id":"123456", "profile_id":100}`,
	`{"network_id":"123456", "profile_id":"123456:prof_12345", "custom_site_section_id":100}`,
	`{"custom_site_section_id":"ss_12345", "profile_id":"123456:prof_12345"}`,
	`{"custom_site_section_id":"ss_12345", "network_id":"123456"}`,
	`{"network_id":"123456", "profile_id":"123456:prof_12345"}`,
}
