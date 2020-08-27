package adapters_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestAppNotSupported(t *testing.T) {
	bidder := &mockBidder{}
	info := adapters.BidderInfo{
		Capabilities: &adapters.CapabilitiesInfo{
			Site: &adapters.PlatformInfo{
				MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner},
			},
		},
	}
	constrained := adapters.EnforceBidderInfo(bidder, info)
	bids, errs := constrained.MakeRequests(&openrtb.BidRequest{
		Imp: []openrtb.Imp{{ID: "imp-1", Banner: &openrtb.Banner{}}},
		App: &openrtb.App{},
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
	info := adapters.BidderInfo{
		Capabilities: &adapters.CapabilitiesInfo{
			App: &adapters.PlatformInfo{
				MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner},
			},
		},
	}
	constrained := adapters.EnforceBidderInfo(bidder, info)
	bids, errs := constrained.MakeRequests(&openrtb.BidRequest{
		Imp:  []openrtb.Imp{{ID: "imp-1", Banner: &openrtb.Banner{}}},
		Site: &openrtb.Site{},
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
	info := adapters.BidderInfo{
		Capabilities: &adapters.CapabilitiesInfo{
			Site: &adapters.PlatformInfo{
				MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeVideo},
			},
			App: &adapters.PlatformInfo{
				MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner},
			},
		},
	}

	constrained := adapters.EnforceBidderInfo(bidder, info)

	testCases := []struct {
		description    string
		inBidRequest   *openrtb.BidRequest
		expectedErrors []error
		expectedImpLen int
	}{
		{
			description: "Empty Imp array. MakeRequest() call not expected",
			inBidRequest: &openrtb.BidRequest{
				Imp:  []openrtb.Imp{},
				Site: &openrtb.Site{},
			},
			expectedErrors: []error{
				&errortypes.BadInput{Message: "Bid request didn't contain media types supported by the bidder"},
			},
			expectedImpLen: 0,
		},
		{
			description: "Sole imp in bid request is of wrong media type. MakeRequest() call not expected",
			inBidRequest: &openrtb.BidRequest{
				Imp: []openrtb.Imp{{ID: "imp-1", Video: &openrtb.Video{}}},
				App: &openrtb.App{},
			},
			expectedErrors: []error{
				&errortypes.BadInput{Message: "request.imp[0] uses video, but this bidder doesn't support it"},
				&errortypes.BadInput{Message: "Bid request didn't contain media types supported by the bidder"},
			},
			expectedImpLen: 0,
		},
		{
			description: "All imps in bid request of wrong media type, MakeRequest() call not expected",
			inBidRequest: &openrtb.BidRequest{
				Imp: []openrtb.Imp{
					{ID: "imp-1", Video: &openrtb.Video{}},
					{ID: "imp-2", Native: &openrtb.Native{}},
					{ID: "imp-3", Audio: &openrtb.Audio{}},
				},
				App: &openrtb.App{},
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
			inBidRequest: &openrtb.BidRequest{
				Imp: []openrtb.Imp{
					{
						ID:    "imp-1",
						Video: &openrtb.Video{},
					},
					{
						Native: &openrtb.Native{},
					},
					{
						ID:     "imp-2",
						Video:  &openrtb.Video{},
						Native: &openrtb.Native{},
					},
					{
						Banner: &openrtb.Banner{},
					},
				},
				Site: &openrtb.Site{},
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
			inBidRequest: &openrtb.BidRequest{
				Imp: []openrtb.Imp{
					{ID: "imp-1", Video: &openrtb.Video{}},
					{ID: "imp-2", Video: &openrtb.Video{}},
				},
				Site: &openrtb.Site{},
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
	gotRequest *openrtb.BidRequest
}

func (m *mockBidder) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var adapterRequests []*adapters.RequestData

	for i := 0; i < len(request.Imp); i++ {
		adapterRequests = append(adapterRequests, &adapters.RequestData{})
	}

	return adapterRequests, nil
}

func (m *mockBidder) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	return nil, []error{errors.New("mock MakeBids error")}
}

func blankAdapterConfig(bidderName openrtb_ext.BidderName) map[string]config.Adapter {
	adapters := make(map[string]config.Adapter)
	adapters[strings.ToLower(string(bidderName))] = config.Adapter{}

	return adapters
}

func TestParsing(t *testing.T) {
	mockBidderName := openrtb_ext.BidderName("someBidder")
	infos := adapters.ParseBidderInfos(blankAdapterConfig(mockBidderName), "./adapterstest/bidder-info", []openrtb_ext.BidderName{mockBidderName})
	if infos[string(mockBidderName)].Maintainer.Email != "some-email@domain.com" {
		t.Errorf("Bad maintainer email. Got %s", infos[string(mockBidderName)].Maintainer.Email)
	}

	assert.Equal(t, true, infos.IsActive(mockBidderName))

	assert.Equal(t, true, infos.HasAppSupport(mockBidderName))
	assert.Equal(t, true, infos.HasSiteSupport(mockBidderName))

	assert.Equal(t, true, infos.SupportsAppMediaType(mockBidderName, openrtb_ext.BidTypeBanner))
	assert.Equal(t, false, infos.SupportsAppMediaType(mockBidderName, openrtb_ext.BidTypeVideo))
	assert.Equal(t, false, infos.SupportsAppMediaType(mockBidderName, openrtb_ext.BidTypeAudio))
	assert.Equal(t, true, infos.SupportsAppMediaType(mockBidderName, openrtb_ext.BidTypeNative))

	assert.Equal(t, true, infos.SupportsWebMediaType(mockBidderName, openrtb_ext.BidTypeBanner))
	assert.Equal(t, true, infos.SupportsWebMediaType(mockBidderName, openrtb_ext.BidTypeVideo))
	assert.Equal(t, false, infos.SupportsWebMediaType(mockBidderName, openrtb_ext.BidTypeAudio))
	assert.Equal(t, true, infos.SupportsWebMediaType(mockBidderName, openrtb_ext.BidTypeNative))
}
