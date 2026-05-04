package doohimpressionvalue

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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
		Timeout:   time.Duration(cfg.TimeoutMS) * time.Millisecond,
		Transport: transport,
	}

	return &Module{
		cfg:      cfg,
		provider: newHTTPValueProvider(cfg, client),
		cache:    newValueCache(cfg.CacheSizeBytes, cfg.CacheTTLSeconds, cfg.NegativeCacheTTLSeconds),
	}, nil
}

type Module struct {
	cfg      moduleConfig
	provider valueProvider
	cache    *valueCache
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

	assignments, uniqueLookups, warnings := resolveImpressionLookups(payload.Request, miCtx.AccountID, m.cfg.LookupPaths)
	result.Warnings = append(result.Warnings, warnings...)
	if len(uniqueLookups) == 0 {
		return result, nil
	}
	if !hasImpressionNeedingQty(payload.Request, assignments, m.cfg.OverwritePolicy) {
		return result, nil
	}

	values, lookupWarnings := m.lookupValues(ctx, miCtx.AccountID, uniqueLookups)
	result.Warnings = append(result.Warnings, lookupWarnings...)
	if len(values) == 0 {
		return result, nil
	}

	if !hasApplicableQtyMutation(payload.Request, assignments, values, m.cfg.OverwritePolicy) {
		return result, nil
	}

	changeSet := hookstage.ChangeSet[hookstage.ProcessedAuctionRequestPayload]{}
	changeSet.AddMutation(func(payload hookstage.ProcessedAuctionRequestPayload) (hookstage.ProcessedAuctionRequestPayload, error) {
		if payload.Request == nil || payload.Request.BidRequest == nil {
			return payload, fmt.Errorf("payload contains a nil bid request")
		}

		currentAssignments, _, _ := resolveImpressionLookups(payload.Request, miCtx.AccountID, m.cfg.LookupPaths)
		applyQtyValues(payload.Request, currentAssignments, values, m.cfg.OverwritePolicy)
		return payload, nil
	}, hookstage.MutationUpdate, "bidrequest", "imp", "qty")

	result.ChangeSet = changeSet
	return result, nil
}

func (m *Module) lookupValues(ctx context.Context, accountID string, lookups []lookupKey) (map[lookupKey]impressionValue, []string) {
	values := make(map[lookupKey]impressionValue, len(lookups))
	warnings := make([]string, 0)
	uncachedLookups := make([]lookupKey, 0, len(lookups))

	for _, lookup := range lookups {
		value, found, cached := m.cache.get(lookup)
		if !cached {
			uncachedLookups = append(uncachedLookups, lookup)
			continue
		}
		if found {
			values[lookup] = value
		} else {
			warnings = append(warnings, fmt.Sprintf("no DOOH impression value found for %s=%q", lookup.Path, lookup.Key))
		}
	}

	if len(uncachedLookups) == 0 {
		return values, warnings
	}

	fetchedValues, providerWarnings, err := m.provider.Lookup(ctx, accountID, uncachedLookups)
	warnings = append(warnings, providerWarnings...)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("DOOH impression value lookup failed: %s", err))
		return values, warnings
	}

	for _, lookup := range uncachedLookups {
		value, ok := fetchedValues[lookup]
		if !ok {
			m.cache.setMiss(lookup)
			warnings = append(warnings, fmt.Sprintf("no DOOH impression value found for %s=%q", lookup.Path, lookup.Key))
			continue
		}

		if err := validateImpressionValue(value); err != nil {
			m.cache.setMiss(lookup)
			warnings = append(warnings, fmt.Sprintf("DOOH impression value skipped for %s=%q: %s", lookup.Path, lookup.Key, err))
			continue
		}

		values[lookup] = value
		m.cache.setValue(lookup, value)
	}

	return values, warnings
}
