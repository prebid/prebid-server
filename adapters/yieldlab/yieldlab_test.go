package yieldlab

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

var testCacheBuster cacheBuster = func() string {
	return "testing"
}

var testWeekGenerator weekGenerator = func() string {
	return "33"
}

func newTestYieldlabBidder(endpoint string) *YieldlabAdapter {
	return &YieldlabAdapter{
		endpoint:    endpoint,
		cacheBuster: testCacheBuster,
		getWeek:     testWeekGenerator,
	}
}

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "yieldlabtest", newTestYieldlabBidder("https://ad.yieldlab.net/testing/"))
}
