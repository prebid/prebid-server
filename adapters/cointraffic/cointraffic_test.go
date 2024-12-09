package cointraffic

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

const testsDir = "cointraffictest"
const testsBidderEndpoint = "http://test-request.com/prebid"

func TestJsonSamples(t *testing.T) {
	ac := config.Adapter{
		Endpoint:         testsBidderEndpoint,
		ExtraAdapterInfo: "",
	}

	sc := config.Server{
		ExternalUrl: "http://hosturl.com",
		GvlID:       1,
		DataCenter:  "2",
	}

	bidder, buildErr := Builder(openrtb_ext.BidderCointraffic, ac, sc)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, testsDir, bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	ac := config.Adapter{
		Endpoint: "{{Malformed}}",
	}

	sc := config.Server{
		ExternalUrl: "http://hosturl.com",
		GvlID:       1,
		DataCenter:  "2",
	}

	_, buildErr := Builder(openrtb_ext.BidderCointraffic, ac, sc)

	assert.Nil(t, buildErr)
}
