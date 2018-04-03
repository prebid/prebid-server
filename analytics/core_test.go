package analytics

import (
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/config"
	"net/http"
	"os"
	"testing"
)

const TEST_DIR string = "testFiles"

func TestSampleModule(t *testing.T) {
	var count int
	am := initAnalytics(&count)
	am.LogAuctionObject(&AuctionObject{http.StatusOK, nil, &openrtb.BidRequest{}, &openrtb.BidResponse{}})
	if count != 1 {
		t.Errorf("PBSAnalyticsModule failed at LogAuctionObejct")
	}

	am.LogSetUIDObject(&SetUIDObject{http.StatusOK, "bidders string", "uid", nil, true})
	if count != 2 {
		t.Errorf("PBSAnalyticsModule failed at LogSetUIDObejct")
	}

	am.LogCookieSyncObject(&CookieSyncObject{})
	if count != 3 {
		t.Errorf("PBSAnalyticsModule failed at LogCookieSyncObejct")
	}

	am.LogAmpObject(&AmpObject{})
	if count != 4 {
		t.Errorf("PBSAnalyticsModule failed at LogAmpObject")
	}
}

type sampleModule struct {
	count *int
}

func (m *sampleModule) LogAuctionObject(ao *AuctionObject) { *m.count++ }

func (m *sampleModule) LogCookieSyncObject(cso *CookieSyncObject) { *m.count++ }

func (m *sampleModule) LogSetUIDObject(so *SetUIDObject) { *m.count++ }

func (m *sampleModule) LogAmpObject(ao *AmpObject) { *m.count++ }

func initAnalytics(count *int) PBSAnalyticsModule {
	modules := make(enabledAnalytics, 0)
	modules = append(modules, &sampleModule{count})
	return &modules
}

func TestNewPBSAnalytics(t *testing.T) {
	if _, err := os.Stat(TEST_DIR); os.IsNotExist(err) {
		if err = os.MkdirAll(TEST_DIR, 0755); err != nil {
			t.Fatalf("Could not create test directory for FileLogger")
		}
	}
	defer os.RemoveAll(TEST_DIR)
	mod := NewPBSAnalytics(&config.Analytics{File: config.FileLogs{Filename: TEST_DIR + "/test"}})
	switch modType := mod.(type) {
	case enabledAnalytics:
		if len(enabledAnalytics(modType)) != 1 {
			t.Fatalf("Failed to add analytics module")
		}
	default:
		t.Fatalf("Failed to initialize analytics module")
	}
}
