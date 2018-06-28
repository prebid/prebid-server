package adapters_test

import (
	"errors"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
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
	if len(errs) != 1 || errs[0].Error() != "this bidder does not support app requests" {
		t.Errorf("Unexpected error: %s", errs[0].Error())
	}
	if len(bids) != 0 {
		t.Errorf("Got %d unexpected bids", len(bids))
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
	if len(errs) != 1 || errs[0].Error() != "this bidder does not support site requests" {
		t.Errorf("Unexpected error: %s", errs[0].Error())
	}
	if len(bids) != 0 {
		t.Errorf("Got %d unexpected bids", len(bids))
	}
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
	assertBoolsEqual(t, true, infos.HasAppSupport(mockBidderName))
	assertBoolsEqual(t, true, infos.HasSiteSupport(mockBidderName))

	assertBoolsEqual(t, true, infos.SupportsAppMediaType(mockBidderName, openrtb_ext.BidTypeBanner))
	assertBoolsEqual(t, false, infos.SupportsAppMediaType(mockBidderName, openrtb_ext.BidTypeVideo))
	assertBoolsEqual(t, false, infos.SupportsAppMediaType(mockBidderName, openrtb_ext.BidTypeAudio))
	assertBoolsEqual(t, true, infos.SupportsAppMediaType(mockBidderName, openrtb_ext.BidTypeNative))

	assertBoolsEqual(t, true, infos.SupportsWebMediaType(mockBidderName, openrtb_ext.BidTypeBanner))
	assertBoolsEqual(t, true, infos.SupportsWebMediaType(mockBidderName, openrtb_ext.BidTypeVideo))
	assertBoolsEqual(t, false, infos.SupportsWebMediaType(mockBidderName, openrtb_ext.BidTypeAudio))
	assertBoolsEqual(t, true, infos.SupportsWebMediaType(mockBidderName, openrtb_ext.BidTypeNative))
}

func assertBoolsEqual(t *testing.T, expected bool, actual bool) {
	t.Helper()
	if actual != expected {
		t.Errorf("expected %t, got %t", expected, actual)
	}
}
