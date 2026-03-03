package aduptech

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderAdUpTech,
		config.Adapter{
			Endpoint:         "https://example.com/rtb/bid",
			ExtraAdapterInfo: `{"target_currency": "EUR"}`,
		},
		config.Server{
			ExternalUrl: "http://hosturl.com",
			GvlID:       1,
			DataCenter:  "2",
		},
	)

	require.NoError(t, buildErr, "Builder returned unexpected error")

	adapterstest.RunJSONBidderTest(t, "aduptechtest", bidder)
}

func TestInvalidExtraAdapterInfo(t *testing.T) {
	_, buildErr := Builder(
		openrtb_ext.BidderAdUpTech,
		config.Adapter{
			Endpoint:         "https://example.com/rtb/bid",
			ExtraAdapterInfo: `{"foo": "bar"}`,
		},
		config.Server{
			ExternalUrl: "http://hosturl.com",
			GvlID:       1,
			DataCenter:  "2",
		},
	)

	assert.EqualError(t, buildErr, "invalid extra info: TargetCurrency is empty, pls check")
}

func TestInvalidTargetCurrency(t *testing.T) {
	_, buildErr := Builder(
		openrtb_ext.BidderAdUpTech,
		config.Adapter{
			Endpoint:         "https://example.com/rtb/bid",
			ExtraAdapterInfo: `{"target_currency": "INVALID"}`,
		},
		config.Server{
			ExternalUrl: "http://hosturl.com",
			GvlID:       1,
			DataCenter:  "2",
		},
	)

	assert.EqualError(t, buildErr, "invalid extra info: invalid TargetCurrency INVALID, pls check")
}
