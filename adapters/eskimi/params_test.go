package eskimi

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderEskimi, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderEskimi, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"placementId":625}`,
	`{"placementId":625,"bidFloor":0.5,"bidFloorCur":"USD"}`,
	`{"placementId":625,"bcat":["IAB1"],"badv":["bad.com"],"bapp":["com.bad.app"]}`,
	`{"placementId":625,"battr":[1,2,17]}`,
}

var invalidParams = []string{
	``,
	`null`,
	`{}`,
	`{"placementId":"625"}`,
	`{"placementId":625.5}`,
	`{"placementId":0}`,
	`{"placementId":-1}`,
	`{"bidFloor":0.5}`,
	`{"placementId":625,"bcat":"IAB1"}`,
	`{"placementId":625,"battr":[0]}`,
	`{"placementId":625,"battr":[18]}`,
	`{"placementId":625,"unknownField":"foo"}`,
}
