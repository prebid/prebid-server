package adapters_test

import (
	"errors"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
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
	assert.IsType(t, &errortypes.BadInput{}, errs[0])
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
	assert.IsType(t, &errortypes.BadInput{}, errs[0])
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
					{ID: "imp-1", Video: &openrtb2.Video{}},
					{ID: "imp-2", Video: &openrtb2.Video{}},
				},
				Site: &openrtb2.Site{},
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
		assert.Len(t, actualAdapterRequests, test.expectedImpLen, "Test failed. Incorrect lenght of filtered imps: %s", test.description)
	}
}

type mockBidder struct {
	gotRequest *openrtb2.BidRequest
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
