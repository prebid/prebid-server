package build

import (
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/iputil"

	"net/http"
	"os"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/privacy"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
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

func (m *sampleModule) Shutdown() { *m.count++ }

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

func TestPBSAnalyticsShutdown(t *testing.T) {
	countA := 0
	countB := 0
	modules := make(enabledAnalytics, 0)
	modules["sampleModuleA"] = &sampleModule{count: &countA}
	modules["sampleModuleB"] = &sampleModule{count: &countB}

	modules.Shutdown()

	assert.Equal(t, 1, countA, "sampleModuleA should have been shutdown")
	assert.Equal(t, 1, countB, "sampleModuleB should have been shutdown")
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

func TestNewModuleHttp(t *testing.T) {
	agmaAnalyticsWithoutError := New(&config.Analytics{
		Agma: config.AgmaAnalytics{
			Enabled: true,
			Endpoint: config.AgmaAnalyticsHttpEndpoint{
				Url:     "http://localhost:8080",
				Timeout: "1s",
			},
			Buffers: config.AgmaAnalyticsBuffer{
				BufferSize: "100KB",
				EventCount: 50,
				Timeout:    "30s",
			},
			Accounts: []config.AgmaAnalyticsAccount{
				{
					PublisherId: "123",
					Code:        "abc",
				},
			},
		},
	})
	instanceWithoutError := agmaAnalyticsWithoutError.(enabledAnalytics)

	assert.Equal(t, len(instanceWithoutError), 1)

	agmaAnalyticsWithError := New(&config.Analytics{
		Agma: config.AgmaAnalytics{
			Enabled: true,
		},
	})
	instanceWithError := agmaAnalyticsWithError.(enabledAnalytics)
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
	lastLoggedAuctionBidRequest *openrtb2.BidRequest
	lastLoggedAmpBidRequest     *openrtb2.BidRequest
	lastLoggedVideoBidRequest   *openrtb2.BidRequest
}

func (m *mockAnalytics) LogAuctionObject(ao *analytics.AuctionObject) {
	m.lastLoggedAuctionBidRequest = ao.RequestWrapper.BidRequest
}

func (m *mockAnalytics) LogAmpObject(ao *analytics.AmpObject) {
	m.lastLoggedAmpBidRequest = ao.RequestWrapper.BidRequest
}

func (m *mockAnalytics) LogVideoObject(vo *analytics.VideoObject) {
	m.lastLoggedVideoBidRequest = vo.RequestWrapper.BidRequest
}

func (m *mockAnalytics) LogCookieSyncObject(ao *analytics.CookieSyncObject) {}

func (m *mockAnalytics) LogSetUIDObject(ao *analytics.SetUIDObject) {}

func (m *mockAnalytics) LogNotificationEventObject(ao *analytics.NotificationEvent) {}

func (m *mockAnalytics) Shutdown() {}

func TestLogObject(t *testing.T) {
	tests := []struct {
		description           string
		givenRequestWrapper   *openrtb_ext.RequestWrapper
		givenEnabledAnalytics enabledAnalytics
		givenActivityControl  bool
		givenAuctionObject    *analytics.AuctionObject
		givenAmpObject        *analytics.AmpObject
		givenVideoObject      *analytics.VideoObject
		expectedBidRequest1   *openrtb2.BidRequest
		expectedBidRequest2   *openrtb2.BidRequest
	}{
		{
			description:           "Multiple analytics modules, clone from evaluate activities, should expect both to have their information to be logged only -- auction",
			givenEnabledAnalytics: enabledAnalytics{"adapter1": &mockAnalytics{}, "adapter2": &mockAnalytics{}},
			givenActivityControl:  true,
			givenAuctionObject: &analytics.AuctionObject{
				Status:   http.StatusOK,
				Errors:   nil,
				Response: &openrtb2.BidResponse{},
				RequestWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID:  "test_request",
						Ext: []byte(`{"prebid":{"analytics":{"adapter1":{"client-analytics":true},"adapter2":{"client-analytics":false}}}}`)},
				},
			},
			expectedBidRequest1: &openrtb2.BidRequest{
				ID:  "test_request",
				Ext: []byte(`{"prebid":{"analytics":{"adapter1":{"client-analytics":true}}}}`)},
			expectedBidRequest2: &openrtb2.BidRequest{
				ID:  "test_request",
				Ext: []byte(`{"prebid":{"analytics":{"adapter2":{"client-analytics":false}}}}`)},
		},
		{
			description:           "Multiple analytics modules, no clone from evaluate activities, should expect both to have their information to be logged only -- amp",
			givenEnabledAnalytics: enabledAnalytics{"adapter1": &mockAnalytics{}, "adapter2": &mockAnalytics{}},
			givenActivityControl:  false,
			givenAmpObject: &analytics.AmpObject{
				Status:          http.StatusOK,
				Errors:          nil,
				AuctionResponse: &openrtb2.BidResponse{},
				RequestWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID:  "test_request",
						Ext: []byte(`{"prebid":{"analytics":{"adapter1":{"client-analytics":true},"adapter2":{"client-analytics":false}}}}`)},
				},
			},
			expectedBidRequest1: &openrtb2.BidRequest{
				ID:  "test_request",
				Ext: []byte(`{"prebid":{"analytics":{"adapter1":{"client-analytics":true}}}}`)},
			expectedBidRequest2: &openrtb2.BidRequest{
				ID:  "test_request",
				Ext: []byte(`{"prebid":{"analytics":{"adapter2":{"client-analytics":false}}}}`)},
		},
		{
			description:           "Single analytics module, clone from evaluate activities, should expect both to have their information to be logged only -- amp",
			givenEnabledAnalytics: enabledAnalytics{"adapter1": &mockAnalytics{}},
			givenActivityControl:  true,
			givenAuctionObject: &analytics.AuctionObject{
				Status:   http.StatusOK,
				Errors:   nil,
				Response: &openrtb2.BidResponse{},
				RequestWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID:  "test_request",
						Ext: []byte(`{"prebid":{"analytics":{"adapter1":{"client-analytics":true},"adapter2":{"client-analytics":false}}}}`)},
				},
			},
			expectedBidRequest1: &openrtb2.BidRequest{
				ID:  "test_request",
				Ext: []byte(`{"prebid":{"analytics":{"adapter1":{"client-analytics":true}}}}`)},
		},
		{
			description:           "Single analytics module, adapter name not found, expect entire analytics object to be nil -- video",
			givenEnabledAnalytics: enabledAnalytics{"unknownAdapter": &mockAnalytics{}},
			givenActivityControl:  true,
			givenVideoObject: &analytics.VideoObject{
				Status:   http.StatusOK,
				Errors:   nil,
				Response: &openrtb2.BidResponse{},
				RequestWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID:  "test_request",
						Ext: []byte(`{"prebid":{"analytics":{"adapter1":{"client-analytics":true},"adapter2":{"client-analytics":false}}}}`)},
				},
			},
			expectedBidRequest1: &openrtb2.BidRequest{
				ID:  "test_request",
				Ext: nil,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			ac := privacy.NewActivityControl(getActivityConfig("sampleModule", test.givenActivityControl, test.givenActivityControl, test.givenActivityControl))

			var loggedBidReq1, loggedBidReq2 *openrtb2.BidRequest
			switch {
			case test.givenAuctionObject != nil:
				test.givenEnabledAnalytics.LogAuctionObject(test.givenAuctionObject, ac)
				loggedBidReq1 = test.givenEnabledAnalytics["adapter1"].(*mockAnalytics).lastLoggedAuctionBidRequest
				if len(test.givenEnabledAnalytics) == 2 {
					loggedBidReq2 = test.givenEnabledAnalytics["adapter2"].(*mockAnalytics).lastLoggedAuctionBidRequest
				}
			case test.givenAmpObject != nil:
				test.givenEnabledAnalytics.LogAmpObject(test.givenAmpObject, ac)
				loggedBidReq1 = test.givenEnabledAnalytics["adapter1"].(*mockAnalytics).lastLoggedAmpBidRequest
				if len(test.givenEnabledAnalytics) == 2 {
					loggedBidReq2 = test.givenEnabledAnalytics["adapter2"].(*mockAnalytics).lastLoggedAmpBidRequest
				}
			case test.givenVideoObject != nil:
				test.givenEnabledAnalytics.LogVideoObject(test.givenVideoObject, ac)
				loggedBidReq1 = test.givenEnabledAnalytics["unknownAdapter"].(*mockAnalytics).lastLoggedVideoBidRequest
			}

			assert.Equal(t, test.expectedBidRequest1, loggedBidReq1)
			if test.expectedBidRequest2 != nil {
				assert.Equal(t, test.expectedBidRequest2, loggedBidReq2)
			}
		})
	}
}

func TestUpdateReqWrapperForAnalytics(t *testing.T) {
	tests := []struct {
		description               string
		givenReqWrapper           *openrtb_ext.RequestWrapper
		givenAdapterName          string
		givenIsCloned             bool
		expectedUpdatedBidRequest *openrtb2.BidRequest
		expectedCloneRequest      *openrtb_ext.RequestWrapper
	}{
		{
			description: "Adapter1 so Adapter2 info should be removed from ext.prebid.analytics",
			givenReqWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: []byte(`{"prebid":{"analytics":{"adapter1":{"client-analytics":true},"adapter2":{"client-analytics":false}}}}`)},
			},
			givenAdapterName: "adapter1",
			givenIsCloned:    false,
			expectedUpdatedBidRequest: &openrtb2.BidRequest{
				Ext: []byte(`{"prebid":{"analytics":{"adapter1":{"client-analytics":true}}}}`),
			},
			expectedCloneRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: []byte(`{"prebid":{"analytics":{"adapter1":{"client-analytics":true},"adapter2":{"client-analytics":false}}}}`)},
			},
		},
		{
			description: "Adapter2 so Adapter1 info should be removed from ext.prebid.analytics",
			givenReqWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: []byte(`{"prebid":{"analytics":{"adapter1":{"client-analytics":true},"adapter2":{"client-analytics":false}}}}`)},
			},
			givenAdapterName: "adapter2",
			givenIsCloned:    true,
			expectedUpdatedBidRequest: &openrtb2.BidRequest{
				Ext: []byte(`{"prebid":{"analytics":{"adapter2":{"client-analytics":false}}}}`),
			},
			expectedCloneRequest: nil,
		},
		{
			description: "Given adapter not found in ext.prebid.analytics so remove entire object",
			givenReqWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: []byte(`{"prebid":{"analytics":{"adapter1":{"client-analytics":true},"adapter2":{"client-analytics":false}}}}`)},
			},
			givenAdapterName:          "adapterNotFound",
			givenIsCloned:             false,
			expectedUpdatedBidRequest: &openrtb2.BidRequest{},
			expectedCloneRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: []byte(`{"prebid":{"analytics":{"adapter1":{"client-analytics":true},"adapter2":{"client-analytics":false}}}}`)},
			},
		},
		{
			description:               "Given request is nil, check there are no exceptions",
			givenReqWrapper:           nil,
			givenAdapterName:          "adapter1",
			givenIsCloned:             false,
			expectedUpdatedBidRequest: nil,
			expectedCloneRequest:      nil,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			cloneReq := updateReqWrapperForAnalytics(test.givenReqWrapper, test.givenAdapterName, test.givenIsCloned)
			if test.givenReqWrapper != nil {
				assert.Equal(t, test.expectedUpdatedBidRequest, test.givenReqWrapper.BidRequest)
			}
			assert.Equal(t, test.expectedCloneRequest, cloneReq)
		})
	}
}
