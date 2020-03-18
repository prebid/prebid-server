package kidoz

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderKidoz, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected kidoz params: %s \n Error: %s", validParam, err)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderKidoz, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"publisher_id":"pub-valid-0", "access_token":"token-valid-0"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"some_random_field":""}`,
	`{"publisher_id":""}`,
	`{"publisher_id": 1}`,
	`{"publisher_id": 1.2}`,
	`{"publisher_id": null}`,
	`{"publisher_id": true}`,
	`{"publisher_id": []}`,
	`{"publisher_id": {}}`,
	`{"publisher_id":"", "access_token":"token-valid-0"}`,
	`{"publisher_id": 1, "access_token":"token-valid-0"}`,
	`{"publisher_id": 1.2, "access_token":"token-valid-0"}`,
	`{"publisher_id": null, "access_token":"token-valid-0"}`,
	`{"publisher_id": true, "access_token":"token-valid-0"}`,
	`{"publisher_id": [], "access_token":"token-valid-0"}`,
	`{"publisher_id": {}, "access_token":"token-valid-0"}`,
	`{"access_token":""}`,
	`{"access_token": 1}`,
	`{"access_token": 1.2}`,
	`{"access_token": null}`,
	`{"access_token": true}`,
	`{"access_token": []}`,
	`{"access_token": {}}`,
	`{"access_token":"", "publisher_id":"pub-valid-0"}`,
	`{"access_token": 1, "publisher_id":"pub-valid-0"}`,
	`{"access_token": 1.2, "publisher_id":"pub-valid-0"}`,
	`{"access_token": null, "publisher_id":"pub-valid-0"}`,
	`{"access_token": true, "publisher_id":"pub-valid-0"}`,
	`{"access_token": [], "publisher_id":"pub-valid-0"}`,
	`{"access_token": {}, "publisher_id":"pub-valid-0"}`,
	`{"access_token": 1, "publisher_id":"pub-valid-0"}`,
	`{"access_token":"token-valid-0", "publisher_id": 1}`,
}
