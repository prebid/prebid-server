package validation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	vastlint "github.com/aleksUIX/vastlint-go"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/hooks/hookanalytics"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeEngine returns canned verdicts keyed by markup, or an error.
type fakeEngine struct {
	verdicts map[string]*vastlint.Result
	err      error
	calls    atomic.Int32
}

func (f *fakeEngine) Validate(_ context.Context, xml string, _ vastlint.Options) (*vastlint.Result, error) {
	f.calls.Add(1)
	if f.err != nil {
		return nil, f.err
	}
	if v, ok := f.verdicts[xml]; ok {
		return v, nil
	}
	return &vastlint.Result{DocumentType: "VAST", InputStatus: "supported", Valid: true}, nil
}
func (f *fakeEngine) Rules() []vastlint.Rule { return nil }
func (f *fakeEngine) Version() string        { return vastlint.ExpectedVersion }
func (f *fakeEngine) Kind() string           { return "fake" }
func (f *fakeEngine) Close() error           { return nil }

const brokenVAST = `<VAST version="4.2"><Ad><InLine></InLine></Ad></VAST>`
const cleanVAST = `<VAST version="4.2"><Ad id="1"><InLine><AdSystem>x</AdSystem><AdTitle>t</AdTitle><Impression><![CDATA[https://e.example/i]]></Impression><Creatives/></InLine></Ad></VAST>`

func brokenVerdict() *vastlint.Result {
	return &vastlint.Result{DocumentType: "VAST", InputStatus: "supported", Valid: false,
		Summary: vastlint.Summary{Errors: 1},
		Issues:  []vastlint.Issue{{ID: "VAST-2.0-inline-impression", Severity: "error"}}}
}

func videoBid(id, adm string) *adapters.TypedBid {
	return &adapters.TypedBid{Bid: &openrtb2.Bid{ID: id, ImpID: "imp-1", AdM: adm, MType: openrtb2.MarkupVideo}, BidType: "video"}
}

func run(t *testing.T, m *Module, account string, bids ...*adapters.TypedBid) hookstage.HookResult[hookstage.RawBidderResponsePayload] {
	t.Helper()
	payload := hookstage.RawBidderResponsePayload{Bidder: "adprime", BidderResponse: &adapters.BidderResponse{Bids: bids}}
	res, err := m.HandleRawBidderResponseHook(context.Background(), hookstage.ModuleInvocationContext{AccountConfig: json.RawMessage(account)}, payload)
	require.NoError(t, err)
	return res
}

func applyChanges(t *testing.T, res hookstage.HookResult[hookstage.RawBidderResponsePayload], payload hookstage.RawBidderResponsePayload) hookstage.RawBidderResponsePayload {
	t.Helper()
	for _, mut := range res.ChangeSet.Mutations() {
		var err error
		payload, err = mut.Apply(payload)
		require.NoError(t, err)
	}
	return payload
}

// waitFor polls cond until it holds or two seconds pass.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() && time.Now().Before(deadline) {
		time.Sleep(2 * time.Millisecond)
	}
	assert.True(t, cond(), "condition not met in time")
}

func cfgWith(mode string, rejectOn ...string) Config {
	t := true
	return Config{Mode: mode, RejectOn: rejectOn, MaxDocumentBytes: 256 << 10, SamplePercent: 100, AnnotateBids: &t, AnnotateRules: &t}
}

func TestMonitorAnnotatesAndNeverDrops(t *testing.T) {
	eng := &fakeEngine{verdicts: map[string]*vastlint.Result{brokenVAST: brokenVerdict()}}
	m := newWithEngine(cfgWith(modeMonitor), eng, nil)
	bids := []*adapters.TypedBid{videoBid("b1", brokenVAST), videoBid("b2", cleanVAST)}
	payload := hookstage.RawBidderResponsePayload{Bidder: "adprime", BidderResponse: &adapters.BidderResponse{Bids: bids}}
	res := run(t, m, "", bids...)
	assert.False(t, res.Reject)
	assert.Empty(t, res.Errors)
	out := applyChanges(t, res, payload)
	require.Len(t, out.BidderResponse.Bids, 2, "monitor mode keeps every bid")
	var ext struct {
		Vastlint bidSummary `json:"vastlint"`
	}
	require.NoError(t, json.Unmarshal(out.BidderResponse.Bids[0].Bid.Ext, &ext))
	assert.False(t, ext.Vastlint.Valid)
	assert.Equal(t, []string{"VAST-2.0-inline-impression"}, ext.Vastlint.Rules)
	assert.Equal(t, vastlint.ExpectedVersion, ext.Vastlint.EngineVersion)
	// Originals untouched: the change flows through the change set only.
	assert.Empty(t, bids[0].Bid.Ext)
	require.Len(t, res.AnalyticsTags.Activities, 1)
	assert.Equal(t, 2, len(res.AnalyticsTags.Activities[0].Results))
	assert.Equal(t, actionChecked, res.AnalyticsTags.Activities[0].Results[0].Values["action"])
	assert.Equal(t, 2, int(eng.calls.Load()))
}

func TestRejectDropsOnlyConfiguredRules(t *testing.T) {
	eng := &fakeEngine{verdicts: map[string]*vastlint.Result{brokenVAST: brokenVerdict()}}
	m := newWithEngine(cfgWith(modeReject, "VAST-2.0-inline-impression"), eng, nil)
	bids := []*adapters.TypedBid{videoBid("b1", brokenVAST), videoBid("b2", cleanVAST)}
	payload := hookstage.RawBidderResponsePayload{Bidder: "adprime", BidderResponse: &adapters.BidderResponse{Bids: bids}}
	res := run(t, m, "", bids...)
	out := applyChanges(t, res, payload)
	require.Len(t, out.BidderResponse.Bids, 1)
	assert.Equal(t, "b2", out.BidderResponse.Bids[0].Bid.ID)
	assert.Equal(t, hookanalytics.ResultStatusBlock, res.AnalyticsTags.Activities[0].Results[0].Status)
	assert.Contains(t, res.DebugMessages[0], "dropped bid b1")

	// Reject mode with a different rule list keeps the bid.
	m2 := newWithEngine(cfgWith(modeReject, "VAST-2.0-inline-adsystem"), eng, nil)
	res2 := run(t, m2, "", bids...)
	out2 := applyChanges(t, res2, payload)
	assert.Len(t, out2.BidderResponse.Bids, 2)
}

func TestAccountConfigOverridesMode(t *testing.T) {
	eng := &fakeEngine{verdicts: map[string]*vastlint.Result{brokenVAST: brokenVerdict()}}
	m := newWithEngine(cfgWith(modeMonitor), eng, nil)
	bids := []*adapters.TypedBid{videoBid("b1", brokenVAST)}
	payload := hookstage.RawBidderResponsePayload{Bidder: "adprime", BidderResponse: &adapters.BidderResponse{Bids: bids}}
	res := run(t, m, `{"mode":"reject","reject_on":["VAST-2.0-inline-impression"]}`, bids...)
	out := applyChanges(t, res, payload)
	assert.Len(t, out.BidderResponse.Bids, 0)
	// Invalid account config: warning, host config used.
	res = run(t, m, `{"mode":"nuke"}`, bids...)
	assert.Len(t, res.Warnings, 1)
	assert.Len(t, applyChanges(t, res, payload).BidderResponse.Bids, 1)
}

func TestSkipsAndEngineFailuresKeepBids(t *testing.T) {
	eng := &fakeEngine{err: errors.New("engine exploded")}
	cfg := cfgWith(modeReject, "VAST-2.0-inline-impression")
	cfg.MaxDocumentBytes = 10
	m := newWithEngine(cfg, eng, nil)
	bids := []*adapters.TypedBid{
		videoBid("url", "https://cdn.example/vast.xml"),
		videoBid("html", "<div>not vast</div>"),
		videoBid("big", brokenVAST),
		{Bid: &openrtb2.Bid{ID: "banner", AdM: "<div/>", MType: openrtb2.MarkupBanner}, BidType: "banner"},
		videoBid("empty", ""),
	}
	payload := hookstage.RawBidderResponsePayload{Bidder: "adprime", BidderResponse: &adapters.BidderResponse{Bids: bids}}
	res := run(t, m, "", bids...)
	out := applyChanges(t, res, payload)
	assert.Len(t, out.BidderResponse.Bids, 5, "nothing is dropped without a verdict")
	actions := map[string]string{}
	for _, r := range res.AnalyticsTags.Activities[0].Results {
		actions[r.AppliedTo.BidIds[0]] = r.Values["action"].(string)
	}
	assert.Equal(t, actionSkippedURL, actions["url"])
	assert.Equal(t, actionSkippedSize, actions["big"])
	assert.Equal(t, int32(0), eng.calls.Load(), "nothing eligible reached the engine")

	// Engine failure on an eligible bid: error reported, bid kept, no drop.
	cfg.MaxDocumentBytes = 1 << 20
	m = newWithEngine(cfg, eng, nil)
	bids = []*adapters.TypedBid{videoBid("b1", brokenVAST)}
	payload = hookstage.RawBidderResponsePayload{Bidder: "adprime", BidderResponse: &adapters.BidderResponse{Bids: bids}}
	res = run(t, m, "", bids...)
	assert.Len(t, applyChanges(t, res, payload).BidderResponse.Bids, 1)
	assert.Empty(t, res.Errors, "the hook itself succeeded")
	assert.Len(t, res.Warnings, 1, "one warning per response, not per bid")
	assert.Equal(t, hookanalytics.ResultStatusError, res.AnalyticsTags.Activities[0].Results[0].Status)
}

func TestSamplingAppliesOnlyInMonitorMode(t *testing.T) {
	eng := &fakeEngine{verdicts: map[string]*vastlint.Result{brokenVAST: brokenVerdict()}}
	cfg := cfgWith(modeMonitor)
	cfg.SamplePercent = 10
	m := newWithEngine(cfg, eng, func() int { return 50 }) // always above the sample threshold
	res := run(t, m, "", videoBid("b1", brokenVAST))
	assert.Equal(t, actionSampledOut, res.AnalyticsTags.Activities[0].Results[0].Values["action"])
	assert.Equal(t, 0, int(eng.calls.Load()))
	cfg.Mode = modeReject
	cfg.RejectOn = []string{"VAST-2.0-inline-impression"}
	m = newWithEngine(cfg, eng, func() int { return 50 })
	run(t, m, "", videoBid("b1", brokenVAST))
	assert.Equal(t, int32(1), eng.calls.Load(), "reject mode checks every bid")
}

func TestAsyncModeQueuesAndLeavesBidsUntouched(t *testing.T) {
	eng := &fakeEngine{verdicts: map[string]*vastlint.Result{brokenVAST: brokenVerdict()}}
	m := newWithEngine(cfgWith(modeAsync), eng, nil)
	defer m.async.close()
	res := run(t, m, "", videoBid("b1", brokenVAST), videoBid("b2", cleanVAST), videoBid("b3", "https://cdn.example/t.xml"))
	assert.Empty(t, res.ChangeSet.Mutations(), "async mode never changes bids")
	assert.Empty(t, res.Errors)
	require.Len(t, res.AnalyticsTags.Activities, 1)
	actions := map[string]int{}
	for _, r := range res.AnalyticsTags.Activities[0].Results {
		actions[r.Values["action"].(string)]++
	}
	assert.Equal(t, map[string]int{actionQueued: 2, actionSkippedURL: 1}, actions)
	waitFor(t, func() bool { return !(m.async.validated.Load() < 2) })
	assert.Equal(t, int64(2), m.async.validated.Load(), "both creatives validated off the path")
	assert.Equal(t, int64(1), m.async.invalid.Load())
	assert.Equal(t, int32(2), eng.calls.Load())
}

func TestAsyncQueueFullIsCountedNotBlocking(t *testing.T) {
	slow := &blockingEngine{release: make(chan struct{})}
	m := newWithEngine(cfgWith(modeAsync), slow, nil)
	m.async.close()
	m.async = newAsyncValidator(slow, nil, 1, 1, 1<<20)
	defer func() { close(slow.release); m.async.close() }()
	var bids []*adapters.TypedBid
	for i := 0; i < 6; i++ {
		bids = append(bids, videoBid("b", cleanVAST))
	}
	done := make(chan struct{})
	go func() { run(t, m, "", bids...); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("hook blocked on a full async queue")
	}
	assert.Greater(t, m.async.full.Load(), int64(0))
}

// blockingEngine holds every validation until released, to fill a queue. When
// entered is set, each call announces itself on it before blocking.
type blockingEngine struct {
	fakeEngine
	release chan struct{}
	entered chan struct{}
}

func (b *blockingEngine) Validate(ctx context.Context, xml string, o vastlint.Options) (*vastlint.Result, error) {
	select {
	case b.entered <- struct{}{}:
	default:
	}
	<-b.release
	return b.fakeEngine.Validate(ctx, xml, o)
}

func TestCacheServesRepeatedCreativesAndKeysOnRules(t *testing.T) {
	eng := &fakeEngine{verdicts: map[string]*vastlint.Result{brokenVAST: brokenVerdict()}}
	cfg := cfgWith(modeMonitor)
	cfg.CacheSize = 1 << 20
	cfg.CacheTTLSeconds = 60
	m := newWithEngine(cfg, eng, nil)

	first := run(t, m, "", videoBid("b1", brokenVAST))
	second := run(t, m, "", videoBid("b2", brokenVAST))
	assert.Equal(t, int32(1), eng.calls.Load(), "the second creative is served from the cache")
	assert.Equal(t, "miss", first.AnalyticsTags.Activities[0].Results[0].Values["cache"])
	res := second.AnalyticsTags.Activities[0].Results[0]
	assert.Equal(t, "hit", res.Values["cache"])
	assert.Equal(t, actionChecked, res.Values["action"])
	assert.Equal(t, false, res.Values["valid"])
	assert.Equal(t, []string{"VAST-2.0-inline-impression"}, res.Values["rules"])
	payload := applyChanges(t, second, hookstage.RawBidderResponsePayload{Bidder: "adprime", BidderResponse: &adapters.BidderResponse{Bids: []*adapters.TypedBid{videoBid("b2", brokenVAST)}}})
	assert.Contains(t, string(payload.BidderResponse.Bids[0].Bid.Ext), `"vastlint"`, "a cached verdict still annotates the bid")

	run(t, m, `{"rules":{"VAST-2.0-inline-impression":"off"}}`, videoBid("b3", brokenVAST))
	assert.Equal(t, int32(2), eng.calls.Load(), "different rules mean a different key")
	assert.Equal(t, int64(1), m.cache.hits.Load())
	assert.Equal(t, int64(2), m.cache.misses.Load())

	// Reject mode uses the cached verdict too.
	cfg2 := cfgWith(modeReject, "VAST-2.0-inline-impression")
	m2 := newWithEngine(cfg2, eng, nil)
	m2.cache = m.cache
	dropped := run(t, m2, "", videoBid("b4", brokenVAST))
	assert.Equal(t, int32(2), eng.calls.Load())
	assert.Len(t, dropped.ChangeSet.Mutations(), 1)
	assert.Equal(t, hookanalytics.ResultStatusBlock, dropped.AnalyticsTags.Activities[0].Results[0].Status)
}

func TestAsyncModeUsesTheCache(t *testing.T) {
	eng := &fakeEngine{verdicts: map[string]*vastlint.Result{brokenVAST: brokenVerdict()}}
	cfg := cfgWith(modeAsync)
	cfg.CacheSize = 1 << 20
	cfg.CacheTTLSeconds = 60
	m := newWithEngine(cfg, eng, nil)
	defer m.async.close()
	run(t, m, "", videoBid("b1", brokenVAST))
	waitFor(t, func() bool { return !(m.async.validated.Load() < 1) })
	res := run(t, m, "", videoBid("b2", brokenVAST))
	r := res.AnalyticsTags.Activities[0].Results[0]
	assert.Equal(t, actionChecked, r.Values["action"], "a cached verdict is reported at once")
	assert.Equal(t, "hit", r.Values["cache"])
	assert.Empty(t, res.ChangeSet.Mutations(), "async mode still never changes bids")
	assert.Equal(t, int32(1), eng.calls.Load())
}

func TestAsyncQueueBytesBoundIsCounted(t *testing.T) {
	slow := &blockingEngine{release: make(chan struct{})}
	m := newWithEngine(cfgWith(modeAsync), slow, nil)
	m.async.close()
	m.async = newAsyncValidator(slow, nil, 1000, 1000, int64(len(cleanVAST)+10)) // room for one creative
	defer func() { close(slow.release); m.async.close() }()
	run(t, m, "", videoBid("b1", cleanVAST), videoBid("b2", cleanVAST), videoBid("b3", cleanVAST))
	assert.Equal(t, int64(1), m.async.submitted.Load())
	assert.Equal(t, int64(2), m.async.full.Load(), "creatives beyond the byte budget are counted, not held")
	assert.LessOrEqual(t, m.async.bytes.Load(), int64(len(cleanVAST)+10))
}

func TestAnnotateRulesSwitchKeepsSummaryOnly(t *testing.T) {
	eng := &fakeEngine{verdicts: map[string]*vastlint.Result{brokenVAST: brokenVerdict()}}
	cfg := cfgWith(modeMonitor)
	f := false
	cfg.AnnotateRules = &f
	m := newWithEngine(cfg, eng, nil)
	bid := videoBid("b1", brokenVAST)
	res := run(t, m, "", bid)
	payload := applyChanges(t, res, hookstage.RawBidderResponsePayload{Bidder: "adprime", BidderResponse: &adapters.BidderResponse{Bids: []*adapters.TypedBid{bid}}})
	ext := string(payload.BidderResponse.Bids[0].Bid.Ext)
	assert.Contains(t, ext, `"valid":false`)
	assert.NotContains(t, ext, `"rules"`, "rule ids stay out of the bid when annotate_rules is false")
	assert.Equal(t, []string{"VAST-2.0-inline-impression"}, res.AnalyticsTags.Activities[0].Results[0].Values["rules"], "analytics tags still carry them")
}

func TestCanceledContextKeepsBids(t *testing.T) {
	eng := &ctxEngine{}
	m := newWithEngine(cfgWith(modeReject, "VAST-2.0-inline-impression"), eng, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	bids := []*adapters.TypedBid{videoBid("b1", brokenVAST), videoBid("b2", cleanVAST)}
	payload := hookstage.RawBidderResponsePayload{Bidder: "adprime", BidderResponse: &adapters.BidderResponse{Bids: bids}}
	res, err := m.HandleRawBidderResponseHook(ctx, hookstage.ModuleInvocationContext{}, payload)
	require.NoError(t, err)
	assert.Empty(t, res.ChangeSet.Mutations(), "nothing is dropped when the engine could not run")
	assert.Empty(t, res.Errors, "the hook itself succeeded")
	assert.Len(t, res.Warnings, 1, "one warning per response, not per bid")
	for _, r := range res.AnalyticsTags.Activities[0].Results {
		assert.Equal(t, actionEngineError, r.Values["action"])
	}
}

// ctxEngine fails as the real engines do when the context is already done.
type ctxEngine struct{ fakeEngine }

func (c *ctxEngine) Validate(ctx context.Context, xml string, o vastlint.Options) (*vastlint.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return c.fakeEngine.Validate(ctx, xml, o)
}

// BenchmarkHookCacheHit measures the on-path cost of a 20-bid response when
// every verdict is cached: hashing, lookup, decoding and annotation.
func BenchmarkHookCacheHit(b *testing.B) {
	eng := &fakeEngine{}
	cfg := cfgWith(modeMonitor)
	cfg.CacheSize = 8 << 20
	cfg.CacheTTLSeconds = 60
	m := newWithEngine(cfg, eng, nil)
	adm := cleanVAST
	for len(adm) < 10<<10 {
		adm += "<!-- " + cleanVAST + " -->"
	}
	bids := make([]*adapters.TypedBid, 20)
	for i := range bids {
		bids[i] = videoBid("b", adm)
	}
	payload := hookstage.RawBidderResponsePayload{Bidder: "adprime", BidderResponse: &adapters.BidderResponse{Bids: bids}}
	m.HandleRawBidderResponseHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.HandleRawBidderResponseHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)
	}
}

func TestBuilderValidatesConfigAgainstTheRealEngine(t *testing.T) {
	_, err := Builder(json.RawMessage(`{"reject_on":["no-such-rule"]}`), moduledeps.ModuleDeps{})
	require.Error(t, err)
	_, err = Builder(json.RawMessage(`{"mode":"sideways"}`), moduledeps.ModuleDeps{})
	require.Error(t, err)
	mod, err := Builder(nil, moduledeps.ModuleDeps{})
	require.NoError(t, err, "an empty config means defaults")
	m := mod.(*Module)
	defer m.Shutdown()
	assert.Equal(t, modeMonitor, m.cfg.Mode)
	assert.Nil(t, m.cache, "the cache is off until sized")
}

func TestRealEngineEndToEnd(t *testing.T) {
	mod, err := Builder(json.RawMessage(`{"mode":"reject","reject_on":["VAST-2.0-inline-impression"],"rules":{"VAST-2.0-adsystem-no-version":"off"}}`), moduledeps.ModuleDeps{})
	require.NoError(t, err)
	m := mod.(*Module)
	defer m.Shutdown()
	fixture, err := os.ReadFile("testdata/missing_impression.xml")
	require.NoError(t, err)
	bids := []*adapters.TypedBid{videoBid("broken", string(fixture)), videoBid("clean", cleanVAST)}
	payload := hookstage.RawBidderResponsePayload{Bidder: "adprime", BidderResponse: &adapters.BidderResponse{Bids: bids}}
	res := run(t, m, "", bids...)
	out := applyChanges(t, res, payload)
	require.Len(t, out.BidderResponse.Bids, 1)
	assert.Equal(t, "clean", out.BidderResponse.Bids[0].Bid.ID)
	var ext struct {
		Vastlint bidSummary `json:"vastlint"`
	}
	require.NoError(t, json.Unmarshal(out.BidderResponse.Bids[0].Bid.Ext, &ext))
	assert.Equal(t, "supported", ext.Vastlint.InputStatus)
	assert.Equal(t, vastlint.ExpectedVersion, ext.Vastlint.EngineVersion)
}

func TestShutdownIsIdempotentAndStopsWorkers(t *testing.T) {
	eng := &fakeEngine{}
	m := newWithEngine(cfgWith(modeAsync), eng, nil)
	run(t, m, "", videoBid("b1", cleanVAST))
	require.NoError(t, m.Shutdown())
	require.NoError(t, m.Shutdown(), "a second shutdown is a no-op")
	// A hook that races the shutdown must neither block nor panic; the creative
	// is counted as not validated.
	res := run(t, m, "", videoBid("b2", cleanVAST))
	require.Len(t, res.AnalyticsTags.Activities, 1)
	assert.Equal(t, actionQueueFull, res.AnalyticsTags.Activities[0].Results[0].Values["action"])
	assert.Equal(t, int64(1), m.async.full.Load())
}

func TestAsyncCloseWaitsForInFlightValidation(t *testing.T) {
	slow := &blockingEngine{release: make(chan struct{}), entered: make(chan struct{}, 1)}
	a := newAsyncValidator(slow, nil, 1, 4, 1<<20)
	require.True(t, a.submit(cleanVAST, nil, nil))
	select {
	case <-slow.entered: // the worker has started the validation
	case <-time.After(time.Second):
		t.Fatal("worker did not pick up the queued creative")
	}
	closed := make(chan struct{})
	go func() { a.close(); close(closed) }()
	select {
	case <-closed:
		t.Fatal("close returned while a validation was in flight")
	case <-time.After(50 * time.Millisecond):
	}
	close(slow.release)
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("close did not return after the validation finished")
	}
	assert.Equal(t, int64(1), a.validated.Load())
	assert.False(t, a.submit(cleanVAST, nil, nil), "a closed validator accepts nothing")
}

func TestAccountOverridesAreValidatedAgainstTheCatalog(t *testing.T) {
	eng := &fakeEngine{verdicts: map[string]*vastlint.Result{brokenVAST: brokenVerdict()}}
	m := newWithEngine(cfgWith(modeMonitor), eng, nil)
	m.catalog = newCatalog([]string{"VAST-2.0-inline-impression"})
	for _, account := range []string{
		`{"rules":{"no-such-rule":"off"}}`,
		`{"rules":{"VAST-2.0-inline-impression":"loud"}}`,
		`{"mode":"reject","reject_on":["no-such-rule"]}`,
		`{"mode":"sideways"}`,
		`{"sample_percent":0}`,
		`{"mode":`,
	} {
		bid := videoBid("b1", brokenVAST)
		res := run(t, m, account, bid)
		require.Len(t, res.Warnings, 1, account)
		out := applyChanges(t, res, hookstage.RawBidderResponsePayload{Bidder: "adprime", BidderResponse: &adapters.BidderResponse{Bids: []*adapters.TypedBid{bid}}})
		assert.Len(t, out.BidderResponse.Bids, 1, "host config (monitor) applies when the account config is invalid: %s", account)
		assert.Equal(t, actionChecked, res.AnalyticsTags.Activities[0].Results[0].Values["action"])
	}
	res := run(t, m, `{"mode":"reject","reject_on":["VAST-2.0-inline-impression"],"annotate_rules":false}`, videoBid("b1", brokenVAST))
	assert.Empty(t, res.Warnings)
	assert.Len(t, res.ChangeSet.Mutations(), 1, "a valid account override takes effect")
}

func TestReportLoopLogsOnChangeAndStopsOnShutdown(t *testing.T) {
	eng := &fakeEngine{verdicts: map[string]*vastlint.Result{brokenVAST: brokenVerdict()}}
	cfg := cfgWith(modeAsync)
	cfg.CacheSize = 1 << 20
	cfg.CacheTTLSeconds = 60
	m := newWithEngine(cfg, eng, nil)
	run(t, m, "", videoBid("b1", brokenVAST), videoBid("b2", cleanVAST))
	stopped := make(chan struct{})
	go func() { m.report(time.Millisecond); close(stopped) }()
	time.Sleep(20 * time.Millisecond) // a few ticks: first with changes, then without
	require.NoError(t, m.Shutdown())
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("report loop did not stop on shutdown")
	}
}

func TestBuilderRejectsBadRuleLevelAndReportsEngine(t *testing.T) {
	_, err := Builder(json.RawMessage(`{"rules":{"VAST-2.0-inline-impression":"loud"}}`), moduledeps.ModuleDeps{})
	require.Error(t, err)
	_, err = Builder(json.RawMessage(`{"rules":{"no-such-rule":"off"}}`), moduledeps.ModuleDeps{})
	require.Error(t, err)
	mod, err := Builder(json.RawMessage(`{"report_interval_seconds":1,"cache_size":1048576}`), moduledeps.ModuleDeps{})
	require.NoError(t, err)
	m := mod.(*Module)
	assert.NotEmpty(t, m.catalog.ids)
	assert.NotNil(t, m.cache)
	require.NoError(t, m.Shutdown())
}

func TestAnnotateMergesAndToleratesBadExt(t *testing.T) {
	s := bidSummary{Valid: true, InputStatus: "supported", Rules: []string{"x"}, EngineVersion: "0.16.0"}
	tb := videoBid("b1", cleanVAST)
	tb.Bid.Ext = json.RawMessage(`{"prebid":{"type":"video"}}`)
	out, ok := annotate(tb, s, true)
	require.True(t, ok)
	assert.Contains(t, string(out.Bid.Ext), `"prebid":{"type":"video"}`, "existing ext is preserved")
	assert.Contains(t, string(out.Bid.Ext), `"vastlint":{`)
	assert.Equal(t, json.RawMessage(`{"prebid":{"type":"video"}}`), tb.Bid.Ext, "the original bid is untouched")
	tb.Bid.Ext = json.RawMessage(`not json`)
	same, ok := annotate(tb, s, true)
	assert.False(t, ok)
	assert.Same(t, tb, same, "a bid whose ext cannot be parsed is left alone")
}

func TestEligibilityHelpers(t *testing.T) {
	assert.True(t, isURL("https://cdn.example/t.xml"))
	assert.True(t, isURL("HTTPS://CDN.EXAMPLE/T.XML"), "scheme is matched case-insensitively")
	assert.False(t, isURL("<VAST/>"))
	assert.False(t, isURL("http"))
	assert.True(t, isVideo(&adapters.TypedBid{BidType: openrtb_ext.BidTypeVideo}))
	assert.True(t, isVideo(&adapters.TypedBid{Bid: &openrtb2.Bid{MType: openrtb2.MarkupVideo}}))
	assert.False(t, isVideo(&adapters.TypedBid{BidType: openrtb_ext.BidTypeBanner, Bid: &openrtb2.Bid{MType: openrtb2.MarkupBanner}}))

	eng := &fakeEngine{}
	m := newWithEngine(cfgWith(modeMonitor), eng, nil)
	res := run(t, m, "",
		&adapters.TypedBid{Bid: &openrtb2.Bid{ID: "banner", AdM: "<div/>", MType: openrtb2.MarkupBanner}, BidType: "banner"},
		videoBid("empty", "   "),
		nil,
		videoBid("json", `{"not":"xml"}`))
	assert.Equal(t, int32(0), eng.calls.Load())
	require.Len(t, res.AnalyticsTags.Activities, 1)
	require.Len(t, res.AnalyticsTags.Activities[0].Results, 1, "only the non-XML video bid is reported")
	assert.Equal(t, actionSkippedNotXML, res.AnalyticsTags.Activities[0].Results[0].Values["action"])
	assert.Empty(t, res.ChangeSet.Mutations())
}

// BenchmarkHookRealEngine measures the on-path cost of validating one and
// twenty distinct creatives per response with the embedded engine (or the
// native one under -tags vastlint_cgo), cache off: the per-call figure a host
// can expect from the hook itself, before any auction overhead.
func BenchmarkHookRealEngine(b *testing.B) {
	fixture, err := os.ReadFile("testdata/missing_impression.xml")
	if err != nil {
		b.Fatal(err)
	}
	mod, err := Builder(json.RawMessage(`{"report_interval_seconds":0}`), moduledeps.ModuleDeps{})
	if err != nil {
		b.Fatal(err)
	}
	m := mod.(*Module)
	defer m.Shutdown()
	for _, n := range []int{1, 20} {
		bids := make([]*adapters.TypedBid, n)
		for i := range bids {
			// Distinct markup per bid so no two bids share an engine result.
			bids[i] = videoBid("b", string(fixture)+"<!-- "+string(rune('a'+i))+" -->")
		}
		payload := hookstage.RawBidderResponsePayload{Bidder: "adprime", BidderResponse: &adapters.BidderResponse{Bids: bids}}
		b.Run(fmt.Sprintf("bids=%d/engine=%s", n, m.engine.Kind()), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := m.HandleRawBidderResponseHook(context.Background(), hookstage.ModuleInvocationContext{}, payload); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// panicEngine panics inside Validate, as a broken binding might.
type panicEngine struct{ fakeEngine }

func (p *panicEngine) Validate(context.Context, string, vastlint.Options) (*vastlint.Result, error) {
	panic("engine exploded")
}

func TestPanicInEngineIsContained(t *testing.T) {
	m := newWithEngine(cfgWith(modeReject, "VAST-2.0-inline-impression"), &panicEngine{}, nil)
	bids := []*adapters.TypedBid{videoBid("b1", brokenVAST), videoBid("b2", cleanVAST), videoBid("b3", cleanVAST)}
	res := run(t, m, "", bids...)
	assert.Empty(t, res.ChangeSet.Mutations(), "nothing is dropped when the engine panics")
	assert.Empty(t, res.Errors, "the hook itself succeeded")
	assert.Len(t, res.Warnings, 1, "one warning per response, not per bid")
	require.Len(t, res.AnalyticsTags.Activities, 1)
	assert.Equal(t, hookanalytics.ActivityStatusError, res.AnalyticsTags.Activities[0].Status, "every eligible bid failed")
	for _, r := range res.AnalyticsTags.Activities[0].Results {
		assert.Equal(t, actionEngineError, r.Values["action"])
	}
	// Off the path the panic is counted, never raised.
	ma := newWithEngine(cfgWith(modeAsync), &panicEngine{}, nil)
	defer ma.Shutdown()
	run(t, ma, "", videoBid("b1", brokenVAST))
	waitFor(t, func() bool { return !(ma.async.errors.Load() < 1) })
	assert.Equal(t, int64(1), ma.async.errors.Load())
}

func TestManyEligibleBidsKeepOrderAndAllAnnotations(t *testing.T) {
	eng := &fakeEngine{verdicts: map[string]*vastlint.Result{brokenVAST: brokenVerdict()}}
	m := newWithEngine(cfgWith(modeMonitor), eng, nil)
	m.parallel = 3
	var bids []*adapters.TypedBid
	for i := 0; i < 25; i++ {
		adm := cleanVAST
		if i%5 == 0 {
			adm = brokenVAST
		}
		bids = append(bids, videoBid(fmt.Sprintf("b%02d", i), adm))
	}
	res := run(t, m, "", bids...)
	out := applyChanges(t, res, hookstage.RawBidderResponsePayload{Bidder: "adprime", BidderResponse: &adapters.BidderResponse{Bids: bids}})
	require.Len(t, out.BidderResponse.Bids, 25)
	for i, tb := range out.BidderResponse.Bids {
		assert.Equal(t, fmt.Sprintf("b%02d", i), tb.Bid.ID, "order preserved")
		assert.Contains(t, string(tb.Bid.Ext), `"vastlint"`)
		if i%5 == 0 {
			assert.Contains(t, string(tb.Bid.Ext), `"valid":false`)
		} else {
			assert.Contains(t, string(tb.Bid.Ext), `"valid":true`)
		}
	}
	assert.Equal(t, int32(25), eng.calls.Load())
	assert.Len(t, res.AnalyticsTags.Activities[0].Results, 25)
}

func TestAnnotateBidsFalseLeavesBidsUntouched(t *testing.T) {
	eng := &fakeEngine{verdicts: map[string]*vastlint.Result{brokenVAST: brokenVerdict()}}
	cfg := cfgWith(modeMonitor)
	f := false
	cfg.AnnotateBids = &f
	m := newWithEngine(cfg, eng, nil)
	res := run(t, m, "", videoBid("b1", brokenVAST))
	assert.Empty(t, res.ChangeSet.Mutations())
	assert.Equal(t, actionChecked, res.AnalyticsTags.Activities[0].Results[0].Values["action"], "tags are still produced")
	res = run(t, m, `{"annotate_bids":true}`, videoBid("b1", brokenVAST))
	assert.Len(t, res.ChangeSet.Mutations(), 1, "an account can turn annotation back on")
}

func TestRejectOnMatchesAnyListedRule(t *testing.T) {
	eng := &fakeEngine{verdicts: map[string]*vastlint.Result{brokenVAST: brokenVerdict()}}
	m := newWithEngine(cfgWith(modeReject, "VAST-2.0-parse-error", "VAST-2.0-inline-impression", "VAST-2.0-mediafile-url"), eng, nil)
	res := run(t, m, "", videoBid("b1", brokenVAST), videoBid("b2", cleanVAST))
	out := applyChanges(t, res, hookstage.RawBidderResponsePayload{Bidder: "adprime", BidderResponse: &adapters.BidderResponse{Bids: []*adapters.TypedBid{videoBid("b1", brokenVAST), videoBid("b2", cleanVAST)}}})
	require.Len(t, out.BidderResponse.Bids, 1)
	assert.Equal(t, "b2", out.BidderResponse.Bids[0].Bid.ID)
}

// TestRace... so Prebid Server's validate.sh runs it under the race detector.
func TestRaceShutdownWhileHooksRunOnTheRealEngine(t *testing.T) {
	mod, err := Builder(json.RawMessage(`{"report_interval_seconds":0,"pool_size":2}`), moduledeps.ModuleDeps{})
	require.NoError(t, err)
	m := mod.(*Module)
	fixture, err := os.ReadFile("testdata/missing_impression.xml")
	require.NoError(t, err)
	bids := []*adapters.TypedBid{videoBid("b1", string(fixture)), videoBid("b2", cleanVAST), videoBid("b3", string(fixture))}
	payload := hookstage.RawBidderResponsePayload{Bidder: "adprime", BidderResponse: &adapters.BidderResponse{Bids: bids}}
	var wg sync.WaitGroup
	stop := make(chan struct{})
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				// Must never panic, before or after Shutdown.
				_, err := m.HandleRawBidderResponseHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)
				assert.NoError(t, err)
			}
		}()
	}
	time.Sleep(30 * time.Millisecond)
	require.NoError(t, m.Shutdown())
	time.Sleep(30 * time.Millisecond)
	close(stop)
	wg.Wait()
	res, err := m.HandleRawBidderResponseHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)
	require.NoError(t, err)
	assert.Empty(t, res.ChangeSet.Mutations(), "after shutdown bids pass through unvalidated")
}
