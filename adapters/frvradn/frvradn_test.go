package frvradn

import (
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
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
