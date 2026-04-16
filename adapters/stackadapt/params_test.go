package stackadapt

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderStackAdapt, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected stackadapt params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderStackAdapt, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"publisherId":"pub-123","supplyId":"ssp-1"}`,
	`{"publisherId":"pub-123","supplyId":"ssp-1","placementId":"placement-456"}`,
	`{"publisherId":"pub-123","supplyId":"ssp-1","banner":{"expdir":[1,3]}}`,
	`{"publisherId":"pub-123","supplyId":"ssp-1","bidfloor":1.5}`,
	`{"publisherId":"pub-123","supplyId":"ssp-1","placementId":"pl-1","banner":{"expdir":[1,2,3]},"bidfloor":0.5}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`[]`,
	`{}`,
	`{"placementId":"placement-456"}`,
	`{"publisherId":"pub-123"}`,
	`{"supplyId":"ssp-1"}`,
	`{"publisherId":""}`,
	`{"publisherId":"pub-123","supplyId":""}`,
	`{"publisherId":"pub-123","supplyId":"ssp-1","bidfloor":-1}`,
	`{"publisherId":"pub-123","supplyId":"ssp-1","banner":"invalid"}`,
}
