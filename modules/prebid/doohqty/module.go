package doohqty

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/prebid-server/v4/hooks/hookexecution"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
)

var _ hookstage.ProcessedAuctionRequest = (*Module)(nil)

func Builder(rawConfig json.RawMessage, deps moduledeps.ModuleDeps) (interface{}, error) {
	cfg, err := parseModuleConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	transport := http.DefaultTransport
	if deps.HTTPClient != nil && deps.HTTPClient.Transport != nil {
		transport = deps.HTTPClient.Transport
	}

	client := &http.Client{
		Transport: transport,
	}

	return &Module{
		cfg:          cfg,
		provider:     newHTTPValueProvider(client),
		requestCache: newValueCache(cfg.CacheSizeBytes),
		csvSource:    newCSVSnapshotSource(context.Background(), client),
	}, nil
}

type Module struct {
	cfg          moduleConfig
	provider     valueProvider
	requestCache *valueCache
	csvSource    *csvSnapshotSource
}

func (m *Module) HandleProcessedAuctionHook(
	ctx context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.ProcessedAuctionRequestPayload,
) (hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], error) {
	result := hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]{}

	if payload.Request == nil || payload.Request.BidRequest == nil {
		return result, hookexecution.NewFailure("payload contains a nil bid request")
	}

	if payload.Request.DOOH == nil {
		return result, nil
	}

	cfg, err := applyAccountConfig(m.cfg, miCtx.AccountConfig)
	if err != nil {
		return result, hookexecution.NewFailure(err.Error())
	}
	if !cfg.Enabled {
		return result, nil
	}
	if cfg.Source.Endpoint == "" {
		result.Warnings = append(result.Warnings, "DOOH qty source endpoint is not configured")
		return result, nil
	}

	assignments, uniqueLookups, warnings := resolveImpressionLookups(payload.Request, miCtx.AccountID, cfg.LookupPaths)
	result.Warnings = append(result.Warnings, warnings...)
	if len(uniqueLookups) == 0 {
		return result, nil
	}
	if !hasImpressionNeedingQty(payload.Request, assignments, cfg.OverwritePolicy) {
		return result, nil
	}

	values, lookupWarnings := m.lookupValues(ctx, cfg, miCtx.AccountID, uniqueLookups)
	result.Warnings = append(result.Warnings, lookupWarnings...)
	if len(values) == 0 {
		return result, nil
	}

	if !hasApplicableQtyMutation(payload.Request, assignments, values, cfg.OverwritePolicy) {
		return result, nil
	}

	changeSet := hookstage.ChangeSet[hookstage.ProcessedAuctionRequestPayload]{}
	changeSet.AddMutation(func(payload hookstage.ProcessedAuctionRequestPayload) (hookstage.ProcessedAuctionRequestPayload, error) {
		if payload.Request == nil || payload.Request.BidRequest == nil {
			return payload, fmt.Errorf("payload contains a nil bid request")
		}

		currentAssignments, _, _ := resolveImpressionLookups(payload.Request, miCtx.AccountID, cfg.LookupPaths)
		applyQtyValues(payload.Request, currentAssignments, values, cfg.OverwritePolicy)
		return payload, nil
	}, hookstage.MutationUpdate, "bidrequest", "imp", "qty")

	result.ChangeSet = changeSet
	return result, nil
}

func (m *Module) lookupValues(ctx context.Context, cfg moduleConfig, accountID string, lookups []lookupKey) (map[lookupKey]impressionValue, []string) {
	switch cfg.Source.Type {
	case sourceTypeCSVSnapshot:
		return m.csvSource.Lookup(cfg, accountID, lookups)
	case sourceTypeRequestLookup:
		return m.lookupRequestValues(ctx, cfg, accountID, lookups)
	default:
		return nil, []string{fmt.Sprintf("DOOH qty source type %q is not supported", cfg.Source.Type)}
	}
}

func (m *Module) lookupRequestValues(ctx context.Context, cfg moduleConfig, accountID string, lookups []lookupKey) (map[lookupKey]impressionValue, []string) {
	values := make(map[lookupKey]impressionValue, len(lookups))
	warnings := make([]string, 0)
	uncachedLookups := make([]lookupKey, 0, len(lookups))

	for _, lookup := range lookups {
		value, found, cached := m.requestCache.get(lookup)
		if !cached {
			uncachedLookups = append(uncachedLookups, lookup)
			continue
		}
		if found {
			values[lookup] = value
		} else {
			warnings = append(warnings, fmt.Sprintf("no DOOH qty found for %s=%q", lookup.Path, lookup.Key))
		}
	}

	if len(uncachedLookups) == 0 {
		return values, warnings
	}

	fetchedValues, providerWarnings, err := m.provider.Lookup(ctx, cfg, accountID, uncachedLookups)
	warnings = append(warnings, providerWarnings...)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("DOOH qty lookup failed: %s", err))
		return values, warnings
	}

	for _, lookup := range uncachedLookups {
		value, ok := fetchedValues[lookup]
		if !ok {
			m.requestCache.setMissWithTTL(lookup, cfg.NegativeCacheTTLSeconds)
			warnings = append(warnings, fmt.Sprintf("no DOOH qty found for %s=%q", lookup.Path, lookup.Key))
			continue
		}

		if err := validateImpressionValue(value); err != nil {
			m.requestCache.setMissWithTTL(lookup, cfg.NegativeCacheTTLSeconds)
			warnings = append(warnings, fmt.Sprintf("DOOH qty skipped for %s=%q: %s", lookup.Path, lookup.Key, err))
			continue
		}

		values[lookup] = value
		m.requestCache.setValueWithTTL(lookup, value, cfg.CacheTTLSeconds)
	}

	return values, warnings
}

func (m *Module) Shutdown() error {
	if m.csvSource != nil {
		m.csvSource.Shutdown()
	}
	return nil
}
