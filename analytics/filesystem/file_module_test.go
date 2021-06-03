package filesystem

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/config"

	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/usersync"
)

const TEST_DIR string = "testFiles"

func TestAmpObject_ToJson(t *testing.T) {
	ao := &analytics.AmpObject{
		Status:             http.StatusOK,
		Errors:             make([]error, 0),
		AuctionResponse:    &openrtb2.BidResponse{},
		AmpTargetingValues: map[string]string{},
	}
	if aoJson := jsonifyAmpObject(ao); strings.Contains(aoJson, "Transactional Logs Error") {
		t.Fatalf("AmpObject failed to convert to json")
	}
}

func TestAuctionObject_ToJson(t *testing.T) {
	ao := &analytics.AuctionObject{
		Status: http.StatusOK,
	}
	if aoJson := jsonifyAuctionObject(ao); strings.Contains(aoJson, "Transactional Logs Error") {
		t.Fatalf("AuctionObject failed to convert to json")
	}
}

func TestVideoObject_ToJson(t *testing.T) {
	vo := &analytics.VideoObject{
		Status: http.StatusOK,
	}
	if voJson := jsonifyVideoObject(vo); strings.Contains(voJson, "Transactional Logs Error") {
		t.Fatalf("AuctionObject failed to convert to json")
	}
}

func TestSetUIDObject_ToJson(t *testing.T) {
	so := &analytics.SetUIDObject{
		Status: http.StatusOK,
		Bidder: "any-bidder",
		UID:    "uid string",
	}
	if soJson := jsonifySetUIDObject(so); strings.Contains(soJson, "Transactional Logs Error") {
		t.Fatalf("SetUIDObject failed to convert to json")
	}
}

func TestCookieSyncObject_ToJson(t *testing.T) {
	cso := &analytics.CookieSyncObject{
		Status:       http.StatusOK,
		BidderStatus: []*usersync.CookieSyncBidders{},
	}
	if csoJson := jsonifyCookieSync(cso); strings.Contains(csoJson, "Transactional Logs Error") {
		t.Fatalf("CookieSyncObject failed to convert to json")
	}
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
	if neoJson := jsonifyNotificationEventObject(neo); strings.Contains(neoJson, "Transactional Logs Error") {
		t.Fatalf("NotificationEventObject failed to convert to json")
	}
}

func TestFileLogger_LogObjects(t *testing.T) {
	if _, err := os.Stat(TEST_DIR); os.IsNotExist(err) {
		if err = os.MkdirAll(TEST_DIR, 0755); err != nil {
			t.Fatalf("Could not create test directory for FileLogger")
		}
	}
	defer os.RemoveAll(TEST_DIR)
	if fl, err := NewFileLogger(TEST_DIR + "//test"); err == nil {
		fl.LogAuctionObject(&analytics.AuctionObject{})
		fl.LogVideoObject(&analytics.VideoObject{})
		fl.LogAmpObject(&analytics.AmpObject{})
		fl.LogSetUIDObject(&analytics.SetUIDObject{})
		fl.LogCookieSyncObject(&analytics.CookieSyncObject{})
		fl.LogNotificationEventObject(&analytics.NotificationEvent{})
	} else {
		t.Fatalf("Couldn't initialize file logger: %v", err)
	}
}
