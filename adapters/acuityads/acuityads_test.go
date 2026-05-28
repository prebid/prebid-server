package acuityads

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAcuityAds, config.Adapter{
		Endpoint: "http://{{.Host}}.example.com/bid?token={{.AccountID}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "acuityadstest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderAcuityAds, config.Adapter{
		Endpoint: "{{Malformed}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
}

func TestBuildEndpointURLRejectsUnsafeHost(t *testing.T) {
	bidder := &AcuityAdsAdapter{endpoint: template.Must(template.New("endpointTemplate").Parse("http://{{.Host}}.example.com/bid"))}
	_, err := bidder.buildEndpointURL(&openrtb_ext.ExtAcuityAds{Host: "127.0.0.1:6060/debug/pprof#", AccountID: "account"})
	assert.Error(t, err)
}
