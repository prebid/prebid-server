package tappx

import (
	"regexp"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderTappx, config.Adapter{
		Endpoint: "http://{{.Host}}"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "tappxtest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderTappx, config.Adapter{
		Endpoint: "{{Malformed}}"})

	assert.Error(t, buildErr)
}

func TestTsValue(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderTappx, config.Adapter{
		Endpoint: "http://{{.Host}}"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	bidderTappx := bidder.(*TappxAdapter)

	var test int
	test = 0
	var tappxExt openrtb_ext.ExtImpTappx
	tappxExt.Host = "example.host.tappx.com"
	tappxExt.Endpoint = "DUMMYENDPOINT"
	tappxExt.TappxKey = "dummy-tappx-key"

	url, err := bidderTappx.buildEndpointURL(&tappxExt, test)

	match, err := regexp.MatchString(`http://example\.host\.tappx\.com/DUMMYENDPOINT\?tappxkey=dummy-tappx-key&ts=[0-9]{13}&type_cnn=prebid&v=1\.3`, url)
	if err != nil {
		t.Errorf("Error while running regex validation: %s", err.Error())
		return
	}
	assert.True(t, match)
}
