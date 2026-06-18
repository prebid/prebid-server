package doohcreativeapproval

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"

	"github.com/prebid/prebid-server/v4/exchange/entities"
	"github.com/prebid/prebid-server/v4/hooks/hookexecution"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

const activeContextKey = "doohcreativeapproval.active"

var _ hookstage.ProcessedAuctionRequest = (*Module)(nil)
var _ hookstage.AllProcessedBidResponses = (*Module)(nil)

func Builder(rawConfig json.RawMessage, deps moduledeps.ModuleDeps) (interface{}, error) {
	cfg, err := parseModuleConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	client := deps.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	return &Module{
		cfg:      cfg,
		provider: newHTTPApprovalProvider(client),
		cache:    newApprovalCache(cfg.CacheSizeBytes),
	}, nil
}

type Module struct {
	cfg      moduleConfig
	provider approvalProvider
	cache    *approvalCache
}

func (m *Module) HandleProcessedAuctionHook(
	_ context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.ProcessedAuctionRequestPayload,
) (hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], error) {
	result := hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]{}

	cfg, err := applyAccountConfig(m.cfg, miCtx.AccountConfig)
	if err != nil {
		return result, hookexecution.NewFailure(err.Error())
	}
	if !cfg.Enabled {
		return result, nil
	}
	if cfg.Endpoint == "" {
		result.Warnings = append(result.Warnings, "DOOH creative approval endpoint is not configured")
		return result, nil
	}
	if payload.Request == nil || payload.Request.BidRequest == nil || payload.Request.DOOH == nil {
		return result, nil
	}

	moduleContext := hookstage.NewModuleContext()
	moduleContext.Set(activeContextKey, true)
	result.ModuleContext = moduleContext
	return result, nil
}

func (m *Module) HandleAllProcessedBidResponsesHook(
	ctx context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.AllProcessedBidResponsesPayload,
) (hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload], error) {
	result := hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload]{}
	if !isModuleContextActive(miCtx.ModuleContext) {
		return result, nil
	}

	cfg, err := applyAccountConfig(m.cfg, miCtx.AccountConfig)
	if err != nil {
		return result, hookexecution.NewFailure(err.Error())
	}
	if !cfg.Enabled || cfg.Endpoint == "" || len(payload.Responses) == 0 {
		return result, nil
	}

	statuses, warnings := m.resolveApprovalStatuses(ctx, cfg, miCtx.AccountID, payload.Responses)
	result.Warnings = append(result.Warnings, warnings...)
	if !needsApprovalFilter(payload.Responses, cfg, miCtx.AccountID, statuses) {
		return result, nil
	}

	changeSet := hookstage.ChangeSet[hookstage.AllProcessedBidResponsesPayload]{}
	changeSet.AddMutation(func(payload hookstage.AllProcessedBidResponsesPayload) (hookstage.AllProcessedBidResponsesPayload, error) {
		filterResponsesByApproval(payload.Responses, cfg, miCtx.AccountID, statuses)
		return payload, nil
	}, hookstage.MutationUpdate, "responses", "bids")
	result.ChangeSet = changeSet
	return result, nil
}

func isModuleContextActive(moduleContext *hookstage.ModuleContext) bool {
	activeValue, ok := moduleContext.Get(activeContextKey)
	if !ok {
		return false
	}
	active, ok := activeValue.(bool)
	return ok && active
}

func (m *Module) resolveApprovalStatuses(
	ctx context.Context,
	cfg moduleConfig,
	accountID string,
	responses map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid,
) (map[string]approvalStatus, []string) {
	statuses := make(map[string]approvalStatus)
	creativesByID, warnings := collectCreativeApprovals(responses, cfg, accountID)
	if len(creativesByID) == 0 {
		return statuses, warnings
	}

	ids := make([]string, 0, len(creativesByID))
	for id := range creativesByID {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	cachedLookups := make(map[string]cachedApprovalLookup)
	refreshCreatives := make([]creativeApproval, 0, len(ids))
	for _, id := range ids {
		if cached, ok := m.cache.get(id); ok {
			statuses[id] = cached.Status
			cachedLookups[id] = cached
			if !cached.RefreshDue {
				continue
			}
		}
		refreshCreatives = append(refreshCreatives, creativesByID[id])
	}
	if len(refreshCreatives) == 0 {
		return statuses, warnings
	}

	fetchedStatuses, providerWarnings, err := m.provider.Lookup(ctx, cfg, accountID, refreshCreatives)
	warnings = append(warnings, providerWarnings...)
	if err != nil {
		warnings = append(warnings, "DOOH creative approval lookup failed: "+err.Error())
		for _, creative := range refreshCreatives {
			status := approvalStatusPending
			if cached, ok := cachedLookups[creative.CreativeApprovalID]; ok {
				status = cached.Status
			}
			statuses[creative.CreativeApprovalID] = status
			if err := m.cache.set(creative.CreativeApprovalID, status, cfg.PendingTTLSeconds); err != nil {
				warnings = append(warnings, cacheWriteWarning(creative.CreativeApprovalID, err))
			}
		}
		return statuses, warnings
	}

	for _, creative := range refreshCreatives {
		status, ok := fetchedStatuses[creative.CreativeApprovalID]
		if !ok {
			status = approvalStatusPending
			warnings = append(warnings, "DOOH creative approval response missing creative_approval_id "+creative.CreativeApprovalID)
		}
		statuses[creative.CreativeApprovalID] = status
		if err := m.cache.set(creative.CreativeApprovalID, status, ttlForStatus(cfg, status)); err != nil {
			warnings = append(warnings, cacheWriteWarning(creative.CreativeApprovalID, err))
		}
	}

	return statuses, warnings
}

func cacheWriteWarning(creativeApprovalID string, err error) string {
	return "DOOH creative approval cache write failed for creative_approval_id " + creativeApprovalID + ": " + err.Error()
}
