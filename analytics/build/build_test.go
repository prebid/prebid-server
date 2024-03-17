package build

import (
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/prebid/prebid-server/v2/util/iputil"

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
		Status:         http.StatusOK,
		RequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: getDefaultBidRequest()},
		Errors:         nil,
		Response:       &openrtb2.BidResponse{},
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

	am.LogAmpObject(&analytics.AmpObject{RequestWrapper: &openrtb_ext.RequestWrapper{}}, privacy.ActivityControl{})
	if count != 4 {
		t.Errorf("PBSAnalyticsModule failed at LogAmpObject")
	}

	am.LogVideoObject(&analytics.VideoObject{RequestWrapper: &openrtb_ext.RequestWrapper{}}, privacy.ActivityControl{})
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

	acAllowed := privacy.NewActivityControl(getActivityConfig("sampleModule", true, true, true))

	ao := &analytics.AuctionObject{
		Status:         http.StatusOK,
		RequestWrapper: &openrtb_ext.RequestWrapper{},
		Errors:         nil,
		Response:       &openrtb2.BidResponse{},
	}

	am.LogAuctionObject(ao, acAllowed)
	if count != 1 {
		t.Errorf("PBSAnalyticsModule failed at LogAuctionObject")
	}

	am.LogAmpObject(&analytics.AmpObject{RequestWrapper: &openrtb_ext.RequestWrapper{}}, acAllowed)
	if count != 2 {
		t.Errorf("PBSAnalyticsModule failed at LogAmpObject")
	}

	am.LogVideoObject(&analytics.VideoObject{RequestWrapper: &openrtb_ext.RequestWrapper{}}, acAllowed)
	if count != 3 {
		t.Errorf("PBSAnalyticsModule failed at LogVideoObject")
	}

	am.LogNotificationEventObject(&analytics.NotificationEvent{}, acAllowed)
	if count != 4 {
		t.Errorf("PBSAnalyticsModule failed at LogNotificationEventObject")
	}
}

func TestSampleModuleActivitiesAllowedAndDenied(t *testing.T) {
	var count int
	am := initAnalytics(&count)

	acAllowed := privacy.NewActivityControl(getActivityConfig("sampleModule", true, false, true))

	rw := &openrtb_ext.RequestWrapper{BidRequest: getDefaultBidRequest()}
	ao := &analytics.AuctionObject{
		RequestWrapper: rw,
		Status:         http.StatusOK,
		Errors:         nil,
		Response:       &openrtb2.BidResponse{},
	}

	am.LogAuctionObject(ao, acAllowed)
	if count != 1 {
		t.Errorf("PBSAnalyticsModule failed at LogAuctionObject")
	}

	am.LogAmpObject(&analytics.AmpObject{RequestWrapper: rw}, acAllowed)
	if count != 2 {
		t.Errorf("PBSAnalyticsModule failed at LogAmpObject")
	}

	am.LogVideoObject(&analytics.VideoObject{RequestWrapper: rw}, acAllowed)
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

	acDenied := privacy.NewActivityControl(getActivityConfig("sampleModule", false, true, true))

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

func TestEvaluateActivities(t *testing.T) {
	testCases := []struct {
		description             string
		givenActivityControl    privacy.ActivityControl
		expectedRequest         *openrtb_ext.RequestWrapper
		expectedAllowActivities bool
	}{
		{
			description:             "all blocked",
			givenActivityControl:    privacy.NewActivityControl(getActivityConfig("sampleModule", false, false, false)),
			expectedRequest:         nil,
			expectedAllowActivities: false,
		},
		{
			description:             "all allowed",
			givenActivityControl:    privacy.NewActivityControl(getActivityConfig("sampleModule", true, true, true)),
			expectedRequest:         nil,
			expectedAllowActivities: true,
		},

		{
			description:          "ActivityTransmitUserFPD and ActivityTransmitPreciseGeo disabled",
			givenActivityControl: privacy.NewActivityControl(getActivityConfig("sampleModule", true, false, false)),
			expectedRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{ID: "test_request", User: &openrtb2.User{ID: ""}, Device: &openrtb2.Device{IFA: "", IP: "127.0.0.0"}},
			},
			expectedAllowActivities: true,
		},
		{
			description:          "ActivityTransmitUserFPD enabled, ActivityTransmitPreciseGeo disabled",
			givenActivityControl: privacy.NewActivityControl(getActivityConfig("sampleModule", true, true, false)),
			expectedRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{ID: "test_request", User: &openrtb2.User{ID: "user-id"}, Device: &openrtb2.Device{IFA: "device-ifa", IP: "127.0.0.0"}},
			},
			expectedAllowActivities: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			rw := &openrtb_ext.RequestWrapper{BidRequest: getDefaultBidRequest()}
			resActivityAllowed, resRequest := evaluateActivities(rw, test.givenActivityControl, "sampleModule")
			assert.Equal(t, test.expectedAllowActivities, resActivityAllowed)
			if test.expectedRequest != nil {
				assert.Equal(t, test.expectedRequest.User.ID, resRequest.User.ID)
				assert.Equal(t, test.expectedRequest.Device.IFA, resRequest.Device.IFA)
				assert.Equal(t, test.expectedRequest.Device.IP, resRequest.Device.IP)
			} else {
				assert.Nil(t, resRequest)
			}

		})
	}

}

func getDefaultBidRequest() *openrtb2.BidRequest {
	return &openrtb2.BidRequest{
		ID:     "test_request",
		User:   &openrtb2.User{ID: "user-id"},
		Device: &openrtb2.Device{IFA: "device-ifa", IP: "127.0.0.1"},
	}
}

func getActivityConfig(componentName string, allowReportAnalytics, allowTransmitUserFPD, allowTransmitPreciseGeo bool) *config.AccountPrivacy {
	return &config.AccountPrivacy{
		AllowActivities: &config.AllowActivities{
			ReportAnalytics: config.Activity{
				Default: ptrutil.ToPtr(true),
				Rules: []config.ActivityRule{
					{
						Allow: allowReportAnalytics,
						Condition: config.ActivityCondition{
							ComponentName: []string{componentName},
							ComponentType: []string{"analytics"},
						},
					},
				},
			},
			TransmitUserFPD: config.Activity{
				Default: ptrutil.ToPtr(true),
				Rules: []config.ActivityRule{
					{
						Allow: allowTransmitUserFPD,
						Condition: config.ActivityCondition{
							ComponentName: []string{componentName},
							ComponentType: []string{"analytics"},
						},
					},
				},
			},
			TransmitPreciseGeo: config.Activity{
				Default: ptrutil.ToPtr(true),
				Rules: []config.ActivityRule{
					{
						Allow: allowTransmitPreciseGeo,
						Condition: config.ActivityCondition{
							ComponentName: []string{componentName},
							ComponentType: []string{"analytics"},
						},
					},
				},
			},
		},
		IPv4Config: config.IPv4{
			AnonKeepBits: iputil.IPv4DefaultMaskingBitSize,
		},
		IPv6Config: config.IPv6{
			AnonKeepBits: iputil.IPv6DefaultMaskingBitSize,
		},
	}
}

type mockAnalytics struct {
	lastLoggedAuctionObject    *analytics.AuctionObject
	lastLoggedAmpObject        *analytics.AmpObject
	lastLoggedCookieSyncObject *analytics.CookieSyncObject
	lastLoggedSetUIDObject     *analytics.SetUIDObject
	lastLoggedVideoObject      *analytics.VideoObject
	lastLoggedEventObject      *analytics.NotificationEvent
}

func (m *mockAnalytics) LogAuctionObject(ao *analytics.AuctionObject) {
	m.lastLoggedAuctionObject = ao
}

func (m *mockAnalytics) LogAmpObject(ao *analytics.AmpObject) {
	m.lastLoggedAmpObject = ao
}

func (m *mockAnalytics) LogCookieSyncObject(ao *analytics.CookieSyncObject) {
	m.lastLoggedCookieSyncObject = ao
}

func (m *mockAnalytics) LogSetUIDObject(ao *analytics.SetUIDObject) {
	m.lastLoggedSetUIDObject = ao
}

func (m *mockAnalytics) LogVideoObject(ao *analytics.VideoObject) {
	m.lastLoggedVideoObject = ao
}

func (m *mockAnalytics) LogNotificationEventObject(ao *analytics.NotificationEvent) {
	m.lastLoggedEventObject = ao
}

// TODO: Change test structure so that we're calling multiple modules at once where evaluateActivites didn't make a clone
func TestLogAnalyticsObjectExtRequestPrebidAnalytics(t *testing.T) {
	tests := []struct {
		description         string
		givenRequestWrapper *openrtb_ext.RequestWrapper
		givenAdapterName    string
		expectedBidRequest  *openrtb2.BidRequest
	}{
		{
			description: "Adapter1 so Adapter2 info should be removed from ext.prebid.analytics",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:  "test_request",
					Ext: []byte(`{"prebid":{"analytics":{"adapter1":{"client-analytics":true},"adapter2":{"client-analytics":false}}}}`)},
			},
			givenAdapterName: "adapter1",
			expectedBidRequest: &openrtb2.BidRequest{
				ID:  "test_request",
				Ext: []byte(`{"prebid":{"analytics":{"adapter1":{"client-analytics":true}}}}`)},
		},
		{
			description: "Adapter 2 so Adapter1 info should be removed from ext.prebid.analytics",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:  "test_request",
					Ext: []byte(`{"prebid":{"analytics":{"adapter1":{"client-analytics":true},"adapter2":{"client-analytics":false}}}}`)},
			},
			givenAdapterName: "adapter2",
			expectedBidRequest: &openrtb2.BidRequest{
				ID:  "test_request",
				Ext: []byte(`{"prebid":{"analytics":{"adapter2":{"client-analytics":false}}}}`)},
		},
		{
			description: "Given adapter not found in ext.prebid.analytics so remove entire object",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:  "test_request",
					Ext: []byte(`{"prebid":{"analytics":{"adapter1":{"client-analytics":true},"adapter2":{"client-analytics":false}}}}`)},
			},
			givenAdapterName: "adapterNotFound",
			expectedBidRequest: &openrtb2.BidRequest{
				ID:  "test_request",
				Ext: nil,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			ea := enabledAnalytics{test.givenAdapterName: &mockAnalytics{}}

			ac := privacy.NewActivityControl(getActivityConfig("sampleModule", true, true, true))

			ao := &analytics.AuctionObject{
				Status:         http.StatusOK,
				Errors:         nil,
				Response:       &openrtb2.BidResponse{},
				RequestWrapper: test.givenRequestWrapper,
			}

			amp := &analytics.AmpObject{
				Status:          http.StatusOK,
				Errors:          nil,
				AuctionResponse: &openrtb2.BidResponse{},
				RequestWrapper:  test.givenRequestWrapper,
			}

			video := &analytics.VideoObject{
				Status:         http.StatusOK,
				Errors:         nil,
				Response:       &openrtb2.BidResponse{},
				RequestWrapper: test.givenRequestWrapper,
			}

			ea.LogAuctionObject(ao, ac)
			ea.LogAmpObject(amp, ac)
			ea.LogVideoObject(video, ac)

			// Retrieve the logged analytics from the mockAnalytics module, and assert that the logged object was properly altered
			loggedAo := ea[test.givenAdapterName].(*mockAnalytics).lastLoggedAuctionObject
			loggedAmp := ea[test.givenAdapterName].(*mockAnalytics).lastLoggedAmpObject
			loggedVideo := ea[test.givenAdapterName].(*mockAnalytics).lastLoggedVideoObject

			assert.Equal(t, test.expectedBidRequest, loggedAo.RequestWrapper.BidRequest)
			assert.Equal(t, test.expectedBidRequest, loggedAmp.RequestWrapper.BidRequest)
			assert.Equal(t, test.expectedBidRequest, loggedVideo.RequestWrapper.BidRequest)
		})
	}
}

func TestUpdateReqWrapperForAnalytics(t *testing.T) {
	tests := []struct {
		description      string
		givenReqWrapper  *openrtb_ext.RequestWrapper
		givenAdapterName string
		expected         *openrtb2.BidRequest
	}{
		{
			description: "Adapter1 so Adapter2 info should be removed from ext.prebid.analytics",
			givenReqWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: []byte(`{"prebid":{"analytics":{"adapter1":{"client-analytics":true},"adapter2":{"client-analytics":false}}}}`)},
			},
			givenAdapterName: "adapter1",
			expected: &openrtb2.BidRequest{
				Ext: []byte(`{"prebid":{"analytics":{"adapter1":{"client-analytics":true}}}}`),
			},
		},
		{
			description: "Adapter2 so Adapter1 info should be removed from ext.prebid.analytics",
			givenReqWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: []byte(`{"prebid":{"analytics":{"adapter1":{"client-analytics":true},"adapter2":{"client-analytics":false}}}}`)},
			},
			givenAdapterName: "adapter2",
			expected: &openrtb2.BidRequest{
				Ext: []byte(`{"prebid":{"analytics":{"adapter2":{"client-analytics":false}}}}`),
			},
		},
		{
			description: "Given adapter not found in ext.prebid.analytics so remove entire object",
			givenReqWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: []byte(`{"prebid":{"analytics":{"adapter1":{"client-analytics":true},"adapter2":{"client-analytics":false}}}}`)},
			},
			givenAdapterName: "adapterNotFound",
			expected:         &openrtb2.BidRequest{},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			updateReqWrapperForAnalytics(test.givenReqWrapper, test.givenAdapterName)
			assert.Equal(t, test.expected, test.givenReqWrapper.BidRequest)
		})
	}
}
