package eplanning

import (
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	eplanningAdapter := NewEPlanningBidder(new(http.Client), "http://rtb.e-planning.net/pbs/1")
	eplanningAdapter.testing = true
	adapterstest.RunJSONBidderTest(t, "eplanningtest", eplanningAdapter)
}
