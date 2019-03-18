package openrtb_ext

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/xeipuuv/gojsonschema"
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

// TestBidderParamSchemas makes sure that the validator.Schema() function
// returns valid JSON for all known BidderNames.
func TestBidderParamSchemas(t *testing.T) {
	for _, bidderName := range BidderMap {
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
	if err := validator.Validate(BidderAppnexus, json.RawMessage(`{"placementId":123}`)); err != nil {
		t.Errorf("These params should be valid. Error was: %v", err)
	}
}

func TestInvalidParams(t *testing.T) {
	if err := validator.Validate(BidderAppnexus, json.RawMessage(`{}`)); err == nil {
		t.Error("These params should be invalid.")
	}
}

func TestBidderList(t *testing.T) {
	list := BidderList()
	for _, bidderName := range BidderMap {
		adapterInList(t, bidderName, list)
	}
}

func adapterInList(t *testing.T, a BidderName, l []BidderName) {
	found := false
	for _, n := range l {
		if a == n {
			found = true
		}
	}
	if !found {
		t.Errorf("Adapter %s not found in the adapter map!", a)
	}
}
