package endpoints

import (
	"database/sql"
	"encoding/base64"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/cache"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"
)

// Mock cache
type eventsMockCache struct {
	accounts *eventsMockAccountService
}

func (c *eventsMockCache) Accounts() cache.AccountsService {
	return c.accounts
}
func (c *eventsMockCache) Config() cache.ConfigService {
	return nil
}
func (c *eventsMockCache) Close() error {
	return nil
}

type eventsMockAccountService struct {
	Fail          bool
	Error         error
	EventsEnabled bool
}

func (s *eventsMockAccountService) Get(id string) (*cache.Account, error) {
	if s.Fail {
		return nil, s.Error
	}

	if s.EventsEnabled {
		return &cache.Account{
			ID:            id,
			EventsEnabled: true,
		}, nil
	}

	return &cache.Account{
		ID:            id,
		EventsEnabled: false,
	}, nil
}
func (s *eventsMockAccountService) Set(account *cache.Account) error {
	return nil
}

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

// Tests

func TestShouldReturnBadRequestWhenTypeIsMissing(t *testing.T) {

	// mock cache
	mockCache := &eventsMockCache{
		accounts: &eventsMockAccountService{},
	}

	// mock PBS Analytics Module
	mockAnalyticsModule := &eventsMockAnalyticsModule{
		Fail: false,
	}

	// prepare
	reqData := ""

	req := httptest.NewRequest("GET", "/event", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := NewEventEndpoint(mockCache, mockAnalyticsModule)

	// execute
	e(recorder, req, nil)

	d, _ := ioutil.ReadAll(recorder.Result().Body)

	// validate
	assert.Equal(t, 400, recorder.Result().StatusCode, "Expected 400 on request with missing type parameter")
	assert.Equal(t, "invalid request: parameter 't' is required\n", string(d))
}

func TestShouldReturnBadRequestWhenTypeIsInvalid(t *testing.T) {

	// mock cache
	mockCache := &eventsMockCache{
		accounts: &eventsMockAccountService{},
	}

	// mock PBS Analytics Module
	mockAnalyticsModule := &eventsMockAnalyticsModule{
		Fail: false,
	}

	// prepare
	reqData := ""

	req := httptest.NewRequest("GET", "/event?t=test", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := NewEventEndpoint(mockCache, mockAnalyticsModule)

	// execute
	e(recorder, req, nil)

	d, _ := ioutil.ReadAll(recorder.Result().Body)

	// validate
	assert.Equal(t, 400, recorder.Result().StatusCode, "Expected 400 on request with invalid type parameter")
	assert.Equal(t, "invalid request: unknown type: 'test'\n", string(d))
}

func TestShouldReturnBadRequestWhenBidIdIsMissing(t *testing.T) {

	// mock cache
	mockCache := &eventsMockCache{
		accounts: &eventsMockAccountService{},
	}

	// mock PBS Analytics Module
	mockAnalyticsModule := &eventsMockAnalyticsModule{
		Fail: false,
	}

	// prepare
	reqData := ""

	req := httptest.NewRequest("GET", "/event?t=win", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := NewEventEndpoint(mockCache, mockAnalyticsModule)

	// execute
	e(recorder, req, nil)

	d, _ := ioutil.ReadAll(recorder.Result().Body)

	// validate
	assert.Equal(t, 400, recorder.Result().StatusCode, "Expected 400 on request with missing bidid parameter")
	assert.Equal(t, "invalid request: parameter 'b' is required\n", string(d))
}

func TestShouldReturnBadRequestWhenTimestampIsInvalid(t *testing.T) {

	// mock cache
	mockCache := &eventsMockCache{
		accounts: &eventsMockAccountService{},
	}

	// mock PBS Analytics Module
	mockAnalyticsModule := &eventsMockAnalyticsModule{
		Fail: false,
	}

	// prepare
	reqData := ""

	req := httptest.NewRequest("GET", "/event?t=win&b=test&ts=q", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := NewEventEndpoint(mockCache, mockAnalyticsModule)

	// execute
	e(recorder, req, nil)

	d, _ := ioutil.ReadAll(recorder.Result().Body)

	// validate
	assert.Equal(t, 400, recorder.Result().StatusCode, "Expected 400 on request with invalid timestamp parameter")
	assert.Equal(t, "invalid request: invalid request: error parsing timestamp 'q'\n", string(d))
}

func TestShouldReturnUnauthorizedWhenAccountIsMissing(t *testing.T) {

	// mock cache
	mockCache := &eventsMockCache{
		accounts: &eventsMockAccountService{},
	}

	// mock PBS Analytics Module
	mockAnalyticsModule := &eventsMockAnalyticsModule{
		Fail: false,
	}

	// prepare
	reqData := ""

	req := httptest.NewRequest("GET", "/event?t=win&b=test&ts=1234", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := NewEventEndpoint(mockCache, mockAnalyticsModule)

	// execute
	e(recorder, req, nil)

	d, _ := ioutil.ReadAll(recorder.Result().Body)

	// validate
	assert.Equal(t, 401, recorder.Result().StatusCode, "Expected 401 on request with missing account id parameter")
	assert.Equal(t, "Account 'a' is required query parameter and can't be empty", string(d))
}

func TestShouldReturnBadRequestWhenFormatValueIsInvalid(t *testing.T) {

	// mock cache
	mockCache := &eventsMockCache{
		accounts: &eventsMockAccountService{},
	}

	// mock PBS Analytics Module
	mockAnalyticsModule := &eventsMockAnalyticsModule{
		Fail: false,
	}

	// prepare
	reqData := ""

	req := httptest.NewRequest("GET", "/event?t=win&b=test&ts=1234&f=q", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := NewEventEndpoint(mockCache, mockAnalyticsModule)

	// execute
	e(recorder, req, nil)

	d, _ := ioutil.ReadAll(recorder.Result().Body)

	// validate
	assert.Equal(t, 400, recorder.Result().StatusCode, "Expected 400 on request with invalid format parameter")
	assert.Equal(t, "invalid request: unknown format: 'q'\n", string(d))
}

func TestShouldReturnBadRequestWhenAnalyticsValueIsInvalid(t *testing.T) {

	// mock cache
	mockCache := &eventsMockCache{
		accounts: &eventsMockAccountService{},
	}

	// mock PBS Analytics Module
	mockAnalyticsModule := &eventsMockAnalyticsModule{
		Fail: false,
	}

	// prepare
	reqData := ""

	req := httptest.NewRequest("GET", "/event?t=win&b=test&ts=1234&f=b&x=4", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := NewEventEndpoint(mockCache, mockAnalyticsModule)

	// execute
	e(recorder, req, nil)

	d, _ := ioutil.ReadAll(recorder.Result().Body)

	// validate
	assert.Equal(t, 400, recorder.Result().StatusCode, "Expected 400 on request with invalid analytics parameter")
	assert.Equal(t, "invalid request: unknown analytics: '4'\n", string(d))
}

func TestShouldNotPassEventToAnalyticsReporterWhenAccountNotFound(t *testing.T) {

	// mock cache
	mockCache := &eventsMockCache{
		accounts: &eventsMockAccountService{
			Fail:  true,
			Error: sql.ErrNoRows,
		},
	}

	// mock PBS Analytics Module
	mockAnalyticsModule := &eventsMockAnalyticsModule{
		Fail: false,
	}

	// prepare
	reqData := ""

	req := httptest.NewRequest("GET", "/event?t=win&b=test&ts=1234&f=b&x=1&a=testacc", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := NewEventEndpoint(mockCache, mockAnalyticsModule)

	// execute
	e(recorder, req, nil)
	d, _ := ioutil.ReadAll(recorder.Result().Body)

	// validate
	assert.Equal(t, 401, recorder.Result().StatusCode, "Expected 401 on account not found")
	assert.Equal(t, "Account 'testacc' doesn't support events", string(d))
}

func TestShouldNotPassEventToAnalyticsReporterWhenAccountEventNotEnabled(t *testing.T) {

	// mock cache
	mockCache := &eventsMockCache{
		accounts: &eventsMockAccountService{
			Fail:          false,
			EventsEnabled: false,
		},
	}

	// mock PBS Analytics Module
	mockAnalyticsModule := &eventsMockAnalyticsModule{
		Fail: false,
	}

	// prepare
	reqData := ""

	req := httptest.NewRequest("GET", "/event?t=win&b=test&ts=1234&f=b&x=1&a=testacc", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := NewEventEndpoint(mockCache, mockAnalyticsModule)

	// execute
	e(recorder, req, nil)
	d, _ := ioutil.ReadAll(recorder.Result().Body)

	// validate
	assert.Equal(t, 401, recorder.Result().StatusCode, "Expected 401 on account with events disabled")
	assert.Equal(t, "Account 'testacc' doesn't support events", string(d))
}

func TestShouldPassEventToAnalyticsReporterWhenAccountEventEnabled(t *testing.T) {

	// mock cache
	mockCache := &eventsMockCache{
		accounts: &eventsMockAccountService{
			Fail:          false,
			EventsEnabled: true,
		},
	}

	// mock PBS Analytics Module
	mockAnalyticsModule := &eventsMockAnalyticsModule{
		Fail: false,
	}

	// prepare
	reqData := ""

	req := httptest.NewRequest("GET", "/event?t=win&b=test&ts=1234&f=b&x=1&a=testacc", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := NewEventEndpoint(mockCache, mockAnalyticsModule)

	// execute
	e(recorder, req, nil)

	// validate
	assert.Equal(t, 200, recorder.Result().StatusCode, "Expected 200 when account has events enabled")
	assert.Equal(t, true, mockAnalyticsModule.Invoked)
}

func TestShouldNotPassEventToAnalyticsReporterWhenAnalyticsValueIsZero(t *testing.T) {

	// mock cache
	mockCache := &eventsMockCache{
		accounts: &eventsMockAccountService{
			Fail:          false,
			EventsEnabled: true,
		},
	}

	// mock PBS Analytics Module
	mockAnalyticsModule := &eventsMockAnalyticsModule{
		Fail: false,
	}

	// prepare
	reqData := ""

	req := httptest.NewRequest("GET", "/event?t=win&b=test&ts=1234&f=b&x=0&a=testacc", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := NewEventEndpoint(mockCache, mockAnalyticsModule)

	// execute
	e(recorder, req, nil)

	// validate
	assert.Equal(t, 200, recorder.Result().StatusCode)
	assert.Equal(t, true, mockAnalyticsModule.Invoked != true)
}

func TestShouldRespondWithPixelAndContentTypeWhenRequestFormatIsImage(t *testing.T) {

	// mock cache
	mockCache := &eventsMockCache{
		accounts: &eventsMockAccountService{
			Fail:          false,
			EventsEnabled: true,
		},
	}

	// mock PBS Analytics Module
	mockAnalyticsModule := &eventsMockAnalyticsModule{
		Fail: false,
	}

	// prepare
	reqData := ""

	req := httptest.NewRequest("GET", "/event?t=win&b=test&ts=1234&f=i&x=1&a=testacc", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := NewEventEndpoint(mockCache, mockAnalyticsModule)

	// execute
	e(recorder, req, nil)

	d, _ := ioutil.ReadAll(recorder.Result().Body)

	// validate
	assert.Equal(t, 200, recorder.Result().StatusCode, "Expected 200 with tracking pixel when format is imp")
	assert.Equal(t, true, mockAnalyticsModule.Invoked)
	assert.Equal(t, "image/png", recorder.Header().Get("Content-Type"))
	assert.Equal(t, "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAABHNCSVQICAgIfAhkiAAAAA1JREFUCJljYGBgYAAAAAUAAYehTtQAAAAASUVORK5CYII=", base64.URLEncoding.EncodeToString(d))
}

func TestShouldRespondWithNoContentWhenRequestFormatIsNotDefined(t *testing.T) {

	// mock cache
	mockCache := &eventsMockCache{
		accounts: &eventsMockAccountService{
			Fail:          false,
			EventsEnabled: true,
		},
	}

	// mock PBS Analytics Module
	mockAnalyticsModule := &eventsMockAnalyticsModule{
		Fail: false,
	}

	// prepare
	reqData := ""

	req := httptest.NewRequest("GET", "/event?t=imp&b=test&ts=1234&x=1&a=testacc", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := NewEventEndpoint(mockCache, mockAnalyticsModule)

	// execute
	e(recorder, req, nil)

	d, _ := ioutil.ReadAll(recorder.Result().Body)

	// validate
	assert.Equal(t, 200, recorder.Result().StatusCode, "Expected 200 with empty response")
	assert.Equal(t, true, mockAnalyticsModule.Invoked)
	assert.Equal(t, "", recorder.Header().Get("Content-Type"))
	assert.Equal(t, 0, len(d))
}

func TestShouldParseEventCorrectly(t *testing.T) {

	req := httptest.NewRequest("GET", "/event?t=win&b=bidId&f=b&ts=1000&x=1&a=accountId&bidder=bidder", strings.NewReader(""))

	expected := &EventRequest{
		Type:      WIN,
		Bidid:     "bidId",
		Timestamp: 1000,
		Bidder:    "bidder",
		AccountId: "",
		Format:    BLANK,
		Analytics: ENABLED,
	}

	// execute
	er, err := ParseEventRequest(req)

	// validate
	assert.Equal(t, nil, err)
	assert.EqualValues(t, expected, er)
}
