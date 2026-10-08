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
	// networkId — "n" and digits; it becomes a hostname label. host is optional.
	`{"networkId":"n1","placementKey":"a4f21c9e7b"}`,

	// host — optional and unused by this adapter, but validated as the Prebid.js adapter
	// validates it, so the same params pass on both sides.
	`{"networkId":"n3057","host":"ads.example.com","placementKey":"a4f21c9e7b"}`,
	`{"networkId":"n3057","host":"ads.example.com:8080","placementKey":"a4f21c9e7b"}`,
	`{"networkId":"n3057","host":"ads.example.com:65535","placementKey":"a4f21c9e7b"}`,
	`{"networkId":"n3057","host":"ads-eu.example.co.uk","placementKey":"a4f21c9e7b"}`,
	// A single-label host is a real deployment shape (an internal name, or
	// localhost in a staging rig), not a malformed one.
	`{"networkId":"n3057","host":"localhost","placementKey":"a4f21c9e7b"}`,
	`{"networkId":"n3057","host":"api-us","placementKey":"a4f21c9e7b"}`,

	// placementKey — minLength 1, so a single character is the boundary.
	`{"networkId":"n3057","host":"ads.example.com","placementKey":"a"}`,

	// channel — free-form, and deliberately uncapped: the ad server applies its
	// own ingest limits rather than the adapter rejecting the impression.
	`{"networkId":"n3057","host":"ads.example.com","placementKey":"a4f21c9e7b","channel":"sports-uk"}`,
	`{"networkId":"n3057","host":"ads.example.com","placementKey":"a4f21c9e7b","channel":""}`,
	`{"networkId":"n3057","host":"ads.example.com","placementKey":"a4f21c9e7b","channel":"` + strings.Repeat("c", 300) + `"}`,

	// customParams — an object of scalars, in every scalar flavour.
	`{"networkId":"n3057","host":"ads.example.com","placementKey":"a4f21c9e7b","customParams":{"section":"sport","tier":2,"premium":true}}`,
	`{"networkId":"n3057","host":"ads.example.com","placementKey":"a4f21c9e7b","customParams":{"ratio":1.75,"empty":""}}`,
	`{"networkId":"n3057","host":"ads.example.com","placementKey":"a4f21c9e7b","customParams":{}}`,

	// bidFloor — minimum 0, so 0 is the boundary and means "no floor".
	`{"networkId":"n3057","host":"ads.example.com","placementKey":"a4f21c9e7b","bidFloor":0}`,
	`{"networkId":"n3057","host":"ads.example.com","placementKey":"a4f21c9e7b","bidFloor":0.01}`,
	`{"networkId":"n3057","host":"ads.example.com","placementKey":"a4f21c9e7b","bidFloor":1.75,"bidFloorCur":"EUR"}`,

	// bidFloorCur — a plain string; the schema declares no pattern, so it must
	// not reject a currency it merely does not recognise.
	`{"networkId":"n3057","host":"ads.example.com","placementKey":"a4f21c9e7b","bidFloorCur":"USD"}`,

	// Everything at once.
	`{"networkId":"n3057","host":"ads.example.com:8443","placementKey":"a4f21c9e7b","channel":"sports-uk","customParams":{"section":"sport"},"bidFloor":2.5,"bidFloorCur":"GBP"}`,
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
	`{"networkId":"n3057"}`,
	`{"placementKey":"a4f21c9e7b"}`,
	// host does not stand in for networkId on the server side.
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b"}`,

	// networkId — wrong type, wrong shape, and anything that could leave eashb.com.
	`{"networkId":3057,"placementKey":"a4f21c9e7b"}`,
	`{"networkId":"","placementKey":"a4f21c9e7b"}`,
	`{"networkId":"3057","placementKey":"a4f21c9e7b"}`,
	`{"networkId":"N3057","placementKey":"a4f21c9e7b"}`,
	`{"networkId":"n3057.evil.com","placementKey":"a4f21c9e7b"}`,
	`{"networkId":"n30a57","placementKey":"a4f21c9e7b"}`,

	// host — wrong type, and the empty string, which the pattern rejects
	// because it demands at least one label character.
	`{"networkId":"n3057","host":42,"placementKey":"a4f21c9e7b"}`,
	`{"networkId":"n3057","host":"","placementKey":"a4f21c9e7b"}`,
	// A host must not be able to rewrite the outbound URL.
	`{"networkId":"n3057","host":"https://ads.example.com","placementKey":"a4f21c9e7b"}`,
	`{"networkId":"n3057","host":"ads.example.com/collect","placementKey":"a4f21c9e7b"}`,
	`{"networkId":"n3057","host":"user@ads.example.com","placementKey":"a4f21c9e7b"}`,
	`{"networkId":"n3057","host":"ads.example.com?x=1","placementKey":"a4f21c9e7b"}`,
	`{"networkId":"n3057","host":"ads.example.com#frag","placementKey":"a4f21c9e7b"}`,
	`{"networkId":"n3057","host":"ads.example.com:80a","placementKey":"a4f21c9e7b"}`,
	`{"networkId":"n3057","host":"ads example.com","placementKey":"a4f21c9e7b"}`,

	// placementKey — wrong type, and the empty string just under minLength 1.
	`{"networkId":"n3057","host":"ads.example.com","placementKey":42}`,
	`{"networkId":"n3057","host":"ads.example.com","placementKey":""}`,

	// channel — wrong type.
	`{"networkId":"n3057","host":"ads.example.com","placementKey":"a4f21c9e7b","channel":42}`,
	`{"networkId":"n3057","host":"ads.example.com","placementKey":"a4f21c9e7b","channel":["sports-uk"]}`,

	// customParams — must be an object of scalars. A nested object or array
	// would be stringified into targeting as a Go rendering of a map, so the
	// schema rejects the impression instead.
	`{"networkId":"n3057","host":"ads.example.com","placementKey":"a4f21c9e7b","customParams":"not-an-object"}`,
	`{"networkId":"n3057","host":"ads.example.com","placementKey":"a4f21c9e7b","customParams":[]}`,
	`{"networkId":"n3057","host":"ads.example.com","placementKey":"a4f21c9e7b","customParams":{"nested":{"a":1}}}`,
	`{"networkId":"n3057","host":"ads.example.com","placementKey":"a4f21c9e7b","customParams":{"list":[1,2]}}`,
	`{"networkId":"n3057","host":"ads.example.com","placementKey":"a4f21c9e7b","customParams":{"nothing":null}}`,

	// bidFloor — wrong type, and one step under the minimum of 0.
	`{"networkId":"n3057","host":"ads.example.com","placementKey":"a4f21c9e7b","bidFloor":-1}`,
	`{"networkId":"n3057","host":"ads.example.com","placementKey":"a4f21c9e7b","bidFloor":-0.01}`,
	`{"networkId":"n3057","host":"ads.example.com","placementKey":"a4f21c9e7b","bidFloor":"1.75"}`,

	// bidFloorCur — wrong type.
	`{"networkId":"n3057","host":"ads.example.com","placementKey":"a4f21c9e7b","bidFloorCur":978}`,
}
