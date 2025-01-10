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
	tests := []struct {
		description   string
		imp           openrtb2.Imp
		expectedType  openrtb_ext.BidType
		expectedError string
	}{
		{
			description:   "Error case: empty imp",
			imp:           openrtb2.Imp{},
			expectedError: "Could not infer bid type from imp",
		},
		{
			description:  "Banner type",
			imp:          openrtb2.Imp{Banner: &openrtb2.Banner{}},
			expectedType: openrtb_ext.BidTypeBanner,
		},
		{
			description:  "Native type",
			imp:          openrtb2.Imp{Native: &openrtb2.Native{}},
			expectedType: openrtb_ext.BidTypeNative,
		},
		{
			description:  "Video type",
			imp:          openrtb2.Imp{Video: &openrtb2.Video{}},
			expectedType: openrtb_ext.BidTypeVideo,
		},
		{
			description:  "Audio type",
			imp:          openrtb2.Imp{Audio: &openrtb2.Audio{}},
			expectedType: openrtb_ext.BidTypeAudio,
		},
		{
			description:   "Unsupported type",
			imp:           openrtb2.Imp{PMP: &openrtb2.PMP{}},
			expectedError: "Could not infer bid type from imp",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			inferredType, err := inferBidTypeFromImp(test.imp)

			if test.expectedError != "" {
				assert.EqualError(t, err[0], test.expectedError)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.expectedType, inferredType)
			}
		})
	}
}

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdhese, config.Adapter{
		Endpoint: "https://{{.AccountID}}.foo.bar/"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

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
