package account

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/fetcher"
	"github.com/prebid/prebid-server/v4/metrics"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/stored_requests"
)

type fakeTime struct {
	now time.Time
}

func newFakeTime() *fakeTime {
	return &fakeTime{now: time.Unix(1000, 0)}
}

func (f *fakeTime) Now() time.Time {
	return f.now
}

func (f *fakeTime) Add(d time.Duration) {
	f.now = f.now.Add(d)
}

type mockSource struct {
	accounts     map[string]json.RawMessage
	accountCalls atomic.Int32
	bulkCalls    atomic.Int32
	bulkErr      error
}

func (m *mockSource) Fetch(_ context.Context, accountID string) (json.RawMessage, bool, error) {
	m.accountCalls.Add(1)
	raw, found := m.accounts[accountID]
	return raw, found, nil
}

func (m *mockSource) FetchAll(_ context.Context) (map[string]json.RawMessage, error) {
	m.bulkCalls.Add(1)
	return m.accounts, m.bulkErr
}

func newV2Fetcher(t *testing.T, source fetcher.Source[string]) *FetcherAccountFetcher {
	t.Helper()
	cfg := config.FetcherConfig{Type: "lru", MaxEntries: 100, TTLSeconds: 3600}
	v2, err := NewFetcherAccountFetcher(source, cfg, json.RawMessage(`{}`), newFakeTime(), nil)
	require.NoError(t, err)
	return v2
}

func TestV2GetAccountUsesTypedCacheAfterFirstFetch(t *testing.T) {
	source := &mockSource{accounts: map[string]json.RawMessage{
		"pub-1": json.RawMessage(`{"id":"pub-1"}`),
	}}
	v2 := newV2Fetcher(t, source)
	cfg := &config.Configuration{}

	account, errs := GetAccount(context.Background(), cfg, v2, "pub-1", nil)
	require.Empty(t, errs)
	require.NotNil(t, account)
	assert.Equal(t, "pub-1", account.ID)
	// Derived config is computed once at cache insert.
	assert.NotNil(t, account.GDPR.PurposeConfigs, "derived config should be populated")

	// Second lookup is served from the typed cache without hitting the source.
	account, errs = GetAccount(context.Background(), cfg, v2, "pub-1", nil)
	require.Empty(t, errs)
	assert.Equal(t, "pub-1", account.ID)
	assert.Equal(t, int32(1), source.accountCalls.Load(), "second GetAccount should be a cache hit")
}

func TestV2GetAccountCachesDefaultsAndDerivedConfig(t *testing.T) {
	defaults := json.RawMessage(`{
		"gdpr": {
			"basic_enforcement_vendors": ["appnexus"],
			"purpose1": {
				"enforce_algo": "basic",
				"vendor_exceptions": ["rubicon"]
			},
			"special_feature1": {
				"vendor_exceptions": ["appnexus"]
			}
		},
		"privacy": {
			"dsa": {
				"default": "{\"dsarequired\":1,\"pubrender\":2,\"transparency\":[{\"domain\":\"test.com\"}]}"
			}
		}
	}`)
	source := &mockSource{accounts: map[string]json.RawMessage{
		"pub-1": json.RawMessage(`{"id":"pub-1"}`),
	}}
	v2, err := NewFetcherAccountFetcher(source, config.FetcherConfig{
		Type:       "lru",
		MaxEntries: 100,
		TTLSeconds: 3600,
	}, defaults, newFakeTime(), nil)
	require.NoError(t, err)

	account, errs := GetAccount(context.Background(), &config.Configuration{}, v2, "pub-1", nil)
	require.Empty(t, errs)
	require.NotNil(t, account)

	assert.Contains(t, account.GDPR.BasicEnforcementVendorsMap, "appnexus")
	assert.Contains(t, account.GDPR.Purpose1.VendorExceptionMap, "rubicon")
	assert.Contains(t, account.GDPR.SpecialFeature1.VendorExceptionMap, openrtb_ext.BidderName("appnexus"))
	assert.Equal(t, config.TCF2BasicEnforcement, account.GDPR.Purpose1.EnforceAlgoID)
	require.NotNil(t, account.Privacy.DSA)
	require.NotNil(t, account.Privacy.DSA.DefaultUnpacked)
	assert.Equal(t, int8(1), *account.Privacy.DSA.DefaultUnpacked.Required)
	assert.Equal(t, int8(2), *account.Privacy.DSA.DefaultUnpacked.PubRender)
	assert.Equal(t, "test.com", account.Privacy.DSA.DefaultUnpacked.Transparency[0].Domain)

	// A second lookup is a typed-cache hit: the source is not called again, and
	// defaults/DSA/derived map work is not repeated through the fetcher path.
	account, errs = GetAccount(context.Background(), &config.Configuration{}, v2, "pub-1", nil)
	require.Empty(t, errs)
	require.NotNil(t, account)
	assert.Equal(t, int32(1), source.accountCalls.Load())
}

func TestV2GetAccountUsesTTLRefreshByDefault(t *testing.T) {
	clk := newFakeTime()
	source := &mockSource{accounts: map[string]json.RawMessage{
		"pub-1": json.RawMessage(`{"id":"pub-1","disabled":false}`),
	}}
	v2, err := NewFetcherAccountFetcher(source, config.FetcherConfig{
		Type:       "lru",
		MaxEntries: 100,
		TTLSeconds: 1,
	}, json.RawMessage(`{}`), clk, nil)
	require.NoError(t, err)

	account, errs := GetAccount(context.Background(), &config.Configuration{}, v2, "pub-1", nil)
	require.Empty(t, errs)
	require.NotNil(t, account)
	assert.False(t, account.Disabled)
	assert.Equal(t, int32(1), source.accountCalls.Load())

	source.accounts["pub-1"] = json.RawMessage(`{"id":"pub-1","disabled":true}`)
	clk.Add(2 * time.Second)

	account, errs = GetAccount(context.Background(), &config.Configuration{}, v2, "pub-1", nil)
	require.Empty(t, errs)
	require.NotNil(t, account)
	assert.False(t, account.Disabled, "ttl mode should return stale data immediately")
	assert.Eventually(t, func() bool { return source.accountCalls.Load() == 2 }, time.Second, 5*time.Millisecond)
}

func TestV2GetAccountUsesDefaultsWhenTypedAccountIsMissing(t *testing.T) {
	source := &mockSource{accounts: map[string]json.RawMessage{}}
	v2 := newV2Fetcher(t, source)
	cfg := &config.Configuration{}

	account, errs := GetAccount(context.Background(), cfg, v2, "missing", nil)
	require.Empty(t, errs)
	require.NotNil(t, account)
	assert.Equal(t, "missing", account.ID, "not-found should fall back to AccountDefaults with the requested ID")
}

func TestV2GetAccountReturnsRequiredErrorWhenTypedAccountIsMissing(t *testing.T) {
	source := &mockSource{accounts: map[string]json.RawMessage{}}
	v2 := newV2Fetcher(t, source)
	cfg := &config.Configuration{
		AccountRequired: true,
		AccountDefaults: config.Account{Disabled: true},
	}

	account, errs := GetAccount(context.Background(), cfg, v2, "missing", nil)

	require.Nil(t, account)
	require.Len(t, errs, 1)
	assert.IsType(t, &errortypes.AcctRequired{}, errs[0])
}

func TestV2FetchAccountSerializesTypedAccount(t *testing.T) {
	source := &mockSource{accounts: map[string]json.RawMessage{
		"pub-1": json.RawMessage(`{"id":"pub-1"}`),
	}}
	v2 := newV2Fetcher(t, source)

	raw, errs := v2.FetchAccount(context.Background(), nil, "pub-1")
	require.Empty(t, errs)
	var account config.Account
	require.NoError(t, json.Unmarshal(raw, &account))
	assert.Equal(t, "pub-1", account.ID)
}

func TestV2FetchAccountReturnsFetchError(t *testing.T) {
	v2 := newV2Fetcher(t, &mockSource{accounts: map[string]json.RawMessage{}})

	raw, errs := v2.FetchAccount(context.Background(), nil, "missing")
	require.Nil(t, raw)
	require.Len(t, errs, 1)
	assert.IsType(t, stored_requests.NotFoundError{}, errs[0])
}

func TestV2FetchAccountReturnsMarshalError(t *testing.T) {
	source := &mockSource{accounts: map[string]json.RawMessage{
		"pub-1": json.RawMessage(`{"id":"pub-1"}`),
	}}
	v2 := newV2Fetcher(t, source)
	fetchedAccount, fetchErrs := v2.Fetch(context.Background(), "pub-1")
	require.Empty(t, fetchErrs)
	fetchedAccount.BidAdjustments = &openrtb_ext.ExtRequestPrebidBidAdjustments{
		MediaType: openrtb_ext.MediaType{
			Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
				"appnexus": {
					"deal": []openrtb_ext.Adjustment{{Value: math.NaN()}},
				},
			},
		},
	}
	raw, errs := v2.FetchAccount(context.Background(), nil, "pub-1")
	require.Nil(t, raw)
	require.Len(t, errs, 1)
	assert.IsType(t, &errortypes.FailedToMarshal{}, errs[0])
}

func TestV2GetAccountReturnsMalformedErrorForInvalidTypedAccount(t *testing.T) {
	source := &mockSource{accounts: map[string]json.RawMessage{
		"bad": json.RawMessage(`{`),
	}}
	v2 := newV2Fetcher(t, source)
	cfg := &config.Configuration{}

	account, errs := GetAccount(context.Background(), cfg, v2, "bad", nil)
	require.Nil(t, account)
	require.NotEmpty(t, errs)
	_, isMalformed := errs[0].(*errortypes.MalformedAcct)
	assert.True(t, isMalformed, "malformed account JSON should surface a MalformedAcct error")
}

func TestV2GetAccountRejectsDisabledTypedAccount(t *testing.T) {
	source := &mockSource{accounts: map[string]json.RawMessage{
		"off": json.RawMessage(`{"id":"off","disabled":true}`),
	}}
	v2 := newV2Fetcher(t, source)
	cfg := &config.Configuration{}

	account, errs := GetAccount(context.Background(), cfg, v2, "off", nil)
	require.Nil(t, account)
	require.NotEmpty(t, errs)
	_, isDisabled := errs[0].(*errortypes.AccountDisabled)
	assert.True(t, isDisabled, "disabled account should surface an AccountDisabled error")
}

func TestV2NewFetcherAccountFetcherPreloadsAccounts(t *testing.T) {
	source := &mockSource{accounts: map[string]json.RawMessage{
		"pub-1": json.RawMessage(`{"id":"pub-1"}`),
	}}
	cfg := config.FetcherConfig{Type: "lru", MaxEntries: 100, TTLSeconds: 3600, Refresh: "preload"}
	v2, err := NewFetcherAccountFetcher(source, cfg, json.RawMessage(`{}`), newFakeTime(), nil)
	require.NoError(t, err)
	assert.Equal(t, int32(1), source.bulkCalls.Load(), "preload should perform a single bulk fetch at startup")

	account, errs := GetAccount(context.Background(), &config.Configuration{}, v2, "pub-1", nil)
	require.Empty(t, errs)
	assert.Equal(t, "pub-1", account.ID)
	assert.Equal(t, int32(0), source.accountCalls.Load(), "preloaded account should be served without a per-key fetch")
}

func TestV2GetAccountUsesExplicitTTLRefresh(t *testing.T) {
	clk := newFakeTime()
	source := &mockSource{accounts: map[string]json.RawMessage{
		"pub-1": json.RawMessage(`{"id":"pub-1","disabled":false}`),
	}}
	cfg := config.FetcherConfig{Type: "lru", MaxEntries: 100, TTLSeconds: 1, Refresh: "ttl"}
	v2, err := NewFetcherAccountFetcher(source, cfg, json.RawMessage(`{}`), clk, nil)
	require.NoError(t, err)

	account, errs := GetAccount(context.Background(), &config.Configuration{}, v2, "pub-1", nil)
	require.Empty(t, errs)
	require.NotNil(t, account)
	assert.False(t, account.Disabled)
	assert.Equal(t, int32(1), source.accountCalls.Load())

	source.accounts["pub-1"] = json.RawMessage(`{"id":"pub-1","disabled":true}`)
	clk.Add(2 * time.Second)

	account, errs = GetAccount(context.Background(), &config.Configuration{}, v2, "pub-1", nil)
	require.Empty(t, errs)
	require.NotNil(t, account)
	assert.False(t, account.Disabled, "ttl mode should return stale data immediately and refresh in the background")
	assert.Eventually(t, func() bool { return source.accountCalls.Load() == 2 }, time.Second, 5*time.Millisecond)
}

func TestV2GetAccountUnboundedCacheRetainsAllEntries(t *testing.T) {
	source := &mockSource{accounts: map[string]json.RawMessage{}}
	for i := 0; i < 1000; i++ {
		id := fmt.Sprintf("pub-%d", i)
		source.accounts[id] = json.RawMessage(fmt.Sprintf(`{"id":%q}`, id))
	}
	cfg := config.FetcherConfig{Type: "unbounded", TTLSeconds: 3600}
	v2, err := NewFetcherAccountFetcher(source, cfg, json.RawMessage(`{}`), newFakeTime(), nil)
	require.NoError(t, err)

	for i := 0; i < 1000; i++ {
		id := fmt.Sprintf("pub-%d", i)
		account, errs := GetAccount(context.Background(), &config.Configuration{}, v2, id, nil)
		require.Empty(t, errs)
		require.NotNil(t, account)
		assert.Equal(t, id, account.ID)
	}
	assert.Equal(t, int32(1000), source.accountCalls.Load())

	for i := 0; i < 1000; i++ {
		id := fmt.Sprintf("pub-%d", i)
		account, errs := GetAccount(context.Background(), &config.Configuration{}, v2, id, nil)
		require.Empty(t, errs)
		require.NotNil(t, account)
		assert.Equal(t, id, account.ID)
	}
	assert.Equal(t, int32(1000), source.accountCalls.Load(), "unbounded cache should retain every fetched account")
}

func TestV2NewFetcherAccountFetcherRejectsPreloadWithoutBulkSource(t *testing.T) {
	plain := notBulkSource{}
	cfg := config.FetcherConfig{Type: "lru", MaxEntries: 100, TTLSeconds: 3600, Refresh: "preload"}
	_, err := NewFetcherAccountFetcher(plain, cfg, json.RawMessage(`{}`), newFakeTime(), nil)
	require.Error(t, err)
}

func TestV2NewFetcherAccountFetcherRejectsUnknownRefreshMode(t *testing.T) {
	source := &mockSource{accounts: map[string]json.RawMessage{}}
	cfg := config.FetcherConfig{Type: "lru", MaxEntries: 100, TTLSeconds: 3600, Refresh: "bogus"}
	_, err := NewFetcherAccountFetcher(source, cfg, json.RawMessage(`{}`), newFakeTime(), nil)
	require.Error(t, err)
}

func TestV2NewFetcherAccountFetcherRejectsInvalidNegativeCacheConfig(t *testing.T) {
	source := &mockSource{accounts: map[string]json.RawMessage{}}
	cfg := config.FetcherConfig{
		Type:       "lru",
		MaxEntries: 100,
		Negative: config.NegativeCacheConfig{
			Enabled:    true,
			MaxEntries: 0,
			TTLSeconds: 60,
		},
	}

	_, err := NewFetcherAccountFetcher(source, cfg, json.RawMessage(`{}`), newFakeTime(), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "negative")
}

func TestV2NewFetcherAccountFetcherRequiresTime(t *testing.T) {
	source := &mockSource{accounts: map[string]json.RawMessage{}}

	_, err := NewFetcherAccountFetcher(source, config.FetcherConfig{Type: "none"}, json.RawMessage(`{}`), nil, nil)

	require.EqualError(t, err, "accounts.cache: time is required")
}

func TestV2NewFetcherAccountFetcherReturnsPreloadSourceError(t *testing.T) {
	preloadErr := errors.New("preload failed")
	source := &mockSource{
		accounts: map[string]json.RawMessage{},
		bulkErr:  preloadErr,
	}

	accountFetcher, err := NewFetcherAccountFetcher(
		source,
		config.FetcherConfig{Type: "lru", MaxEntries: 100, Refresh: "preload"},
		json.RawMessage(`{}`),
		newFakeTime(),
		nil,
	)

	require.Nil(t, accountFetcher)
	require.ErrorIs(t, err, preloadErr)
	assert.Contains(t, err.Error(), "accounts.cache.preload")
}

func TestV2AccountTransformRejectsMalformedDefaults(t *testing.T) {
	transform := newAccountTransform(json.RawMessage(`{`))

	account, err := transform("pub-1", json.RawMessage(`{}`))

	require.Nil(t, account)
	assert.IsType(t, &errortypes.MalformedAcct{}, err)
}

func TestV2AccountTransformRejectsMalformedDSADefault(t *testing.T) {
	transform := newAccountTransform(nil)

	account, err := transform("pub-1", json.RawMessage(`{"privacy":{"dsa":{"default":"not-json"}}}`))

	require.Nil(t, account)
	assert.IsType(t, &errortypes.MalformedAcct{}, err)
}

func TestV2AccountTransformUsesRequestedIDWhenSourceOmitsID(t *testing.T) {
	transform := newAccountTransform(nil)

	account, err := transform("pub-1", json.RawMessage(`{"disabled":false}`))

	require.NoError(t, err)
	assert.Equal(t, "pub-1", account.ID)
}

func TestV2MetricsRecorderForwardsFetcherEvents(t *testing.T) {
	engine := &metrics.MetricsEngineMock{}
	engine.Mock.On("RecordFetcherResult", fetcherSubsystem, metrics.FetcherResultHit).Once()
	engine.Mock.On("RecordFetcherResult", fetcherSubsystem, metrics.FetcherResultMiss).Once()
	engine.Mock.On("RecordFetcherResult", fetcherSubsystem, metrics.FetcherResultNegative).Once()
	engine.Mock.On(
		"RecordFetcherBackendFetch",
		fetcherSubsystem,
		metrics.FetcherOperationGet,
		metrics.FetcherBackendOK,
		time.Second,
	).Once()
	recorder := metricsRecorder{engine: engine, subsystem: fetcherSubsystem}

	recorder.CacheHit()
	recorder.CacheMiss()
	recorder.CacheNegative()
	recorder.BackendFetch(metrics.FetcherOperationGet, metrics.FetcherBackendOK, time.Second)

	engine.AssertExpectations(t)
}

type notBulkSource struct{}

func (notBulkSource) Fetch(_ context.Context, accountID string) (json.RawMessage, bool, error) {
	return nil, false, nil
}
