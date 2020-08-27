package endpoints

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/cache"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/prebid_cache_client"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"
)

const maxSize = 1024 * 256

// Mock pbs cache client
type mockCacheClient struct {
	Fail  bool
	Error error
	Uuids []string
}

func (m *mockCacheClient) PutJson(ctx context.Context, values []prebid_cache_client.Cacheable) ([]string, []error) {
	if m.Fail {
		return []string{}, []error{m.Error}
	}
	return m.Uuids, []error{}
}
func (m *mockCacheClient) GetExtCacheData() (string, string) {
	return "", ""
}

// Mock cache
type MockCache struct {
	accounts *mockAccountService
}

func (c *MockCache) Accounts() cache.AccountsService {
	return c.accounts
}
func (c *MockCache) Config() cache.ConfigService {
	return nil
}
func (c *MockCache) Close() error {
	return nil
}

type mockAccountService struct {
	Fail  bool
	Error error
}

func (s *mockAccountService) Get(id string) (*cache.Account, error) {
	if s.Fail {
		return nil, s.Error
	}

	return &cache.Account{
		ID:            id,
		EventsEnabled: true,
	}, nil
}
func (s *mockAccountService) Set(account *cache.Account) error {
	return nil
}

// Test

func TestShouldRespondWithBadRequestWhenAccountParameterIsMissing(t *testing.T) {
	// mock pbs cache client
	mockCacheClient := &mockCacheClient{}

	// mock cache
	mockCache := &MockCache{
		accounts: &mockAccountService{},
	}

	// prepare
	reqData := ""

	req := httptest.NewRequest("POST", "/vtrack", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := vtrackEndpoint{
		Cfg:         nil,
		BidderInfos: nil,
		PbsCache:    mockCacheClient,
		DataCache:   mockCache,
	}

	// execute
	e.Handle(recorder, req, nil)

	d, _ := ioutil.ReadAll(recorder.Result().Body)

	// validate
	assert.Equal(t, 400, recorder.Result().StatusCode, "Expected 400 on request with missing account parameter")
	assert.Equal(t, "Account 'a' is required query parameter and can't be empty", string(d))
}

func TestShouldRespondWithBadRequestWhenRequestBodyIsEmpty(t *testing.T) {
	// mock pbs cache client
	mockCacheClient := &mockCacheClient{}

	// mock cache
	mockCache := &MockCache{
		accounts: &mockAccountService{},
	}

	// config
	cfg := &config.Configuration{MaxRequestSize: maxSize}

	// prepare
	reqData := ""

	req := httptest.NewRequest("POST", "/vtrack?a=1235", strings.NewReader(reqData))

	recorder := httptest.NewRecorder()

	e := vtrackEndpoint{
		Cfg:         cfg,
		BidderInfos: nil,
		PbsCache:    mockCacheClient,
		DataCache:   mockCache,
	}

	// execute
	e.Handle(recorder, req, nil)

	d, _ := ioutil.ReadAll(recorder.Result().Body)

	// validate
	assert.Equal(t, 400, recorder.Result().StatusCode, "Expected 400 on request with empty body")
	assert.Equal(t, "Invalid request: request body is empty\n", string(d))
}

func TestShouldRespondWithBadRequestWhenRequestBodyIsInvalid(t *testing.T) {
	// mock pbs cache client
	mockCacheClient := &mockCacheClient{}

	// mock cache
	mockCache := &MockCache{
		accounts: &mockAccountService{},
	}

	// config
	cfg := &config.Configuration{MaxRequestSize: maxSize}

	// prepare
	reqData := "invalid"

	req := httptest.NewRequest("POST", "/vtrack?a=1235", strings.NewReader(reqData))

	recorder := httptest.NewRecorder()

	e := vtrackEndpoint{
		Cfg:         cfg,
		BidderInfos: nil,
		PbsCache:    mockCacheClient,
		DataCache:   mockCache,
	}

	// execute
	e.Handle(recorder, req, nil)

	// validate
	assert.Equal(t, 400, recorder.Result().StatusCode, "Expected 400 on request with invalid body")
}

func TestShouldRespondWithBadRequestWhenBidIdIsMissing(t *testing.T) {
	// mock pbs cache client
	mockCacheClient := &mockCacheClient{}

	// mock cache
	mockCache := &MockCache{
		accounts: &mockAccountService{},
	}

	// config
	cfg := &config.Configuration{MaxRequestSize: maxSize}

	// prepare
	data := &BidCacheRequest{
		Puts: []prebid_cache_client.Cacheable{
			{},
		},
	}

	reqData, _ := json.Marshal(data)

	req := httptest.NewRequest("POST", "/vtrack?a=1235", strings.NewReader(string(reqData)))

	recorder := httptest.NewRecorder()

	e := vtrackEndpoint{
		Cfg:         cfg,
		BidderInfos: nil,
		PbsCache:    mockCacheClient,
		DataCache:   mockCache,
	}

	// execute
	e.Handle(recorder, req, nil)

	d, _ := ioutil.ReadAll(recorder.Result().Body)

	// validate
	assert.Equal(t, 400, recorder.Result().StatusCode, "Expected 400 on request with elements missing bidid")
	assert.Equal(t, "Invalid request: 'bidid' is required field and can't be empty\n", string(d))
}

func TestShouldRespondWithBadRequestWhenBidderIsMissing(t *testing.T) {
	// mock pbs cache client
	mockCacheClient := &mockCacheClient{}

	// mock cache
	mockCache := &MockCache{
		accounts: &mockAccountService{},
	}

	// config
	cfg := &config.Configuration{MaxRequestSize: maxSize}

	// prepare
	data := &BidCacheRequest{
		Puts: []prebid_cache_client.Cacheable{
			{
				BidID: "test",
			},
		},
	}

	reqData, _ := json.Marshal(data)

	req := httptest.NewRequest("POST", "/vtrack?a=1235", strings.NewReader(string(reqData)))

	recorder := httptest.NewRecorder()

	e := vtrackEndpoint{
		Cfg:         cfg,
		BidderInfos: nil,
		PbsCache:    mockCacheClient,
		DataCache:   mockCache,
	}

	// execute
	e.Handle(recorder, req, nil)

	d, _ := ioutil.ReadAll(recorder.Result().Body)

	// validate
	assert.Equal(t, 400, recorder.Result().StatusCode, "Expected 400 on request with elements missing bidder")
	assert.Equal(t, "Invalid request: 'bidder' is required field and can't be empty\n", string(d))
}

func TestShouldRespondWithInternalServerErrorWhenFetchingAccountFails(t *testing.T) {
	// mock pbs cache client
	mockCacheClient := &mockCacheClient{}

	// mock cache
	mockCache := &MockCache{
		accounts: &mockAccountService{
			Fail:  true,
			Error: fmt.Errorf("failed retrieving account details"),
		},
	}

	// config
	cfg := &config.Configuration{MaxRequestSize: maxSize}

	// prepare
	data := &BidCacheRequest{
		Puts: []prebid_cache_client.Cacheable{
			{
				BidID:  "test",
				Bidder: "test",
			},
		},
	}

	reqData, _ := json.Marshal(data)

	req := httptest.NewRequest("POST", "/vtrack?a=1235", strings.NewReader(string(reqData)))

	recorder := httptest.NewRecorder()

	e := vtrackEndpoint{
		Cfg:         cfg,
		BidderInfos: nil,
		PbsCache:    mockCacheClient,
		DataCache:   mockCache,
	}

	// execute
	e.Handle(recorder, req, nil)

	d, _ := ioutil.ReadAll(recorder.Result().Body)

	// validate
	assert.Equal(t, 500, recorder.Result().StatusCode, "Expected 500 when failing to retrieve account details")
	assert.Equal(t, "Invalid request: failed retrieving account details\n", string(d))
}

func TestShouldRespondWithInternalServerErrorWhenPbsCacheClientFails(t *testing.T) {
	// mock pbs cache client
	mockCacheClient := &mockCacheClient{
		Fail:  true,
		Error: fmt.Errorf("pbs cache client failed"),
	}

	// mock cache
	mockCache := &MockCache{
		accounts: &mockAccountService{},
	}

	// config
	cfg := &config.Configuration{
		MaxRequestSize: maxSize, VTrack: config.VTrack{
			TimeoutMs: int64(2000), AllowUnknownBidder: true,
		},
	}

	// prepare
	data := &BidCacheRequest{
		Puts: []prebid_cache_client.Cacheable{
			{
				BidID:  "test",
				Bidder: "test",
			},
		},
	}

	reqData, _ := json.Marshal(data)

	req := httptest.NewRequest("POST", "/vtrack?a=1235", strings.NewReader(string(reqData)))

	recorder := httptest.NewRecorder()

	e := vtrackEndpoint{
		Cfg:         cfg,
		BidderInfos: nil,
		PbsCache:    mockCacheClient,
		DataCache:   mockCache,
	}

	// execute
	e.Handle(recorder, req, nil)

	d, _ := ioutil.ReadAll(recorder.Result().Body)

	// validate
	assert.Equal(t, 500, recorder.Result().StatusCode, "Expected 500 when pbs cache client fails")
	assert.Equal(t, "Error(s) updating vast: pbs cache client failed\n", string(d))
}

func TestShouldTolerateAccountNotFound(t *testing.T) {
	// mock pbs cache client
	mockCacheClient := &mockCacheClient{}

	// mock cache
	mockCache := &MockCache{
		accounts: &mockAccountService{
			Fail:  true,
			Error: sql.ErrNoRows,
		},
	}

	// config
	cfg := &config.Configuration{
		MaxRequestSize: maxSize, VTrack: config.VTrack{
			TimeoutMs: int64(2000), AllowUnknownBidder: false,
		},
	}

	// prepare
	data := &BidCacheRequest{
		Puts: []prebid_cache_client.Cacheable{
			{
				BidID:  "test",
				Bidder: "test",
			},
		},
	}

	reqData, _ := json.Marshal(data)
	req := httptest.NewRequest("POST", "/vtrack?a=1235", strings.NewReader(string(reqData)))

	recorder := httptest.NewRecorder()

	e := vtrackEndpoint{
		Cfg:         cfg,
		BidderInfos: nil,
		PbsCache:    mockCacheClient,
		DataCache:   mockCache,
	}

	// execute
	e.Handle(recorder, req, nil)

	// validate
	assert.Equal(t, 200, recorder.Result().StatusCode, "Expected 200 when account is not found and request is valid")
}

func TestShouldSendToCacheExpectedPutsAndUpdatableBiddersWhenBidderVastNotAllowed(t *testing.T) {
	// mock pbs cache client
	mockCacheClient := &mockCacheClient{
		Fail:  false,
		Uuids: []string{"uuid1"},
	}

	// mock cache
	mockCache := &MockCache{
		accounts: &mockAccountService{
			Fail: false,
		},
	}

	// config
	cfg := &config.Configuration{
		MaxRequestSize: maxSize, VTrack: config.VTrack{
			TimeoutMs: int64(2000), AllowUnknownBidder: false,
		},
	}

	// bidder info
	bidderInfos := make(adapters.BidderInfos)
	bidderInfos["bidder"] = adapters.BidderInfo{
		Status:                  adapters.StatusActive,
		ModifyingVastXmlAllowed: false,
	}
	bidderInfos["updatable_bidder"] = adapters.BidderInfo{
		Status:                  adapters.StatusActive,
		ModifyingVastXmlAllowed: true,
	}

	// prepare
	data := &BidCacheRequest{
		Puts: []prebid_cache_client.Cacheable{
			{
				BidID:  "bidId1",
				Bidder: "bidder",
				Data:   []byte("{\"key\":\"val\"}"),
			},
			{
				BidID:  "bidId2",
				Bidder: "updatable_bidder",
				Data:   []byte("{\"key\":\"val\"}"),
			},
		},
	}

	reqData, _ := json.Marshal(data)
	req := httptest.NewRequest("POST", "/vtrack?a=1235", strings.NewReader(string(reqData)))

	recorder := httptest.NewRecorder()

	e := vtrackEndpoint{
		Cfg:         cfg,
		BidderInfos: bidderInfos,
		PbsCache:    mockCacheClient,
		DataCache:   mockCache,
	}

	// execute
	e.Handle(recorder, req, nil)

	d, _ := ioutil.ReadAll(recorder.Result().Body)

	// validate
	assert.Equal(t, 200, recorder.Result().StatusCode, "Expected 200 when account is not found and request is valid")
	assert.Equal(t, "{\"responses\":[{\"uuid\":\"uuid1\"}]}", string(d), "Expected 200 when account is found and request is valid")
}

func TestShouldSendToCacheExpectedPutsAndUpdatableBiddersWhenBidderVastAllowed(t *testing.T) {
	// mock pbs cache client
	mockCacheClient := &mockCacheClient{
		Fail:  false,
		Uuids: []string{"uuid1", "uuid2"},
	}

	// mock cache
	mockCache := &MockCache{
		accounts: &mockAccountService{
			Fail: false,
		},
	}

	// config
	cfg := &config.Configuration{
		MaxRequestSize: maxSize, VTrack: config.VTrack{
			TimeoutMs: int64(2000), AllowUnknownBidder: false,
		},
	}

	// bidder info
	bidderInfos := make(adapters.BidderInfos)
	bidderInfos["bidder"] = adapters.BidderInfo{
		Status:                  adapters.StatusActive,
		ModifyingVastXmlAllowed: true,
	}
	bidderInfos["updatable_bidder"] = adapters.BidderInfo{
		Status:                  adapters.StatusActive,
		ModifyingVastXmlAllowed: true,
	}

	// prepare
	data := &BidCacheRequest{
		Puts: []prebid_cache_client.Cacheable{
			{
				BidID:  "bidId1",
				Bidder: "bidder",
				Data:   []byte("{\"key\":\"val\"}"),
			},
			{
				BidID:  "bidId2",
				Bidder: "updatable_bidder",
				Data:   []byte("{\"key\":\"val\"}"),
			},
		},
	}

	reqData, _ := json.Marshal(data)
	req := httptest.NewRequest("POST", "/vtrack?a=1235", strings.NewReader(string(reqData)))

	recorder := httptest.NewRecorder()

	e := vtrackEndpoint{
		Cfg:         cfg,
		BidderInfos: bidderInfos,
		PbsCache:    mockCacheClient,
		DataCache:   mockCache,
	}

	// execute
	e.Handle(recorder, req, nil)

	d, _ := ioutil.ReadAll(recorder.Result().Body)

	// validate
	assert.Equal(t, 200, recorder.Result().StatusCode, "Expected 200 when account is not found and request is valid")
	assert.Equal(t, "{\"responses\":[{\"uuid\":\"uuid1\"},{\"uuid\":\"uuid2\"}]}", string(d), "Expected 200 when account is found and request is valid")
}

func TestShouldSendToCacheExpectedPutsAndUpdatableUnknownBiddersWhenUnknownBidderIsAllowed(t *testing.T) {
	// mock pbs cache client
	mockCacheClient := &mockCacheClient{
		Fail:  false,
		Uuids: []string{"uuid1", "uuid2"},
	}

	// mock cache
	mockCache := &MockCache{
		accounts: &mockAccountService{
			Fail: false,
		},
	}

	// config
	cfg := &config.Configuration{
		MaxRequestSize: maxSize, VTrack: config.VTrack{
			TimeoutMs: int64(2000), AllowUnknownBidder: true,
		},
	}

	// bidder info
	bidderInfos := make(adapters.BidderInfos)

	// prepare
	data := &BidCacheRequest{
		Puts: []prebid_cache_client.Cacheable{
			{
				BidID:  "bidId1",
				Bidder: "bidder",
				Data:   []byte("{\"key\":\"val\"}"),
			},
			{
				BidID:  "bidId2",
				Bidder: "updatable_bidder",
				Data:   []byte("{\"key\":\"val\"}"),
			},
		},
	}

	reqData, _ := json.Marshal(data)
	req := httptest.NewRequest("POST", "/vtrack?a=1235", strings.NewReader(string(reqData)))

	recorder := httptest.NewRecorder()

	e := vtrackEndpoint{
		Cfg:         cfg,
		BidderInfos: bidderInfos,
		PbsCache:    mockCacheClient,
		DataCache:   mockCache,
	}

	// execute
	e.Handle(recorder, req, nil)

	d, _ := ioutil.ReadAll(recorder.Result().Body)

	// validate
	assert.Equal(t, 200, recorder.Result().StatusCode, "Expected 200 when account is not found and request is valid")
	assert.Equal(t, "{\"responses\":[{\"uuid\":\"uuid1\"},{\"uuid\":\"uuid2\"}]}", string(d), "Expected 200 when account is found, request has unknown bidders but allowUnknownBidders is enabled")
}
