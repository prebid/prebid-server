package filesystem

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/prebid/prebid-server/v4/analytics"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/hooks/hookanalytics"
	"github.com/prebid/prebid-server/v4/hooks/hookexecution"
	"github.com/stretchr/testify/mock"

	"github.com/prebid/openrtb/v20/openrtb2"
)

const TEST_DIR string = "testFiles"

type MockLogger struct {
	mock.Mock
}

func (ml *MockLogger) Debug(v ...interface{}) {
	ml.Called(v)
}

func (ml *MockLogger) Flush() {
	ml.Called()
}

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

func TestAuctionObject_ToJsonIncludesRulesEngineWarningAnalytics(t *testing.T) {
	ao := &analytics.AuctionObject{
		Status: http.StatusOK,
		HookExecutionOutcome: []hookexecution.StageOutcome{
			{
				Groups: []hookexecution.GroupOutcome{
					{
						InvocationResults: []hookexecution.HookOutcome{
							{
								HookID: hookexecution.HookID{
									ModuleCode:   "prebid.rulesengine",
									HookImplCode: "rulesengine",
								},
								AnalyticsTags: hookanalytics.Analytics{
									Activities: []hookanalytics.Activity{{
										Name:   "rules_engine_bidder_filtering",
										Status: hookanalytics.ActivityStatusSuccess,
										Results: []hookanalytics.Result{{
											Status: hookanalytics.ResultStatusBlock,
											Values: map[string]interface{}{
												"code":   errortypes.RulesEngineBidderExcludedWarningCode,
												"reason": "excluded_by_rule",
											},
										}},
									}},
								},
							},
						},
					},
				},
			},
		},
	}

	aoJSON := jsonifyAuctionObject(ao)

	if strings.Contains(aoJSON, "Transactional Logs Error") {
		t.Fatalf("AuctionObject failed to convert to json")
	}
	if !strings.Contains(aoJSON, `"name":"rules_engine_bidder_filtering"`) ||
		!strings.Contains(aoJSON, `"code":`+strconv.Itoa(errortypes.RulesEngineBidderExcludedWarningCode)) ||
		!strings.Contains(aoJSON, `"reason":"excluded_by_rule"`) {
		t.Fatalf("AuctionObject json did not include structured rules engine warning analytics: %s", aoJSON)
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
		BidderStatus: []*analytics.CookieSyncBidder{},
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

func TestFileLoggerShutdown(t *testing.T) {
	mockLogger := &MockLogger{}
	fl := &FileLogger{
		Logger: mockLogger,
	}
	mockLogger.On("Flush").Return(nil)

	fl.Shutdown()

	mockLogger.AssertNumberOfCalls(t, "Flush", 1)
}
