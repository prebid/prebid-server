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

	match, err := regexp.MatchString(`https://example\.host\.tappx\.com/DUMMYENDPOINT\?tappxkey=dummy-tappx-key&ts=[0-9]{13}&type_cnn=prebid&v=1\.1`, url)
	if err != nil {
		t.Errorf("Error while running regex validation: %s", err.Error())
		return
	}
	assert.True(t, match)
}
