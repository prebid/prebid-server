package doohcreativeapproval

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilder(t *testing.T) {
	module, err := Builder(json.RawMessage(`{
		"enabled": true,
		"endpoint": "http://approval.example.com",
		"cache_size_bytes": 1048576,
		"max_concurrent_lookups": 2
	}`), moduledeps.ModuleDeps{HTTPClient: http.DefaultClient})

	require.NoError(t, err)
	creativeApprovalModule, ok := module.(*Module)
	require.True(t, ok)
	assert.Equal(t, "http://approval.example.com", creativeApprovalModule.cfg.Endpoint)
	assert.NotNil(t, creativeApprovalModule.provider)
	assert.NotNil(t, creativeApprovalModule.cache)
	assert.Equal(t, 2, cap(creativeApprovalModule.refreshes.slots))
}

func TestBuilderInvalidConfig(t *testing.T) {
	module, err := Builder(json.RawMessage(`{"platforms":["site"]}`), moduledeps.ModuleDeps{})

	require.Error(t, err)
	assert.Nil(t, module)
}

func TestModuleImplementsHooks(t *testing.T) {
	module := &Module{}

	assert.Implements(t, (*hookstage.ProcessedAuctionRequest)(nil), module)
	assert.Implements(t, (*hookstage.AllProcessedBidResponses)(nil), module)
}

func TestHandleProcessedAuctionHookActivation(t *testing.T) {
	tests := []struct {
		name          string
		module        *Module
		accountConfig json.RawMessage
		request       *openrtb_ext.RequestWrapper
		active        bool
		warnings      []string
	}{
		{
			name:    "dooh-active",
			module:  &Module{cfg: testModuleConfig()},
			request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{DOOH: &openrtb2.DOOH{ID: "screen"}}},
			active:  true,
		},
		{
			name:    "site-inactive",
			module:  &Module{cfg: testModuleConfig()},
			request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{Site: &openrtb2.Site{ID: "site"}}},
			active:  false,
		},
		{
			name:    "app-inactive",
			module:  &Module{cfg: testModuleConfig()},
			request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{App: &openrtb2.App{ID: "app"}}},
			active:  false,
		},
		{
			name:    "nil-request-inactive",
			module:  &Module{cfg: testModuleConfig()},
			request: nil,
			active:  false,
		},
		{
			name:          "account-disabled",
			module:        &Module{cfg: testModuleConfig()},
			accountConfig: testAccountConfig(t, `{"enabled":false}`),
			request:       &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{DOOH: &openrtb2.DOOH{ID: "screen"}}},
			active:        false,
		},
		{
			name:     "missing-endpoint",
			module:   &Module{cfg: moduleConfig{Enabled: true, Platforms: []string{defaultPlatformDOOH}}},
			request:  &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{DOOH: &openrtb2.DOOH{ID: "screen"}}},
			active:   false,
			warnings: []string{"DOOH creative approval endpoint is not configured"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{AccountConfig: test.accountConfig}, hookstage.ProcessedAuctionRequestPayload{Request: test.request})

			require.NoError(t, err)
			assert.Equal(t, test.warnings, result.Warnings)
			assert.Equal(t, test.active, isModuleContextActive(result.ModuleContext))
		})
	}
}

func TestHandleProcessedAuctionHookInvalidAccountConfig(t *testing.T) {
	module := &Module{cfg: testModuleConfig()}

	_, err := module.HandleProcessedAuctionHook(
		context.Background(),
		hookstage.ModuleInvocationContext{AccountConfig: json.RawMessage(`{"platforms":["site"]}`)},
		hookstage.ProcessedAuctionRequestPayload{Request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{DOOH: &openrtb2.DOOH{ID: "screen"}}}},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), `platforms must contain only "dooh"`)
}

func TestHandleAllProcessedBidResponsesHookNoActiveContext(t *testing.T) {
	provider := &fakeApprovalProvider{}
	module := newTestModule(provider, nil)
	payload := hookstage.AllProcessedBidResponsesPayload{Responses: testResponses(openrtb_ext.BidderName("appnexus"), testBid("cr-1"))}

	result, err := module.HandleAllProcessedBidResponsesHook(context.Background(), hookstage.ModuleInvocationContext{AccountID: "acct"}, payload)

	require.NoError(t, err)
	assert.Empty(t, result.ChangeSet.Mutations())
	assert.Equal(t, 0, provider.callCount())
}

func TestHandleAllProcessedBidResponsesHookUsesLookupResultsOnLaterAuctions(t *testing.T) {
	accountID := "acct"
	bidder := openrtb_ext.BidderName("appnexus")
	approvedID := creativeApprovalID(accountID, bidder, "approved")
	rejectedID := creativeApprovalID(accountID, bidder, "rejected")
	provider := &fakeApprovalProvider{
		statuses: map[string]approvalStatus{
			approvedID: approvalStatusApproved,
			rejectedID: approvalStatusRejected,
		},
	}
	module := newTestModule(provider, nil)

	firstPayload := hookstage.AllProcessedBidResponsesPayload{Responses: testResponses(bidder, testBid("approved"), testBid("rejected"))}
	firstResult, err := module.HandleAllProcessedBidResponsesHook(context.Background(), hookstage.ModuleInvocationContext{AccountID: accountID, ModuleContext: testActiveModuleContext()}, firstPayload)
	firstPayload = applyAllProcessedMutations(firstPayload, firstResult)

	require.NoError(t, err)
	assert.NotContains(t, firstPayload.Responses, bidder)
	waitForApprovalRefreshes(t, module)

	secondPayload := hookstage.AllProcessedBidResponsesPayload{Responses: testResponses(bidder, testBid("approved"), testBid("rejected"))}
	secondResult, err := module.HandleAllProcessedBidResponsesHook(context.Background(), hookstage.ModuleInvocationContext{AccountID: accountID, ModuleContext: testActiveModuleContext()}, secondPayload)
	secondPayload = applyAllProcessedMutations(secondPayload, secondResult)

	require.NoError(t, err)
	assert.Len(t, secondResult.ChangeSet.Mutations(), 1)
	require.Contains(t, secondPayload.Responses, bidder)
	require.Len(t, secondPayload.Responses[bidder].Bids, 1)
	assert.Equal(t, "approved", secondPayload.Responses[bidder].Bids[0].Bid.CrID)
	assert.Equal(t, 1, provider.callCount())
}

func TestHandleAllProcessedBidResponsesHookMissingResponseCachesPending(t *testing.T) {
	accountID := "acct"
	bidder := openrtb_ext.BidderName("appnexus")
	creativeID := creativeApprovalID(accountID, bidder, "missing")
	provider := &fakeApprovalProvider{}
	module := newTestModule(provider, nil)
	payload := hookstage.AllProcessedBidResponsesPayload{Responses: testResponses(bidder, testBid("missing"))}

	result, err := module.HandleAllProcessedBidResponsesHook(context.Background(), hookstage.ModuleInvocationContext{AccountID: accountID, ModuleContext: testActiveModuleContext()}, payload)
	payload = applyAllProcessedMutations(payload, result)

	require.NoError(t, err)
	assert.Len(t, result.ChangeSet.Mutations(), 1)
	assert.NotContains(t, payload.Responses, bidder)
	waitForApprovalRefreshes(t, module)
	lookup, ok := module.cache.get(creativeID)
	assert.True(t, ok)
	assert.Equal(t, approvalStatusPending, lookup.Status)
	assert.False(t, lookup.RefreshDue)
}

func TestHandleAllProcessedBidResponsesHookProviderErrorCachesPending(t *testing.T) {
	accountID := "acct"
	bidder := openrtb_ext.BidderName("appnexus")
	creativeID := creativeApprovalID(accountID, bidder, "error")
	provider := &fakeApprovalProvider{err: errApprovalProvider}
	module := newTestModule(provider, nil)
	payload := hookstage.AllProcessedBidResponsesPayload{Responses: testResponses(bidder, testBid("error"))}

	result, err := module.HandleAllProcessedBidResponsesHook(context.Background(), hookstage.ModuleInvocationContext{AccountID: accountID, ModuleContext: testActiveModuleContext()}, payload)
	payload = applyAllProcessedMutations(payload, result)

	require.NoError(t, err)
	assert.NotContains(t, payload.Responses, bidder)
	waitForApprovalRefreshes(t, module)
	lookup, ok := module.cache.get(creativeID)
	assert.True(t, ok)
	assert.Equal(t, approvalStatusPending, lookup.Status)
	assert.Equal(t, 1, provider.callCount())
}

func TestHandleAllProcessedBidResponsesHookUsesCachedApproval(t *testing.T) {
	accountID := "acct"
	bidder := openrtb_ext.BidderName("appnexus")
	creativeID := creativeApprovalID(accountID, bidder, "cached")
	provider := &fakeApprovalProvider{}
	module := newTestModule(provider, nil)
	module.cache.set(creativeID, approvalStatusApproved, 60)
	payload := hookstage.AllProcessedBidResponsesPayload{Responses: testResponses(bidder, testBid("cached"))}

	result, err := module.HandleAllProcessedBidResponsesHook(context.Background(), hookstage.ModuleInvocationContext{AccountID: accountID, ModuleContext: testActiveModuleContext()}, payload)

	require.NoError(t, err)
	assert.Empty(t, result.ChangeSet.Mutations())
	assert.Equal(t, 0, provider.callCount())
}

func TestHandleAllProcessedBidResponsesHookUsesStaleCachedApprovalOnProviderError(t *testing.T) {
	accountID := "acct"
	bidder := openrtb_ext.BidderName("appnexus")
	creativeID := creativeApprovalID(accountID, bidder, "stale-approved")
	now := time.Unix(1000, 0)
	provider := &fakeApprovalProvider{err: errApprovalProvider}
	cache := newApprovalCache(1024 * 1024)
	cache.now = func() time.Time {
		return now
	}
	module := newTestModule(provider, cache)
	require.NoError(t, module.cache.set(creativeID, approvalStatusApproved, 1))
	now = now.Add(2 * time.Second)
	payload := hookstage.AllProcessedBidResponsesPayload{Responses: testResponses(bidder, testBid("stale-approved"))}

	result, err := module.HandleAllProcessedBidResponsesHook(context.Background(), hookstage.ModuleInvocationContext{AccountID: accountID, ModuleContext: testActiveModuleContext()}, payload)

	require.NoError(t, err)
	assert.Empty(t, result.ChangeSet.Mutations())
	assert.Empty(t, result.Warnings)
	waitForApprovalRefreshes(t, module)
	assert.Equal(t, 1, provider.callCount())

	lookup, ok := module.cache.get(creativeID)
	require.True(t, ok)
	assert.Equal(t, approvalStatusApproved, lookup.Status)
	assert.False(t, lookup.RefreshDue)
}

func TestHandleAllProcessedBidResponsesHookUsesStaleStatusUntilRefreshCompletes(t *testing.T) {
	accountID := "acct"
	bidder := openrtb_ext.BidderName("appnexus")
	creativeID := creativeApprovalID(accountID, bidder, "fresh-rejected")
	now := time.Unix(1000, 0)
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	provider := &fakeApprovalProvider{
		statuses: map[string]approvalStatus{
			creativeID: approvalStatusRejected,
		},
		started: started,
		release: release,
	}
	cache := newApprovalCache(1024 * 1024)
	cache.now = func() time.Time {
		return now
	}
	module := newTestModule(provider, cache)
	require.NoError(t, module.cache.set(creativeID, approvalStatusApproved, 1))
	now = now.Add(2 * time.Second)
	firstPayload := hookstage.AllProcessedBidResponsesPayload{Responses: testResponses(bidder, testBid("fresh-rejected"))}

	firstResult, err := module.HandleAllProcessedBidResponsesHook(context.Background(), hookstage.ModuleInvocationContext{AccountID: accountID, ModuleContext: testActiveModuleContext()}, firstPayload)

	require.NoError(t, err)
	assert.Empty(t, firstResult.ChangeSet.Mutations())
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("approval refresh did not start")
	}

	secondPayload := hookstage.AllProcessedBidResponsesPayload{Responses: testResponses(bidder, testBid("fresh-rejected"))}
	secondResult, err := module.HandleAllProcessedBidResponsesHook(context.Background(), hookstage.ModuleInvocationContext{AccountID: accountID, ModuleContext: testActiveModuleContext()}, secondPayload)
	require.NoError(t, err)
	assert.Empty(t, secondResult.ChangeSet.Mutations())
	assert.Equal(t, 1, provider.callCount())

	close(release)
	waitForApprovalRefreshes(t, module)

	thirdPayload := hookstage.AllProcessedBidResponsesPayload{Responses: testResponses(bidder, testBid("fresh-rejected"))}
	thirdResult, err := module.HandleAllProcessedBidResponsesHook(context.Background(), hookstage.ModuleInvocationContext{AccountID: accountID, ModuleContext: testActiveModuleContext()}, thirdPayload)
	thirdPayload = applyAllProcessedMutations(thirdPayload, thirdResult)
	require.NoError(t, err)
	assert.NotContains(t, thirdPayload.Responses, bidder)

	lookup, ok := module.cache.get(creativeID)
	require.True(t, ok)
	assert.Equal(t, approvalStatusRejected, lookup.Status)
	assert.False(t, lookup.RefreshDue)
}

func TestHandleAllProcessedBidResponsesHookKeepsStaleStatusForIncompleteRefresh(t *testing.T) {
	accountID := "acct"
	bidder := openrtb_ext.BidderName("appnexus")
	creativeID := creativeApprovalID(accountID, bidder, "stale-approved")
	now := time.Unix(1000, 0)
	cache := newApprovalCache(1024 * 1024)
	cache.now = func() time.Time { return now }
	module := newTestModule(&fakeApprovalProvider{}, cache)
	require.NoError(t, module.cache.set(creativeID, approvalStatusApproved, 1))
	now = now.Add(2 * time.Second)
	payload := hookstage.AllProcessedBidResponsesPayload{Responses: testResponses(bidder, testBid("stale-approved"))}

	result, err := module.HandleAllProcessedBidResponsesHook(context.Background(), hookstage.ModuleInvocationContext{AccountID: accountID, ModuleContext: testActiveModuleContext()}, payload)

	require.NoError(t, err)
	assert.Empty(t, result.ChangeSet.Mutations())
	waitForApprovalRefreshes(t, module)
	lookup, ok := module.cache.get(creativeID)
	require.True(t, ok)
	assert.Equal(t, approvalStatusApproved, lookup.Status)
	assert.False(t, lookup.RefreshDue)
}

func TestHandleAllProcessedBidResponsesHookCoalescesConcurrentMisses(t *testing.T) {
	const requests = 20
	accountID := "acct"
	bidder := openrtb_ext.BidderName("appnexus")
	creativeID := creativeApprovalID(accountID, bidder, "same")
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	provider := &fakeApprovalProvider{
		statuses: map[string]approvalStatus{creativeID: approvalStatusApproved},
		started:  started,
		release:  release,
	}
	module := newTestModule(provider, nil)

	var wg sync.WaitGroup
	errs := make(chan error, requests)
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			payload := hookstage.AllProcessedBidResponsesPayload{Responses: testResponses(bidder, testBid("same"))}
			result, err := module.HandleAllProcessedBidResponsesHook(context.Background(), hookstage.ModuleInvocationContext{AccountID: accountID, ModuleContext: testActiveModuleContext()}, payload)
			if err != nil {
				errs <- err
				return
			}
			payload = applyAllProcessedMutations(payload, result)
			if _, ok := payload.Responses[bidder]; ok {
				errs <- fmt.Errorf("first-seen creative was not suppressed")
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("approval refresh did not start")
	}
	assert.Equal(t, 1, provider.callCount())
	close(release)
	waitForApprovalRefreshes(t, module)
}

func TestHandleAllProcessedBidResponsesHookSuppressesUnknownWhenCacheWriteFails(t *testing.T) {
	accountID := "acct"
	bidder := openrtb_ext.BidderName("appnexus")
	creativeID := creativeApprovalID(accountID, bidder, "approved")
	provider := &fakeApprovalProvider{
		statuses: map[string]approvalStatus{
			creativeID: approvalStatusApproved,
		},
	}
	cache := newApprovalCache(1024 * 1024)
	cache.marshal = func(v any) ([]byte, error) {
		return nil, errApprovalProvider
	}
	module := newTestModule(provider, cache)
	payload := hookstage.AllProcessedBidResponsesPayload{Responses: testResponses(bidder, testBid("approved"))}

	result, err := module.HandleAllProcessedBidResponsesHook(context.Background(), hookstage.ModuleInvocationContext{AccountID: accountID, ModuleContext: testActiveModuleContext()}, payload)
	payload = applyAllProcessedMutations(payload, result)

	require.NoError(t, err)
	assert.NotContains(t, payload.Responses, bidder)
	assert.Empty(t, result.Warnings)
	waitForApprovalRefreshes(t, module)
	assert.Equal(t, 1, provider.callCount())
	_, ok := module.cache.get(creativeID)
	assert.False(t, ok)
}
