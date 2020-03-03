package kidoz

import (
	"math"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "kidoztest", NewKidozBidder("http://example.com/prebid"))
}

func TestMakeRequests(t *testing.T) {
	kidoz := NewKidozBidder("http://example.com/prebid")

	t.Run("Handles Request marshal failure", func(t *testing.T) {
		request := &openrtb.BidRequest{
			ID: "test-req-id-0",
			Imp: []openrtb.Imp{
				{
					ID: "test-imp-id-0",
					Banner: &openrtb.Banner{
						Format: []openrtb.Format{
							{
								W: 320,
								H: 50,
							},
						},
					},
					Ext:      []byte(`{"bidder":{"access_token":"token-0","publisher_id":"pub-0"}}`),
					BidFloor: math.Inf(1), // cant be marshalled
				},
			},
		}
		extra := &adapters.ExtraRequestInfo{}
		reqs, errs := kidoz.MakeRequests(request, extra)
		assert.Equal(t, 1, len(errs))
		assert.Equal(t, 0, len(reqs))
	})
}

func TestGetMediaTypeForImp(t *testing.T) {
	imps := []openrtb.Imp{
		{
			ID:     "1",
			Banner: &openrtb.Banner{},
		},
		{
			ID:    "2",
			Video: &openrtb.Video{},
		},
		{
			ID:     "3",
			Native: &openrtb.Native{},
		},
		{
			ID:    "4",
			Audio: &openrtb.Audio{},
		},
	}

	t.Run("Bid not found is type empty string", func(t *testing.T) {
		actual := GetMediaTypeForImp("ARGLE_BARGLE", imps)
		assert.Equal(t, UndefinedMediaType, actual)
	})
	t.Run("Can find banner type", func(t *testing.T) {
		actual := GetMediaTypeForImp("1", imps)
		assert.Equal(t, openrtb_ext.BidTypeBanner, actual)
	})
	t.Run("Can find video type", func(t *testing.T) {
		actual := GetMediaTypeForImp("2", imps)
		assert.Equal(t, openrtb_ext.BidTypeVideo, actual)
	})
	t.Run("Can find native type", func(t *testing.T) {
		actual := GetMediaTypeForImp("3", imps)
		assert.Equal(t, openrtb_ext.BidTypeNative, actual)
	})
	t.Run("Can find audio type", func(t *testing.T) {
		actual := GetMediaTypeForImp("4", imps)
		assert.Equal(t, openrtb_ext.BidTypeAudio, actual)
	})
}
