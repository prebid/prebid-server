package anzuprebid

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
		if err := validator.Validate(openrtb_ext.BidderAnzuPrebid, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderAnzuPrebid, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"zone":"zone1","queryString":"pid=0123456789abcdef0123456789abcdef"}`,
	`{"zone":"zone2","queryString":"c=rtb&m=req&key=00000000-0000-4000-8000-000000000000"}`,
	`{"zone":"zone1","queryString":"?pid=123"}`,
	`{"zone":"zone2","queryString":"k=v%20w&list=1,2&path=/x/y"}`,
	`{"zone":"zone1","queryString":"pid=123","extra":"ignored"}`,
	`{"zone":"zone2","queryString":"` + strings.Repeat("a", 2048) + `"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`{}`,
	`[]`,
	`{"zone":"zone1"}`,
	`{"queryString":"pid=123"}`,
	`{"zone":"","queryString":"pid=123"}`,
	`{"zone":"ZONE1","queryString":"pid=123"}`,
	`{"zone":"zone3","queryString":"pid=123"}`,
	`{"zone":"https://evil.example.com/","queryString":"pid=123"}`,
	`{"zone":1,"queryString":"pid=123"}`,
	`{"zone":"zone1","QueryString":"pid=123"}`,
	`{"zone":"zone1","queryString":""}`,
	`{"zone":"zone1","queryString":123}`,
	`{"zone":"zone1","queryString":"a=1#frag"}`,
	`{"zone":"zone1","queryString":"a=1 b"}`,
	`{"zone":"zone1","queryString":"a=1;b=2"}`,
	`{"zone":"zone1","queryString":"a=\"x\""}`,
	`{"zone":"zone1","queryString":"a=<b>"}`,
	`{"zone":"zone1","queryString":"` + strings.Repeat("a", 2049) + `"}`,
}
