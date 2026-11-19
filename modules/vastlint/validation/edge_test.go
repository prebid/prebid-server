package validation

// Edge cases at the boundaries of eligibility, annotation and configuration,
// most of them against the real engine.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	vastlint "github.com/aleksUIX/vastlint-go"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func realModule(t *testing.T, cfg string) *Module {
	t.Helper()
	mod, err := Builder(json.RawMessage(cfg), moduledeps.ModuleDeps{})
	require.NoError(t, err)
	m := mod.(*Module)
	t.Cleanup(func() { m.Shutdown() })
	return m
}

func actionsOf(res hookstage.HookResult[hookstage.RawBidderResponsePayload]) map[string]string {
	out := map[string]string{}
	for id, v := range valuesOf(res) {
		out[id] = v["action"].(string)
	}
	return out
}

func valuesOf(res hookstage.HookResult[hookstage.RawBidderResponsePayload]) map[string]map[string]interface{} {
	out := map[string]map[string]interface{}{}
	if len(res.AnalyticsTags.Activities) == 0 {
		return out
	}
	for _, r := range res.AnalyticsTags.Activities[0].Results {
		out[r.AppliedTo.BidIds[0]] = r.Values
	}
	return out
}

// slowEngine blocks until released or until the caller's context ends, as
// the real engines do while waiting for a pool slot.
type slowEngine struct {
	fakeEngine
	release chan struct{}
}

func (e *slowEngine) Validate(ctx context.Context, xml string, o vastlint.Options) (*vastlint.Result, error) {
	select {
	case <-e.release:
		return e.fakeEngine.Validate(ctx, xml, o)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestEligibilityAtTheEdges(t *testing.T) {
	m := realModule(t, `{"max_document_bytes":4096}`)
	broken, err := os.ReadFile("testdata/missing_impression.xml")
	require.NoError(t, err)
	pad := func(n int) string { return "<!--" + strings.Repeat("p", n-7) + "-->" }
	exact := string(broken) + pad(4096-len(broken)) // exactly the cap
	over := exact + pad(8)                          // eight bytes over
	bomDecl := "\ufeff\n  <?xml version=\"1.0\"?>" + string(broken)
	vmap := `<vmap:VMAP xmlns:vmap="http://www.iab.net/videosuite/vmap" version="1.0"><vmap:AdBreak timeOffset="start" breakType="linear" breakId="pre"><vmap:AdSource id="1" allowMultipleAds="true" followRedirects="true"><vmap:AdTagURI templateType="vast3"><![CDATA[https://ads.example/vast]]></vmap:AdTagURI></vmap:AdSource></vmap:AdBreak></vmap:VMAP>`
	notUTF8 := "<VAST version=\"3.0\"><Ad><InLine><AdTitle>\xff\xfe</AdTitle></InLine></Ad></VAST>"
	bannerType := &adapters.TypedBid{Bid: &openrtb2.Bid{ID: "mtype-only", ImpID: "", AdM: string(broken), MType: openrtb2.MarkupVideo}, BidType: openrtb_ext.BidTypeBanner}
	bids := []*adapters.TypedBid{
		videoBid("exact", exact), videoBid("over", over), videoBid("bom", bomDecl), videoBid("vmap", vmap), videoBid("bytes", notUTF8), bannerType,
		videoBid("dup", string(broken)), videoBid("dup", string(broken)),
	}
	res := run(t, m, "", bids...)
	a := actionsOf(res)
	assert.Equal(t, actionChecked, a["exact"], "a creative exactly at the cap is validated")
	assert.Equal(t, actionSkippedSize, a["over"], "over the cap is skipped and kept")
	assert.Equal(t, actionChecked, a["bom"], "a BOM and whitespace before the declaration are trimmed before validation")
	assert.Equal(t, "supported", valuesOf(res)["bom"]["input_status"], "and the trimmed document parses")
	assert.Equal(t, actionChecked, a["vmap"], "VMAP is eligible and validated")
	assert.Equal(t, "VMAP", valuesOf(res)["vmap"]["document_type"], "the document type is reported so a playlist is not mistaken for an ad")
	assert.Equal(t, actionSkippedNotUTF, a["bytes"], "invalid UTF-8 is skipped before the engine, kept, and not an engine error")
	assert.Equal(t, actionChecked, a["mtype-only"], "mtype=2 makes a bid eligible even with a banner BidType")
	out := applyChanges(t, res, hookstage.RawBidderResponsePayload{Bidder: "adprime", BidderResponse: &adapters.BidderResponse{Bids: bids}})
	assert.Len(t, out.BidderResponse.Bids, len(bids), "monitor mode keeps every bid, duplicates and empty imp ids included")
	assert.Len(t, res.AnalyticsTags.Activities[0].Results, len(bids), "one result per video bid, duplicates reported separately")
}

func TestAnnotationReplacesAndRespectsExistingExt(t *testing.T) {
	eng := &fakeEngine{verdicts: map[string]*vastlint.Result{brokenVAST: brokenVerdict()}}
	m := newWithEngine(cfgWith(modeMonitor), eng, nil)
	stale := videoBid("stale", brokenVAST)
	stale.Bid.Ext = json.RawMessage(`{"vastlint":{"valid":true,"stale":1},"prebid":{"type":"video"}}`)
	arr := videoBid("array", brokenVAST)
	arr.Bid.Ext = json.RawMessage(`[1,2]`)
	spaced := videoBid("spaced", brokenVAST)
	spaced.Bid.Ext = json.RawMessage("  {\"a\":1}  ")
	res := run(t, m, "", stale, arr, spaced)
	out := applyChanges(t, res, hookstage.RawBidderResponsePayload{Bidder: "adprime", BidderResponse: &adapters.BidderResponse{Bids: []*adapters.TypedBid{stale, arr, spaced}}})
	var got struct {
		Vastlint map[string]interface{} `json:"vastlint"`
		Prebid   map[string]interface{} `json:"prebid"`
	}
	require.NoError(t, json.Unmarshal(out.BidderResponse.Bids[0].Bid.Ext, &got))
	assert.Equal(t, false, got.Vastlint["valid"], "a stale vastlint key is replaced, not merged")
	assert.NotContains(t, got.Vastlint, "stale")
	assert.Equal(t, "video", got.Prebid["type"], "other ext keys survive")
	assert.Equal(t, json.RawMessage(`[1,2]`), out.BidderResponse.Bids[1].Bid.Ext, "a non-object ext is left alone")
	assert.Contains(t, strings.Join(res.DebugMessages, "\n"), "not annotated")
	assert.Contains(t, string(out.BidderResponse.Bids[2].Bid.Ext), `"vastlint"`, "surrounding whitespace is tolerated")
}

func TestPoolSizeOneHandlesTwentyBids(t *testing.T) {
	m := realModule(t, `{"pool_size":1}`)
	assert.Equal(t, 1, m.parallel, "fan-out never exceeds the pool")
	broken, err := os.ReadFile("testdata/missing_impression.xml")
	require.NoError(t, err)
	var bids []*adapters.TypedBid
	for i := 0; i < 20; i++ {
		bids = append(bids, videoBid(fmt.Sprintf("b%02d", i), string(broken)+fmt.Sprintf("<!-- %d -->", i)))
	}
	res := run(t, m, "", bids...)
	out := applyChanges(t, res, hookstage.RawBidderResponsePayload{Bidder: "adprime", BidderResponse: &adapters.BidderResponse{Bids: bids}})
	for i, tb := range out.BidderResponse.Bids {
		assert.Equal(t, fmt.Sprintf("b%02d", i), tb.Bid.ID)
		assert.Contains(t, string(tb.Bid.Ext), `"valid":false`)
	}
	assert.Empty(t, res.Warnings)
}

func TestAccountsSwitchModesOverOneCache(t *testing.T) {
	eng := &fakeEngine{verdicts: map[string]*vastlint.Result{brokenVAST: brokenVerdict()}}
	cfg := cfgWith(modeMonitor)
	cfg.CacheSize = 1 << 20
	cfg.CacheTTLSeconds = 60
	m := newWithEngine(cfg, eng, nil)
	m.catalog = newCatalog([]string{"VAST-2.0-inline-impression"})
	defer m.Shutdown()
	// Account A validates off the path: nothing on the bid, one engine call.
	resA := run(t, m, `{"mode":"async"}`, videoBid("a", brokenVAST))
	assert.Equal(t, actionQueued, actionsOf(resA)["a"])
	assert.Empty(t, resA.ChangeSet.Mutations())
	waitFor(t, func() bool { return m.async.validated.Load() == 1 })
	// Account B rejects: the verdict A produced serves B from the cache.
	resB := run(t, m, `{"mode":"reject","reject_on":["VAST-2.0-inline-impression"]}`, videoBid("b", brokenVAST))
	assert.Equal(t, actionDropped, actionsOf(resB)["b"])
	assert.Equal(t, "hit", resB.AnalyticsTags.Activities[0].Results[0].Values["cache"])
	// The host (monitor) keeps and annotates the same creative.
	resH := run(t, m, "", videoBid("h", brokenVAST))
	assert.Equal(t, actionChecked, actionsOf(resH)["h"])
	assert.Len(t, resH.ChangeSet.Mutations(), 1)
	assert.Equal(t, int32(1), eng.calls.Load(), "one engine call served three accounts")
}

func TestSamplingNeverAppliesToReject(t *testing.T) {
	eng := &fakeEngine{verdicts: map[string]*vastlint.Result{brokenVAST: brokenVerdict()}}
	cfg := cfgWith(modeMonitor)
	cfg.SamplePercent = 1
	m := newWithEngine(cfg, eng, func() int { return 99 }) // always sampled out in monitor mode
	assert.Equal(t, actionSampledOut, actionsOf(run(t, m, "", videoBid("x", brokenVAST)))["x"])
	assert.Equal(t, actionDropped, actionsOf(run(t, m, `{"mode":"reject","reject_on":["VAST-2.0-inline-impression"]}`, videoBid("y", brokenVAST)))["y"], "an account in reject mode checks every bid")
}

// TestRaceMixedAccountsOnTheRealEngine runs concurrent hooks with different
// account overrides against one engine and one cache.
func TestRaceMixedAccountsOnTheRealEngine(t *testing.T) {
	m := realModule(t, `{"cache_size":1048576,"pool_size":2}`)
	broken, err := os.ReadFile("testdata/missing_impression.xml")
	require.NoError(t, err)
	accounts := []string{"", `{"mode":"async"}`, `{"mode":"reject","reject_on":["VAST-2.0-inline-impression"]}`, `{"rules":{"VAST-2.0-inline-impression":"off"}}`, `{"annotate_bids":false}`}
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 30; i++ {
				bids := []*adapters.TypedBid{videoBid("b1", string(broken)), videoBid("b2", string(broken)+"<!-- v -->")}
				payload := hookstage.RawBidderResponsePayload{Bidder: "adprime", BidderResponse: &adapters.BidderResponse{Bids: bids}}
				_, err := m.HandleRawBidderResponseHook(context.Background(), hookstage.ModuleInvocationContext{AccountConfig: json.RawMessage(accounts[(g+i)%len(accounts)])}, payload)
				assert.NoError(t, err)
			}
		}(g)
	}
	wg.Wait()
}

func TestBidderSuppliedVerdictIsRemovedOnEveryPath(t *testing.T) {
	spoof := func(id, adm string) *adapters.TypedBid {
		tb := videoBid(id, adm)
		tb.Bid.Ext = json.RawMessage(`{"vastlint":{"valid":true,"spoof":1},"prebid":{"type":"video"}}`)
		return tb
	}
	check := func(t *testing.T, name string, m *Module, account string, tb *adapters.TypedBid) {
		t.Helper()
		res := run(t, m, account, tb)
		out := applyChanges(t, res, hookstage.RawBidderResponsePayload{Bidder: "adprime", BidderResponse: &adapters.BidderResponse{Bids: []*adapters.TypedBid{tb}}})
		require.Len(t, out.BidderResponse.Bids, 1, name)
		ext := string(out.BidderResponse.Bids[0].Bid.Ext)
		assert.NotContains(t, ext, `"spoof"`, "%s: the bidder's verdict is gone", name)
		assert.Contains(t, ext, `"prebid":{"type":"video"}`, "%s: other ext keys survive", name)
		assert.Contains(t, ext, `"vastlint"`, "%s: the module's own verdict is present", name)
	}
	eng := &fakeEngine{verdicts: map[string]*vastlint.Result{brokenVAST: brokenVerdict()}}
	check(t, "annotated", newWithEngine(cfgWith(modeMonitor), eng, nil), "", spoof("a", brokenVAST))

	// Paths that write no verdict must still remove the bidder's.
	strip := func(t *testing.T, name string, m *Module, account string, tb *adapters.TypedBid, want string) {
		t.Helper()
		res := run(t, m, account, tb)
		assert.Equal(t, want, actionsOf(res)[tb.Bid.ID], name)
		out := applyChanges(t, res, hookstage.RawBidderResponsePayload{Bidder: "adprime", BidderResponse: &adapters.BidderResponse{Bids: []*adapters.TypedBid{tb}}})
		require.Len(t, out.BidderResponse.Bids, 1, name)
		ext := string(out.BidderResponse.Bids[0].Bid.Ext)
		assert.NotContains(t, ext, `"vastlint"`, "%s: no verdict key at all", name)
		assert.Contains(t, ext, `"prebid":{"type":"video"}`, "%s: other ext keys survive", name)
		assert.Equal(t, json.RawMessage(`{"vastlint":{"valid":true,"spoof":1},"prebid":{"type":"video"}}`), tb.Bid.Ext, "%s: the original bid is untouched", name)
	}
	strip(t, "skipped_url", newWithEngine(cfgWith(modeMonitor), eng, nil), "", spoof("u", "https://cdn.example/t.xml"), actionSkippedURL)
	strip(t, "annotate_bids false", newWithEngine(cfgWith(modeMonitor), eng, nil), `{"annotate_bids":false}`, spoof("n", brokenVAST), actionChecked)
	strip(t, "engine_error", newWithEngine(cfgWith(modeMonitor), &fakeEngine{err: vastlint.ErrEngine}, nil), "", spoof("e", brokenVAST), actionEngineError)
	strip(t, "sampled_out", newWithEngine(func() Config { c := cfgWith(modeMonitor); c.SamplePercent = 1; return c }(), eng, func() int { return 99 }), "", spoof("s", brokenVAST), actionSampledOut)
	ma := newWithEngine(cfgWith(modeAsync), eng, nil)
	defer ma.Shutdown()
	strip(t, "async", ma, "", spoof("q", brokenVAST), actionQueued)
}

func TestNullExtIsAnnotatedAndUppercaseSchemeIsAURL(t *testing.T) {
	eng := &fakeEngine{verdicts: map[string]*vastlint.Result{brokenVAST: brokenVerdict()}}
	m := newWithEngine(cfgWith(modeMonitor), eng, nil)
	nul := videoBid("null", brokenVAST)
	nul.Bid.Ext = json.RawMessage(`null`)
	up := videoBid("up", "HTTPS://CDN.EXAMPLE/T.XML")
	res := run(t, m, "", nul, up)
	out := applyChanges(t, res, hookstage.RawBidderResponsePayload{Bidder: "adprime", BidderResponse: &adapters.BidderResponse{Bids: []*adapters.TypedBid{nul, up}}})
	assert.Contains(t, string(out.BidderResponse.Bids[0].Bid.Ext), `"vastlint":{`, "a null ext is treated as empty")
	assert.Equal(t, actionSkippedURL, actionsOf(res)["up"])
}

func TestDeadlineMidFlightKeepsRemainingBids(t *testing.T) {
	slow := &slowEngine{release: make(chan struct{})}
	m := newWithEngine(cfgWith(modeReject, "VAST-2.0-inline-impression"), slow, nil)
	m.parallel = 1
	var bids []*adapters.TypedBid
	for i := 0; i < 20; i++ {
		bids = append(bids, videoBid(fmt.Sprintf("b%02d", i), brokenVAST))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	start := time.Now()
	res, err := m.HandleRawBidderResponseHook(ctx, hookstage.ModuleInvocationContext{}, hookstage.RawBidderResponsePayload{Bidder: "adprime", BidderResponse: &adapters.BidderResponse{Bids: bids}})
	require.NoError(t, err)
	assert.Less(t, time.Since(start), 2*time.Second, "the hook returns once the budget is gone instead of validating the rest")
	assert.Empty(t, res.ChangeSet.Mutations(), "nothing is dropped without a verdict")
	assert.Len(t, res.Warnings, 1)
	for _, a := range actionsOf(res) {
		assert.Equal(t, actionEngineError, a)
	}
	close(slow.release)
}
