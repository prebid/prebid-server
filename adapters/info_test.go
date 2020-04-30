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
	_, errs := constrained.MakeRequests(&openrtb.BidRequest{
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
	}, &adapters.ExtraRequestInfo{})
	if !assert.Len(t, errs, 6) {
		return
	}
	assert.EqualError(t, errs[0], "request.imp[1] uses native, but this bidder doesn't support it")
	assert.EqualError(t, errs[1], "request.imp[2] uses native, but this bidder doesn't support it")
	assert.EqualError(t, errs[2], "request.imp[3] uses banner, but this bidder doesn't support it")
	assert.EqualError(t, errs[3], "request.imp[1] has no supported MediaTypes. It will be ignored")
	assert.EqualError(t, errs[4], "request.imp[3] has no supported MediaTypes. It will be ignored")
	assert.EqualError(t, errs[5], "mock MakeRequests error")
	assert.IsType(t, &errortypes.BadInput{}, errs[0])
	assert.IsType(t, &errortypes.BadInput{}, errs[1])
	assert.IsType(t, &errortypes.BadInput{}, errs[2])
	assert.IsType(t, &errortypes.BadInput{}, errs[3])
	assert.IsType(t, &errortypes.BadInput{}, errs[4])

	req := bidder.gotRequest
	if !assert.Len(t, req.Imp, 2) {
		return
	}
	assert.Equal(t, "imp-1", req.Imp[0].ID)
	assert.Equal(t, "imp-2", req.Imp[1].ID)
	assert.Nil(t, req.Imp[1].Native)
}

type mockBidder struct {
	gotRequest *openrtb.BidRequest
}

func (m *mockBidder) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	m.gotRequest = request
	return nil, []error{errors.New("mock MakeRequests error")}
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
