package epom_as

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderEpomAs, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderEpomAs, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	// host — the pattern is byte-identical to util/urlutil.IsSafeHost, which the
	// adapter gates on, so everything it accepts must validate here too.
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b"}`,
	`{"host":"ads.example.com:8080","placementKey":"a4f21c9e7b"}`,
	`{"host":"ads.example.com:65535","placementKey":"a4f21c9e7b"}`,
	`{"host":"ads-eu.example.co.uk","placementKey":"a4f21c9e7b"}`,
	// A single-label host is a real deployment shape (an internal name, or
	// localhost in a staging rig), not a malformed one.
	`{"host":"localhost","placementKey":"a4f21c9e7b"}`,
	`{"host":"api-us","placementKey":"a4f21c9e7b"}`,

	// placementKey — minLength 1, so a single character is the boundary.
	`{"host":"ads.example.com","placementKey":"a"}`,

	// channel — free-form, and deliberately uncapped: the ad server applies its
	// own ingest limits rather than the adapter rejecting the impression.
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b","channel":"sports-uk"}`,
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b","channel":""}`,
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b","channel":"` + strings.Repeat("c", 300) + `"}`,

	// customParams — an object of scalars, in every scalar flavour.
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b","customParams":{"section":"sport","tier":2,"premium":true}}`,
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b","customParams":{"ratio":1.75,"empty":""}}`,
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b","customParams":{}}`,

	// bidFloor — minimum 0, so 0 is the boundary and means "no floor".
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b","bidFloor":0}`,
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b","bidFloor":0.01}`,
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b","bidFloor":1.75,"bidFloorCur":"EUR"}`,

	// bidFloorCur — a plain string; the schema declares no pattern, so it must
	// not reject a currency it merely does not recognise.
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b","bidFloorCur":"USD"}`,

	// Everything at once.
	`{"host":"ads.example.com:8443","placementKey":"a4f21c9e7b","channel":"sports-uk","customParams":{"section":"sport"},"bidFloor":2.5,"bidFloorCur":"GBP"}`,
}

var invalidParams = []string{
	// Non-object roots.
	``,
	`null`,
	`true`,
	`5`,
	`[]`,
	`"{}"`,

	// Required params.
	`{}`,
	`{"host":"ads.example.com"}`,
	`{"placementKey":"a4f21c9e7b"}`,

	// host — wrong type, and the empty string, which the pattern rejects
	// because it demands at least one label character.
	`{"host":42,"placementKey":"a4f21c9e7b"}`,
	`{"host":"","placementKey":"a4f21c9e7b"}`,
	// A host must not be able to rewrite the outbound URL.
	`{"host":"https://ads.example.com","placementKey":"a4f21c9e7b"}`,
	`{"host":"ads.example.com/collect","placementKey":"a4f21c9e7b"}`,
	`{"host":"user@ads.example.com","placementKey":"a4f21c9e7b"}`,
	`{"host":"ads.example.com?x=1","placementKey":"a4f21c9e7b"}`,
	`{"host":"ads.example.com#frag","placementKey":"a4f21c9e7b"}`,
	`{"host":"ads.example.com:80a","placementKey":"a4f21c9e7b"}`,
	`{"host":"ads example.com","placementKey":"a4f21c9e7b"}`,

	// placementKey — wrong type, and the empty string just under minLength 1.
	`{"host":"ads.example.com","placementKey":42}`,
	`{"host":"ads.example.com","placementKey":""}`,

	// channel — wrong type.
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b","channel":42}`,
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b","channel":["sports-uk"]}`,

	// customParams — must be an object of scalars. A nested object or array
	// would be stringified into targeting as a Go rendering of a map, so the
	// schema rejects the impression instead.
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b","customParams":"not-an-object"}`,
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b","customParams":[]}`,
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b","customParams":{"nested":{"a":1}}}`,
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b","customParams":{"list":[1,2]}}`,
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b","customParams":{"nothing":null}}`,

	// bidFloor — wrong type, and one step under the minimum of 0.
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b","bidFloor":-1}`,
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b","bidFloor":-0.01}`,
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b","bidFloor":"1.75"}`,

	// bidFloorCur — wrong type.
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b","bidFloorCur":978}`,
}
