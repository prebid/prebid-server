package account

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	jsonpatch "gopkg.in/evanphx/json-patch.v5"

	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/fetcher"
	fetchercache "github.com/prebid/prebid-server/v4/fetcher/cache"
	"github.com/prebid/prebid-server/v4/metrics"
	"github.com/prebid/prebid-server/v4/stored_requests"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
	"github.com/prebid/prebid-server/v4/util/timeutil"
)

// fetcherSubsystem is the metrics subsystem label for the account cache.
const fetcherSubsystem = "account"

// FetcherAccountFetcher is the Fetchers 2.0 account fetcher. It embeds the
// underlying source fetcher (file / db / http / multi) so it continues to satisfy
// stored_requests.AllFetcher, and adds Fetch which serves fully-derived,
// immutable *config.Account values from a fetcher engine.
type FetcherAccountFetcher struct {
	stored_requests.AllFetcher
	engine *fetcher.Fetcher[string, *config.Account]
}

// Fetch implements account.Fetcher[*config.Account].
func (f *FetcherAccountFetcher) Fetch(ctx context.Context, accountID string) (*config.Account, []error) {
	account, err := f.engine.Get(ctx, accountID)
	if err != nil {
		if errors.Is(err, fetcher.ErrNotFound) {
			return nil, []error{stored_requests.NotFoundError{ID: accountID, DataType: "Account"}}
		}
		return nil, []error{err}
	}
	return account, nil
}

// NewFetcherAccountFetcher wires a fetcher engine in front of an existing account
// source. The sources emit raw, unmerged account rows and the shared transform applies
// the account defaults once (at cache insert); this fetcher adds the typed cache,
// single-flight coalescing and optional negative caching. metricsEngine may be nil
// (metrics are not recorded).
func NewFetcherAccountFetcher(source stored_requests.AllFetcher, cfg config.FetcherConfig, defaults json.RawMessage, t timeutil.Time, metricsEngine metrics.MetricsEngine) (*FetcherAccountFetcher, error) {
	if t == nil {
		t = &timeutil.RealTime{}
	}
	var cache fetcher.Cache[string, *config.Account]
	switch cfg.Type {
	case "none":
		cache = fetchercache.NilCache[string, *config.Account]{}
	case "unbounded":
		unbounded, err := fetchercache.NewUnboundedCache[string, *config.Account](cfg.TTL(), t)
		if err != nil {
			return nil, err
		}
		cache = unbounded
	case "", "lru":
		lru, err := fetchercache.NewLRUCache[string, *config.Account](cfg.MaxEntries, cfg.TTL(), t)
		if err != nil {
			return nil, err
		}
		cache = lru
	default:
		return nil, fmt.Errorf("accounts.cache.type %q is not supported (expected none, unbounded or lru)", cfg.Type)
	}

	var negatives *fetcher.NegativeStore[string]
	if cfg.Negative.Enabled {
		n, err := fetcher.NewNegativeStore[string](cfg.Negative.MaxEntries, cfg.Negative.TTL(), t)
		if err != nil {
			return nil, fmt.Errorf("accounts.cache.negative: %w", err)
		}
		negatives = n
	}

	var recorder fetcher.Recorder
	if metricsEngine != nil {
		recorder = metricsRecorder{engine: metricsEngine, subsystem: fetcherSubsystem}
	}

	// Freshness (refresh) axis: ttl (serve-stale + background revalidation), none
	// (never revalidate / load-once), or preload (bulk warm at startup then ttl).
	effectiveTTL := cfg.TTL()
	serveStale := cfg.ServeStale
	var preload fetcher.BulkSource[string]
	switch cfg.Refresh {
	case "", config.RefreshTTL:
		serveStale = true
	case config.RefreshNone:
		effectiveTTL = 0 // never revalidate
	case config.RefreshPreload:
		serveStale = true
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

	engine := fetcher.New(fetcher.Params[string, *config.Account]{
		Source:            accountSource{fetcher: source},
		Transform:         newAccountTransform(defaults),
		Cache:             cache,
		TTL:               effectiveTTL,
		Negatives:         negatives,
		Coalesce:          cfg.CoalesceRequests,
		ServeStale:        serveStale,
		RevalidateTimeout: cfg.RevalidateTimeout(),
		Preload:           preload,
		Time:              t,
		Metrics:           recorder,
	})
	engine.Start(context.Background())

	return &FetcherAccountFetcher{AllFetcher: source, engine: engine}, nil
}

// metricsRecorder adapts a metrics.MetricsEngine to the fetcher.Recorder interface,
// emitting the dedicated fetcher_* metrics under the given subsystem label.
type metricsRecorder struct {
	engine    metrics.MetricsEngine
	subsystem string
}

func (r metricsRecorder) CacheHit() {
	r.engine.RecordFetcherResult(r.subsystem, metrics.FetcherResultHit)
}
func (r metricsRecorder) CacheMiss() {
	r.engine.RecordFetcherResult(r.subsystem, metrics.FetcherResultMiss)
}
func (r metricsRecorder) CacheNegative() {
	r.engine.RecordFetcherResult(r.subsystem, metrics.FetcherResultNegative)
}

func (r metricsRecorder) BackendFetch(result string, d time.Duration) {
	var mapped metrics.FetcherBackendResult
	switch result {
	case "ok":
		mapped = metrics.FetcherBackendOK
	case "notfound":
		mapped = metrics.FetcherBackendNotFound
	default:
		mapped = metrics.FetcherBackendError
	}
	r.engine.RecordFetcherBackendFetch(r.subsystem, mapped, d)
}

// accountSource adapts an existing stored_requests account fetcher into a
// fetcher.Source. It requests the raw, unmerged account row (defaults are applied
// once, downstream, by the shared transform) and reuses the backend's not-found
// classification: a NotFoundError becomes an absent map key (fetcher's "not found"
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

// accountBulkSource adapts a stored_requests.AllAccountsFetcher into a fetcher.BulkSource.
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
func newAccountTransform(defaults json.RawMessage) fetcher.TransformFunc[string, *config.Account] {
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
