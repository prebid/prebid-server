package helpers

import (
	"net/http"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/usersync"
)

func TestJsonifyAuctionObject(t *testing.T) {
	ao := &analytics.AuctionObject{
		Status: http.StatusOK,
	}
	if _, err := JsonifyAuctionObject(ao, "scopeId"); err != nil {
		t.Fail()
	}

}

func TestJsonifyVideoObject(t *testing.T) {
	vo := &analytics.VideoObject{
		Status: http.StatusOK,
	}
	if _, err := JsonifyVideoObject(vo, "scopeId"); err != nil {
		t.Fail()
	}
}

func TestJsonifyCookieSync(t *testing.T) {
	cso := &analytics.CookieSyncObject{
		Status:       http.StatusOK,
		BidderStatus: []*usersync.CookieSyncBidders{},
	}
	if _, err := JsonifyCookieSync(cso, "scopeId"); err != nil {
		t.Fail()
	}
}

func TestJsonifySetUIDObject(t *testing.T) {
	so := &analytics.SetUIDObject{
		Status: http.StatusOK,
		Bidder: "any-bidder",
		UID:    "uid string",
	}
	if _, err := JsonifySetUIDObject(so, "scopeId"); err != nil {
		t.Fail()
	}
}

func TestJsonifyAmpObject(t *testing.T) {
	ao := &analytics.AmpObject{
		Status:             http.StatusOK,
		Errors:             make([]error, 0),
		AuctionResponse:    &openrtb2.BidResponse{},
		AmpTargetingValues: map[string]string{},
	}
	if _, err := JsonifyAmpObject(ao, "scopeId"); err != nil {
		t.Fail()
	}
}
