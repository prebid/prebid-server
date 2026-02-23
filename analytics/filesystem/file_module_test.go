package filesystem

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/stretchr/testify/require"
)

func TestImplementation(t *testing.T) {
	require.Implements(t, (*analytics.Module)(nil), new(fileLogger))
}

func TestAmpObject_ToJson(t *testing.T) {
	ao := &analytics.AmpObject{
		Status:             http.StatusOK,
		Errors:             make([]error, 0),
		AuctionResponse:    &openrtb2.BidResponse{},
		AmpTargetingValues: map[string]string{},
	}
	b := new(bytes.Buffer)
	jsonifyAmpObject(b, ao)
	require.NotContains(t, b.String(), "Transactional Logs Error")
}

func TestAuctionObject_ToJson(t *testing.T) {
	ao := &analytics.AuctionObject{
		Status: http.StatusOK,
	}
	b := new(bytes.Buffer)
	jsonifyAuctionObject(b, ao)
	require.NotContains(t, b.String(), "Transactional Logs Error")
}

func TestVideoObject_ToJson(t *testing.T) {
	vo := &analytics.VideoObject{
		Status: http.StatusOK,
	}
	b := new(bytes.Buffer)
	jsonifyVideoObject(b, vo)
	require.NotContains(t, b.String(), "Transactional Logs Error")
}

func TestSetUIDObject_ToJson(t *testing.T) {
	so := &analytics.SetUIDObject{
		Status: http.StatusOK,
		Bidder: "any-bidder",
		UID:    "uid string",
	}
	b := new(bytes.Buffer)
	jsonifySetUIDObject(b, so)
	require.NotContains(t, b.String(), "Transactional Logs Error")
}

func TestCookieSyncObject_ToJson(t *testing.T) {
	cso := &analytics.CookieSyncObject{
		Status:       http.StatusOK,
		BidderStatus: []*analytics.CookieSyncBidder{},
	}
	b := new(bytes.Buffer)
	jsonifyCookieSync(b, cso)
	require.NotContains(t, b.String(), "Transactional Logs Error")
}

func TestLogNotificationEventObject_ToJson(t *testing.T) {
	neo := &analytics.NotificationEvent{
		Request: &analytics.EventRequest{
			Bidder: "bidder",
		},
		Account: &config.Account{
			ID: "id",
		},
	}
	b := new(bytes.Buffer)
	jsonifyNotificationEventObject(b, neo)
	require.NotContains(t, b.String(), "Transactional Logs Error")
}

func TestFileLogger_LogObjects(t *testing.T) {
	if fl, err := NewFileLogger(t.TempDir() + "/test"); err == nil {
		fl.LogAuctionObject(&analytics.AuctionObject{})
		fl.LogVideoObject(&analytics.VideoObject{})
		fl.LogAmpObject(&analytics.AmpObject{})
		fl.LogSetUIDObject(&analytics.SetUIDObject{})
		fl.LogCookieSyncObject(&analytics.CookieSyncObject{})
		fl.LogNotificationEventObject(&analytics.NotificationEvent{})
		fl.Shutdown()
	} else {
		t.Fatalf("Couldn't initialize file logger: %v", err)
	}
}
