package helpers

import (
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/stretchr/testify/assert"
)

func TestJsonifyAuctionObject(t *testing.T) {
	ao := &analytics.AuctionObject{
		Status: http.StatusOK,
	}

	_, err := JsonifyAuctionObject(ao, "scopeId")
	assert.NoError(t, err)
}

func TestJsonifyVideoObject(t *testing.T) {
	vo := &analytics.VideoObject{
		Status: http.StatusOK,
	}

	_, err := JsonifyVideoObject(vo, "scopeId")
	assert.NoError(t, err)
}

func TestJsonifyCookieSync(t *testing.T) {
	cso := &analytics.CookieSyncObject{
		Status:       http.StatusOK,
		BidderStatus: []*analytics.CookieSyncBidder{},
	}

	_, err := JsonifyCookieSync(cso, "scopeId")
	assert.NoError(t, err)
}

func TestJsonifySetUIDObject(t *testing.T) {
	so := &analytics.SetUIDObject{
		Status: http.StatusOK,
		Bidder: "any-bidder",
		UID:    "uid string",
	}

	_, err := JsonifySetUIDObject(so, "scopeId")
	assert.NoError(t, err)
}

func TestJsonifyAmpObject(t *testing.T) {
	ao := &analytics.AmpObject{
		Status:             http.StatusOK,
		Errors:             make([]error, 0),
		AuctionResponse:    &openrtb2.BidResponse{},
		AmpTargetingValues: map[string]string{},
	}

	_, err := JsonifyAmpObject(ao, "scopeId")
	assert.NoError(t, err)
}
