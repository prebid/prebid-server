package doohcreativeapproval

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/exchange/entities"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func testModuleConfig() moduleConfig {
	return moduleConfig{
		Enabled:              true,
		Platforms:            []string{defaultPlatformDOOH},
		Endpoint:             "http://approval.example.com/creative-approval",
		TimeoutMS:            defaultTimeoutMS,
		CacheSizeBytes:       defaultCacheSizeBytes,
		MaxConcurrentLookups: defaultMaxConcurrentLookups,
		ApprovedTTLSeconds:   defaultApprovedTTLSeconds,
		RejectedTTLSeconds:   defaultRejectedTTLSeconds,
		PendingTTLSeconds:    defaultPendingTTLSeconds,
	}
}

func testBid(crid string) *entities.PbsOrtbBid {
	return &entities.PbsOrtbBid{
		Bid: &openrtb2.Bid{
			ID:      "bid-" + crid,
			ImpID:   "imp-1",
			Price:   1.23,
			CrID:    crid,
			AdID:    "ad-" + crid,
			CID:     "campaign-" + crid,
			ADomain: []string{"advertiser.example"},
			Cat:     []string{"IAB1"},
			W:       1920,
			H:       1080,
			Dur:     15,
			DealID:  "deal-" + crid,
			IURL:    "https://example.com/" + crid + ".jpg",
		},
		BidType: openrtb_ext.BidTypeVideo,
	}
}

func testResponses(bidder openrtb_ext.BidderName, bids ...*entities.PbsOrtbBid) map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid {
	return map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
		bidder: {
			Bids:     bids,
			Currency: "USD",
			Seat:     bidder.String(),
		},
	}
}

func testActiveModuleContext() *hookstage.ModuleContext {
	moduleContext := hookstage.NewModuleContext()
	moduleContext.Set(activeContextKey, true)
	return moduleContext
}

func applyAllProcessedMutations(payload hookstage.AllProcessedBidResponsesPayload, result hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload]) hookstage.AllProcessedBidResponsesPayload {
	for _, mutation := range result.ChangeSet.Mutations() {
		payload, _ = mutation.Apply(payload)
	}
	return payload
}

type fakeApprovalProvider struct {
	mu        sync.Mutex
	statuses  map[string]approvalStatus
	warnings  []string
	err       error
	calls     int
	creatives []creativeApproval
	started   chan<- struct{}
	release   <-chan struct{}
}

func (p *fakeApprovalProvider) Lookup(ctx context.Context, _ moduleConfig, _ string, creatives []creativeApproval) (map[string]approvalStatus, []string, error) {
	p.mu.Lock()
	p.calls++
	p.creatives = append([]creativeApproval(nil), creatives...)
	statuses := make(map[string]approvalStatus, len(p.statuses))
	for id, status := range p.statuses {
		statuses[id] = status
	}
	warnings := append([]string(nil), p.warnings...)
	err := p.err
	started := p.started
	release := p.release
	p.mu.Unlock()

	if started != nil {
		select {
		case started <- struct{}{}:
		default:
		}
	}
	if release != nil {
		select {
		case <-release:
		case <-ctx.Done():
			return nil, warnings, ctx.Err()
		}
	}
	if err != nil {
		return nil, warnings, err
	}
	return statuses, warnings, nil
}

func (p *fakeApprovalProvider) callCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

func newTestModule(provider approvalProvider, cache *approvalCache) *Module {
	cfg := testModuleConfig()
	if cache == nil {
		cache = newApprovalCache(cfg.CacheSizeBytes)
	}
	return &Module{
		cfg:       cfg,
		provider:  provider,
		cache:     cache,
		refreshes: newApprovalRefreshCoordinator(cfg.MaxConcurrentLookups),
	}
}

func waitForApprovalRefreshes(t *testing.T, module *Module) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		module.refreshes.wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for approval refreshes")
	}
}

func testAccountConfig(t interface{ Helper() }, value string) json.RawMessage {
	t.Helper()
	return json.RawMessage(value)
}

var errApprovalProvider = errors.New("approval service unavailable")
