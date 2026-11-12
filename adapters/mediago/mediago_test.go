package mediago

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestMakeRequestCurrency(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderMediaGo, config.Adapter{
		Endpoint: "https://REGION.mediago.io/api/bid?tn={{.AccountID}}",
	}, config.Server{})
	if err != nil {
		t.Fatalf("Builder returned unexpected error %v", err)
	}

	tests := []struct {
		name       string
		requestCur []string
		expected   []string
	}{
		{name: "defaults to USD", expected: []string{"USD"}},
		{name: "passes through requested currencies", requestCur: []string{"EUR", "USD"}, expected: []string{"EUR", "USD"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			width, height := int64(300), int64(250)
			request := &openrtb2.BidRequest{
				Cur: tt.requestCur,
				Imp: []openrtb2.Imp{{
					ID:     "imp-1",
					Banner: &openrtb2.Banner{W: &width, H: &height},
					Ext:    json.RawMessage(`{"bidder":{"token":"test-token"}}`),
				}},
			}
			requestData, errs := bidder.MakeRequests(request, nil)
			if len(errs) != 0 {
				t.Fatalf("MakeRequests returned errors: %v", errs)
			}
			if len(requestData) != 1 {
				t.Fatalf("MakeRequests returned %d requests, expected 1", len(requestData))
			}

			var forwarded openrtb2.BidRequest
			if err := json.Unmarshal(requestData[0].Body, &forwarded); err != nil {
				t.Fatalf("failed to decode forwarded request: %v", err)
			}
			assert.Equal(t, tt.expected, forwarded.Cur)
		})
	}
}

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderMediaGo, config.Adapter{
		Endpoint: "https://REGION.mediago.io/api/bid?tn={{.AccountID}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "mediagotest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderMediaGo, config.Adapter{Endpoint: "{{Malformed}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
}
