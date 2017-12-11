package openrtb_ext

import (
	"github.com/mxmCherry/openrtb"
	"github.com/xeipuuv/gojsonschema"
	"os"
	"testing"
)

// TestMain does the expensive setup so we don't keep re-reading the files in static/bidder-params for each test.
func TestMain(m *testing.M) {
	bidderParams, err := NewBidderParamsValidator("../static/bidder-params")
	if err != nil {
		os.Exit(1)
	}
	validator = bidderParams
	os.Exit(m.Run())
}

var validator BidderParamValidator

// TestGetBidderName makes sure the GetBidderNames method works properly.
func TestGetBidderName(t *testing.T) {
	for bidderString, bidderName := range bidderMap {
		converted, isValid := GetBidderName(bidderString)
		if !isValid {
			t.Errorf("GetBidderName thinks \"%s\" is not a valid bidder", bidderString)
		}
		if converted != bidderName {
			t.Errorf("GetBidderName parsed %s into %s", bidderString, converted.String())
		}
	}
}

// TestBidderParamSchemas makes sure that the validator.Schema() function
// returns valid JSON for all known BidderNames.
func TestBidderParamSchemas(t *testing.T) {
	for _, bidderName := range bidderMap {
		schema := validator.Schema(bidderName)
		if schema == "" {
			t.Errorf("No schema exists for bidder %s. Does static/bidder-params/%s.json exist?", bidderName, bidderName)
		}

		if _, err := gojsonschema.NewBytesLoader([]byte(schema)).LoadJSON(); err != nil {
			t.Errorf("static/bidder-params/%s.json does not have a valid json-schema. %v", bidderName, err)
		}
	}
}

// TestValidParams and TestInvalidParams overlap with adapters/appnexus/params_test... but those tests
// from the other packages don't show up in code coverage.
func TestValidParams(t *testing.T) {
	if err := validator.Validate(BidderAppnexus, openrtb.RawJSON(`{"placementId":123}`)); err != nil {
		t.Errorf("These params should be valid. Error was: %v", err)
	}
}

func TestInvalidParams(t *testing.T) {
	if err := validator.Validate(BidderAppnexus, openrtb.RawJSON(`{}`)); err == nil {
		t.Error("These params should be invalid.")
	}
}
