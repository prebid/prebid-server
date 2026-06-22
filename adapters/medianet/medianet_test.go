package medianet

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderMedianet, config.Adapter{
		Endpoint:         "https://example.media.net/rtb/prebid",
		ExtraAdapterInfo: "http://localhost:8080/extrnal_url",
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "medianettest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderMedianet, config.Adapter{
		Endpoint: "{{Malformed}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
}

func TestRegionalEndpoint(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderMedianet, config.Adapter{
		Endpoint: "https://{{.Host}}/rtb/pb/prebids2s"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	assert.NoError(t, buildErr)

	tests := []struct {
		name        string
		region      string
		expectedURI string
	}{
		{"US East", "USE", "https://prebid-adapter-useast.media.net/rtb/pb/prebids2s"},
		{"US West", "USW", "https://prebid-adapter-uswest.media.net/rtb/pb/prebids2s"},
		{"Asia", "APAC", "https://prebid-adapter-asia.media.net/rtb/pb/prebids2s"},
		{"Europe Central", "EUC", "https://prebid-adapter-eu.media.net/rtb/pb/prebids2s"},
		{"lowercase region", "use", "https://prebid-adapter-useast.media.net/rtb/pb/prebids2s"},
		{"no region", "", "https://prebid-adapter.media.net/rtb/pb/prebids2s"},
		{"unknown region", "XYZ", "https://prebid-adapter.media.net/rtb/pb/prebids2s"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bidderParams := `{"cid":"c","crid":"r"}`
			if test.region != "" {
				bidderParams = fmt.Sprintf(`{"cid":"c","crid":"r","region":%q}`, test.region)
			}
			request := &openrtb2.BidRequest{
				ID: "test-request-id",
				Imp: []openrtb2.Imp{{
					ID:  "1",
					Ext: json.RawMessage(fmt.Sprintf(`{"bidder":%s}`, bidderParams)),
				}},
			}

			reqs, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})
			assert.Empty(t, errs)
			assert.Len(t, reqs, 1)
			assert.Equal(t, test.expectedURI, reqs[0].Uri)
		})
	}
}
