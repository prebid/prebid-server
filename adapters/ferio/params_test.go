package ferio

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("failed to fetch json schemas: %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderFerio, json.RawMessage(validParam)); err != nil {
			t.Errorf("schema rejected ferio params: %s\nerror: %v", validParam, err)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("failed to fetch json schemas: %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderFerio, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"publisherId":"publisher-1","adUnitId":"adunit-1","tenantId":"tenant-1"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`[]`,
	`{}`,
	`{"publisherId":"publisher-1","adUnitId":"adunit-1"}`,
	`{"publisherId":"publisher-1"}`,
	`{"adUnitId":"adunit-1"}`,
	`{"publisherId":"publisher-1","tenantId":"tenant-1"}`,
	`{"publisherId":"","adUnitId":"adunit-1","tenantId":"tenant-1"}`,
	`{"publisherId":"publisher-1","adUnitId":"","tenantId":"tenant-1"}`,
	`{"publisherId":"publisher-1","adUnitId":"adunit-1","tenantId":""}`,
	`{"publisherId":123,"adUnitId":"adunit-1","tenantId":"tenant-1"}`,
	`{"publisherId":"publisher-1","adUnitId":123,"tenantId":"tenant-1"}`,
	`{"publisherId":"publisher-1","adUnitId":"adunit-1","tenantId":123}`,
	`{"publisherId":"publisher-1","adUnitId":"adunit-1","tenantId":"tenant-1","unknown":"value"}`,
	`{"publisherId":"publisher-1","placementId":"placement-1","tenantId":"tenant-1"}`,
}
