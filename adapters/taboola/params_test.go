package taboola

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderTaboola, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderTaboola, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"publisherId" : "1", "tagid": "tag-id-for-example"}`,
	`{"publisherId" : "1", "tagId": "tag-id-for-example"}`,
	`{"publisherId" : "1", "tagid": "tag-id-for-example","position":1}`,
	`{"publisherId" : "1", "tagid": "tag-id-for-example","pageType":"pageType"}`,
	`{"publisherId" : "1", "tagid": "tag-id-for-example", "bcat": ["excluded-category"], "badv": ["excluded-advertiser"], "bidfloor": 1.2, "publisherDomain": "http://domain.com"}`,
}

var invalidParams = []string{
	`{}`,
	`{"tagId" : "1"}`,
	`{"publisherId" : "1", "bcat": ["excluded-category"], "badv": ["excluded-advertiser"], "bidfloor": 1.2, "publisherDomain": "http://domain.com"}`,
	`{"publisherId" : 1, "tagid": "tag-id-for-example"}`,
	`{"publisherId" : "1"", "tagid": 2}`,
	`{"publisherId" : "1", "tagid": "tag-id-for-example", "bcat":"incorrect-type"}`,
	`{"publisherId" : "1", "tagid": "tag-id-for-example", "badv":"incorrect-type"}`,
	`{"publisherId" : "1", "tagid": "tag-id-for-example", "bidfloor":"incorrect-type"}`,
	`{"publisherId" : "1", "tagid": "tag-id-for-example", "publisherDomain":1}`,
	`{"tagid": "tag-id-for-example", "bcat": ["excluded-category"], "badv": ["excluded-advertiser"], "bidfloor": 1.2, "publisherDomain": "http://domain.com"}`,
	`{"publisherId" : "1", "tagid": "tag-id-for-example","position":null}`,
	`{"publisherId" : "1", "tagid": "tag-id-for-example","position":"1"}`,
	`{"publisherId" : "1", "tagid": "tag-id-for-example","pageType":1}`,
	`{"publisherId" : "1", "tagid": "tag-id-for-example","pageType":null}`,
}
