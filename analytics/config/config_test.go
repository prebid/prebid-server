package config

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"os"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
)

const TEST_DIR string = "testFiles"

func TestSampleModule(t *testing.T) {
	var count int
	am := initAnalytics(&count)
	am.LogAuctionObject(&analytics.AuctionObject{
		Status:   http.StatusOK,
		Errors:   nil,
		Request:  &openrtb.BidRequest{},
		Response: &openrtb.BidResponse{},
	})
	if count != 1 {
		t.Errorf("PBSAnalyticsModule failed at LogAuctionObject")
	}

	am.LogSetUIDObject(&analytics.SetUIDObject{
		Status:  http.StatusOK,
		Bidder:  "bidders string",
		UID:     "uid",
		Errors:  nil,
		Success: true,
	})
	if count != 2 {
		t.Errorf("PBSAnalyticsModule failed at LogSetUIDObject")
	}

	am.LogCookieSyncObject(&analytics.CookieSyncObject{})
	if count != 3 {
		t.Errorf("PBSAnalyticsModule failed at LogCookieSyncObject")
	}

	am.LogAmpObject(&analytics.AmpObject{})
	if count != 4 {
		t.Errorf("PBSAnalyticsModule failed at LogAmpObject")
	}

	am.LogVideoObject(&analytics.VideoObject{})
	if count != 5 {
		t.Errorf("PBSAnalyticsModule failed at LogVideoObject")
	}
}

type sampleModule struct {
	count *int
}

func (m *sampleModule) LogAuctionObject(ao *analytics.AuctionObject) { *m.count++ }

func (m *sampleModule) LogVideoObject(vo *analytics.VideoObject) { *m.count++ }

func (m *sampleModule) LogCookieSyncObject(cso *analytics.CookieSyncObject) { *m.count++ }

func (m *sampleModule) LogSetUIDObject(so *analytics.SetUIDObject) { *m.count++ }

func (m *sampleModule) LogAmpObject(ao *analytics.AmpObject) { *m.count++ }

func initAnalytics(count *int) analytics.PBSAnalyticsModule {
	instance := &pbsAnalyticsModule{
		enabledModules: make([]analyticsModule, 0),
	}
	instance.enabledModules = append(instance.enabledModules, &sampleModule{count})
	return instance
}

func TestNewPBSAnalytics(t *testing.T) {
	pbsAnalytics := NewPBSAnalytics(&config.Analytics{})
	instance := pbsAnalytics.(*pbsAnalyticsModule)

	assert.Equal(t, len(instance.enabledModules), 0)
}

func TestNewPBSAnalytics_FileLogger(t *testing.T) {
	if _, err := os.Stat(TEST_DIR); os.IsNotExist(err) {
		if err = os.MkdirAll(TEST_DIR, 0755); err != nil {
			t.Fatalf("Could not create test directory for FileLogger")
		}
	}
	defer os.RemoveAll(TEST_DIR)

	pbsAnalytics := NewPBSAnalytics(&config.Analytics{File: config.FileLogs{Filename: TEST_DIR + "/test"}})
	instance := pbsAnalytics.(*pbsAnalyticsModule)

	assert.Equal(t, len(instance.enabledModules), 1)
	assert.NotNil(t, instance.enabledModules[0].(analyticsModule))
}

func TestNewPBSAnalytics_Pubstack(t *testing.T) {

	pbsAnalyticsWithoutError := NewPBSAnalytics(&config.Analytics{
		Pubstack: config.Pubstack{
			Enabled:   true,
			ScopeId:   "scopeId",
			IntakeUrl: "https://pubstack.io/intake",
			Buffers: config.PubstackBuffer{
				BufferSize: "100KB",
				EventCount: 0,
				Timeout:    "30s",
			},
			ConfRefresh: "2h",
		},
	})
	instanceWithoutError := pbsAnalyticsWithoutError.(*pbsAnalyticsModule)

	assert.Equal(t, len(instanceWithoutError.enabledModules), 1)
	assert.NotNil(t, instanceWithoutError.enabledModules[0].(analyticsModule))

	pbsAnalyticsWithError := NewPBSAnalytics(&config.Analytics{
		Pubstack: config.Pubstack{
			Enabled: true,
		},
	})
	instanceWithError := pbsAnalyticsWithError.(*pbsAnalyticsModule)
	assert.Equal(t, len(instanceWithError.enabledModules), 0)
}
