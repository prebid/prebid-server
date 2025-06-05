package exco

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderExco,
		config.Adapter{
			Endpoint: "https://testjsonsample.com",
		},
		config.Server{
			ExternalUrl: "http://hosturl.com",
			GvlID:       1,
			DataCenter:  "2",
		},
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "excotest", bidder)
}

func TestMediaTypeForBid(t *testing.T) {
	_, err := getMediaTypeForBid(&openrtb2.Bid{
		MType: 0,
	})
	require.Error(t, err, "Should raise Error in case of unsupported Media Type")

	mediaType, err := getMediaTypeForBid(&openrtb2.Bid{
		MType: 1,
	})
	require.NoError(t, err, "Failed to detect Media Type")
	assert.Equal(t, openrtb_ext.BidTypeBanner, mediaType, "Failed to detect Media Type")

	mediaType, err = getMediaTypeForBid(&openrtb2.Bid{
		MType: 2,
	})
	require.NoError(t, err, "Failed to detect Media Type")
	assert.Equal(t, openrtb_ext.BidTypeVideo, mediaType, "Failed to detect Media Type")
}
