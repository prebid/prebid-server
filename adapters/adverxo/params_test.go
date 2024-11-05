package adverxo

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAdverxo, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected adverxo params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAdverxo, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{ "adUnitId": 5, "auth": "fb71a1ec1d4c0b7e3f0a21703fece91d8b65be44"}`,
	`{ "adUnitId": 402053, "auth": "fb71a1ec1d4c0b7e3f0a21703fece91d8b65be44"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`{}`,
	`{"adCode": "string", "seatCode": 5, "originalPublisherid": "string"}`,
	`{ "adUnitId": 5}`,
	`{ "auth": "fb71a1ec1d4c0b7e3f0a21703fece91d8b65be44"}`,
	`{ "adUnitId": 0, "auth": "fb71a1ec1d4c0b7e3f0a21703fece91d8b65be44"}`,
	`{ "adUnitId": "5", "auth": "fb71a1ec1d4c0b7e3f0a21703fece91d8b65be44"}`,
	`{ "adUnitId": 5, "auth": ""}`,
	`{ "adUnitId": 5, "auth": "12345"}`,
}
