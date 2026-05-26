package synapseHX

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch json-schemas. %v", err)
	}

	for _, param := range validParams {
		if err := validator.Validate(openrtb_ext.BidderSynapseHX, json.RawMessage(param)); err != nil {
			t.Errorf("Schema rejected valid params: %s", param)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch json-schemas. %v", err)
	}

	for _, param := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderSynapseHX, json.RawMessage(param)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", param)
		}
	}
}

var validParams = []string{
	`{"tenantId":"a"}`,
	`{"tenantId":"8cd85aed-25a6-4db0-ad98-4a3af1f7601c"}`,
	`{"tenantId":"a","adUnitId":"b"}`,
	`{"tenantId":"a","adUnitId":"8cd85aed-25a6-4db0-ad98-4a3af1f7601c"}`,
}

var invalidParams = []string{
	`{"tenantId":1}`,
	`{"tenantId":""}`,
	`{"tenantId":"a","adUnitId":1}`,
	`{"tenantId":"a","adUnitId":""}`,
}
