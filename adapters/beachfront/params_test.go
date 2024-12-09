package beachfront

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderBeachfront, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected beachfront params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderBeachfront, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"appId":"eitherBannerOrVideoAppId", "bidfloor":9.21}`,
	`{"appId":"eitherBannerOrVideoAppId", "bidfloor":9.23129837662781}`,
	`{"appId":"eitherBannerOrVideoAppId", "bidfloor":0.01}`,
	`{"appId":"eitherBannerOrVideoAppId", "bidfloor":0}`,
	`{"appId":"eitherBannerOrVideoAppId"}`,
	`{"appIds": {
		"banner":"aBannerAppId"
		}
	}`,
	`{"appIds": {
		"video":"aVideoAppId"
		}
	}`,
	`{"appIds": {
		"video":"aVideoAppId",
		"banner":"aBannerAppId"
		}
	}`,
	`{"bidfloor":2.50,
		"appIds": {
			"video":"aVideoAppId",
			"banner":"aBannerAppId"
		},
		"videoResponseType":"nurl"
	}`,
}

var invalidParams = []string{
	`{"appId":1176, "bidfloor":0.01}`,
	`{"appIds":"eitherBannerOrVideoAppId"}`,
	`{"appIds":{"cerebralUplink":"cerebralUplinkAppId"}}`,
	`{"appId":"eitherBannerOrVideoAppId",
		"appIds": {
			"video":"aVideoAppId",
			"banner":"aBannerAppId"
		}
	}`,
	`{}`,
}
