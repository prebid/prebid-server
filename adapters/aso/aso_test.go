package aso

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAso, config.Adapter{
		Endpoint: "https://srv.aso1.net/pbs/bidder?zid={{.ZoneID}}"}, config.Server{ExternalUrl: "http://hosturl.com"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "asotest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderAso, config.Adapter{
		Endpoint: "zid={{ZoneID}}"}, config.Server{ExternalUrl: "http://hosturl.com"})

	assert.Error(t, buildErr)
}
