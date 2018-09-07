package eplanning

import (
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	eplanningAdapter := NewEPlanningBidder(new(http.Client), "http://ads.us.e-planning.net/dsp/obr/1")
	adapterstest.RunJSONBidderTest(t, "eplanningtest", eplanningAdapter)
}
