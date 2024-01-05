package build

import (
	"net/http"
	"os"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/analytics"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/privacy"
	"github.com/prebid/prebid-server/v2/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

const TEST_DIR string = "testFiles"

func TestSampleModule(t *testing.T) {
	var count int
	am := initAnalytics(&count)
	am.LogAuctionObject(&analytics.AuctionObject{
		Status:   http.StatusOK,
		Errors:   nil,
		Response: &openrtb2.BidResponse{},
	}, privacy.ActivityControl{})
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

	am.LogAmpObject(&analytics.AmpObject{}, privacy.ActivityControl{})
	if count != 4 {
		t.Errorf("PBSAnalyticsModule failed at LogAmpObject")
	}

	am.LogVideoObject(&analytics.VideoObject{}, privacy.ActivityControl{})
	if count != 5 {
		t.Errorf("PBSAnalyticsModule failed at LogVideoObject")
	}

	am.LogNotificationEventObject(&analytics.NotificationEvent{}, privacy.ActivityControl{})
	if count != 6 {
		t.Errorf("PBSAnalyticsModule failed at LogNotificationEventObject")
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

func (m *sampleModule) LogNotificationEventObject(ne *analytics.NotificationEvent) { *m.count++ }

func initAnalytics(count *int) analytics.Runner {
	modules := make(enabledAnalytics, 0)
	modules["sampleModule"] = &sampleModule{count}
	return &modules
}

func TestNewPBSAnalytics(t *testing.T) {
	pbsAnalytics := New(&config.Analytics{})
	instance := pbsAnalytics.(enabledAnalytics)

	assert.Equal(t, len(instance), 0)
}

func TestNewPBSAnalytics_FileLogger(t *testing.T) {
	if _, err := os.Stat(TEST_DIR); os.IsNotExist(err) {
		if err = os.MkdirAll(TEST_DIR, 0755); err != nil {
			t.Fatalf("Could not create test directory for FileLogger")
		}
	}
	defer os.RemoveAll(TEST_DIR)
	mod := New(&config.Analytics{File: config.FileLogs{Filename: TEST_DIR + "/test"}})
	switch modType := mod.(type) {
	case enabledAnalytics:
		if len(enabledAnalytics(modType)) != 1 {
			t.Fatalf("Failed to add analytics module")
		}
	default:
		t.Fatalf("Failed to initialize analytics module")
	}

	pbsAnalytics := New(&config.Analytics{File: config.FileLogs{Filename: TEST_DIR + "/test"}})
	instance := pbsAnalytics.(enabledAnalytics)

	assert.Equal(t, len(instance), 1)
}

func TestNewPBSAnalytics_Pubstack(t *testing.T) {

	pbsAnalyticsWithoutError := New(&config.Analytics{
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
	instanceWithoutError := pbsAnalyticsWithoutError.(enabledAnalytics)

	assert.Equal(t, len(instanceWithoutError), 1)

	pbsAnalyticsWithError := New(&config.Analytics{
		Pubstack: config.Pubstack{
			Enabled: true,
		},
	})
	instanceWithError := pbsAnalyticsWithError.(enabledAnalytics)
	assert.Equal(t, len(instanceWithError), 0)
}

func TestSampleModuleActivitiesAllowed(t *testing.T) {
	var count int
	am := initAnalytics(&count)

	acAllowed := privacy.NewActivityControl(getDefaultActivityConfig("sampleModule", true))

	ao := &analytics.AuctionObject{
		Status:   http.StatusOK,
		Errors:   nil,
		Response: &openrtb2.BidResponse{},
	}

	am.LogAuctionObject(ao, acAllowed)
	if count != 1 {
		t.Errorf("PBSAnalyticsModule failed at LogAuctionObject")
	}

	am.LogAmpObject(&analytics.AmpObject{}, acAllowed)
	if count != 2 {
		t.Errorf("PBSAnalyticsModule failed at LogAmpObject")
	}

	am.LogVideoObject(&analytics.VideoObject{}, acAllowed)
	if count != 3 {
		t.Errorf("PBSAnalyticsModule failed at LogVideoObject")
	}

	am.LogNotificationEventObject(&analytics.NotificationEvent{}, acAllowed)
	if count != 4 {
		t.Errorf("PBSAnalyticsModule failed at LogNotificationEventObject")
	}
}

func TestSampleModuleActivitiesDenied(t *testing.T) {
	var count int
	am := initAnalytics(&count)

	acDenied := privacy.NewActivityControl(getDefaultActivityConfig("sampleModule", false))

	ao := &analytics.AuctionObject{
		Status:   http.StatusOK,
		Errors:   nil,
		Response: &openrtb2.BidResponse{},
	}

	am.LogAuctionObject(ao, acDenied)
	if count != 0 {
		t.Errorf("PBSAnalyticsModule failed at LogAuctionObject")
	}

	am.LogAmpObject(&analytics.AmpObject{}, acDenied)
	if count != 0 {
		t.Errorf("PBSAnalyticsModule failed at LogAmpObject")
	}

	am.LogVideoObject(&analytics.VideoObject{}, acDenied)
	if count != 0 {
		t.Errorf("PBSAnalyticsModule failed at LogVideoObject")
	}

	am.LogNotificationEventObject(&analytics.NotificationEvent{}, acDenied)
	if count != 0 {
		t.Errorf("PBSAnalyticsModule failed at LogNotificationEventObject")
	}
}

func getDefaultActivityConfig(componentName string, allow bool) *config.AccountPrivacy {
	return &config.AccountPrivacy{
		AllowActivities: &config.AllowActivities{
			ReportAnalytics: config.Activity{
				Default: ptrutil.ToPtr(true),
				Rules: []config.ActivityRule{
					{
						Allow: allow,
						Condition: config.ActivityCondition{
							ComponentName: []string{componentName},
							ComponentType: []string{"analytics"},
						},
					},
				},
			},
		},
	}
}
