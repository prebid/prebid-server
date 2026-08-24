package account

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/fetcher"
	"github.com/prebid/prebid-server/v4/metrics"
	"github.com/prebid/prebid-server/v4/stored_requests"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
	"github.com/prebid/prebid-server/v4/util/timeutil"
)

// fetcherSubsystem is the metrics subsystem label for the account cache.
const fetcherSubsystem = "account"

// FetcherAccountFetcher is the Fetchers 2.0 account fetcher. It embeds the
// typed fetcher engine and serves fully-derived, immutable *config.Account values.
type FetcherAccountFetcher struct {
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

// FetchAccount keeps FetcherAccountFetcher compatible with the current account
// lookup wiring. GetAccount uses Fetch first, so this method is only a bridge for
// call sites still typed as stored_requests.AccountFetcher.
func (f *FetcherAccountFetcher) FetchAccount(ctx context.Context, _ json.RawMessage, accountID string) (json.RawMessage, []error) {
	account, errs := f.Fetch(ctx, accountID)
	if len(errs) > 0 {
		return nil, errs
	}
	raw, err := json.Marshal(account)
	if err != nil {
		return nil, []error{err}
	}
	return raw, nil
}

// NewFetcherAccountFetcher wires the generic fetcher engine to a Fetchers 2.0
// account source. The source emits raw, unmerged account rows and the shared
// transform applies account defaults once at cache insert; this fetcher adds the
// typed cache, request coalescing and optional negative caching. metricsEngine
// may be nil (metrics are not recorded).
func NewFetcherAccountFetcher(source Source, cfg config.FetcherConfig, defaults json.RawMessage, t timeutil.Time, metricsEngine metrics.MetricsEngine) (*FetcherAccountFetcher, error) {
	if t == nil {
		t = &timeutil.RealTime{}
	}

	var recorder fetcher.Recorder = fetcher.NoopRecorder{}
	if metricsEngine != nil {
		recorder = metricsRecorder{engine: metricsEngine, subsystem: fetcherSubsystem}
	}

	engine, err := fetcher.New(fetcher.Params[string, *config.Account]{
		Source:    newAccountFetcherSource(source),
		Transform: newAccountTransform(defaults),
		Config:    newFetcherConfig(cfg),
		Time:      t,
		Metrics:   recorder,
	})
	if err != nil {
		return nil, fmt.Errorf("accounts.cache: %w", err)
	}
	if err := engine.Start(context.Background()); err != nil {
		return nil, fmt.Errorf("accounts.cache.preload: %w", err)
	}

	return &FetcherAccountFetcher{engine: engine}, nil
}

func newAccountFetcherSource(source Source) fetcherSource {
	if bulk, ok := source.(BulkSource); ok {
		return accountBulkSource{source: bulk}
	}
	return accountSource{source: source}
}

type fetcherSource interface {
	Fetch(ctx context.Context, key string) (json.RawMessage, bool, error)
}

func newFetcherConfig(cfg config.FetcherConfig) fetcher.Config {
	refreshMode := string(cfg.Refresh)
	if refreshMode == "" {
		refreshMode = string(config.RefreshTTL)
	}
	return fetcher.Config{
		Cache: fetcher.CacheConfig{
			Type:       cfg.Type,
			MaxEntries: cfg.MaxEntries,
			TTL:        cfg.TTL(),
		},
		Refresh: fetcher.RefreshConfig{
			Mode:                     refreshMode,
			ServeStale:               cfg.ServeStale,
			BackgroundRefreshTimeout: cfg.BackgroundRefreshTimeout(),
			BackgroundRefreshBackoff: cfg.BackgroundRefreshBackoff(),
		},
		Negative: fetcher.NegativeConfig{
			Enabled:    cfg.Negative.Enabled,
			Type:       cfg.Negative.Type,
			MaxEntries: cfg.Negative.MaxEntries,
			TTL:        cfg.Negative.TTL(),
		},
		CoalesceRequests: cfg.CoalesceRequests,
	}
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

func (r metricsRecorder) BackendFetch(operation string, result string, d time.Duration) {
	var mappedOperation metrics.FetcherOperation
	switch operation {
	case "start":
		mappedOperation = metrics.FetcherOperationStart
	case "background_refresh":
		mappedOperation = metrics.FetcherOperationBackgroundRefresh
	default:
		mappedOperation = metrics.FetcherOperationGet
	}
	var mapped metrics.FetcherBackendResult
	switch result {
	case "ok":
		mapped = metrics.FetcherBackendOK
	case "notfound":
		mapped = metrics.FetcherBackendNotFound
	default:
		mapped = metrics.FetcherBackendError
	}
	r.engine.RecordFetcherBackendFetch(r.subsystem, mappedOperation, mapped, d)
}

// accountSource is the account-specific raw source stage used by the generic
// fetcher engine. A source-level not-found becomes found=false; systemic errors
// are returned so they are not cached as definitive misses.
type accountSource struct {
	source Source
}

func (s accountSource) Fetch(ctx context.Context, key string) (json.RawMessage, bool, error) {
	raw, err := s.source.Fetch(ctx, key)
	if err != nil {
		if errors.Is(err, errAccountNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return raw, true, nil
}

// accountBulkSource adapts an account BulkSource into a fetcher.BulkSource.
// It returns the raw, unmerged account rows as-is; the shared transform applies defaults
// downstream, so this path and the single-key path merge in exactly one place.
type accountBulkSource struct {
	source BulkSource
}

func (s accountBulkSource) Fetch(ctx context.Context, key string) (json.RawMessage, bool, error) {
	return accountSource{source: s.source}.Fetch(ctx, key)
}

func (s accountBulkSource) FetchAll(ctx context.Context) (map[string]json.RawMessage, error) {
	return s.source.FetchAll(ctx)
}

// newAccountTransform returns the single normalization step for accounts. It
// unmarshals account defaults, overlays the raw row, unpacks DSA defaults, fills
// the ID, and computes the derived + IP-masking config. It runs once per id at
// cache insert, so the per-request read path does no unmarshal, derive or IP
// masking. Both the single-key and bulk sources feed it raw, unmerged rows, so
// defaults are applied here in exactly one place.
func newAccountTransform(defaults json.RawMessage) fetcher.TransformFunc[string, *config.Account] {
	return func(accountID string, raw json.RawMessage) (*config.Account, error) {
		account := &config.Account{}
		if defaults != nil {
			if err := jsonutil.MergeClone(account, defaults); err != nil {
				return nil, &errortypes.MalformedAcct{
					Message: fmt.Sprintf("The prebid-server account config for account id \"%s\" is malformed. Please reach out to the prebid server host.", accountID),
				}
			}
		}
		if err := jsonutil.MergeClone(account, raw); err != nil {
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
