package bwx

import (
	"testing"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderBWX,
		config.Adapter{
			Endpoint: "http://rtb.boldwin.live/?pid={{.SourceId}}&host={{.Host}}&pbs=1",
		},
		config.Server{
			ExternalUrl: "http://hosturl.com",
			GvlID:       1,
			DataCenter:  "2",
		},
	)

	assert.NoError(t, buildErr)
	adapterstest.RunJSONBidderTest(t, "bwxtest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderBWX, config.Adapter{
		Endpoint: "{{Malformed}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
}
