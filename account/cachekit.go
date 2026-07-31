package account

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/benbjohnson/clock"
	jsonpatch "gopkg.in/evanphx/json-patch.v5"

	"github.com/prebid/prebid-server/v4/cachekit"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/logger"
	"github.com/prebid/prebid-server/v4/metrics"
	"github.com/prebid/prebid-server/v4/stored_requests"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

// cacheKitSubsystem is the metrics subsystem label for the account cache.
const cacheKitSubsystem = "account"

// CacheKitAccountFetcher is the Fetchers 2.0 account fetcher. It embeds the
// underlying source fetcher (file / db / http / multi) so it continues to satisfy
// stored_requests.AllFetcher, and adds FetchAccountTyped which serves fully-derived,
// immutable *config.Account values from a cachekit engine.
type CacheKitAccountFetcher struct {
	stored_requests.AllFetcher
	engine *cachekit.Fetcher[string, *config.Account]
}

// FetchAccountTyped implements account.TypedAccountFetcher.
func (f *CacheKitAccountFetcher) FetchAccountTyped(ctx context.Context, accountID string) (*config.Account, []error) {
	account, err := f.engine.Get(ctx, accountID)
	if err != nil {
		if errors.Is(err, cachekit.ErrNotFound) {
			return nil, []error{stored_requests.NotFoundError{ID: accountID, DataType: "Account"}}
		}
		return nil, []error{err}
	}
	return account, nil
}

// NewCacheKitAccountFetcher wires a cachekit engine in front of an existing account
// source. The sources emit raw, unmerged account rows and the shared transform applies
// the account defaults once (at cache insert); this fetcher adds the typed cache,
// single-flight coalescing and optional negative caching. clk may be nil (a real clock
// is used). metricsEngine may be nil (metrics are not recorded).
func NewCacheKitAccountFetcher(source stored_requests.AllFetcher, cfg config.CacheKitConfig, defaults json.RawMessage, clk clock.Clock, metricsEngine metrics.MetricsEngine) (*CacheKitAccountFetcher, error) {
	var cache cachekit.Cache[string, *config.Account]
	switch cfg.Type {
	case "none":
		cache = cachekit.NoCache[string, *config.Account]{}
	case "", "lru":
		lru, err := cachekit.NewLRUCache[string, *config.Account](cfg.MaxEntries, clk)
		if err != nil {
			return nil, err
		}
		cache = lru
	default:
		return nil, fmt.Errorf("accounts.cache.type %q is not supported (expected none or lru)", cfg.Type)
	}

	var negatives *cachekit.NegativeStore[string]
	if cfg.Negative.Enabled {
		n, err := cachekit.NewNegativeStore[string](cfg.Negative.MaxEntries, cfg.Negative.TTL(), clk)
		if err != nil {
			// Negative caching is an optimization, not a correctness requirement. If it
			// can't be built (e.g. a bad accounts.cache.negative.max_entries), warn and
			// continue without it rather than aborting account fetcher startup; not-found
			// lookups will just fall through to the backend each time.
			logger.Warnf("account cachekit: negative caching disabled, failed to initialize: %v", err)
		} else {
			negatives = n
		}
	}

	var recorder cachekit.Recorder
	if metricsEngine != nil {
		recorder = metricsRecorder{engine: metricsEngine, subsystem: cacheKitSubsystem}
	}

	// Freshness (refresh) axis: ttl (serve-stale + background revalidation), none
	// (never revalidate / load-once), or preload (bulk warm at startup then ttl).
	effectiveTTL := cfg.TTL()
	var preload cachekit.BulkSource[string]
	switch cfg.Refresh {
	case "", config.RefreshTTL:
		// serve-stale via ttl; nothing to preload.
	case config.RefreshNone:
		effectiveTTL = 0 // never revalidate
	case config.RefreshPreload:
		bulk, ok := source.(stored_requests.AllAccountsFetcher)
		if !ok {
			return nil, fmt.Errorf("accounts.cache.refresh %q requires an account source that supports bulk loading (FetchAllAccounts)", cfg.Refresh)
		}
		preload = accountBulkSource{fetcher: bulk}
	// NOTE: "delta-poll" (event-driven Save/Invalidation, mirroring v1's http_events /
	// cache-events producers) is intentionally not implemented: no known deployment
	// pushes live account updates, so it would be untested, unused code. It can be added
	// later as a background mechanism without touching Source/Transform/Cache.
	default:
		return nil, fmt.Errorf("accounts.cache.refresh %q is not supported (expected ttl, none or preload)", cfg.Refresh)
	}

	engine := cachekit.New(cachekit.Params[string, *config.Account]{
		Source:     accountSource{fetcher: source},
		Transform:  newAccountTransform(defaults),
		Cache:      cache,
		TTL:        effectiveTTL,
		Negatives:  negatives,
		Coalesce:   cfg.CoalesceRequests,
		ServeStale: cfg.ServeStale,
		Preload:    preload,
		Clock:      clk,
		Metrics:    recorder,
	})
	engine.Start(context.Background())

	return &CacheKitAccountFetcher{AllFetcher: source, engine: engine}, nil
}

// metricsRecorder adapts a metrics.MetricsEngine to the cachekit.Recorder interface,
// emitting the dedicated cachekit_* metrics under the given subsystem label.
type metricsRecorder struct {
	engine    metrics.MetricsEngine
	subsystem string
}

func (r metricsRecorder) CacheHit() {
	r.engine.RecordCacheKitResult(r.subsystem, metrics.CacheKitResultHit)
}
func (r metricsRecorder) CacheMiss() {
	r.engine.RecordCacheKitResult(r.subsystem, metrics.CacheKitResultMiss)
}
func (r metricsRecorder) CacheNegative() {
	r.engine.RecordCacheKitResult(r.subsystem, metrics.CacheKitResultNegative)
}

func (r metricsRecorder) BackendFetch(result string, d time.Duration) {
	var mapped metrics.CacheKitBackendResult
	switch result {
	case "ok":
		mapped = metrics.CacheKitBackendOK
	case "notfound":
		mapped = metrics.CacheKitBackendNotFound
	default:
		mapped = metrics.CacheKitBackendError
	}
	r.engine.RecordCacheKitBackendFetch(r.subsystem, mapped, d)
}

// accountSource adapts an existing stored_requests account fetcher into a
// cachekit.Source. It requests the raw, unmerged account row (defaults are applied
// once, downstream, by the shared transform) and reuses the backend's not-found
// classification: a NotFoundError becomes an absent map key (cachekit's "not found"
// convention); any other error is a systemic failure and is not cached.
type accountSource struct {
	fetcher stored_requests.AccountFetcher
}

func (s accountSource) Fetch(ctx context.Context, keys []string) (map[string]json.RawMessage, error) {
	out := make(map[string]json.RawMessage, len(keys))
	for _, id := range keys {
		// nil defaults => the backend returns the raw row without merging. The shared
		// transform applies defaults, so the single-key and bulk paths merge in one place.
		raw, errs := s.fetcher.FetchAccount(ctx, nil, id)
		if len(errs) > 0 {
			if isNotFoundErr(errs) {
				continue // absent key => definitive not-found for this id
			}
			return nil, errors.Join(errs...)
		}
		out[id] = raw
	}
	return out, nil
}

func isNotFoundErr(errs []error) bool {
	for _, e := range errs {
		if _, ok := e.(stored_requests.NotFoundError); ok {
			return true
		}
	}
	return false
}

// accountBulkSource adapts a stored_requests.AllAccountsFetcher into a cachekit.BulkSource.
// It returns the raw, unmerged account rows as-is; the shared transform applies defaults
// downstream, so this path and the single-key path merge in exactly one place.
type accountBulkSource struct {
	fetcher stored_requests.AllAccountsFetcher
}

func (s accountBulkSource) FetchAll(ctx context.Context) (map[string]json.RawMessage, error) {
	data, errs := s.fetcher.FetchAllAccounts(ctx)
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return data, nil
}

// newAccountTransform returns the single normalization step for accounts. It merges the
// account defaults into the raw row, then unmarshals, unpacks DSA defaults, fills the ID,
// and computes the derived + IP-masking config. It runs once per id at cache insert, so
// the per-request read path does no merge, unmarshal, DSA unpack, derive or IP masking.
// Both the single-key and bulk sources feed it raw, unmerged rows, so the defaults-merge
// lives here in exactly one place.
func newAccountTransform(defaults json.RawMessage) cachekit.TransformFunc[string, *config.Account] {
	return func(accountID string, raw json.RawMessage) (*config.Account, error) {
		merged := raw
		if defaults != nil {
			m, err := jsonpatch.MergePatch(defaults, raw)
			if err != nil {
				return nil, &errortypes.MalformedAcct{
					Message: fmt.Sprintf("The prebid-server account config for account id \"%s\" is malformed. Please reach out to the prebid server host.", accountID),
				}
			}
			merged = m
		}
		account := &config.Account{}
		if err := jsonutil.UnmarshalValid(merged, account); err != nil {
			return nil, &errortypes.MalformedAcct{
				Message: fmt.Sprintf("The prebid-server account config for account id \"%s\" is malformed. Please reach out to the prebid server host.", accountID),
			}
		}
		if err := config.UnpackDSADefault(account.Privacy.DSA); err != nil {
			return nil, &errortypes.MalformedAcct{
				Message: fmt.Sprintf("The prebid-server account config DSA for account id \"%s\" is malformed. Please reach out to the prebid server host.", accountID),
			}
		}
		if len(account.ID) == 0 {
			account.ID = accountID
		}
		setDerivedConfig(account)
		applyIPMaskingDefaults(account)
		return account, nil
	}
}
