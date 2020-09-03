package brightroll

import (
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestEmptyConfig(t *testing.T) {
	output := NewBrightrollBidder("http://test-bid.ybp.yahoo.com/bid/appnexuspbs", "")
	ex := ExtraInfo{
		Accounts: []Account{},
	}
	expected := &BrightrollAdapter{
		URI:       "http://test-bid.ybp.yahoo.com/bid/appnexuspbs",
		extraInfo: ex,
	}
	assert.Equal(t, expected, output, "")
}

func TestNonEmptyConfig(t *testing.T) {
	output := NewBrightrollBidder("http://test-bid.ybp.yahoo.com/bid/appnexuspbs", "{\"accounts\": [{\"id\": \"test\",\"bidfloor\":0.1}]}")
	ex := ExtraInfo{
		Accounts: []Account{{ID: "test", BidFloor: 0.1}},
	}

	expected := &BrightrollAdapter{
		URI:       "http://test-bid.ybp.yahoo.com/bid/appnexuspbs",
		extraInfo: ex,
	}
	assert.Equal(t, expected, output, "")
}

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "brightrolltest", NewBrightrollBidder("http://test-bid.ybp.yahoo.com/bid/appnexuspbs", "{\"accounts\": [{\"id\": \"adthrive\",\"badv\": [], \"bcat\": [\"IAB8-5\",\"IAB8-18\"],\"battr\": [1,2,3], \"bidfloor\":0.0}]}"))
}
