package adapters

import (
	"errors"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppNotSupported(t *testing.T) {
	bidder := &mockBidder{}
	info := config.BidderInfo{
		Capabilities: &config.CapabilitiesInfo{
			Site: &config.PlatformInfo{
				MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner},
			},
		},
	}
	constrained := BuildInfoAwareBidder(bidder, info)
	bids, errs := constrained.MakeRequests(&openrtb2.BidRequest{
		Imp: []openrtb2.Imp{{ID: "imp-1", Banner: &openrtb2.Banner{}}},
		App: &openrtb2.App{},
	}, &ExtraRequestInfo{})
	if !assert.Len(t, errs, 1) {
		return
	}
	assert.EqualError(t, errs[0], "this bidder does not support app requests")
	assert.IsType(t, &errortypes.Warning{}, errs[0])
	assert.Len(t, bids, 0)
}

func TestSiteNotSupported(t *testing.T) {
	bidder := &mockBidder{}
	info := config.BidderInfo{
		Capabilities: &config.CapabilitiesInfo{
			App: &config.PlatformInfo{
				MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner},
			},
		},
	}
	constrained := BuildInfoAwareBidder(bidder, info)
	bids, errs := constrained.MakeRequests(&openrtb2.BidRequest{
		Imp:  []openrtb2.Imp{{ID: "imp-1", Banner: &openrtb2.Banner{}}},
		Site: &openrtb2.Site{},
	}, &ExtraRequestInfo{})
	if !assert.Len(t, errs, 1) {
		return
	}
	assert.EqualError(t, errs[0], "this bidder does not support site requests")
	assert.IsType(t, &errortypes.Warning{}, errs[0])
	assert.Len(t, bids, 0)
}

func TestDOOHNotSupported(t *testing.T) {
	bidder := &mockBidder{}
	info := config.BidderInfo{
		Capabilities: &config.CapabilitiesInfo{
			Site: &config.PlatformInfo{
				MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner},
			},
		},
	}
	constrained := BuildInfoAwareBidder(bidder, info)
	bids, errs := constrained.MakeRequests(&openrtb2.BidRequest{
		Imp:  []openrtb2.Imp{{ID: "imp-1", Banner: &openrtb2.Banner{}}},
		DOOH: &openrtb2.DOOH{},
	}, &ExtraRequestInfo{})
	require.Len(t, errs, 1)
	assert.EqualError(t, errs[0], "this bidder does not support dooh requests")
	assert.IsType(t, &errortypes.Warning{}, errs[0])
	assert.Len(t, bids, 0)
}

func TestImpFiltering(t *testing.T) {
	bidder := &mockBidder{}
	info := config.BidderInfo{
		Capabilities: &config.CapabilitiesInfo{
			Site: &config.PlatformInfo{
				MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeVideo},
			},
			App: &config.PlatformInfo{
				MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner},
			},
			DOOH: &config.PlatformInfo{
				MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeNative},
			},
		},
	}

	constrained := BuildInfoAwareBidder(bidder, info)

	testCases := []struct {
		description    string
		inBidRequest   *openrtb2.BidRequest
		expectedErrors []error
		expectedImpLen int
	}{
		{
			description: "Empty Imp array. MakeRequest() call not expected",
			inBidRequest: &openrtb2.BidRequest{
				Imp:  []openrtb2.Imp{},
				Site: &openrtb2.Site{},
			},
			expectedErrors: []error{
				&errortypes.BadInput{Message: "Bid request didn't contain media types supported by the bidder"},
			},
			expectedImpLen: 0,
		},
		{
			description: "Sole imp in bid request is of wrong media type. MakeRequest() call not expected",
			inBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "imp-1", Video: &openrtb2.Video{}}},
				App: &openrtb2.App{},
			},
			expectedErrors: []error{
				&errortypes.BadInput{Message: "request.imp[0] uses video, but this bidder doesn't support it"},
				&errortypes.BadInput{Message: "request.imp[0] has no supported MediaTypes. It will be ignored"},
				&errortypes.BadInput{Message: "Bid request didn't contain media types supported by the bidder"},
			},
			expectedImpLen: 0,
		},
		{
			description: "All imps in bid request of wrong media type, MakeRequest() call not expected",
			inBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-1", Video: &openrtb2.Video{}},
					{ID: "imp-2", Native: &openrtb2.Native{}},
					{ID: "imp-3", Audio: &openrtb2.Audio{}},
				},
				App: &openrtb2.App{},
			},
			expectedErrors: []error{
				&errortypes.BadInput{Message: "request.imp[0] uses video, but this bidder doesn't support it"},
				&errortypes.BadInput{Message: "request.imp[0] has no supported MediaTypes. It will be ignored"},
				&errortypes.BadInput{Message: "request.imp[1] uses native, but this bidder doesn't support it"},
				&errortypes.BadInput{Message: "request.imp[1] has no supported MediaTypes. It will be ignored"},
				&errortypes.BadInput{Message: "request.imp[2] uses audio, but this bidder doesn't support it"},
				&errortypes.BadInput{Message: "request.imp[2] has no supported MediaTypes. It will be ignored"},
				&errortypes.BadInput{Message: "Bid request didn't contain media types supported by the bidder"},
			},
			expectedImpLen: 0,
		},
		{
			description: "Some imps with correct media type, MakeRequest() call expected",
			inBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:    "imp-1",
						Video: &openrtb2.Video{},
					},
					{
						Native: &openrtb2.Native{},
					},
					{
						ID:     "imp-2",
						Video:  &openrtb2.Video{},
						Native: &openrtb2.Native{},
					},
					{
						Banner: &openrtb2.Banner{},
					},
				},
				Site: &openrtb2.Site{},
			},
			expectedErrors: []error{
				&errortypes.BadInput{Message: "request.imp[1] uses native, but this bidder doesn't support it"},
				&errortypes.BadInput{Message: "request.imp[1] has no supported MediaTypes. It will be ignored"},
				&errortypes.BadInput{Message: "request.imp[2] uses native, but this bidder doesn't support it"},
				&errortypes.BadInput{Message: "request.imp[3] uses banner, but this bidder doesn't support it"},
				&errortypes.BadInput{Message: "request.imp[3] has no supported MediaTypes. It will be ignored"},
			},
			expectedImpLen: 2,
		},
		{
			description: "All imps with correct media type, MakeRequest() call expected",
			inBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-1", Native: &openrtb2.Native{}},
					{ID: "imp-2", Native: &openrtb2.Native{}},
				},
				DOOH: &openrtb2.DOOH{},
			},
			expectedErrors: nil,
			expectedImpLen: 2,
		},
	}

	for _, test := range testCases {
		actualAdapterRequests, actualErrs := constrained.MakeRequests(test.inBidRequest, &ExtraRequestInfo{})

		// Assert the request.Imp slice was correctly filtered and if MakeRequest() was called by asserting
		// the corresponding error messages were returned
		for i, expectedErr := range test.expectedErrors {
			assert.EqualError(t, expectedErr, actualErrs[i].Error(), "Test failed. Error[%d] in error list mismatch: %s", i, test.description)
		}

		// Extra MakeRequests() call check: our mockBidder returns an adapter request for every imp
		assert.Len(t, actualAdapterRequests, test.expectedImpLen, "Test failed. Incorrect length of filtered imps: %s", test.description)
	}
}

func TestImpFilteringForMultiFormatRequests(t *testing.T) {
	bidder := &mockBidder{}
	var falseValue bool
	info := config.BidderInfo{
		Capabilities: &config.CapabilitiesInfo{
			Site: &config.PlatformInfo{
				MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeVideo, openrtb_ext.BidTypeNative, openrtb_ext.BidTypeAudio},
			},
			App: &config.PlatformInfo{
				MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner},
			},
			DOOH: &config.PlatformInfo{
				MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeNative},
			},
		},
		OpenRTB: &config.OpenRTBInfo{
			MultiformatSupported: &falseValue,
		},
	}

	constrained := BuildInfoAwareBidder(bidder, info)

	testCases := []struct {
		description        string
		inBidRequest       *openrtb2.BidRequest
		inExtraRequestInfo *ExtraRequestInfo
		expectedErrors     []error
		expectedImpLen     int
	}{
		{
			description: "All imps with preferred media type, MakeRequest() call expected",
			inBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:     "imp-1",
						Banner: &openrtb2.Banner{},
						Video:  &openrtb2.Video{},
					},
					{
						ID:     "imp-2",
						Banner: &openrtb2.Banner{},
						Native: &openrtb2.Native{},
					},
				},
				Site: &openrtb2.Site{},
			},
			inExtraRequestInfo: &ExtraRequestInfo{
				PreferredMediaType: openrtb_ext.BidTypeBanner,
			},
			expectedErrors: []error{
				&errortypes.Warning{Message: "Imp imp-1 uses video, removing video as the bidder doesn't support multi-format"},
				&errortypes.Warning{Message: "Imp imp-2 uses native, removing native as the bidder doesn't support multi-format"},
			},
			expectedImpLen: 2,
		},
		{
			description: "Some imps with preferred media type, MakeRequest() call expected",
			inBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:     "imp-1",
						Banner: &openrtb2.Banner{},
						Video:  &openrtb2.Video{},
					},
					{
						ID:     "imp-2",
						Video:  &openrtb2.Video{},
						Native: &openrtb2.Native{},
					},
					{
						ID:    "imp-3",
						Video: &openrtb2.Video{},
						Audio: &openrtb2.Audio{},
					},
				},
				Site: &openrtb2.Site{},
			},
			inExtraRequestInfo: &ExtraRequestInfo{
				PreferredMediaType: openrtb_ext.BidTypeBanner,
			},
			expectedErrors: []error{
				&errortypes.Warning{Message: "Imp imp-1 uses video, removing video as the bidder doesn't support multi-format"},
				&errortypes.BadInput{Message: "Imp imp-2 does not have a preferred BANNER media type. It will be ignored."},
				&errortypes.BadInput{Message: "Imp imp-3 does not have a preferred BANNER media type. It will be ignored."},
			},
			expectedImpLen: 1,
		},
		{
			description: "No imps with preferred media type, MakeRequest() call not expected",
			inBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:     "imp-1",
						Banner: &openrtb2.Banner{},
						Video:  &openrtb2.Video{},
					},
					{
						ID:     "imp-2",
						Video:  &openrtb2.Video{},
						Native: &openrtb2.Native{},
					},
				},
				Site: &openrtb2.Site{},
			},
			inExtraRequestInfo: &ExtraRequestInfo{
				PreferredMediaType: openrtb_ext.BidTypeAudio,
			},
			expectedErrors: []error{
				&errortypes.BadInput{Message: "Imp imp-1 does not have a preferred AUDIO media type. It will be ignored."},
				&errortypes.BadInput{Message: "Imp imp-2 does not have a preferred AUDIO media type. It will be ignored."},
				&errortypes.BadInput{Message: "Bid request didn't contain media types supported by the bidder"},
			},
			expectedImpLen: 0,
		},
		{
			description: "preferred media type not defined, MakeRequest() call not expected",
			inBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:     "imp-1",
						Banner: &openrtb2.Banner{},
						Video:  &openrtb2.Video{},
					},
				},
				Site: &openrtb2.Site{},
			},
			inExtraRequestInfo: &ExtraRequestInfo{
				PreferredMediaType: "",
			},
			expectedErrors: []error{
				&errortypes.BadInput{Message: "Removing the imp imp-1 as the bidder does not support multi-format and preferred media type is not defined for the bidder"},
			},
			expectedImpLen: 0,
		},
	}

	for _, test := range testCases {
		actualAdapterRequests, actualErrs := constrained.MakeRequests(test.inBidRequest, test.inExtraRequestInfo)

		for i, expectedErr := range test.expectedErrors {
			assert.EqualError(t, expectedErr, actualErrs[i].Error(), "Test failed. Error[%d] in error list mismatch: %s", i, test.description)
		}
		assert.Len(t, actualAdapterRequests, test.expectedImpLen, "Test failed. Incorrect length of filtered imps: %s", test.description)
	}
}

func TestAdjustImpForPreferredMediaType(t *testing.T) {
	testCases := []struct {
		description        string
		inImp              openrtb2.Imp
		preferredMediaType openrtb_ext.BidType
		expectedImp        openrtb2.Imp
		expectedErrors     []error
	}{
		{
			description: "Multiformat impression with all media types and preferred media type Banner",
			inImp: openrtb2.Imp{
				ID:     "imp-1",
				Banner: &openrtb2.Banner{},
				Video:  &openrtb2.Video{},
				Audio:  &openrtb2.Audio{},
				Native: &openrtb2.Native{},
			},
			preferredMediaType: openrtb_ext.BidTypeBanner,
			expectedImp: openrtb2.Imp{
				ID:     "imp-1",
				Banner: &openrtb2.Banner{},
			},
			expectedErrors: []error{
				&errortypes.Warning{Message: "Imp imp-1 uses video, removing video as the bidder doesn't support multi-format"},
				&errortypes.Warning{Message: "Imp imp-1 uses audio, removing audio as the bidder doesn't support multi-format"},
				&errortypes.Warning{Message: "Imp imp-1 uses native, removing native as the bidder doesn't support multi-format"},
			},
		},
		{
			description: "Multiformat impression with all media types and preferred media type Video",
			inImp: openrtb2.Imp{
				ID:     "imp-1",
				Banner: &openrtb2.Banner{},
				Video:  &openrtb2.Video{},
				Audio:  &openrtb2.Audio{},
				Native: &openrtb2.Native{},
			},
			preferredMediaType: openrtb_ext.BidTypeVideo,
			expectedImp: openrtb2.Imp{
				ID:    "imp-1",
				Video: &openrtb2.Video{},
			},
			expectedErrors: []error{
				&errortypes.Warning{Message: "Imp imp-1 uses banner, removing banner as the bidder doesn't support multi-format"},
				&errortypes.Warning{Message: "Imp imp-1 uses audio, removing audio as the bidder doesn't support multi-format"},
				&errortypes.Warning{Message: "Imp imp-1 uses native, removing native as the bidder doesn't support multi-format"},
			},
		},
		{
			description: "Multiformat impression with all media types and preferred media type Audio",
			inImp: openrtb2.Imp{
				ID:     "imp-1",
				Banner: &openrtb2.Banner{},
				Video:  &openrtb2.Video{},
				Audio:  &openrtb2.Audio{},
				Native: &openrtb2.Native{},
			},
			preferredMediaType: openrtb_ext.BidTypeAudio,
			expectedImp: openrtb2.Imp{
				ID:    "imp-1",
				Audio: &openrtb2.Audio{},
			},
			expectedErrors: []error{
				&errortypes.Warning{Message: "Imp imp-1 uses banner, removing banner as the bidder doesn't support multi-format"},
				&errortypes.Warning{Message: "Imp imp-1 uses video, removing video as the bidder doesn't support multi-format"},
				&errortypes.Warning{Message: "Imp imp-1 uses native, removing native as the bidder doesn't support multi-format"},
			},
		},
		{
			description: "Multiformat impression with all media types and preferred media type Native",
			inImp: openrtb2.Imp{
				ID:     "imp-1",
				Banner: &openrtb2.Banner{},
				Video:  &openrtb2.Video{},
				Audio:  &openrtb2.Audio{},
				Native: &openrtb2.Native{},
			},
			preferredMediaType: openrtb_ext.BidTypeNative,
			expectedImp: openrtb2.Imp{
				ID:     "imp-1",
				Native: &openrtb2.Native{},
			},
			expectedErrors: []error{
				&errortypes.Warning{Message: "Imp imp-1 uses banner, removing banner as the bidder doesn't support multi-format"},
				&errortypes.Warning{Message: "Imp imp-1 uses video, removing video as the bidder doesn't support multi-format"},
				&errortypes.Warning{Message: "Imp imp-1 uses audio, removing audio as the bidder doesn't support multi-format"},
			},
		},
		{
			description: "Invalid Banner media type",
			inImp: openrtb2.Imp{
				ID:    "imp-1",
				Video: &openrtb2.Video{},
			},
			preferredMediaType: openrtb_ext.BidTypeBanner,
			expectedImp:        openrtb2.Imp{},
			expectedErrors: []error{
				&errortypes.BadInput{Message: "Imp imp-1 does not have a preferred BANNER media type. It will be ignored."},
			},
		},
		{
			description: "Invalid Video media type",
			inImp: openrtb2.Imp{
				ID:     "imp-1",
				Banner: &openrtb2.Banner{},
			},
			preferredMediaType: openrtb_ext.BidTypeVideo,
			expectedImp:        openrtb2.Imp{},
			expectedErrors: []error{
				&errortypes.BadInput{Message: "Imp imp-1 does not have a preferred VIDEO media type. It will be ignored."},
			},
		},
		{
			description: "Invalid Audio media type",
			inImp: openrtb2.Imp{
				ID:     "imp-1",
				Banner: &openrtb2.Banner{},
			},
			preferredMediaType: openrtb_ext.BidTypeAudio,
			expectedImp:        openrtb2.Imp{},
			expectedErrors: []error{
				&errortypes.BadInput{Message: "Imp imp-1 does not have a preferred AUDIO media type. It will be ignored."},
			},
		},
		{
			description: "Invalid Native media type",
			inImp: openrtb2.Imp{
				ID:     "imp-1",
				Banner: &openrtb2.Banner{},
			},
			preferredMediaType: openrtb_ext.BidTypeNative,
			expectedImp:        openrtb2.Imp{},
			expectedErrors: []error{
				&errortypes.BadInput{Message: "Imp imp-1 does not have a preferred NATIVE media type. It will be ignored."},
			},
		},
		{
			description: "Invalid preferred media type",
			inImp: openrtb2.Imp{
				ID:     "imp-1",
				Banner: &openrtb2.Banner{},
			},
			preferredMediaType: "invalid",
			expectedImp:        openrtb2.Imp{},
			expectedErrors: []error{
				&errortypes.BadInput{Message: "Imp imp-1 has an invalid preferred media type: invalid. It will be ignored."},
			},
		},
		{
			description: "Multiformat impression with preferred media type not defined",
			inImp: openrtb2.Imp{
				ID:     "imp-1",
				Banner: &openrtb2.Banner{},
				Video:  &openrtb2.Video{},
				Audio:  &openrtb2.Audio{},
				Native: &openrtb2.Native{},
			},
			preferredMediaType: "",
			expectedImp:        openrtb2.Imp{},
			expectedErrors: []error{
				&errortypes.BadInput{Message: "Removing the imp imp-1 as the bidder does not support multi-format and preferred media type is not defined for the bidder"},
			},
		},
	}

	for _, test := range testCases {
		_, actualErrs := adjustImpForPreferredMediaType(&test.inImp, test.preferredMediaType)
		if test.expectedErrors != nil {
			for i, expectedErr := range test.expectedErrors {
				assert.EqualError(t, expectedErr, actualErrs[i].Error(), "Test failed. Error[%d] in error list mismatch: %s", i, test.description)
			}
		} else {
			assert.Equal(t, test.expectedImp, test.inImp, test.description)
		}
	}
}

func TestIsMultiFormatSupported(t *testing.T) {
	trueValue, falseValue := true, false
	testCases := []struct {
		description string
		bidderInfo  config.BidderInfo
		expected    bool
	}{
		{
			description: "MultiformatSupported is true",
			bidderInfo: config.BidderInfo{
				OpenRTB: &config.OpenRTBInfo{
					MultiformatSupported: &trueValue,
				},
			},
			expected: true,
		},
		{
			description: "MultiformatSupported is false",
			bidderInfo: config.BidderInfo{
				OpenRTB: &config.OpenRTBInfo{
					MultiformatSupported: &falseValue,
				},
			},
			expected: false,
		},
		{
			description: "MultiformatSupported is nil",
			bidderInfo: config.BidderInfo{
				OpenRTB: &config.OpenRTBInfo{
					MultiformatSupported: nil,
				},
			},
			expected: true,
		},
		{
			description: "OpenRTB is nil",
			bidderInfo:  config.BidderInfo{},
			expected:    true,
		},
	}

	for _, test := range testCases {
		result := IsMultiFormatSupported(test.bidderInfo)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestPruneImps(t *testing.T) {
	testCases := []struct {
		description        string
		imps               []openrtb2.Imp
		allowedTypes       parsedSupports
		multiformatSupport bool
		preferredMediaType openrtb_ext.BidType
		expectedUpdated    bool
		expectedImps       []openrtb2.Imp
		expectedErrors     []error
	}{
		{
			description: "No imps to filter",
			imps: []openrtb2.Imp{
				{ID: "imp-1", Banner: &openrtb2.Banner{}},
			},
			allowedTypes: parsedSupports{
				banner: true,
			},
			multiformatSupport: true,
			expectedUpdated:    false,
			expectedImps: []openrtb2.Imp{
				{ID: "imp-1", Banner: &openrtb2.Banner{}},
			},
			expectedErrors: nil,
		},
		{
			description: "Filter out unsupported banner",
			imps: []openrtb2.Imp{
				{ID: "imp-1", Banner: &openrtb2.Banner{}},
			},
			allowedTypes: parsedSupports{
				banner: false,
			},
			multiformatSupport: true,
			expectedUpdated:    true,
			expectedImps:       []openrtb2.Imp{},
			expectedErrors: []error{
				&errortypes.Warning{Message: "request.imp[0] uses banner, but this bidder doesn't support it"},
				&errortypes.BadInput{Message: "request.imp[0] has no supported MediaTypes. It will be ignored"},
			},
		},
		{
			description: "Filter out unsupported video",
			imps: []openrtb2.Imp{
				{ID: "imp-1", Video: &openrtb2.Video{}},
			},
			allowedTypes: parsedSupports{
				video: false,
			},
			multiformatSupport: true,
			expectedUpdated:    true,
			expectedImps:       []openrtb2.Imp{},
			expectedErrors: []error{
				&errortypes.Warning{Message: "request.imp[0] uses video, but this bidder doesn't support it"},
				&errortypes.BadInput{Message: "request.imp[0] has no supported MediaTypes. It will be ignored"},
			},
		},
		{
			description: "Filter out unsupported audio",
			imps: []openrtb2.Imp{
				{ID: "imp-1", Audio: &openrtb2.Audio{}},
			},
			allowedTypes: parsedSupports{
				audio: false,
			},
			multiformatSupport: true,
			expectedUpdated:    true,
			expectedImps:       []openrtb2.Imp{},
			expectedErrors: []error{
				&errortypes.Warning{Message: "request.imp[0] uses audio, but this bidder doesn't support it"},
				&errortypes.BadInput{Message: "request.imp[0] has no supported MediaTypes. It will be ignored"},
			},
		},
		{
			description: "Filter out unsupported native",
			imps: []openrtb2.Imp{
				{ID: "imp-1", Native: &openrtb2.Native{}},
			},
			allowedTypes: parsedSupports{
				native: false,
			},
			multiformatSupport: true,
			expectedUpdated:    true,
			expectedImps:       []openrtb2.Imp{},
			expectedErrors: []error{
				&errortypes.Warning{Message: "request.imp[0] uses native, but this bidder doesn't support it"},
				&errortypes.BadInput{Message: "request.imp[0] has no supported MediaTypes. It will be ignored"},
			},
		},
		{
			description: "Filter out all unsupported types",
			imps: []openrtb2.Imp{
				{ID: "imp-1", Banner: &openrtb2.Banner{}, Video: &openrtb2.Video{}, Audio: &openrtb2.Audio{}, Native: &openrtb2.Native{}},
			},
			allowedTypes: parsedSupports{
				banner: false,
				video:  false,
				audio:  false,
				native: false,
			},
			multiformatSupport: true,
			expectedUpdated:    true,
			expectedImps:       []openrtb2.Imp{},
			expectedErrors: []error{
				&errortypes.Warning{Message: "request.imp[0] uses banner, but this bidder doesn't support it"},
				&errortypes.Warning{Message: "request.imp[0] uses video, but this bidder doesn't support it"},
				&errortypes.Warning{Message: "request.imp[0] uses audio, but this bidder doesn't support it"},
				&errortypes.Warning{Message: "request.imp[0] uses native, but this bidder doesn't support it"},
				&errortypes.BadInput{Message: "request.imp[0] has no supported MediaTypes. It will be ignored"},
			},
		},
		{
			description: "Filter out unsupported multiformat, preferred media type is banner",
			imps: []openrtb2.Imp{
				{ID: "imp-1", Banner: &openrtb2.Banner{}, Video: &openrtb2.Video{}},
			},
			allowedTypes: parsedSupports{
				banner: true,
				video:  true,
			},
			multiformatSupport: false,
			preferredMediaType: openrtb_ext.BidTypeBanner,
			expectedUpdated:    false,
			expectedImps:       []openrtb2.Imp{{ID: "imp-1", Banner: &openrtb2.Banner{}}},
			expectedErrors: []error{
				&errortypes.Warning{Message: "Imp imp-1 uses video, removing video as the bidder doesn't support multi-format"},
			},
		},
		{
			description: "Filter out unsupported multiformat, preferred media type is video",
			imps: []openrtb2.Imp{
				{ID: "imp-1", Banner: &openrtb2.Banner{}, Video: &openrtb2.Video{}},
			},
			allowedTypes: parsedSupports{
				banner: true,
				video:  true,
			},
			multiformatSupport: false,
			preferredMediaType: openrtb_ext.BidTypeVideo,
			expectedUpdated:    false,
			expectedImps:       []openrtb2.Imp{{ID: "imp-1", Video: &openrtb2.Video{}}},
			expectedErrors: []error{
				&errortypes.Warning{Message: "Imp imp-1 uses banner, removing banner as the bidder doesn't support multi-format"},
			},
		},
		{
			description: "Filter out unsupported multiformat, preferred media type is video and not present in any imp",
			imps: []openrtb2.Imp{
				{ID: "imp-1", Banner: &openrtb2.Banner{}, Native: &openrtb2.Native{}},
			},
			allowedTypes: parsedSupports{
				banner: true,
				video:  true,
				native: true,
			},
			multiformatSupport: false,
			preferredMediaType: openrtb_ext.BidTypeVideo,
			expectedUpdated:    true,
			expectedImps:       []openrtb2.Imp{},
			expectedErrors: []error{
				&errortypes.BadInput{Message: "Imp imp-1 does not have a preferred VIDEO media type. It will be ignored."},
			},
		},
		{
			description: "Multi-imp, Filter out unsupported multiformat, preferred media type is video and present in one imp",
			imps: []openrtb2.Imp{
				{ID: "imp-1", Banner: &openrtb2.Banner{}, Video: &openrtb2.Video{}},
				{ID: "imp-2", Banner: &openrtb2.Banner{}, Native: &openrtb2.Native{}},
			},
			allowedTypes: parsedSupports{
				banner: true,
				video:  true,
				native: true,
			},
			multiformatSupport: false,
			preferredMediaType: openrtb_ext.BidTypeVideo,
			expectedUpdated:    true,
			expectedImps:       []openrtb2.Imp{{ID: "imp-1", Video: &openrtb2.Video{}}},
			expectedErrors: []error{
				&errortypes.Warning{Message: "Imp imp-1 uses banner, removing banner as the bidder doesn't support multi-format"},
				&errortypes.BadInput{Message: "Imp imp-2 does not have a preferred VIDEO media type. It will be ignored."},
			},
		},
		{
			description: "Filter out unsupported type and unsupported multiformat, preferred media type is video",
			imps: []openrtb2.Imp{
				{ID: "imp-1", Banner: &openrtb2.Banner{}, Video: &openrtb2.Video{}, Native: &openrtb2.Native{}},
			},
			allowedTypes: parsedSupports{
				banner: true,
				video:  true,
				native: false,
			},
			multiformatSupport: false,
			preferredMediaType: openrtb_ext.BidTypeVideo,
			expectedUpdated:    false,
			expectedImps:       []openrtb2.Imp{{ID: "imp-1", Video: &openrtb2.Video{}}},
			expectedErrors: []error{
				&errortypes.Warning{Message: "request.imp[0] uses native, but this bidder doesn't support it"},
				&errortypes.Warning{Message: "Imp imp-1 uses banner, removing banner as the bidder doesn't support multi-format"},
			},
		},
	}

	for _, test := range testCases {
		updated, imps, errs := pruneImps(test.imps, test.allowedTypes, test.multiformatSupport, test.preferredMediaType)
		assert.Equal(t, test.expectedUpdated, updated, test.description+":Updated")
		assert.Equal(t, test.expectedImps, imps, test.description+":Imps")
		assert.Equal(t, test.expectedErrors, errs, test.description+":Errors")
	}
}

type mockBidder struct {
}

func (m *mockBidder) MakeRequests(request *openrtb2.BidRequest, reqInfo *ExtraRequestInfo) ([]*RequestData, []error) {
	var adapterRequests []*RequestData

	for i := 0; i < len(request.Imp); i++ {
		adapterRequests = append(adapterRequests, &RequestData{})
	}

	return adapterRequests, nil
}

func (m *mockBidder) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *RequestData, response *ResponseData) (*BidderResponse, []error) {
	return nil, []error{errors.New("mock MakeBids error")}
}
