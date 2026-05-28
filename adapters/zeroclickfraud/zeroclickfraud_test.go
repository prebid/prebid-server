package zeroclickfraud

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderZeroClickFraud, config.Adapter{
		Endpoint: "http://{{.Host}}/openrtb2?sid={{.SourceId}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "zeroclickfraudtest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderZeroClickFraud, config.Adapter{
		Endpoint: "{{Malformed}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
}

func TestGetBidderParamsRejectsUnsafeHost(t *testing.T) {
	imp := openrtb2.Imp{Ext: []byte(`{"bidder":{"host":"127.0.0.1:6060/debug/pprof#","sourceId":1}}`)}
	_, err := getBidderParams(&imp)
	assert.Error(t, err)
}
