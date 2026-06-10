package ezoic

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

const testsBidderEndpoint = "https://g.ezoic.net/ezoic/prebid/adapter/ortb"

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderEzoic, config.Adapter{
		Endpoint: testsBidderEndpoint}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 347, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "ezoictest", bidder)
}

func TestNoContentResponse(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderEzoic, config.Adapter{
		Endpoint: testsBidderEndpoint}, config.Server{})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	bidResponse, errs := bidder.MakeBids(nil, nil, &adapters.ResponseData{StatusCode: 204})
	assert.Nil(t, bidResponse)
	assert.Empty(t, errs)
}

func TestGetMediaTypeForBid(t *testing.T) {
	bidType, err := getMediaTypeForBid(openrtb2.Bid{MType: openrtb2.MarkupBanner})
	assert.NoError(t, err)
	assert.Equal(t, openrtb_ext.BidTypeBanner, bidType)

	bidType, err = getMediaTypeForBid(openrtb2.Bid{MType: openrtb2.MarkupVideo})
	assert.NoError(t, err)
	assert.Equal(t, openrtb_ext.BidTypeVideo, bidType)

	bidType, err = getMediaTypeForBid(openrtb2.Bid{MType: openrtb2.MarkupNative})
	assert.NoError(t, err)
	assert.Equal(t, openrtb_ext.BidTypeNative, bidType)

	_, err = getMediaTypeForBid(openrtb2.Bid{ID: "no-mtype"})
	assert.Error(t, err)

	_, err = getMediaTypeForBid(openrtb2.Bid{ID: "audio", MType: openrtb2.MarkupAudio})
	assert.Error(t, err)
}
