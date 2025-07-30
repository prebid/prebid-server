package events

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/prebid_cache_client"
	"github.com/prebid/prebid-server/v3/stored_requests"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/stretchr/testify/assert"
)

const (
	maxSize = 1024 * 256

	vastXmlWithImpressionWithContent    = "<VAST version=\"3.0\"><Ad><Wrapper><AdSystem>prebid.org wrapper</AdSystem><VASTAdTagURI><![CDATA[adm2]]></VASTAdTagURI><Impression>content</Impression><Creatives></Creatives></Wrapper></Ad></VAST>"
	vastXmlWithImpressionWithoutContent = "<VAST version=\"3.0\"><Ad><Wrapper><AdSystem>prebid.org wrapper</AdSystem><VASTAdTagURI><![CDATA[adm2]]></VASTAdTagURI><Impression></Impression><Creatives></Creatives></Wrapper></Ad></VAST>"
	vastXmlWithoutImpression            = "<VAST version=\"3.0\"><Ad><Wrapper><AdSystem>prebid.org wrapper</AdSystem><VASTAdTagURI><![CDATA[adm2]]></VASTAdTagURI><Creatives></Creatives></Wrapper></Ad></VAST>"
)

// Mock pbs cache client
type vtrackMockCacheClient struct {
	Fail   bool
	Error  error
	Uuids  []string
	Values []prebid_cache_client.Cacheable
}

func (m *vtrackMockCacheClient) PutJson(ctx context.Context, values []prebid_cache_client.Cacheable) ([]string, []error) {
	if m.Fail {
		return []string{}, []error{m.Error}
	}
	m.Values = values
	return m.Uuids, []error{}
}
func (m *vtrackMockCacheClient) GetExtCacheData() (scheme string, host string, path string) {
	return
}

// Test
func TestShouldRespondWithBadRequestWhenAccountParameterIsMissing(t *testing.T) {
	// mock pbs cache client
	mockCacheClient := &vtrackMockCacheClient{}

	// mock AccountsFetcher
	mockAccountsFetcher := &mockAccountsFetcher{}

	// mock config
	cfg := &config.Configuration{
		AccountDefaults: config.Account{},
	}
	cfg.MarshalAccountDefaults()

	// prepare
	reqData := ""

	req := httptest.NewRequest("POST", "/vtrack", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	e := vtrackEndpoint{
		Cfg:                 cfg,
		BidderInfos:         nil,
		Cache:               mockCacheClient,
		Accounts:            mockAccountsFetcher,
		normalizeBidderName: openrtb_ext.NormalizeBidderName,
	}

	// execute
	e.Handle(recorder, req, nil)

	d, err := io.ReadAll(recorder.Result().Body)
	if err != nil {
		t.Fatal(err)
	}

	// validate
	assert.Equal(t, 400, recorder.Result().StatusCode, "Expected 400 on request with missing account parameter")
	assert.Equal(t, "Account 'a' is required query parameter and can't be empty", string(d))
}

func TestShouldRespondWithBadRequestWhenRequestBodyIsEmpty(t *testing.T) {
	// mock pbs cache client
	mockCacheClient := &vtrackMockCacheClient{}

	// mock AccountsFetcher
	mockAccountsFetcher := &mockAccountsFetcher{}

	// config
	cfg := &config.Configuration{
		MaxRequestSize:  maxSize,
		AccountDefaults: config.Account{},
	}
	cfg.MarshalAccountDefaults()

	// prepare
	reqData := ""

	req := httptest.NewRequest("POST", "/vtrack?a=events_enabled", strings.NewReader(reqData))

	recorder := httptest.NewRecorder()

	e := vtrackEndpoint{
		Cfg:                 cfg,
		BidderInfos:         nil,
		Cache:               mockCacheClient,
		Accounts:            mockAccountsFetcher,
		normalizeBidderName: openrtb_ext.NormalizeBidderName,
	}

	// execute
	e.Handle(recorder, req, nil)

	d, err := io.ReadAll(recorder.Result().Body)
	if err != nil {
		t.Fatal(err)
	}

	// validate
	assert.Equal(t, 400, recorder.Result().StatusCode, "Expected 400 on request with empty body")
	assert.Equal(t, "Invalid request: request body is empty\n", string(d))
}

func TestShouldRespondWithBadRequestWhenRequestBodyIsInvalid(t *testing.T) {
	// mock pbs cache client
	mockCacheClient := &vtrackMockCacheClient{}

	// mock AccountsFetcher
	mockAccountsFetcher := &mockAccountsFetcher{}

	// config
	cfg := &config.Configuration{
		MaxRequestSize:  maxSize,
		AccountDefaults: config.Account{},
	}
	cfg.MarshalAccountDefaults()

	// prepare
	reqData := "invalid"

	req := httptest.NewRequest("POST", "/vtrack?a=events_enabled", strings.NewReader(reqData))

	recorder := httptest.NewRecorder()

	e := vtrackEndpoint{
		Cfg:                 cfg,
		BidderInfos:         nil,
		Cache:               mockCacheClient,
		Accounts:            mockAccountsFetcher,
		normalizeBidderName: openrtb_ext.NormalizeBidderName,
	}

	// execute
	e.Handle(recorder, req, nil)

	// validate
	assert.Equal(t, 400, recorder.Result().StatusCode, "Expected 400 on request with invalid body")
}

func TestShouldRespondWithBadRequestWhenBidIdIsMissing(t *testing.T) {
	// mock pbs cache client
	mockCacheClient := &vtrackMockCacheClient{}

	// mock AccountsFetcher
	mockAccountsFetcher := &mockAccountsFetcher{}

	// config
	cfg := &config.Configuration{
		MaxRequestSize:  maxSize,
		AccountDefaults: config.Account{},
	}
	cfg.MarshalAccountDefaults()

	// prepare
	data := &BidCacheRequest{
		Puts: []prebid_cache_client.Cacheable{
			{},
		},
	}

	reqData, err := jsonutil.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/vtrack?a=events_enabled", strings.NewReader(string(reqData)))

	recorder := httptest.NewRecorder()

	e := vtrackEndpoint{
		Cfg:                 cfg,
		BidderInfos:         nil,
		Cache:               mockCacheClient,
		Accounts:            mockAccountsFetcher,
		normalizeBidderName: openrtb_ext.NormalizeBidderName,
	}

	// execute
	e.Handle(recorder, req, nil)

	d, err := io.ReadAll(recorder.Result().Body)
	if err != nil {
		t.Fatal(err)
	}

	// validate
	assert.Equal(t, 400, recorder.Result().StatusCode, "Expected 400 on request with elements missing bidid")
	assert.Equal(t, "Invalid request: 'bidid' is required field and can't be empty\n", string(d))
}

func TestShouldRespondWithBadRequestWhenBidderIsMissing(t *testing.T) {
	// mock pbs cache client
	mockCacheClient := &vtrackMockCacheClient{}

	// mock AccountsFetcher
	mockAccountsFetcher := &mockAccountsFetcher{}

	// config
	cfg := &config.Configuration{
		MaxRequestSize:  maxSize,
		AccountDefaults: config.Account{},
	}
	cfg.MarshalAccountDefaults()

	// prepare
	data := &BidCacheRequest{
		Puts: []prebid_cache_client.Cacheable{
			{
				BidID: "test",
			},
		},
	}

	reqData, err := jsonutil.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/vtrack?a=events_enabled", strings.NewReader(string(reqData)))

	recorder := httptest.NewRecorder()

	e := vtrackEndpoint{
		Cfg:                 cfg,
		BidderInfos:         nil,
		Cache:               mockCacheClient,
		Accounts:            mockAccountsFetcher,
		normalizeBidderName: openrtb_ext.NormalizeBidderName,
	}

	// execute
	e.Handle(recorder, req, nil)

	d, err := io.ReadAll(recorder.Result().Body)
	if err != nil {
		t.Fatal(err)
	}

	// validate
	assert.Equal(t, 400, recorder.Result().StatusCode, "Expected 400 on request with elements missing bidder")
	assert.Equal(t, "Invalid request: 'bidder' is required field and can't be empty\n", string(d))
}

func TestShouldRespondWithInternalServerErrorWhenPbsCacheClientFails(t *testing.T) {
	// mock pbs cache client
	mockCacheClient := &vtrackMockCacheClient{
		Fail:  true,
		Error: fmt.Errorf("pbs cache client failed"),
	}

	// mock AccountsFetcher
	mockAccountsFetcher := &mockAccountsFetcher{}

	// config
	cfg := &config.Configuration{
		MaxRequestSize: maxSize, VTrack: config.VTrack{
			TimeoutMS: int64(2000), AllowUnknownBidder: true,
		},
		AccountDefaults: config.Account{},
	}
	cfg.MarshalAccountDefaults()

	// prepare
	data, err := getValidVTrackRequestBody(false, false)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/vtrack?a=events_enabled", strings.NewReader(data))

	recorder := httptest.NewRecorder()

	e := vtrackEndpoint{
		Cfg:                 cfg,
		BidderInfos:         nil,
		Cache:               mockCacheClient,
		Accounts:            mockAccountsFetcher,
		normalizeBidderName: openrtb_ext.NormalizeBidderName,
	}

	// execute
	e.Handle(recorder, req, nil)

	d, err := io.ReadAll(recorder.Result().Body)
	if err != nil {
		t.Fatal(err)
	}

	// validate
	assert.Equal(t, 500, recorder.Result().StatusCode, "Expected 500 when pbs cache client fails")
	assert.Equal(t, "Error(s) updating vast: pbs cache client failed\n", string(d))
}

func TestShouldTolerateAccountNotFound(t *testing.T) {
	// mock pbs cache client
	mockCacheClient := &vtrackMockCacheClient{}

	// mock AccountsFetcher
	mockAccountsFetcher := &mockAccountsFetcher{
		Fail:  true,
		Error: stored_requests.NotFoundError{},
	}

	// config
	cfg := &config.Configuration{
		MaxRequestSize: maxSize, VTrack: config.VTrack{
			TimeoutMS: int64(2000), AllowUnknownBidder: false,
		},
		AccountDefaults: config.Account{},
	}
	cfg.MarshalAccountDefaults()

	// prepare
	data, err := getValidVTrackRequestBody(true, false)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/vtrack?a=1235", strings.NewReader(data))

	recorder := httptest.NewRecorder()

	e := vtrackEndpoint{
		Cfg:                 cfg,
		BidderInfos:         nil,
		Cache:               mockCacheClient,
		Accounts:            mockAccountsFetcher,
		normalizeBidderName: openrtb_ext.NormalizeBidderName,
	}

	// execute
	e.Handle(recorder, req, nil)

	// validate
	assert.Equal(t, 200, recorder.Result().StatusCode, "Expected 200 when account is not found and request is valid")
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
}

func TestShouldSendToCacheExpectedPutsAndUpdatableBiddersWhenBidderVastNotAllowed(t *testing.T) {
	// mock pbs cache client
	mockCacheClient := &vtrackMockCacheClient{
		Fail:  false,
		Uuids: []string{"uuid1"},
	}

	// mock AccountsFetcher
	mockAccountsFetcher := &mockAccountsFetcher{
		Fail: false,
	}

	// config
	cfg := &config.Configuration{
		MaxRequestSize: maxSize, VTrack: config.VTrack{
			TimeoutMS: int64(2000), AllowUnknownBidder: false,
		},
		AccountDefaults: config.Account{},
	}
	cfg.MarshalAccountDefaults()

	// bidder info
	bidderInfos := make(config.BidderInfos)
	bidderInfos["bidder"] = config.BidderInfo{
		Disabled:                false,
		ModifyingVastXmlAllowed: false,
	}
	bidderInfos["updatable_bidder"] = config.BidderInfo{
		Disabled:                false,
		ModifyingVastXmlAllowed: true,
	}

	// prepare
	data, err := getValidVTrackRequestBody(false, false)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/vtrack?a=events_enabled", strings.NewReader(data))

	recorder := httptest.NewRecorder()

	e := vtrackEndpoint{
		Cfg:                 cfg,
		BidderInfos:         bidderInfos,
		Cache:               mockCacheClient,
		Accounts:            mockAccountsFetcher,
		normalizeBidderName: openrtb_ext.NormalizeBidderName,
	}

	// execute
	e.Handle(recorder, req, nil)

	d, err := io.ReadAll(recorder.Result().Body)
	if err != nil {
		t.Fatal(err)
	}

	// validate
	assert.Equal(t, 200, recorder.Result().StatusCode, "Expected 200 when account is not found and request is valid")
	assert.Equal(t, "{\"responses\":[{\"uuid\":\"uuid1\"}]}", string(d), "Expected 200 when account is found and request is valid")
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
}

func TestShouldSendToCacheExpectedPutsAndUpdatableBiddersWhenBidderVastAllowed(t *testing.T) {
	// mock pbs cache client
	mockCacheClient := &vtrackMockCacheClient{
		Fail:  false,
		Uuids: []string{"uuid1", "uuid2"},
	}

	// mock AccountsFetcher
	mockAccountsFetcher := &mockAccountsFetcher{
		Fail: false,
	}

	// config
	cfg := &config.Configuration{
		MaxRequestSize: maxSize, VTrack: config.VTrack{
			TimeoutMS: int64(2000), AllowUnknownBidder: false,
		},
		AccountDefaults: config.Account{},
	}
	cfg.MarshalAccountDefaults()

	// bidder info
	bidderInfos := make(config.BidderInfos)
	bidderInfos["bidder"] = config.BidderInfo{
		Disabled:                false,
		ModifyingVastXmlAllowed: true,
	}
	bidderInfos["updatable_bidder"] = config.BidderInfo{
		Disabled:                false,
		ModifyingVastXmlAllowed: true,
	}

	// prepare
	data, err := getValidVTrackRequestBody(true, true)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/vtrack?a=events_enabled", strings.NewReader(data))

	recorder := httptest.NewRecorder()

	var mockNormalizeBidderName openrtb_ext.BidderNameNormalizer = func(name string) (openrtb_ext.BidderName, bool) {
		return openrtb_ext.BidderName(name), true
	}
	e := vtrackEndpoint{
		Cfg:                 cfg,
		BidderInfos:         bidderInfos,
		Cache:               mockCacheClient,
		Accounts:            mockAccountsFetcher,
		normalizeBidderName: mockNormalizeBidderName,
	}

	// execute
	e.Handle(recorder, req, nil)

	d, err := io.ReadAll(recorder.Result().Body)
	if err != nil {
		t.Fatal(err)
	}

	// validate
	assert.Equal(t, 200, recorder.Result().StatusCode, "Expected 200 when account is not found and request is valid")
	assert.Equal(t, "{\"responses\":[{\"uuid\":\"uuid1\"},{\"uuid\":\"uuid2\"}]}", string(d), "Expected 200 when account is found and request is valid")
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	assert.Len(t, mockCacheClient.Values, 2)
	assert.Contains(t, string(mockCacheClient.Values[0].Data), "bidder=bidder")
	assert.Contains(t, string(mockCacheClient.Values[1].Data), "bidder=updatable_bidder")
}

func TestShouldSendToCacheExpectedPutsAndUpdatableCaseSensitiveBiddersWhenBidderVastAllowed(t *testing.T) {
	// mock pbs cache client
	mockCacheClient := &vtrackMockCacheClient{
		Fail:  false,
		Uuids: []string{"uuid1", "uuid2"},
	}

	// mock AccountsFetcher
	mockAccountsFetcher := &mockAccountsFetcher{
		Fail: false,
	}

	// config
	cfg := &config.Configuration{
		MaxRequestSize: maxSize, VTrack: config.VTrack{
			TimeoutMS: int64(2000), AllowUnknownBidder: false,
		},
		AccountDefaults: config.Account{},
	}
	cfg.MarshalAccountDefaults()

	// bidder info
	bidderInfos := make(config.BidderInfos)
	bidderInfos["appnexus"] = config.BidderInfo{
		Disabled:                false,
		ModifyingVastXmlAllowed: true,
	}

	d, err := getVTrackRequestData(true, true)
	assert.NoError(t, err)

	cacheReq := &BidCacheRequest{
		Puts: []prebid_cache_client.Cacheable{
			{
				Type:       prebid_cache_client.TypeXML,
				BidID:      "bidId1",
				Bidder:     "APPNEXUS", // case sensitive name
				Data:       d,
				TTLSeconds: 3600,
				Timestamp:  1000,
			},
			{
				Type:       prebid_cache_client.TypeXML,
				BidID:      "bidId2",
				Bidder:     "ApPnExUs", // case sensitive name
				Data:       d,
				TTLSeconds: 3600,
				Timestamp:  1000,
			},
		},
	}
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	err = enc.Encode(cacheReq)
	assert.NoError(t, err)
	data := buf.String()

	req := httptest.NewRequest("POST", "/vtrack?a=events_enabled", strings.NewReader(data))

	recorder := httptest.NewRecorder()
	e := vtrackEndpoint{
		Cfg:                 cfg,
		BidderInfos:         bidderInfos,
		Cache:               mockCacheClient,
		Accounts:            mockAccountsFetcher,
		normalizeBidderName: openrtb_ext.NormalizeBidderName,
	}

	// execute
	e.Handle(recorder, req, nil)

	d, err = io.ReadAll(recorder.Result().Body)
	if err != nil {
		t.Fatal(err)
	}

	// validate
	assert.Equal(t, 200, recorder.Result().StatusCode, "Expected 200 when account is not found and request is valid")
	assert.Equal(t, "{\"responses\":[{\"uuid\":\"uuid1\"},{\"uuid\":\"uuid2\"}]}", string(d), "Expected 200 when account is found and request is valid")
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	assert.Len(t, mockCacheClient.Values, 2)
	assert.Contains(t, string(mockCacheClient.Values[0].Data), "bidder=APPNEXUS")
	assert.Contains(t, string(mockCacheClient.Values[1].Data), "bidder=ApPnExUs")
}

func TestShouldSendToCacheExpectedPutsAndUpdatableUnknownBiddersWhenUnknownBidderIsAllowed(t *testing.T) {
	// mock pbs cache client
	mockCacheClient := &vtrackMockCacheClient{
		Fail:  false,
		Uuids: []string{"uuid1", "uuid2"},
	}

	// mock AccountsFetcher
	mockAccountsFetcher := &mockAccountsFetcher{
		Fail: false,
	}

	// config
	cfg := &config.Configuration{
		MaxRequestSize: maxSize, VTrack: config.VTrack{
			TimeoutMS: int64(2000), AllowUnknownBidder: true,
		},
		AccountDefaults: config.Account{},
	}
	cfg.MarshalAccountDefaults()

	// bidder info
	bidderInfos := make(config.BidderInfos)

	// prepare
	data, err := getValidVTrackRequestBody(true, false)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/vtrack?a=events_enabled", strings.NewReader(data))

	recorder := httptest.NewRecorder()

	e := vtrackEndpoint{
		Cfg:                 cfg,
		BidderInfos:         bidderInfos,
		Cache:               mockCacheClient,
		Accounts:            mockAccountsFetcher,
		normalizeBidderName: openrtb_ext.NormalizeBidderName,
	}

	// execute
	e.Handle(recorder, req, nil)

	d, err := io.ReadAll(recorder.Result().Body)
	if err != nil {
		t.Fatal(err)
	}

	// validate
	assert.Equal(t, 200, recorder.Result().StatusCode, "Expected 200 when account is not found and request is valid")
	assert.Equal(t, "{\"responses\":[{\"uuid\":\"uuid1\"},{\"uuid\":\"uuid2\"}]}", string(d), "Expected 200 when account is found, request has unknown bidders but allowUnknownBidders is enabled")
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
}

func TestShouldReturnBadRequestWhenRequestExceedsMaxRequestSize(t *testing.T) {
	// mock pbs cache client
	mockCacheClient := &vtrackMockCacheClient{
		Fail:  false,
		Uuids: []string{"uuid1", "uuid2"},
	}

	// mock AccountsFetcher
	mockAccountsFetcher := &mockAccountsFetcher{
		Fail: false,
	}

	// config
	cfg := &config.Configuration{
		MaxRequestSize: 1,
		VTrack: config.VTrack{
			TimeoutMS: int64(2000), AllowUnknownBidder: true,
		},
		AccountDefaults: config.Account{},
	}
	cfg.MarshalAccountDefaults()

	// bidder info
	bidderInfos := make(config.BidderInfos)

	// prepare
	data, err := getValidVTrackRequestBody(true, false)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/vtrack?a=events_enabled", strings.NewReader(data))

	recorder := httptest.NewRecorder()

	e := vtrackEndpoint{
		Cfg:                 cfg,
		BidderInfos:         bidderInfos,
		Cache:               mockCacheClient,
		Accounts:            mockAccountsFetcher,
		normalizeBidderName: openrtb_ext.NormalizeBidderName,
	}

	// execute
	e.Handle(recorder, req, nil)

	d, err := io.ReadAll(recorder.Result().Body)
	if err != nil {
		t.Fatal(err)
	}

	// validate
	assert.Equal(t, 400, recorder.Result().StatusCode, "Expected 400 when request exceeds max request size")
	assert.Equal(t, "Invalid request: request size exceeded max size of 1 bytes\n", string(d))
}

func TestShouldRespondWithInternalErrorPbsCacheIsNotConfigured(t *testing.T) {
	// mock AccountsFetcher
	mockAccountsFetcher := &mockAccountsFetcher{
		Fail: false,
	}

	// config
	cfg := &config.Configuration{
		MaxRequestSize: maxSize, VTrack: config.VTrack{
			TimeoutMS: int64(2000), AllowUnknownBidder: false,
		},
		AccountDefaults: config.Account{},
	}
	cfg.MarshalAccountDefaults()

	// prepare
	data, err := getValidVTrackRequestBody(true, true)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/vtrack?a=events_enabled", strings.NewReader(data))
	recorder := httptest.NewRecorder()

	e := vtrackEndpoint{
		Cfg:                 cfg,
		BidderInfos:         nil,
		Cache:               nil,
		Accounts:            mockAccountsFetcher,
		normalizeBidderName: openrtb_ext.NormalizeBidderName,
	}

	// execute
	e.Handle(recorder, req, nil)

	d, err := io.ReadAll(recorder.Result().Body)
	if err != nil {
		t.Fatal(err)
	}

	// validate
	assert.Equal(t, 500, recorder.Result().StatusCode, "Expected 500 when pbs cache is not configured")
	assert.Equal(t, "PBS Cache client is not configured", string(d))
}

func TestVastUrlShouldReturnExpectedUrl(t *testing.T) {
	url := GetVastUrlTracking("http://external-url", "bidId", "bidder", "accountId", 1000, "integrationType")
	assert.Equal(t, "http://external-url/event?t=imp&b=bidId&a=accountId&bidder=bidder&f=b&int=integrationType&ts=1000", url, "Invalid vast url")
}

func getValidVTrackRequestBody(withImpression bool, withContent bool) (string, error) {
	d, e := getVTrackRequestData(withImpression, withContent)

	if e != nil {
		return "", e
	}

	req := &BidCacheRequest{
		Puts: []prebid_cache_client.Cacheable{
			{
				Type:       prebid_cache_client.TypeXML,
				BidID:      "bidId1",
				Bidder:     "bidder",
				Data:       d,
				TTLSeconds: 3600,
				Timestamp:  1000,
			},
			{
				Type:       prebid_cache_client.TypeXML,
				BidID:      "bidId2",
				Bidder:     "updatable_bidder",
				Data:       d,
				TTLSeconds: 3600,
				Timestamp:  1000,
			},
		},
	}

	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)

	e = enc.Encode(req)

	return buf.String(), e
}

func getVTrackRequestData(wi bool, wic bool) (db []byte, e error) {
	data := &bytes.Buffer{}
	enc := json.NewEncoder(data)
	enc.SetEscapeHTML(false)

	if wi && wic {
		e = enc.Encode(vastXmlWithImpressionWithContent)
		return data.Bytes(), e
	} else if wi {
		e = enc.Encode(vastXmlWithImpressionWithoutContent)
	} else {
		enc.Encode(vastXmlWithoutImpression)
	}

	return data.Bytes(), e
}

func TestGetIntegrationType(t *testing.T) {
	testCases := []struct {
		description             string
		givenHttpRequest        *http.Request
		expectedIntegrationType string
		expectedError           error
	}{
		{
			description:             "Integration type in http request is valid, expect same integration time and no errors",
			givenHttpRequest:        httptest.NewRequest("GET", "/event?t=win&b=bidId&f=b&ts=1000&x=1&a=accountId&bidder=bidder&int=TestIntegrationType", strings.NewReader("")),
			expectedIntegrationType: "TestIntegrationType",
			expectedError:           nil,
		},
		{
			description:      "Integration type in http request is too long, expect too long error",
			givenHttpRequest: httptest.NewRequest("GET", "/event?t=win&b=bidId&f=b&ts=1000&x=1&a=accountId&bidder=bidder&int=TestIntegrationTypeTooLongTestIntegrationTypeTooLongTestIntegrationType", strings.NewReader("")),
			expectedError:    errors.New("integration type length is too long"),
		},
		{
			description:      "Integration type in http request contains invalid character, expect invalid character error",
			givenHttpRequest: httptest.NewRequest("GET", "/event?t=win&b=bidId&f=b&ts=1000&x=1&a=accountId&bidder=bidder&int=Te$tIntegrationType", strings.NewReader("")),
			expectedError:    errors.New("integration type can only contain numbers, letters and these characters '-', '_'"),
		},
	}

	for _, test := range testCases {
		integrationType, err := getIntegrationType(test.givenHttpRequest)
		if test.expectedError != nil {
			assert.Equal(t, test.expectedError, err, test.description)
		} else {
			assert.Empty(t, err, test.description)
			assert.Equalf(t, test.expectedIntegrationType, integrationType, test.description)
		}
	}
}
