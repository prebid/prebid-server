package pubnative

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderPubnative, config.Adapter{
		Endpoint: "http://example.com/prebid"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "pubnativetest", bidder)
}

func TestMakeRequestsDoesNotModifyRequestCur(t *testing.T) {
	bidder := &PubnativeAdapter{URI: "http://example.com/prebid"}
	request := &openrtb2.BidRequest{
		ID:     "request-id",
		Cur:    []string{"EUR"},
		Device: &openrtb2.Device{OS: "android"},
		Imp: []openrtb2.Imp{{
			ID:     "imp-id",
			Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
			Ext:    json.RawMessage(`{"bidder":{"zone_id":1,"app_auth_token":"4fd53a12b78af4b39835de9e449c3082"}}`),
		}},
	}

	requests, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs)
	assert.Len(t, requests, 1)
	assert.Equal(t, []string{"EUR"}, request.Cur)
}
