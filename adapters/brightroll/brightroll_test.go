package brightroll

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestEmptyConfig(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBrightroll, config.Adapter{
		Endpoint:         `http://test-bid.ybp.yahoo.com/bid/appnexuspbs`,
		ExtraAdapterInfo: ``,
	})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	ex := ExtraInfo{
		Accounts: []Account{},
	}
	expected := &BrightrollAdapter{
		URI:       "http://test-bid.ybp.yahoo.com/bid/appnexuspbs",
		extraInfo: ex,
	}
	assert.Equal(t, expected, bidder)
}

func TestNonEmptyConfig(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBrightroll, config.Adapter{
		Endpoint:         `http://test-bid.ybp.yahoo.com/bid/appnexuspbs`,
		ExtraAdapterInfo: `{"accounts": [{"id": "test","bidfloor":0.1}]}`,
	})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	ex := ExtraInfo{
		Accounts: []Account{{ID: "test", BidFloor: 0.1}},
	}
	expected := &BrightrollAdapter{
		URI:       "http://test-bid.ybp.yahoo.com/bid/appnexuspbs",
		extraInfo: ex,
	}
	assert.Equal(t, expected, bidder)
}

func TestMalformedEmpty(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderBrightroll, config.Adapter{
		Endpoint:         `http://test-bid.ybp.yahoo.com/bid/appnexuspbs`,
		ExtraAdapterInfo: `malformed`,
	})

	assert.Error(t, buildErr)
}

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBrightroll, config.Adapter{
		Endpoint:         `http://test-bid.ybp.yahoo.com/bid/appnexuspbs`,
		ExtraAdapterInfo: `{"accounts": [{"id": "adthrive","badv": [], "bcat": ["IAB8-5","IAB8-18"],"battr": [1,2,3], "bidfloor":0.0}]}`,
	})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "brightrolltest", bidder)
}
