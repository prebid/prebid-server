// Package rtd implements the ZeroGPU Real Time Data module for Prebid Server.
//
// The module resolves the publisher domain from an incoming OpenRTB request,
// classifies it against the IAB content taxonomy using ZeroGPU's
// zlm-v1-iab-domain-classifier model, and injects the resulting categories into
// {site,app,dooh}.content.data so every bidder in the auction can read them.
//
// Classification results are cached in-process, and every failure mode is
// fail-open: a slow or unavailable ZeroGPU API leaves the auction unenriched
// but never delays or rejects it.
package rtd

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/coocood/freecache"
	"github.com/prebid/prebid-server/v4/hooks/hookanalytics"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
)

// Builder is the module entry point invoked by Prebid Server at startup.
func Builder(rawCfg json.RawMessage, deps moduledeps.ModuleDeps) (interface{}, error) {
	cfg, err := newConfig(rawCfg)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{Timeout: time.Duration(cfg.Timeout) * time.Millisecond}
	if deps.HTTPClient != nil {
		// Reuse the host's pooled transport so connections are shared.
		httpClient.Transport = deps.HTTPClient.Transport
	}

	bgCtx, bgCancel := context.WithCancel(context.Background())

	return &Module{
		cfg:        cfg,
		httpClient: httpClient,
		cache:      freecache.NewCache(cfg.CacheSize),
		bgCtx:      bgCtx,
		bgCancel:   bgCancel,
	}, nil
}

// Module implements the ZeroGPU RTD module.
type Module struct {
	cfg        Config
	httpClient *http.Client
	cache      *freecache.Cache

	// bgCtx bounds the lifetime of background cache warm-ups. It is deliberately
	// independent of any hook context, which is cancelled at the execution
	// plan's group timeout.
	bgCtx    context.Context
	bgCancel context.CancelFunc
	wg       sync.WaitGroup

	// inFlight collapses concurrent warm-ups for the same domain.
	inFlight sync.Map
}

var (
	_ hookstage.ProcessedAuctionRequest = (*Module)(nil)
	_ shutdowner                        = (*Module)(nil)
)

// shutdowner mirrors modules.Shutdowner, which the host calls on teardown.
type shutdowner interface {
	Shutdown() error
}

// Shutdown cancels any in-flight cache warm-ups and waits for them to finish.
func (m *Module) Shutdown() error {
	m.bgCancel()
	m.wg.Wait()
	return nil
}

const analyticsActivity = "zerogpu-rtd-domain-classification"

// HandleProcessedAuctionHook enriches the request from cached classifications
// and stages a mutation adding the resulting IAB segments.
//
// This stage is chosen deliberately: it is the last point at which the request
// is still shared by every bidder, stored requests have already been merged,
// and account-level config is available. Running at bidder_request instead
// would warm the same domain once per bidder.
//
// The hook context is intentionally unused. Nothing on this path performs I/O,
// so the hook cannot time out and adds no latency to the auction.
func (m *Module) HandleProcessedAuctionHook(
	_ context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.ProcessedAuctionRequestPayload,
) (hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], error) {
	var result hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]

	if payload.Request == nil || payload.Request.BidRequest == nil {
		return result, nil
	}
	if !m.cfg.AccountFilter.isAllowed(miCtx.AccountID) {
		return result, nil
	}

	domain := resolveDomain(payload.Request.BidRequest)
	if domain == "" {
		result.AnalyticsTags = skippedTags("no domain available on the request")
		return result, nil
	}

	// The cache is the only source consulted on the auction path. A miss
	// schedules a background warm-up and leaves this auction unenriched rather
	// than making bidders wait on a network round trip.
	segs, cached := m.lookup(domain)
	if !cached {
		m.warm(domain)
		result.AnalyticsTags = skippedTags("domain not yet cached; warming in the background")
		return result, nil
	}
	if segs.isEmpty() {
		result.AnalyticsTags = skippedTags("no categories available for this domain")
		return result, nil
	}

	// Enrichment is applied inside the mutation rather than here so that the
	// change is recorded in the module trace and can be reverted by core.
	result.ChangeSet.AddMutation(
		func(p hookstage.ProcessedAuctionRequestPayload) (hookstage.ProcessedAuctionRequestPayload, error) {
			m.enrich(p.Request.BidRequest, segs)
			return p, nil
		},
		hookstage.MutationAdd,
		mutationKey(payload.Request.BidRequest)...,
	)

	result.AnalyticsTags = successTags(domain, segs)
	return result, nil
}

func successTags(domain string, segs segments) hookanalytics.Analytics {
	return activity(hookanalytics.ActivityStatusSuccess, hookanalytics.ResultStatusModify, map[string]interface{}{
		"domain":            domain,
		"content_2_2_count": len(segs.Content22),
		"content_1_0_count": len(segs.Content10),
		"audience_count":    len(segs.Audience),
	})
}

func skippedTags(reason string) hookanalytics.Analytics {
	return activity(hookanalytics.ActivityStatusSuccess, hookanalytics.ResultStatusAllow, map[string]interface{}{
		"reason": reason,
	})
}

func activity(status hookanalytics.ActivityStatus, resultStatus hookanalytics.ResultStatus, values map[string]interface{}) hookanalytics.Analytics {
	return hookanalytics.Analytics{
		Activities: []hookanalytics.Activity{{
			Name:   analyticsActivity,
			Status: status,
			Results: []hookanalytics.Result{{
				Status: resultStatus,
				Values: values,
			}},
		}},
	}
}
