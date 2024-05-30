package metax

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters/adapterstest"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/prebid/prebid-server/v2/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderMetaX,
		config.Adapter{
			Endpoint: "https://hb.metaxads.com/prebid?sid={{.PublisherID}}&adunit={{.AdUnit}}&source=prebid-server",
		},
		config.Server{
			ExternalUrl: "http://hosturl.com",
			GvlID:       1301,
			DataCenter:  "2",
		},
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "metaxtest", bidder)
}

func TestAssignBannerSize(t *testing.T) {
	b1 := &openrtb2.Banner{
		Format: []openrtb2.Format{
			{W: 300, H: 250},
			{W: 728, H: 90},
		},
	}
	b1n, err := assignBannerSize(b1)
	assert.Equal(t, b1n.W, ptrutil.ToPtr(int64(300)))
	assert.Equal(t, b1n.H, ptrutil.ToPtr(int64(250)))
	assert.Nil(t, err)

	b2 := &openrtb2.Banner{
		Format: []openrtb2.Format{
			{W: 300, H: 250},
			{W: 728, H: 90},
		},
		W: ptrutil.ToPtr(int64(1080)),
		H: ptrutil.ToPtr(int64(720)),
	}
	b2n, err := assignBannerSize(b2)
	assert.Equal(t, b2n.W, ptrutil.ToPtr(int64(1080)))
	assert.Equal(t, b2n.H, ptrutil.ToPtr(int64(720)))
	assert.Nil(t, err)
}
