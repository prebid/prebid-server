package account

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
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
	accountCalls int
	bulkCalls    int
}

func (m *mockSource) Fetch(_ context.Context, accountID string) (json.RawMessage, error) {
	m.accountCalls++
	raw, ok := m.accounts[accountID]
	if !ok {
		return nil, errAccountNotFound
	}
	return raw, nil
}

func (m *mockSource) FetchAll(_ context.Context) (map[string]json.RawMessage, error) {
	m.bulkCalls++
	return m.accounts, nil
}

func newV2Fetcher(t *testing.T, source Source) *FetcherAccountFetcher {
	t.Helper()
	cfg := config.FetcherConfig{Type: "lru", MaxEntries: 100, TTLSeconds: 3600}
	v2, err := NewFetcherAccountFetcher(source, cfg, json.RawMessage(`{}`), newFakeTime(), nil)
	require.NoError(t, err)
	return v2
}

func TestV2GetAccountTypedHit(t *testing.T) {
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
	assert.Equal(t, 1, source.accountCalls, "second GetAccount should be a cache hit")
}

func TestV2GetAccountAppliesDefaultsAndDerivedConfigOnce(t *testing.T) {
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
	assert.Equal(t, 1, source.accountCalls)
}

func TestV2RefreshTTLReloadsFromSource(t *testing.T) {
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
	assert.Equal(t, 1, source.accountCalls)

	source.accounts["pub-1"] = json.RawMessage(`{"id":"pub-1","disabled":true}`)
	clk.Add(2 * time.Second)

	account, errs = GetAccount(context.Background(), &config.Configuration{}, v2, "pub-1", nil)
	require.Empty(t, errs)
	require.NotNil(t, account)
	assert.False(t, account.Disabled, "ttl mode should return stale data immediately")
	assert.Eventually(t, func() bool { return source.accountCalls == 2 }, time.Second, 5*time.Millisecond)
}

func TestV2GetAccountNotFoundFallsBackToDefaults(t *testing.T) {
	source := &mockSource{accounts: map[string]json.RawMessage{}}
	v2 := newV2Fetcher(t, source)
	cfg := &config.Configuration{}

	account, errs := GetAccount(context.Background(), cfg, v2, "unknown", nil)
	require.Empty(t, errs)
	require.NotNil(t, account)
	assert.Equal(t, "unknown", account.ID, "not-found should fall back to AccountDefaults with the requested ID")
}

func TestV2GetAccountMalformedReturnsError(t *testing.T) {
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

func TestV2GetAccountDisabled(t *testing.T) {
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

func TestV2RefreshPreloadWarmsCache(t *testing.T) {
	source := &mockSource{accounts: map[string]json.RawMessage{
		"pub-1": json.RawMessage(`{"id":"pub-1"}`),
	}}
	cfg := config.FetcherConfig{Type: "lru", MaxEntries: 100, TTLSeconds: 3600, Refresh: "preload"}
	v2, err := NewFetcherAccountFetcher(source, cfg, json.RawMessage(`{}`), newFakeTime(), nil)
	require.NoError(t, err)
	assert.Equal(t, 1, source.bulkCalls, "preload should perform a single bulk fetch at startup")

	account, errs := GetAccount(context.Background(), &config.Configuration{}, v2, "pub-1", nil)
	require.Empty(t, errs)
	assert.Equal(t, "pub-1", account.ID)
	assert.Equal(t, 0, source.accountCalls, "preloaded account should be served without a per-key fetch")
}

func TestV2RefreshTTLServesStaleByDefault(t *testing.T) {
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
	assert.Equal(t, 1, source.accountCalls)

	source.accounts["pub-1"] = json.RawMessage(`{"id":"pub-1","disabled":true}`)
	clk.Add(2 * time.Second)

	account, errs = GetAccount(context.Background(), &config.Configuration{}, v2, "pub-1", nil)
	require.Empty(t, errs)
	require.NotNil(t, account)
	assert.False(t, account.Disabled, "ttl mode should return stale data immediately and refresh in the background")
	assert.Eventually(t, func() bool { return source.accountCalls == 2 }, time.Second, 5*time.Millisecond)
}

func TestV2UnboundedCacheDoesNotEvictByEntryCount(t *testing.T) {
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
	assert.Equal(t, 1000, source.accountCalls)

	for i := 0; i < 1000; i++ {
		id := fmt.Sprintf("pub-%d", i)
		account, errs := GetAccount(context.Background(), &config.Configuration{}, v2, id, nil)
		require.Empty(t, errs)
		require.NotNil(t, account)
		assert.Equal(t, id, account.ID)
	}
	assert.Equal(t, 1000, source.accountCalls, "unbounded cache should retain every fetched account")
}

func TestV2RefreshPreloadUnsupportedSourceErrors(t *testing.T) {
	plain := notBulkSource{}
	cfg := config.FetcherConfig{Type: "lru", MaxEntries: 100, TTLSeconds: 3600, Refresh: "preload"}
	_, err := NewFetcherAccountFetcher(plain, cfg, json.RawMessage(`{}`), newFakeTime(), nil)
	require.Error(t, err)
}

func TestV2RefreshUnknownModeErrors(t *testing.T) {
	source := &mockSource{accounts: map[string]json.RawMessage{}}
	cfg := config.FetcherConfig{Type: "lru", MaxEntries: 100, TTLSeconds: 3600, Refresh: "bogus"}
	_, err := NewFetcherAccountFetcher(source, cfg, json.RawMessage(`{}`), newFakeTime(), nil)
	require.Error(t, err)
}

func TestV2NegativeCacheInvalidConfigErrors(t *testing.T) {
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

type notBulkSource struct{}

func (notBulkSource) Fetch(_ context.Context, accountID string) (json.RawMessage, error) {
	return nil, errAccountNotFound
}
