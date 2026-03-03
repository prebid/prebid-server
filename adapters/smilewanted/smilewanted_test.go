package smilewanted

import (
	"encoding/json"
	"math"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderSmileWanted, config.Adapter{
		Endpoint: "http://example.com/go/"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "smilewantedtest", bidder)
}

// TestMakeRequestsJSONMarshalError tests the error handling when JSON marshalling fails
func TestMakeRequestsJSONMarshalError(t *testing.T) {
	bidder := &adapter{
		URI: "http://example.com/go/",
	}

	// Create a request with a float value that cannot be marshalled to JSON (NaN)
	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID:       "test-imp-id",
			BidFloor: math.NaN(), // NaN cannot be marshalled to JSON
			Ext:      json.RawMessage(`{"bidder": {"zoneId": "test"}}`),
		}},
	}

	_, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})
	assert.NotNil(t, errs)
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "Json not encoded")
}

// TestMakeBidsJSONUnmarshalError tests the error handling when unmarshalling external request fails
func TestMakeBidsJSONUnmarshalError(t *testing.T) {
	bidder := &adapter{
		URI: "http://example.com/go/",
	}

	internalRequest := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "test-imp-id",
		}},
	}

	// Create external request with invalid JSON that will fail unmarshalling
	externalRequest := &adapters.RequestData{
		Body: []byte(`{invalid json}`),
	}

	response := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"id":"test-response-id","seatbid":[{"bid":[{"id":"test-bid-id","impid":"test-imp-id","price":1.0}]}]}`),
	}

	_, errs := bidder.MakeBids(internalRequest, externalRequest, response)
	assert.NotNil(t, errs)
	assert.Len(t, errs, 1)
}
