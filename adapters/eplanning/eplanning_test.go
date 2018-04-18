package eplanning

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"testing"

	"net/http"
)

func TestJsonSamples(t *testing.T) {
	eplanningAdapter := NewEPlanningBidder(new(http.Client), "http://ads.us.e-planning.net/dsp/obr/1")
	adapterstest.RunJSONBidderTest(t, "eplanningtest", eplanningAdapter)
}
