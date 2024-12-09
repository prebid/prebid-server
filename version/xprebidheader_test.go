package version

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

func TestBuildXPrebidHeader(t *testing.T) {
	testCases := []struct {
		description string
		version     string
		result      string
	}{
		{
			description: "No Version",
			version:     "",
			result:      "pbs-go/unknown",
		},
		{
			description: "Version",
			version:     "0.100.0",
			result:      "pbs-go/0.100.0",
		},
	}

	for _, test := range testCases {
		result := BuildXPrebidHeader(test.version)
		assert.Equal(t, test.result, result, test.description)
	}
}

func TestBuildXPrebidHeaderForRequest(t *testing.T) {
	testCases := []struct {
		description   string
		version       string
		requestExt    *openrtb_ext.ExtRequest
		requestAppExt *openrtb_ext.ExtApp
		result        string
	}{
		{
			description: "No versions",
			version:     "",
			result:      "pbs-go/unknown",
		},
		{
			description: "pbs",
			version:     "test-version",
			result:      "pbs-go/test-version",
		},
		{
			description: "prebid.js",
			version:     "test-version",
			requestExt: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Channel: &openrtb_ext.ExtRequestPrebidChannel{
						Name:    "pbjs",
						Version: "test-pbjs-version",
					},
				},
			},
			result: "pbs-go/test-version,pbjs/test-pbjs-version",
		},
		{
			description: "unknown prebid.js",
			version:     "test-version",
			requestExt: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Channel: &openrtb_ext.ExtRequestPrebidChannel{
						Name: "pbjs",
					},
				},
			},
			result: "pbs-go/test-version,pbjs/unknown",
		},
		{
			description: "channel without a name",
			version:     "test-version",
			requestExt: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Channel: &openrtb_ext.ExtRequestPrebidChannel{
						Version: "test-version",
					},
				},
			},
			result: "pbs-go/test-version",
		},
		{
			description: "prebid-mobile",
			version:     "test-version",
			requestAppExt: &openrtb_ext.ExtApp{
				Prebid: openrtb_ext.ExtAppPrebid{
					Source:  "prebid-mobile",
					Version: "test-prebid-mobile-version",
				},
			},
			result: "pbs-go/test-version,prebid-mobile/test-prebid-mobile-version",
		},
		{
			description: "app ext without a source",
			version:     "test-version",
			requestAppExt: &openrtb_ext.ExtApp{
				Prebid: openrtb_ext.ExtAppPrebid{
					Version: "test-version",
				},
			},
			result: "pbs-go/test-version",
		},
		{
			description: "Version found in both req.Ext and req.App.Ext",
			version:     "test-version",
			requestExt: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Channel: &openrtb_ext.ExtRequestPrebidChannel{
						Name:    "pbjs",
						Version: "test-pbjs-version",
					},
				},
			},
			requestAppExt: &openrtb_ext.ExtApp{
				Prebid: openrtb_ext.ExtAppPrebid{
					Source:  "prebid-mobile",
					Version: "test-prebid-mobile-version",
				},
			},
			result: "pbs-go/test-version,pbjs/test-pbjs-version,prebid-mobile/test-prebid-mobile-version",
		},
	}

	for _, test := range testCases {
		req := &openrtb2.BidRequest{}
		if test.requestExt != nil {
			reqExt, err := jsonutil.Marshal(test.requestExt)
			assert.NoError(t, err, test.description+":err marshalling reqExt")
			req.Ext = reqExt
		}
		if test.requestAppExt != nil {
			reqAppExt, err := jsonutil.Marshal(test.requestAppExt)
			assert.NoError(t, err, test.description+":err marshalling reqAppExt")
			req.App = &openrtb2.App{Ext: reqAppExt}
		}
		result := BuildXPrebidHeaderForRequest(req, test.version)
		assert.Equal(t, test.result, result, test.description+":result")
	}
}
