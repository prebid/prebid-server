package feedad

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func mustMarshal(t *testing.T, v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		t.Errorf("mustMarshal: %s", err)
	}
	return data
}

func provideCompleteParams() *openrtb_ext.ExtImpFeedAd {
	return &openrtb_ext.ExtImpFeedAd{
		ClientToken: "some-clienttoken",
		Decoration:  "some-decoration",
		PlacementId: "some-placementid",
		SdkOptions: &openrtb_ext.ExtImpFeedAdSdkOptions{
			AdvertisingId:   "some-advertisingid",
			AppName:         "some-appname",
			BundleId:        "some-bundleid",
			HybridApp:       true,
			HybridPlatform:  "android",
			LimitAdTracking: true,
		},
	}
}

func TestParams(t *testing.T) {
	type testCase struct {
		_id         string
		_shouldPass bool
		data        []byte
	}

	tests := make([]*testCase, 0)

	// empty-params
	test := &testCase{
		_id:         "empty-params",
		_shouldPass: false,
		data:        []byte("{}"),
	}
	tests = append(tests, test)

	// pass-complete
	params := provideCompleteParams()

	test = &testCase{
		_id:         "pass-complete",
		_shouldPass: true,
		data:        mustMarshal(t, params),
	}
	tests = append(tests, test)

	// pass-minimal
	params = provideCompleteParams()
	params.Decoration = ""
	params.SdkOptions = nil

	test = &testCase{
		_id:         "pass-minimal",
		_shouldPass: true,
		data:        mustMarshal(t, params),
	}
	tests = append(tests, test)

	// fail-missing-clienttoken
	params = provideCompleteParams()
	params.ClientToken = ""

	test = &testCase{
		_id:         "fail-missing-clienttoken",
		_shouldPass: false,
		data:        mustMarshal(t, params),
	}
	tests = append(tests, test)

	// fail-missing-placementid
	params = provideCompleteParams()
	params.PlacementId = ""

	test = &testCase{
		_id:         "fail-missing-placementid",
		_shouldPass: false,
		data:        mustMarshal(t, params),
	}
	tests = append(tests, test)

	// pass-empty-sdkoptions
	params = provideCompleteParams()
	params.SdkOptions = &openrtb_ext.ExtImpFeedAdSdkOptions{}

	test = &testCase{
		_id:         "pass-empty-sdkoptions",
		_shouldPass: true,
		data:        mustMarshal(t, params),
	}
	tests = append(tests, test)

	// Run tests
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas: %v", err)
	}

	for _, test := range tests {
		t.Run(test._id, func(t *testing.T) {
			err := validator.Validate(openrtb_ext.BidderFeedAd, test.data)
			if err == nil && !test._shouldPass {
				t.Error("did not fail")
			} else if err != nil && test._shouldPass {
				t.Errorf("did fail: %s", err)
			}
		})
	}
}
