package triplelift_native

import (
	"testing"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestBadConfig(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderTripleliftNative, config.Adapter{
		Endpoint:         `http://tlx.3lift.net/s2sn/auction?supplier_id=20`,
		ExtraAdapterInfo: `{foo:2}`,
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
}

func TestEmptyConfig(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderTripleliftNative, config.Adapter{
		Endpoint:         `http://tlx.3lift.net/s2sn/auction?supplier_id=20`,
		ExtraAdapterInfo: ``,
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	bidderTripleliftNative := bidder.(*TripleliftNativeAdapter)

	assert.NoError(t, buildErr)
	assert.Empty(t, bidderTripleliftNative.extInfo.PublisherWhitelist)
	assert.Empty(t, bidderTripleliftNative.extInfo.PublisherWhitelistMap)
}

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderTripleliftNative, config.Adapter{
		Endpoint:         `http://tlx.3lift.net/s2sn/auction?supplier_id=20`,
		ExtraAdapterInfo: `{"publisher_whitelist":["foo","bar","baz"]}`,
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "triplelift_nativetest", bidder)
}
