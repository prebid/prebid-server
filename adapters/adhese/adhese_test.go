package adhese

import (
	"github.com/prebid/openrtb/v20/openrtb2"
	"testing"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestInferBidType(t *testing.T) {
	imp := openrtb2.Imp{}

	_, err := inferBidTypeFromImp(imp)

	assert.EqualError(t, err[0], "Could not infer bid type from imp", "Error should be 'Could not infer bid type from imp'")
	assert.NotEmpty(t, err, "Error should not be empty")

	// Test for banner type
	var bannerImp openrtb2.Imp
	bannerImp.Banner = &openrtb2.Banner{}
	inferredType, err := inferBidTypeFromImp(bannerImp)
	assert.Nil(t, err, "Error should be nil")
	assert.Equal(t, openrtb_ext.BidTypeBanner, inferredType, "Inferred type should be 'Banner'")

	// Test for native type
	var nativeImp openrtb2.Imp
	nativeImp.Native = &openrtb2.Native{}
	nativeImp.Native = &openrtb2.Native{}
	inferredType, err = inferBidTypeFromImp(nativeImp)
	assert.Nil(t, err, "Error should be nil")
	assert.Equal(t, openrtb_ext.BidTypeNative, inferredType, "Inferred type should be 'Native'")

	// Test for video type
	var videoImp openrtb2.Imp
	videoImp.Video = &openrtb2.Video{}
	inferredType, err = inferBidTypeFromImp(videoImp)
	assert.Nil(t, err, "Error should be nil")
	assert.Equal(t, openrtb_ext.BidTypeVideo, inferredType, "Inferred type should be 'Video'")

	// Test for audio type
	var audioImp openrtb2.Imp
	audioImp.Audio = &openrtb2.Audio{}
	inferredType, err = inferBidTypeFromImp(audioImp)
	assert.Nil(t, err, "Error should be nil")
	assert.Equal(t, openrtb_ext.BidTypeAudio, inferredType, "Inferred type should be 'Audio'")

	// Test for unsupported type
	var unsupportedImp openrtb2.Imp
	unsupportedImp.PMP = &openrtb2.PMP{}
	inferredType, err = inferBidTypeFromImp(unsupportedImp)
	assert.EqualError(t, err[0], "Could not infer bid type from imp", "Error should be 'Could not infer bid type from imp'")
	assert.NotEmpty(t, err, "Error should not be empty")
}

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdhese, config.Adapter{
		Endpoint: "https://ads-{{.AccountID}}.adhese.com/openrtb2"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "adhesetest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderAdhese, config.Adapter{
		Endpoint: "{{Malformed}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
}
