package tappx

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"net/http"
	"regexp"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "tappxtest", NewTappxBidder(new(http.Client), "https://{{.Host}}"))
}

func TestTsValue(t *testing.T) {
	adapter := NewTappxBidder(new(http.Client), "https://{{.Host}}")
	var test int
	test = 0
	var tappxExt openrtb_ext.ExtImpTappx
	tappxExt.Host = "example.host.tappx.com"
	tappxExt.Endpoint = "DUMMYENDPOINT"
	tappxExt.TappxKey = "dummy-tappx-key"

	url, err := adapter.buildEndpointURL(&tappxExt, test)

	match, err := regexp.MatchString(`&ts=[0-9]{13}`, url)
	if err != nil {
		//something happened during regex validation
	}
	assert.True(t, match)
}
