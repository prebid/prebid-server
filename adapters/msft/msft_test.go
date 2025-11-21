package msft

import (
	"testing"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderMicrosoft,
		config.Adapter{Endpoint: "http://any.url"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "test", bidder)
}

func TestJsonSamplesWithExtraInfo(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderMicrosoft,
		config.Adapter{Endpoint: "http://any.url", ExtraAdapterInfo: `{"hb_source": 50, "hb_source_video": 60}`},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "test-extrainfo", bidder)
}

func TestBadConfig(t *testing.T) {
	t.Run("invalid", func(t *testing.T) {
		_, buildErr := Builder(openrtb_ext.BidderMicrosoft,
			config.Adapter{
				Endpoint:         `http://any.url`,
				ExtraAdapterInfo: `malformed`,
			},
			config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

		assert.Error(t, buildErr)
	})

	t.Run("empty-value", func(t *testing.T) {
		bidder, buildErr := Builder(openrtb_ext.BidderMicrosoft,
			config.Adapter{
				Endpoint:         `http://any.url`,
				ExtraAdapterInfo: ``,
			},
			config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

		adapter := bidder.(*adapter)

		assert.NoError(t, buildErr)
		assert.Equal(t, adapter.hbSource, defaultHBSource)
		assert.Equal(t, adapter.hbSourceVideo, defaultHBSourceVideo)
	})

	t.Run("empty-object", func(t *testing.T) {
		bidder, buildErr := Builder(openrtb_ext.BidderMicrosoft,
			config.Adapter{
				Endpoint:         `http://any.url`,
				ExtraAdapterInfo: `{}`,
			},
			config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

		adapter := bidder.(*adapter)

		assert.NoError(t, buildErr)
		assert.Equal(t, adapter.hbSource, defaultHBSource)
		assert.Equal(t, adapter.hbSourceVideo, defaultHBSourceVideo)
	})
}
