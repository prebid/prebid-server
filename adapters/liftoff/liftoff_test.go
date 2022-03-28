//go:build !integration
// +build !integration

package liftoff

import (
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

const testsDir = "liftofftest"

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, testsDir, NewLiftoffBidder(http.DefaultClient, "http://liftoff.com/givemeads", "http://liftoff-us-east.com/givemeads", "http://liftoff-eu.com/givemeads", "http://liftoff-apac.com/givemeads"))
}
