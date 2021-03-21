package bidmachine

import (
	"math"
	"net/http"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBidmachine, config.Adapter{
		Endpoint: "http://{{.Host}}.example.com/prebid"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "bidmachinetest", bidder)
}

func makeBidRequest() *openrtb.BidRequest {
	request := &openrtb.BidRequest{
		ID: "test-req-id-0",
		Imp: []openrtb.Imp{
			{
				ID: "test-imp-id-0",
				Banner: &openrtb.Banner{
					Format: []openrtb.Format{
						{
							W: 320,
							H: 50,
						},
					},
				},
				Ext: []byte(`{"bidder":{"seller_id":"1", "host": "api-eu", "path":"auction/rtb/v2`),
			},
		},
	}
	return request
}

func TestMakeRequests(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBidmachine, config.Adapter{
		Endpoint: "http://example.com/prebid"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	t.Run("Handles Request marshal failure", func(t *testing.T) {
		request := makeBidRequest()
		request.Imp[0].BidFloor = math.Inf(1) // cant be marshalled
		extra := &adapters.ExtraRequestInfo{}
		reqs, errs := bidder.MakeRequests(request, extra)
		// cant assert message its different on different versions of go
		assert.Equal(t, 1, len(errs))
		assert.Contains(t, errs[0].Error(), "JSON")
		assert.Equal(t, 0, len(reqs))
	})
}

func TestMakeBids(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBidmachine, config.Adapter{
		Endpoint: "http://example.com/prebid"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	t.Run("Handles response marshal failure", func(t *testing.T) {
		request := makeBidRequest()
		requestData := &adapters.RequestData{}
		responseData := &adapters.ResponseData{
			StatusCode: http.StatusOK,
		}

		resp, errs := bidder.MakeBids(request, requestData, responseData)
		// cant assert message its different on different versions of go
		assert.Equal(t, 1, len(errs))
		assert.Contains(t, errs[0].Error(), "JSON")
		assert.Nil(t, resp)
	})
}
