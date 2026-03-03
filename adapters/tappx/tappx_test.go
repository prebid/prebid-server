package tappx

import (
	"regexp"
	"testing"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderTappx, config.Adapter{
		Endpoint: "http://{{.Host}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "tappxtest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderTappx, config.Adapter{
		Endpoint: "{{Malformed}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
}

func TestTsValue(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderTappx, config.Adapter{
		Endpoint: "http://{{.Host}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	bidderTappx := bidder.(*TappxAdapter)

	test := 0
	var tappxExt openrtb_ext.ExtImpTappx
	tappxExt.Endpoint = "DUMMYENDPOINT"
	tappxExt.TappxKey = "dummy-tappx-key"

	url, err := bidderTappx.buildEndpointURL(&tappxExt, test)
	require.NoError(t, err, "buildEndpointURL")

	match, err := regexp.MatchString(`http://ssp\.api\.tappx\.com/rtb/v2/DUMMYENDPOINT\?tappxkey=dummy-tappx-key&ts=[0-9]{13}&type_cnn=prebid&v=1\.6`, url)
	if err != nil {
		t.Errorf("Error while running regex validation: %s", err.Error())
		return
	}
	assert.True(t, match)
}
