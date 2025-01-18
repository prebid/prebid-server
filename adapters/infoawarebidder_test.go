package adapters_test

import (
	"errors"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
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
	constrained := adapters.BuildInfoAwareBidder(bidder, info)
	bids, errs := constrained.MakeRequests(&openrtb2.BidRequest{
		Imp: []openrtb2.Imp{{ID: "imp-1", Banner: &openrtb2.Banner{}}},
		App: &openrtb2.App{},
	}, &adapters.ExtraRequestInfo{})
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
	constrained := adapters.BuildInfoAwareBidder(bidder, info)
	bids, errs := constrained.MakeRequests(&openrtb2.BidRequest{
		Imp:  []openrtb2.Imp{{ID: "imp-1", Banner: &openrtb2.Banner{}}},
		Site: &openrtb2.Site{},
	}, &adapters.ExtraRequestInfo{})
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
	constrained := adapters.BuildInfoAwareBidder(bidder, info)
	bids, errs := constrained.MakeRequests(&openrtb2.BidRequest{
		Imp:  []openrtb2.Imp{{ID: "imp-1", Banner: &openrtb2.Banner{}}},
		DOOH: &openrtb2.DOOH{},
	}, &adapters.ExtraRequestInfo{})
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

	constrained := adapters.BuildInfoAwareBidder(bidder, info)

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
				&errortypes.BadInput{Message: "request.imp[1] uses native, but this bidder doesn't support it"},
				&errortypes.BadInput{Message: "request.imp[2] uses audio, but this bidder doesn't support it"},
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
				&errortypes.BadInput{Message: "request.imp[2] uses native, but this bidder doesn't support it"},
				&errortypes.BadInput{Message: "request.imp[3] uses banner, but this bidder doesn't support it"},
				&errortypes.BadInput{Message: "request.imp[1] has no supported MediaTypes. It will be ignored"},
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
		actualAdapterRequests, actualErrs := constrained.MakeRequests(test.inBidRequest, &adapters.ExtraRequestInfo{})

		// Assert the request.Imp slice was correctly filtered and if MakeRequest() was called by asserting
		// the corresponding error messages were returned
		for i, expectedErr := range test.expectedErrors {
			assert.EqualError(t, expectedErr, actualErrs[i].Error(), "Test failed. Error[%d] in error list mismatch: %s", i, test.description)
		}

		// Extra MakeRequests() call check: our mockBidder returns an adapter request for every imp
		assert.Len(t, actualAdapterRequests, test.expectedImpLen, "Test failed. Incorrect length of filtered imps: %s", test.description)
	}
}

func TestFilterMultiformatImps(t *testing.T) {

	testCases := []struct {
		description        string
		inBidRequest       *openrtb2.BidRequest
		preferredMediaType openrtb_ext.BidType
		expectedErrors     []error
		expectedImps       []openrtb2.Imp
	}{

		{
			description: "Impression with multi-format not present",
			inBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-1", Banner: &openrtb2.Banner{}},
				},
			},
			preferredMediaType: openrtb_ext.BidTypeBanner,
			expectedErrors:     nil,
			expectedImps: []openrtb2.Imp{
				{ID: "imp-1", Banner: &openrtb2.Banner{}},
			},
		},

		{
			description: "Multiformat impression with preferred media type present",
			inBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-1", Banner: &openrtb2.Banner{}, Video: &openrtb2.Video{}},
					{ID: "imp-2", Banner: &openrtb2.Banner{}},
				},
			},
			preferredMediaType: openrtb_ext.BidTypeBanner,
			expectedErrors:     nil,
			expectedImps: []openrtb2.Imp{
				{ID: "imp-1", Banner: &openrtb2.Banner{}},
				{ID: "imp-2", Banner: &openrtb2.Banner{}},
			},
		},
		{
			description: "Multiformat impression with preferred media type not present",
			inBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-1", Banner: &openrtb2.Banner{}, Native: &openrtb2.Native{}},
					{ID: "imp-2", Banner: &openrtb2.Banner{}, Audio: &openrtb2.Audio{}},
				},
			},
			preferredMediaType: openrtb_ext.BidTypeVideo,
			expectedErrors: []error{
				&errortypes.BadInput{Message: "Imp imp-1 does not have a valid VIDEO media type."},
				&errortypes.BadInput{Message: "Imp imp-2 does not have a valid VIDEO media type."},
				&errortypes.BadInput{Message: "Bid request contains 0 impressions after filtering."},
			},
			expectedImps: nil,
		},
		{
			description: "Multiformat impressions with preferred media type present in imp-1 and not in imp-2",
			inBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-1", Banner: &openrtb2.Banner{}, Video: &openrtb2.Video{}},
					{ID: "imp-2", Video: &openrtb2.Video{}, Native: &openrtb2.Native{}},
				},
			},
			preferredMediaType: openrtb_ext.BidTypeBanner,
			expectedErrors: []error{
				&errortypes.BadInput{Message: "Imp imp-2 does not have a valid BANNER media type."},
			},
			expectedImps: []openrtb2.Imp{
				{ID: "imp-1", Banner: &openrtb2.Banner{}},
			},
		},
		{
			description: "Impression with no adformat present",
			inBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-1"},
				},
			},
			preferredMediaType: openrtb_ext.BidTypeBanner,
			expectedErrors:     nil,
			expectedImps: []openrtb2.Imp{
				{ID: "imp-1"},
			},
		},
		{
			description: "Multiformat impression with preferred media type not present in the request or account config",
			inBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-1", Banner: &openrtb2.Banner{}, Video: &openrtb2.Video{}, Audio: &openrtb2.Audio{}, Native: &openrtb2.Native{}},
				},
			},
			preferredMediaType: "",
			expectedErrors:     nil,
			expectedImps: []openrtb2.Imp{
				{ID: "imp-1", Banner: &openrtb2.Banner{}, Video: &openrtb2.Video{}, Audio: &openrtb2.Audio{}, Native: &openrtb2.Native{}},
			},
		},
	}

	for _, test := range testCases {
		actualImps, actualErrs := adapters.FilterMultiformatImps(test.inBidRequest, test.preferredMediaType)
		assert.Equal(t, test.expectedErrors, actualErrs, test.description+":Errors")
		assert.Equal(t, test.expectedImps, actualImps, test.description+":Imps")
	}
}

func TestAdjustImpForPreferredMediaType(t *testing.T) {
	testCases := []struct {
		description        string
		inImp              openrtb2.Imp
		preferredMediaType openrtb_ext.BidType
		expectedImp        *openrtb2.Imp
		expectedError      error
	}{
		{
			description: "Non-multiformat impression, return as-is",
			inImp: openrtb2.Imp{
				ID:     "imp-1",
				Banner: &openrtb2.Banner{},
			},
			preferredMediaType: openrtb_ext.BidTypeBanner,
			expectedImp: &openrtb2.Imp{
				ID:     "imp-1",
				Banner: &openrtb2.Banner{},
			},
			expectedError: nil,
		},
		{
			description: "Multiformat impression with preferred media type Banner",
			inImp: openrtb2.Imp{
				ID:     "imp-1",
				Banner: &openrtb2.Banner{},
				Video:  &openrtb2.Video{},
			},
			preferredMediaType: openrtb_ext.BidTypeBanner,
			expectedImp: &openrtb2.Imp{
				ID:     "imp-1",
				Banner: &openrtb2.Banner{},
			},
			expectedError: nil,
		},
		{
			description: "Multiformat impression with preferred media type Video",
			inImp: openrtb2.Imp{
				ID:     "imp-1",
				Banner: &openrtb2.Banner{},
				Video:  &openrtb2.Video{},
			},
			preferredMediaType: openrtb_ext.BidTypeVideo,
			expectedImp: &openrtb2.Imp{
				ID:    "imp-1",
				Video: &openrtb2.Video{},
			},
			expectedError: nil,
		},
		{
			description: "Multiformat impression with preferred media type Audio",
			inImp: openrtb2.Imp{
				ID:    "imp-1",
				Audio: &openrtb2.Audio{},
				Video: &openrtb2.Video{},
			},
			preferredMediaType: openrtb_ext.BidTypeAudio,
			expectedImp: &openrtb2.Imp{
				ID:    "imp-1",
				Audio: &openrtb2.Audio{},
			},
			expectedError: nil,
		},
		{
			description: "Multiformat impression with preferred media type Native",
			inImp: openrtb2.Imp{
				ID:     "imp-1",
				Native: &openrtb2.Native{},
				Video:  &openrtb2.Video{},
			},
			preferredMediaType: openrtb_ext.BidTypeNative,
			expectedImp: &openrtb2.Imp{
				ID:     "imp-1",
				Native: &openrtb2.Native{},
			},
			expectedError: nil,
		},
		{
			description: "Multiformat impression with no preferred media type",
			inImp: openrtb2.Imp{
				ID:     "imp-1",
				Banner: &openrtb2.Banner{},
				Video:  &openrtb2.Video{},
			},
			preferredMediaType: "",
			expectedImp: &openrtb2.Imp{
				ID:     "imp-1",
				Banner: &openrtb2.Banner{},
				Video:  &openrtb2.Video{},
			},
			expectedError: nil,
		},
		{
			description: "Multiformat impression with invalid preferred media type",
			inImp: openrtb2.Imp{
				ID:     "imp-1",
				Banner: &openrtb2.Banner{},
				Video:  &openrtb2.Video{},
			},
			preferredMediaType: openrtb_ext.BidTypeAudio,
			expectedImp:        nil,
			expectedError:      &errortypes.BadInput{Message: "Imp imp-1 does not have a valid AUDIO media type."},
		},
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
			expectedImp: &openrtb2.Imp{
				ID:     "imp-1",
				Banner: &openrtb2.Banner{},
			},
			expectedError: nil,
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
			expectedImp: &openrtb2.Imp{
				ID:    "imp-1",
				Video: &openrtb2.Video{},
			},
			expectedError: nil,
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
			expectedImp: &openrtb2.Imp{
				ID:    "imp-1",
				Audio: &openrtb2.Audio{},
			},
			expectedError: nil,
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
			expectedImp: &openrtb2.Imp{
				ID:     "imp-1",
				Native: &openrtb2.Native{},
			},
			expectedError: nil,
		},
	}

	for _, test := range testCases {
		actualImp, actualErr := adapters.AdjustImpForPreferredMediaType(test.inImp, test.preferredMediaType)
		assert.Equal(t, test.expectedImp, actualImp, test.description)
		assert.Equal(t, test.expectedError, actualErr, test.description)
	}
}

type mockBidder struct {
}

func (m *mockBidder) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var adapterRequests []*adapters.RequestData

	for i := 0; i < len(request.Imp); i++ {
		adapterRequests = append(adapterRequests, &adapters.RequestData{})
	}

	return adapterRequests, nil
}

func (m *mockBidder) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	return nil, []error{errors.New("mock MakeBids error")}
}
