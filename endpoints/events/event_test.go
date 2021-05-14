package events

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Mock Analytics Module
type eventsMockAnalyticsModule struct {
	Fail    bool
	Error   error
	Invoked bool
}

func (e *eventsMockAnalyticsModule) LogAuctionObject(ao *analytics.AuctionObject) {
	if e.Fail {
		panic(e.Error)
	}
	return
}

func (e *eventsMockAnalyticsModule) LogVideoObject(vo *analytics.VideoObject) {
	if e.Fail {
		panic(e.Error)
	}
	return
}

func (e *eventsMockAnalyticsModule) LogCookieSyncObject(cso *analytics.CookieSyncObject) {
	if e.Fail {
		panic(e.Error)
	}
	return
}

func (e *eventsMockAnalyticsModule) LogSetUIDObject(so *analytics.SetUIDObject) {
	if e.Fail {
		panic(e.Error)
	}
	return
}

func (e *eventsMockAnalyticsModule) LogAmpObject(ao *analytics.AmpObject) {
	if e.Fail {
		panic(e.Error)
	}
	return
}

func (e *eventsMockAnalyticsModule) LogNotificationEventObject(ne *analytics.NotificationEvent) {
	if e.Fail {
		panic(e.Error)
	}
	e.Invoked = true

	return
}

// Mock Account fetcher
var mockAccountData = map[string]json.RawMessage{
	"events_enabled":  json.RawMessage(`{"events_enabled":true}`),
	"events_disabled": json.RawMessage(`{"events_enabled":false}`),
}

type mockAccountsFetcher struct {
	Fail       bool
	Error      error
	DurationMS int
}

func (maf mockAccountsFetcher) FetchAccount(ctx context.Context, accountID string) (json.RawMessage, []error) {
	if maf.DurationMS > 0 {
		select {
		case <-time.After(time.Duration(maf.DurationMS) * time.Millisecond):
			break
		case <-ctx.Done():
			return nil, []error{ctx.Err()}
		}
	}

	if account, ok := mockAccountData[accountID]; ok {
		return account, nil
	}

	if maf.Fail {
		return nil, []error{maf.Error}
	}

	return nil, []error{stored_requests.NotFoundError{accountID, "Account"}}
}

// Tests

func TestShouldReturnBadRequestWhenTypeIsMissing(t *testing.T) {

	// mock AccountsFetcher
	mockAccountsFetcher := &mockAccountsFetcher{}

	// mock PBS Analytics Module
	mockAnalyticsModule := &eventsMockAnalyticsModule{
		Fail: false,
	}

	// mock config
	cfg := &config.Configuration{
		AccountDefaults: config.Account{},
	}

	// prepare
	reqData := ""

	req := httptest.NewRequest("GET", "/event?b=test", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := NewEventEndpoint(cfg, mockAccountsFetcher, mockAnalyticsModule)

	// execute
	e(recorder, req, nil)

	d, err := ioutil.ReadAll(recorder.Result().Body)
	if err != nil {
		t.Fatal(err)
	}

	// validate
	assert.Equal(t, 400, recorder.Result().StatusCode, "Expected 400 on request with missing type parameter")
	assert.Equal(t, "invalid request: parameter 't' is required\n", string(d))
}

func TestShouldReturnBadRequestWhenTypeIsInvalid(t *testing.T) {

	// mock AccountsFetcher
	mockAccounts := &mockAccountsFetcher{}

	// mock PBS Analytics Module
	mockAnalyticsModule := &eventsMockAnalyticsModule{
		Fail: false,
	}

	// mock config
	cfg := &config.Configuration{
		AccountDefaults: config.Account{},
	}

	// prepare
	reqData := ""

	req := httptest.NewRequest("GET", "/event?t=test&b=t", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := NewEventEndpoint(cfg, mockAccounts, mockAnalyticsModule)

	// execute
	e(recorder, req, nil)

	d, err := ioutil.ReadAll(recorder.Result().Body)
	if err != nil {
		t.Fatal(err)
	}

	// validate
	assert.Equal(t, 400, recorder.Result().StatusCode, "Expected 400 on request with invalid type parameter")
	assert.Equal(t, "invalid request: unknown type: 'test'\n", string(d))
}

func TestShouldReturnBadRequestWhenBidIdIsMissing(t *testing.T) {

	// mock AccountsFetcher
	mockAccountsFetcher := &mockAccountsFetcher{}

	// mock PBS Analytics Module
	mockAnalyticsModule := &eventsMockAnalyticsModule{
		Fail: false,
	}

	// mock config
	cfg := &config.Configuration{
		AccountDefaults: config.Account{},
	}

	// prepare
	reqData := ""

	req := httptest.NewRequest("GET", "/event?t=win", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := NewEventEndpoint(cfg, mockAccountsFetcher, mockAnalyticsModule)

	// execute
	e(recorder, req, nil)

	d, err := ioutil.ReadAll(recorder.Result().Body)
	if err != nil {
		t.Fatal(err)
	}

	// validate
	assert.Equal(t, 400, recorder.Result().StatusCode, "Expected 400 on request with missing bidid parameter")
	assert.Equal(t, "invalid request: parameter 'b' is required\n", string(d))
}

func TestShouldReturnBadRequestWhenTimestampIsInvalid(t *testing.T) {

	// mock AccountsFetcher
	mockAccountsFetcher := &mockAccountsFetcher{}

	// mock PBS Analytics Module
	mockAnalyticsModule := &eventsMockAnalyticsModule{
		Fail: false,
	}

	// mock config
	cfg := &config.Configuration{
		AccountDefaults: config.Account{},
	}

	// prepare
	reqData := ""

	req := httptest.NewRequest("GET", "/event?t=win&b=test&ts=q", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := NewEventEndpoint(cfg, mockAccountsFetcher, mockAnalyticsModule)

	// execute
	e(recorder, req, nil)

	d, err := ioutil.ReadAll(recorder.Result().Body)
	if err != nil {
		t.Fatal(err)
	}

	// validate
	assert.Equal(t, 400, recorder.Result().StatusCode, "Expected 400 on request with invalid timestamp parameter")
	assert.Equal(t, "invalid request: invalid request: error parsing timestamp 'q'\n", string(d))
}

func TestShouldReturnUnauthorizedWhenAccountIsMissing(t *testing.T) {

	// mock AccountsFetcher
	mockAccountsFetcher := &mockAccountsFetcher{}

	// mock PBS Analytics Module
	mockAnalyticsModule := &eventsMockAnalyticsModule{
		Fail: false,
	}

	// mock config
	cfg := &config.Configuration{
		AccountDefaults: config.Account{},
	}

	// prepare
	reqData := ""

	req := httptest.NewRequest("GET", "/event?t=win&b=test&ts=1234", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := NewEventEndpoint(cfg, mockAccountsFetcher, mockAnalyticsModule)

	// execute
	e(recorder, req, nil)

	d, err := ioutil.ReadAll(recorder.Result().Body)
	if err != nil {
		t.Fatal(err)
	}

	// validate
	assert.Equal(t, 401, recorder.Result().StatusCode, "Expected 401 on request with missing account id parameter")
	assert.Equal(t, "Account 'a' is required query parameter and can't be empty", string(d))
}

func TestShouldReturnBadRequestWhenFormatValueIsInvalid(t *testing.T) {

	// mock AccountsFetcher
	mockAccountsFetcher := &mockAccountsFetcher{}

	// mock PBS Analytics Module
	mockAnalyticsModule := &eventsMockAnalyticsModule{
		Fail: false,
	}

	// mock config
	cfg := &config.Configuration{
		AccountDefaults: config.Account{},
	}

	// prepare
	reqData := ""

	req := httptest.NewRequest("GET", "/event?t=win&b=test&ts=1234&f=q", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := NewEventEndpoint(cfg, mockAccountsFetcher, mockAnalyticsModule)

	// execute
	e(recorder, req, nil)

	d, err := ioutil.ReadAll(recorder.Result().Body)
	if err != nil {
		t.Fatal(err)
	}

	// validate
	assert.Equal(t, 400, recorder.Result().StatusCode, "Expected 400 on request with invalid format parameter")
	assert.Equal(t, "invalid request: unknown format: 'q'\n", string(d))
}

func TestShouldReturnBadRequestWhenAnalyticsValueIsInvalid(t *testing.T) {

	// mock AccountsFetcher
	mockAccountsFetcher := &mockAccountsFetcher{}

	// mock PBS Analytics Module
	mockAnalyticsModule := &eventsMockAnalyticsModule{
		Fail: false,
	}

	// mock config
	cfg := &config.Configuration{
		AccountDefaults: config.Account{},
	}

	// prepare
	reqData := ""

	req := httptest.NewRequest("GET", "/event?t=win&b=test&ts=1234&f=b&x=4", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := NewEventEndpoint(cfg, mockAccountsFetcher, mockAnalyticsModule)

	// execute
	e(recorder, req, nil)

	d, err := ioutil.ReadAll(recorder.Result().Body)
	if err != nil {
		t.Fatal(err)
	}

	// validate
	assert.Equal(t, 400, recorder.Result().StatusCode, "Expected 400 on request with invalid analytics parameter")
	assert.Equal(t, "invalid request: unknown analytics: '4'\n", string(d))
}

func TestShouldNotPassEventToAnalyticsReporterWhenAccountNotFound(t *testing.T) {

	// mock AccountsFetcher
	mockAccountsFetcher := &mockAccountsFetcher{
		Fail:  true,
		Error: stored_requests.NotFoundError{ID: "testacc"},
	}

	// mock PBS Analytics Module
	mockAnalyticsModule := &eventsMockAnalyticsModule{
		Fail: false,
	}

	// mock config
	cfg := &config.Configuration{
		AccountDefaults: config.Account{},
	}

	// prepare
	reqData := ""

	req := httptest.NewRequest("GET", "/event?t=win&b=test&ts=1234&f=b&x=1&a=testacc", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := NewEventEndpoint(cfg, mockAccountsFetcher, mockAnalyticsModule)

	// execute
	e(recorder, req, nil)
	d, err := ioutil.ReadAll(recorder.Result().Body)
	if err != nil {
		t.Fatal(err)
	}

	// validate
	assert.Equal(t, 401, recorder.Result().StatusCode, "Expected 401 on account not found")
	assert.Equal(t, "Account 'testacc' doesn't support events", string(d))
}

func TestShouldNotPassEventToAnalyticsReporterWhenAccountEventNotEnabled(t *testing.T) {

	// mock AccountsFetcher
	mockAccountsFetcher := &mockAccountsFetcher{
		Fail: false,
	}

	// mock PBS Analytics Module
	mockAnalyticsModule := &eventsMockAnalyticsModule{
		Fail: false,
	}

	// mock config
	cfg := &config.Configuration{
		AccountDefaults: config.Account{},
	}
	cfg.MarshalAccountDefaults()

	// prepare
	reqData := ""

	req := httptest.NewRequest("GET", "/event?t=win&b=test&ts=1234&f=b&x=1&a=events_disabled", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := NewEventEndpoint(cfg, mockAccountsFetcher, mockAnalyticsModule)

	// execute
	e(recorder, req, nil)
	d, err := ioutil.ReadAll(recorder.Result().Body)
	if err != nil {
		t.Fatal(err)
	}

	// validate
	assert.Equal(t, 401, recorder.Result().StatusCode, "Expected 401 on account with events disabled")
	assert.Equal(t, "Account 'events_disabled' doesn't support events", string(d))
}

func TestShouldPassEventToAnalyticsReporterWhenAccountEventEnabled(t *testing.T) {

	// mock AccountsFetcher
	mockAccountsFetcher := &mockAccountsFetcher{
		Fail: false,
	}

	// mock PBS Analytics Module
	mockAnalyticsModule := &eventsMockAnalyticsModule{
		Fail: false,
	}

	// mock config
	cfg := &config.Configuration{
		AccountDefaults: config.Account{},
	}
	cfg.MarshalAccountDefaults()

	// prepare
	reqData := ""

	req := httptest.NewRequest("GET", "/event?t=win&b=test&ts=1234&f=b&x=1&a=events_enabled", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := NewEventEndpoint(cfg, mockAccountsFetcher, mockAnalyticsModule)

	// execute
	e(recorder, req, nil)

	// validate
	assert.Equal(t, 204, recorder.Result().StatusCode, "Expected 204 when account has events enabled")
	assert.Equal(t, true, mockAnalyticsModule.Invoked)
}

func TestShouldNotPassEventToAnalyticsReporterWhenAnalyticsValueIsZero(t *testing.T) {

	// mock AccountsFetcher
	mockAccountsFetcher := &mockAccountsFetcher{
		Fail: false,
	}

	// mock PBS Analytics Module
	mockAnalyticsModule := &eventsMockAnalyticsModule{
		Fail: false,
	}

	// mock config
	cfg := &config.Configuration{
		AccountDefaults: config.Account{},
	}
	cfg.MarshalAccountDefaults()

	// prepare
	reqData := ""

	req := httptest.NewRequest("GET", "/event?t=win&b=test&ts=1234&f=b&x=0&a=events_enabled", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := NewEventEndpoint(cfg, mockAccountsFetcher, mockAnalyticsModule)

	// execute
	e(recorder, req, nil)

	// validate
	assert.Equal(t, 204, recorder.Result().StatusCode)
	assert.Equal(t, true, mockAnalyticsModule.Invoked != true)
}

func TestShouldRespondWithPixelAndContentTypeWhenRequestFormatIsImage(t *testing.T) {

	// mock AccountsFetcher
	mockAccountsFetcher := &mockAccountsFetcher{
		Fail: false,
	}

	// mock PBS Analytics Module
	mockAnalyticsModule := &eventsMockAnalyticsModule{
		Fail: false,
	}

	// mock config
	cfg := &config.Configuration{
		AccountDefaults: config.Account{},
	}
	cfg.MarshalAccountDefaults()

	// prepare
	reqData := ""

	req := httptest.NewRequest("GET", "/event?t=win&b=test&ts=1234&f=i&x=1&a=events_enabled", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := NewEventEndpoint(cfg, mockAccountsFetcher, mockAnalyticsModule)

	// execute
	e(recorder, req, nil)

	d, err := ioutil.ReadAll(recorder.Result().Body)
	if err != nil {
		t.Fatal(err)
	}

	// validate
	assert.Equal(t, 200, recorder.Result().StatusCode, "Expected 200 with tracking pixel when format is imp")
	assert.Equal(t, true, mockAnalyticsModule.Invoked)
	assert.Equal(t, "image/png", recorder.Header().Get("Content-Type"))
	assert.Equal(t, "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAABHNCSVQICAgIfAhkiAAAAA1JREFUCJljYGBgYAAAAAUAAYehTtQAAAAASUVORK5CYII=", base64.URLEncoding.EncodeToString(d))
}

func TestShouldRespondWithNoContentWhenRequestFormatIsNotDefined(t *testing.T) {

	// mock AccountsFetcher
	mockAccountsFetcher := &mockAccountsFetcher{
		Fail: false,
	}

	// mock PBS Analytics Module
	mockAnalyticsModule := &eventsMockAnalyticsModule{
		Fail: false,
	}

	// mock config
	cfg := &config.Configuration{
		AccountDefaults: config.Account{},
	}
	cfg.MarshalAccountDefaults()

	// prepare
	reqData := ""

	req := httptest.NewRequest("GET", "/event?t=imp&b=test&ts=1234&x=1&a=events_enabled", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := NewEventEndpoint(cfg, mockAccountsFetcher, mockAnalyticsModule)

	// execute
	e(recorder, req, nil)

	d, err := ioutil.ReadAll(recorder.Result().Body)
	if err != nil {
		t.Fatal(err)
	}

	// validate
	assert.Equal(t, 204, recorder.Result().StatusCode, "Expected 200 with empty response")
	assert.Equal(t, true, mockAnalyticsModule.Invoked)
	assert.Equal(t, "", recorder.Header().Get("Content-Type"))
	assert.Equal(t, 0, len(d))
}

func TestShouldParseEventCorrectly(t *testing.T) {

	tests := map[string]struct {
		req      *http.Request
		expected *analytics.EventRequest
	}{
		"one": {
			req: httptest.NewRequest("GET", "/event?t=win&b=bidId&f=b&ts=1000&x=1&a=accountId&bidder=bidder", strings.NewReader("")),
			expected: &analytics.EventRequest{
				Type:      analytics.Win,
				BidID:     "bidId",
				Timestamp: 1000,
				Bidder:    "bidder",
				AccountID: "",
				Format:    analytics.Blank,
				Analytics: analytics.Enabled,
			},
		},
		"two": {
			req: httptest.NewRequest("GET", "/event?t=win&b=bidId&ts=0&a=accountId", strings.NewReader("")),
			expected: &analytics.EventRequest{
				Type:      analytics.Win,
				BidID:     "bidId",
				Timestamp: 0,
				Analytics: analytics.Enabled,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			// execute
			er, errs := ParseEventRequest(test.req)

			// validate
			assert.Equal(t, 0, len(errs))
			assert.EqualValues(t, test.expected, er)
		})
	}
}

func TestEventRequestToUrl(t *testing.T) {
	externalUrl := "http://localhost:8000"
	tests := map[string]struct {
		er   *analytics.EventRequest
		want string
	}{
		"one": {
			er: &analytics.EventRequest{
				Type:      analytics.Imp,
				BidID:     "bidid",
				AccountID: "accountId",
				Bidder:    "bidder",
				Timestamp: 1234567,
				Format:    analytics.Blank,
				Analytics: analytics.Enabled,
			},
			want: "http://localhost:8000/event?t=imp&b=bidid&a=accountId&bidder=bidder&f=b&ts=1234567&x=1",
		},
		"two": {
			er: &analytics.EventRequest{
				Type:      analytics.Win,
				BidID:     "bidid",
				AccountID: "accountId",
				Bidder:    "bidder",
				Timestamp: 1234567,
				Format:    analytics.Image,
				Analytics: analytics.Disabled,
			},
			want: "http://localhost:8000/event?t=win&b=bidid&a=accountId&bidder=bidder&f=i&ts=1234567&x=0",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			expected := EventRequestToUrl(externalUrl, test.er)
			// validate
			assert.Equal(t, test.want, expected)
		})
	}
}
