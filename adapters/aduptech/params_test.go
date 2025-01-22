package aduptech

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

var validParams = []string{
	`{ "publisher": "1", "placement": "2" }`,
	`{ "publisher": "123456789", "placement": "234567890" }`,
	`{ "publisher": "123456789", "placement": "234567890", "query": "" }`,
	`{ "publisher": "123456789", "placement": "234567890", "query": "test" }`,
	`{ "publisher": "123456789", "placement": "234567890", "adtest": true }`,
	`{ "publisher": "123456789", "placement": "234567890", "debug": true }`,
	`{ "publisher": "123456789", "placement": "234567890", "query": "test", "adtest": true }`,
	`{ "publisher": "123456789", "placement": "234567890", "ext": {"foo": "bar"} }`,
	`{ "publisher": "123456789", "placement": "234567890", "ext": {} }`,
}

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		assert.NoErrorf(
			t,
			validator.Validate(openrtb_ext.BidderAdUpTech, json.RawMessage(validParam)),
			"Schema rejected Aduptech params: %s",
			validParam,
		)
	}
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{ "publisher": "123456789" }`,
	`{ "placement": "234567890" }`,
	`{ "publisher": null, "placement": null }`,
	`{ "publisher": "", "placement": "" }`,
	`{ "publisher": "123456789", "placement": "234567890", "query": null }`,
	`{ "publisher": "123456789", "placement": "234567890", "adtest": null }`,
	`{ "publisher": "123456789", "placement": "234567890", "debug": null }`,
	`{ "publisher": "123456789", "placement": "234567890", "ext": null }`,
	`{ "publisher": "123456789", "placement": "234567890", "ext": 123 }`,
	`{ "publisher": "123456789", "placement": "234567890", "ext": "abc" }`,
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		assert.Errorf(
			t,
			validator.Validate(openrtb_ext.BidderAdUpTech, json.RawMessage(invalidParam)),
			"Schema allowed unexpected params: %s",
			invalidParam,
		)
	}
}
