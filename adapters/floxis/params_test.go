package floxis

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
		if err := validator.Validate(openrtb_ext.BidderFloxis, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderFloxis, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"seat":"abc"}`,
	`{"seat":"abc","region":"us-e"}`,
	`{"seat":"abc","region":"eu"}`,
	`{"seat":"abc","region":"apac"}`,
	`{"seat":"abc","region":"mars"}`,
	`{"seat":"x","partner":"acme"}`,
	`{"seat":"x","region":"eu","partner":"acme"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`{}`,
	`{"region":"us-e"}`,
	`{"seat":""}`,
	`{"seat":123}`,
	`{"seat":"abc","region":"a.b"}`,
	`{"seat":"abc","region":123}`,
	`{"seat":"x","partner":"a.b/c"}`,
	`{"seat":"x","partner":123}`,
	`{"seat":"abc","unknownField":"foo"}`,
}
