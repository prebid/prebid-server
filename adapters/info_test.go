package adapters_test

import (
	"errors"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
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
	})
	if !assert.Len(t, errs, 1) || !assert.EqualError(t, errs[0], "this bidder does not support app requests") {
		return
	}
	if !assert.Len(t, bids, 0) {
		return
	}
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
	})
	if !assert.Len(t, errs, 1) || !assert.EqualError(t, errs[0], "this bidder does not support site requests") {
		return
	}
	if !assert.Len(t, bids, 0) {
		return
	}
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
	_, _ = constrained.MakeRequests(&openrtb.BidRequest{
		Imp:  []openrtb.Imp{},
		Site: &openrtb.Site{},
	})

}

type mockBidder struct {
	gotRequest *openrtb.BidRequest
}

func (m *mockBidder) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	m.gotRequest = request
	return nil, []error{errors.New("mock MakeRequests error")}
}

func (m *mockBidder) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	return nil, []error{errors.New("mock MakeBids error")}
}

func TestParsing(t *testing.T) {
	mockBidderName := openrtb_ext.BidderName("someBidder")
	infos := adapters.ParseBidderInfos("./adapterstest/bidder-info", []openrtb_ext.BidderName{mockBidderName})
	if infos[string(mockBidderName)].Maintainer.Email != "some-email@domain.com" {
		t.Errorf("Bad maintainer email. Got %s", infos[string(mockBidderName)].Maintainer.Email)
	}
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
