package adapters_test

import (
	"testing"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
)

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
