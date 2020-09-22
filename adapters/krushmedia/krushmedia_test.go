package krushmedia

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "krushmediatest", NewKrushmediaBidder("http://example.com/?c=rtb&m=req&key={{.AccountID}}"))
}
