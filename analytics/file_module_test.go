package analytics

import (
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/usersync"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestAmpObject_ToJson(t *testing.T) {
	ao := AmpObject{
		Status:             http.StatusOK,
		Errors:             make([]error, 0),
		AuctionResponse:    &openrtb.BidResponse{},
		AmpTargetingValues: map[string]string{},
	}
	if aoJson := ao.ToJson(); strings.Contains(aoJson, "Transactional Logs Error") {
		t.Fatalf("AmpObject failed to convert to json")
	}
}

func TestAuctionObject_ToJson(t *testing.T) {
	ao := AuctionObject{
		Status: http.StatusOK,
	}
	if aoJson := ao.ToJson(); strings.Contains(aoJson, "Transactional Logs Error") {
		t.Fatalf("AuctionObject failed to convert to json")
	}
}

func TestSetUIDObject_ToJson(t *testing.T) {
	so := SetUIDObject{
		Status: http.StatusOK,
		Bidder: "any-bidder",
		UID:    "uid string",
	}
	if soJson := so.ToJson(); strings.Contains(soJson, "Transactional Logs Error") {
		t.Fatalf("SetUIDObject failed to convert to json")
	}
}

func TestCookieSyncObject_ToJson(t *testing.T) {
	cso := CookieSyncObject{
		Status:       http.StatusOK,
		BidderStatus: []*usersync.CookieSyncBidders{},
	}
	if csoJson := cso.ToJson(); strings.Contains(csoJson, "Transactional Logs Error") {
		t.Fatalf("CookieSyncObject failed to convert to json")
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
		fl.LogAuctionObject(&AuctionObject{})
		fl.LogAmpObject(&AmpObject{})
		fl.LogSetUIDObject(&SetUIDObject{})
		fl.LogCookieSyncObject(&CookieSyncObject{})
	} else {
		t.Fatalf("Couldn't initialize file logger: %v", err)
	}
}
