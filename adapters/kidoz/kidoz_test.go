package kidoz

import (
	"math"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderKidoz, config.Adapter{
		Endpoint: "http://example.com/prebid"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "kidoztest", bidder)
}

func makeBidRequest() *openrtb2.BidRequest {
	request := &openrtb2.BidRequest{
		ID: "test-req-id-0",
		Imp: []openrtb2.Imp{
			{
				ID: "test-imp-id-0",
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						{
							W: 320,
							H: 50,
						},
					},
				},
				Ext: []byte(`{"bidder":{"access_token":"token-0","publisher_id":"pub-0"}}`),
			},
		},
	}
	return request
}

func TestMakeRequests(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderKidoz, config.Adapter{
		Endpoint: "http://example.com/prebid"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	t.Run("Handles Request marshal failure", func(t *testing.T) {
		request := makeBidRequest()
		request.Imp[0].BidFloor = math.Inf(1) // cant be marshalled
		extra := &adapters.ExtraRequestInfo{}
		reqs, errs := bidder.MakeRequests(request, extra)
		// cant assert message its different on different versions of go
		assert.Equal(t, 1, len(errs))
		assert.Contains(t, errs[0].Error(), "json")
		assert.Equal(t, 0, len(reqs))
	})
}

func TestMakeBids(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderKidoz, config.Adapter{
		Endpoint: "http://example.com/prebid"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	t.Run("Handles response marshal failure", func(t *testing.T) {
		request := makeBidRequest()
		requestData := &adapters.RequestData{}
		responseData := &adapters.ResponseData{
			StatusCode: http.StatusOK,
		}

		resp, errs := bidder.MakeBids(request, requestData, responseData)
		// cant assert message its different on different versions of go
		assert.Equal(t, 1, len(errs))
		assert.Contains(t, errs[0].Error(), "expect { or n, but found")
		assert.Nil(t, resp)
	})
}

func TestGetMediaTypeForImp(t *testing.T) {
	imps := []openrtb2.Imp{
		{
			ID:     "1",
			Banner: &openrtb2.Banner{},
		},
		{
			ID:    "2",
			Video: &openrtb2.Video{},
		},
		{
			ID:     "3",
			Native: &openrtb2.Native{},
		},
		{
			ID:    "4",
			Audio: &openrtb2.Audio{},
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
