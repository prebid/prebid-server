package frvradn

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderFRVRAdNetwork, config.Adapter{
		Endpoint: "https://fran.frvr.com/api/v1/openrtb",
	}, config.Server{ExternalUrl: "https://host.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "frvradntest", bidder)
}

func TestInvalidEndpoint(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderFRVRAdNetwork, config.Adapter{Endpoint: ""}, config.Server{})

	assert.Error(t, buildErr)
}
