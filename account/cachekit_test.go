package account

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	metricsconfig "github.com/prebid/prebid-server/v4/metrics/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/stored_requests"
	"github.com/prebid/prebid-server/v4/stored_requests/caches/memory"
)

// mockAllFetcher is a minimal stored_requests.AllFetcher for the v2 account tests.
// Only FetchAccount is exercised; the other methods satisfy the interface.
type mockAllFetcher struct {
	accounts     map[string]json.RawMessage
	accountCalls int
	bulkCalls    int
}

func (m *mockAllFetcher) FetchRequests(_ context.Context, _ []string, _ []string) (map[string]json.RawMessage, map[string]json.RawMessage, []error) {
	return nil, nil, nil
}

func (m *mockAllFetcher) FetchResponses(_ context.Context, _ []string) (map[string]json.RawMessage, []error) {
	return nil, nil
}

func (m *mockAllFetcher) FetchCategories(_ context.Context, _, _, _ string) (string, error) {
	return "", nil
}

func (m *mockAllFetcher) FetchAccount(_ context.Context, _ json.RawMessage, accountID string) (json.RawMessage, []error) {
	m.accountCalls++
	raw, ok := m.accounts[accountID]
	if !ok {
		return nil, []error{stored_requests.NotFoundError{ID: accountID, DataType: "Account"}}
	}
	return raw, nil
}

// FetchAllAccounts makes the mock a bulk-capable source for refresh: preload.
func (m *mockAllFetcher) FetchAllAccounts(_ context.Context) (map[string]json.RawMessage, []error) {
	m.bulkCalls++
	return m.accounts, nil
}

func newV2Fetcher(t *testing.T, fetcher stored_requests.AllFetcher) *CacheKitAccountFetcher {
	t.Helper()
	cfg := config.CacheKitConfig{Type: "lru", MaxEntries: 100, TTLSeconds: 3600}
	v2, err := NewCacheKitAccountFetcher(fetcher, cfg, json.RawMessage(`{}`), clock.NewMock(), nil)
	require.NoError(t, err)
	return v2
}

func TestV2GetAccountTypedHit(t *testing.T) {
	fetcher := &mockAllFetcher{accounts: map[string]json.RawMessage{
		"pub-1": json.RawMessage(`{"id":"pub-1"}`),
	}}
	v2 := newV2Fetcher(t, fetcher)
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
	assert.Equal(t, 1, fetcher.accountCalls, "second GetAccount should be a cache hit")
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
	fetcher := &mockAllFetcher{accounts: map[string]json.RawMessage{
		"pub-1": json.RawMessage(`{"id":"pub-1"}`),
	}}
	v2, err := NewCacheKitAccountFetcher(fetcher, config.CacheKitConfig{
		Type:       "lru",
		MaxEntries: 100,
		TTLSeconds: 3600,
	}, defaults, clock.NewMock(), nil)
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
	assert.Equal(t, 1, fetcher.accountCalls)
}

func TestV2WrappingLegacyCacheCanMaskBackendChangesAfterTTL(t *testing.T) {
	clk := clock.NewMock()
	source := &mockAllFetcher{accounts: map[string]json.RawMessage{
		"pub-1": json.RawMessage(`{"id":"pub-1","disabled":false}`),
	}}
	legacyCachedFetcher := stored_requests.WithCache(source, stored_requests.Cache{
		Accounts: memory.NewCache(0, 0, "Accounts"),
	}, &metricsconfig.NilMetricsEngine{})
	v2, err := NewCacheKitAccountFetcher(legacyCachedFetcher, config.CacheKitConfig{
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

	// The real source changes, and v2 TTL expires. Because v2 is wrapped around
	// the legacy byte cache, the reload is satisfied by that old cache instead of
	// calling the real source again.
	source.accounts["pub-1"] = json.RawMessage(`{"id":"pub-1","disabled":true}`)
	clk.Add(2 * time.Second)

	account, errs = GetAccount(context.Background(), &config.Configuration{}, v2, "pub-1", nil)
	require.Empty(t, errs)
	require.NotNil(t, account)
	assert.False(t, account.Disabled, "v2 reloaded from the legacy cache, not the changed source")
	assert.Equal(t, 1, source.accountCalls, "backend source is hidden behind the legacy v1 cache")
}

func TestV2GetAccountNotFoundFallsBackToDefaults(t *testing.T) {
	fetcher := &mockAllFetcher{accounts: map[string]json.RawMessage{}}
	v2 := newV2Fetcher(t, fetcher)
	cfg := &config.Configuration{}

	account, errs := GetAccount(context.Background(), cfg, v2, "unknown", nil)
	require.Empty(t, errs)
	require.NotNil(t, account)
	assert.Equal(t, "unknown", account.ID, "not-found should fall back to AccountDefaults with the requested ID")
}

func TestV2GetAccountMalformedReturnsError(t *testing.T) {
	fetcher := &mockAllFetcher{accounts: map[string]json.RawMessage{
		"bad": json.RawMessage(`{`),
	}}
	v2 := newV2Fetcher(t, fetcher)
	cfg := &config.Configuration{}

	account, errs := GetAccount(context.Background(), cfg, v2, "bad", nil)
	require.Nil(t, account)
	require.NotEmpty(t, errs)
	_, isMalformed := errs[0].(*errortypes.MalformedAcct)
	assert.True(t, isMalformed, "malformed account JSON should surface a MalformedAcct error")
}

func TestV2GetAccountDisabled(t *testing.T) {
	fetcher := &mockAllFetcher{accounts: map[string]json.RawMessage{
		"off": json.RawMessage(`{"id":"off","disabled":true}`),
	}}
	v2 := newV2Fetcher(t, fetcher)
	cfg := &config.Configuration{}

	account, errs := GetAccount(context.Background(), cfg, v2, "off", nil)
	require.Nil(t, account)
	require.NotEmpty(t, errs)
	_, isDisabled := errs[0].(*errortypes.AccountDisabled)
	assert.True(t, isDisabled, "disabled account should surface an AccountDisabled error")
}

func TestV2RefreshPreloadWarmsCache(t *testing.T) {
	fetcher := &mockAllFetcher{accounts: map[string]json.RawMessage{
		"pub-1": json.RawMessage(`{"id":"pub-1"}`),
	}}
	cfg := config.CacheKitConfig{Type: "lru", MaxEntries: 100, TTLSeconds: 3600, Refresh: "preload"}
	v2, err := NewCacheKitAccountFetcher(fetcher, cfg, json.RawMessage(`{}`), clock.NewMock(), nil)
	require.NoError(t, err)
	assert.Equal(t, 1, fetcher.bulkCalls, "preload should perform a single bulk fetch at startup")

	account, errs := GetAccount(context.Background(), &config.Configuration{}, v2, "pub-1", nil)
	require.Empty(t, errs)
	assert.Equal(t, "pub-1", account.ID)
	assert.Equal(t, 0, fetcher.accountCalls, "preloaded account should be served without a per-key fetch")
}

func TestV2RefreshTTLServesStaleByDefault(t *testing.T) {
	clk := clock.NewMock()
	fetcher := &mockAllFetcher{accounts: map[string]json.RawMessage{
		"pub-1": json.RawMessage(`{"id":"pub-1","disabled":false}`),
	}}
	cfg := config.CacheKitConfig{Type: "lru", MaxEntries: 100, TTLSeconds: 1, Refresh: "ttl"}
	v2, err := NewCacheKitAccountFetcher(fetcher, cfg, json.RawMessage(`{}`), clk, nil)
	require.NoError(t, err)

	account, errs := GetAccount(context.Background(), &config.Configuration{}, v2, "pub-1", nil)
	require.Empty(t, errs)
	require.NotNil(t, account)
	assert.False(t, account.Disabled)
	assert.Equal(t, 1, fetcher.accountCalls)

	fetcher.accounts["pub-1"] = json.RawMessage(`{"id":"pub-1","disabled":true}`)
	clk.Add(2 * time.Second)

	account, errs = GetAccount(context.Background(), &config.Configuration{}, v2, "pub-1", nil)
	require.Empty(t, errs)
	require.NotNil(t, account)
	assert.False(t, account.Disabled, "ttl mode should return stale data immediately and refresh in the background")
	assert.Eventually(t, func() bool { return fetcher.accountCalls == 2 }, time.Second, 5*time.Millisecond)
}

func TestV2RefreshPreloadUnsupportedSourceErrors(t *testing.T) {
	// A source that does not implement AllAccountsFetcher cannot preload.
	var plain stored_requests.AllFetcher = notBulkFetcher{}
	cfg := config.CacheKitConfig{Type: "lru", MaxEntries: 100, TTLSeconds: 3600, Refresh: "preload"}
	_, err := NewCacheKitAccountFetcher(plain, cfg, json.RawMessage(`{}`), clock.NewMock(), nil)
	require.Error(t, err)
}

func TestV2RefreshUnknownModeErrors(t *testing.T) {
	fetcher := &mockAllFetcher{accounts: map[string]json.RawMessage{}}
	cfg := config.CacheKitConfig{Type: "lru", MaxEntries: 100, TTLSeconds: 3600, Refresh: "bogus"}
	_, err := NewCacheKitAccountFetcher(fetcher, cfg, json.RawMessage(`{}`), clock.NewMock(), nil)
	require.Error(t, err)
}

// notBulkFetcher is an AllFetcher that does NOT implement AllAccountsFetcher.
type notBulkFetcher struct{}

func (notBulkFetcher) FetchRequests(_ context.Context, _ []string, _ []string) (map[string]json.RawMessage, map[string]json.RawMessage, []error) {
	return nil, nil, nil
}
func (notBulkFetcher) FetchResponses(_ context.Context, _ []string) (map[string]json.RawMessage, []error) {
	return nil, nil
}
func (notBulkFetcher) FetchCategories(_ context.Context, _, _, _ string) (string, error) {
	return "", nil
}
func (notBulkFetcher) FetchAccount(_ context.Context, _ json.RawMessage, accountID string) (json.RawMessage, []error) {
	return nil, []error{stored_requests.NotFoundError{ID: accountID, DataType: "Account"}}
}
