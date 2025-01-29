package feedad

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema: %v", err)
	}

	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderFeedAd, json.RawMessage(p)); err != nil {
			t.Errorf("Schema rejected valid params: %s", p)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema: %v", err)
	}

	for _, p := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderFeedAd, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"clientToken":"some-clienttoken","placementId":"some-placementid"}`,
	`{"clientToken":"some-clienttoken","placementId":"some_placementid"}`,
	`{"clientToken":"some-clienttoken","placementId":"some-"}`,
	`{"clientToken":"some-clienttoken","placementId":"some-"}`,
	`{"clientToken":"some-clienttoken","placementId":"42-"}`,
	`{"clientToken":"some-clienttoken","placementId":"42_"}`,
	`{"clientToken":"some-clienttoken","placementId":"some-placementid-42"}`,
	`{"clientToken":"some-clienttoken","placementId":"some_placementid_42"}`,
	`{"clientToken":"some-clienttoken","placementId":"some_placementid-42"}`,
	`{"clientToken":"some-clienttoken","placementId":"some-placementid_42"}`,
	`{"clientToken":"some-clienttoken","placementId":"42-some-placementid"}`,
	`{"clientToken":"some-clienttoken","placementId":"42_some_placementid"}`,
	`{"clientToken":"some-clienttoken","placementId":"42_some-placementid"}`,
	`{"clientToken":"some-clienttoken","placementId":"42-some_placementid"}`,
	`{"clientToken":"some-clienttoken","placementId":"some-placementid","sdkOptions":{}}`,
	`{"clientToken":"some-clienttoken","placementId":"some-placementid","sdkOptions":{"hybrid_platform":"ios"}}`,
	`{"clientToken":"some-clienttoken","placementId":"some-placementid","sdkOptions":{"hybrid_platform":"windows"}}`,
	`{"clientToken":"some-clienttoken","decoration":"some-decoration","placementId":"some-placementid","sdkOptions":{"advertising_id":"","app_name":"","bundle_id":"","hybrid_app":false,"hybrid_platform":"","limit_ad_tracking":false}}`,
	`{"clientToken":"some-clienttoken","decoration":"some-decoration","placementId":"some-placementid","sdkOptions":{"advertising_id":"some-advertisingid","app_name":"some-appname","bundle_id":"some-bundleid","hybrid_app":true,"hybrid_platform":"android","limit_ad_tracking":true}}`,
}

var invalidParams = []string{
	`{}`,
	`{"clientToken":"","placementId":"some-placementid"}`,
	`{"clientToken":"some-clienttoken","placementId":""}`,
	`{"clientToken":"some-clienttoken","placementId":"-"}`,
	`{"clientToken":"some-clienttoken","placementId":"_"}`,
	`{"clientToken":"some-clienttoken","placementId":" "}`,
	`{"clientToken":"some-clienttoken","placementId":"complete garbage"}`,
	`{"clientToken":"some-clienttoken","placementId":"Complete-Garbage"}`,
	`{"clientToken":"some-clienttoken","placementId":"Complete*Garbage"}`,
	`{"clientToken":"some-clienttoken","placementId":"-42"}`,
	`{"clientToken":"some-clienttoken","placementId":"_42"}`,
	`{"clientToken":"some-clienttoken","placementId":"-complete"}`,
	`{"clientToken":"some-clienttoken","placementId":"_garbage"}`,
	`{"clientToken":"some-clienttoken","placementId":"some-placementid","sdkOptions":"complete-garbage"}`,
	`{"clientToken":"some-clienttoken","placementId":"some-placementid","sdkOptions":{"advertising_id":{}}}`,
	`{"clientToken":"some-clienttoken","placementId":"some-placementid","sdkOptions":{"advertising_id":{}}}`,
	`{"clientToken":"some-clienttoken","placementId":"some-placementid","sdkOptions":{"app_name":{}}}`,
	`{"clientToken":"some-clienttoken","placementId":"some-placementid","sdkOptions":{"bundle_id":{}}}`,
	`{"clientToken":"some-clienttoken","placementId":"some-placementid","sdkOptions":{"hybrid_platform":{}}}`,
	`{"clientToken":"some-clienttoken","placementId":"some-placementid","sdkOptions":{"limit_ad_tracking":{}}}`,
}
