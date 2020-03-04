package nanointeractive

import (
	"github.com/magiconair/properties/assert"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "nanointeractivetest", NewNanoIneractiveBidder("https://ad.audiencemanager.de/hbs"))
}

// Test properties of Adapter interface

func TestNanoInteractivetProperties(t *testing.T) {
	ni := NewNanoInteractiveAdapter("https://ad.audiencemanager.de/hbs")

	assert.Equal(t, ni.Name(), "Nano", "missing family name")
	assert.Equal(t, ni.SkipNoCookies(), false, "skip no cookies has to be false")

}
