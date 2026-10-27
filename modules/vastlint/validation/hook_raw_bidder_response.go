package validation

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	vastlint "github.com/aleksUIX/vastlint-go"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/hooks/hookanalytics"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var _ hookstage.RawBidderResponse = (*Module)(nil)

const (
	// activityName is the analytics tag activity this module reports under.
	activityName = "vast_validation"
	// extKey is the bid.ext key that carries the summary.
	extKey = "vastlint"
)

// Per-bid outcomes, reported in analytics tag values under "action".
const (
	actionChecked       = "checked"
	actionDropped       = "dropped"
	actionSkippedURL    = "skipped_url"
	actionSkippedSize   = "skipped_size"
	actionSkippedNotXML = "skipped_not_xml"
	actionSkippedNotUTF = "skipped_not_utf8"
	actionSampledOut    = "sampled_out"
	actionEngineError   = "engine_error"
	actionQueued        = "queued"     // async mode: validation happens after the hook returns
	actionQueueFull     = "queue_full" // async mode: the bounded queue was full, not validated
)

// Phase 1 decisions that are not actions: the bid is left alone without a tag
// (not a video bid, empty markup) or it goes to the engine.
const (
	decideValidate = ""
	decideIgnore   = "-"
)

// bidSummary is written to bid.ext.vastlint.
type bidSummary struct {
	Valid bool `json:"valid"`
	// InputStatus is supported, unsupported or malformed.
	InputStatus string `json:"input_status"`
	// DocumentType is VAST, VMAP or DAAST and DocumentVersion the declared
	// version, so a reader can tell a playlist from an ad.
	DocumentType    string   `json:"document_type,omitempty"`
	DocumentVersion string   `json:"document_version,omitempty"`
	Errors          int      `json:"errors"`
	Warnings        int      `json:"warnings"`
	Infos           int      `json:"infos"`
	Rules           []string `json:"rules,omitempty"`
	// EngineVersion is the engine that produced the verdict, so a reader
	// knows which rule catalog applied.
	EngineVersion string `json:"engine_version"`
}

// maxParallel caps how many bids of one response are validated at once; the
// effective bound is the smaller of this and the engine pool size. Eight keeps
// a 20-bid response of ordinary creatives well inside a 20 ms hook budget
// without starving concurrent requests; the cost per creative grows with its
// size and element count (about 0.1 ms per KB on the wasm engine for dense
// markup), which is what max_document_bytes and the group timeout bound.
const maxParallel = 8

type outcome struct {
	err     error
	summary bidSummary
	cached  bool
}

// cacheTag adds the cache outcome to an analytics result when the cache is on.
func (m *Module) cacheTag(r hookanalytics.Result, cached bool) hookanalytics.Result {
	if m.cache != nil {
		r.Values["cache"] = m.cache.tag(cached)
	}
	return r
}

// HandleRawBidderResponseHook validates every video bid's markup. The auction
// never waits for anything outside this process; the hook returns within the
// group timeout the host configured, and a bid is only removed in reject mode
// when a finding matches reject_on. Bids of one response are validated
// concurrently; results are assembled in the original order so output is
// deterministic.
func (m *Module) HandleRawBidderResponseHook(
	ctx context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.RawBidderResponsePayload,
) (hookstage.HookResult[hookstage.RawBidderResponsePayload], error) {
	result := hookstage.HookResult[hookstage.RawBidderResponsePayload]{}
	if m.engine == nil || payload.BidderResponse == nil || len(payload.BidderResponse.Bids) == 0 {
		return result, nil
	}
	cfg, err := m.cfg.effective(miCtx.AccountConfig, m.catalog)
	if err != nil {
		result.Warnings = append(result.Warnings, err.Error())
	}
	bids := payload.BidderResponse.Bids
	activity := hookanalytics.Activity{Name: activityName, Status: hookanalytics.ActivityStatusSuccess}

	// Phase 1: decide per bid whether it reaches the engine, and why not. The
	// document handed to the engine (and hashed for the cache) is the markup
	// without leading whitespace or a byte order mark, which players tolerate
	// and the XML grammar does not.
	decision := make([]string, len(bids))
	docs := make([]string, len(bids))
	eligible := make([]int, 0, len(bids))
	for i, tb := range bids {
		if tb == nil || tb.Bid == nil || !isVideo(tb) {
			decision[i] = decideIgnore
			continue
		}
		adm := strings.TrimSpace(strings.TrimLeft(tb.Bid.AdM, "\ufeff"))
		switch {
		case adm == "":
			decision[i] = decideIgnore
		case isURL(adm):
			decision[i] = actionSkippedURL
		case !strings.HasPrefix(adm, "<"):
			decision[i] = actionSkippedNotXML
		case len(adm) > cfg.MaxDocumentBytes:
			decision[i] = actionSkippedSize
		case !utf8.ValidString(adm):
			decision[i] = actionSkippedNotUTF
		case cfg.Mode != modeReject && cfg.SamplePercent < 100 && m.sample() >= cfg.SamplePercent:
			decision[i] = actionSampledOut
		default:
			docs[i] = adm
			eligible = append(eligible, i)
		}
	}

	var rk []byte
	if m.cache != nil && len(eligible) > 0 {
		rk = rulesKey(cfg.Rules, m.engine.Version())
	}

	if cfg.Mode == modeAsync && m.async != nil {
		// Off the bid path: queue the eligible creatives and return at once.
		// The auction does not wait, and the only change to a bid is the
		// removal of a vastlint key a bidder may have put in ext itself. A
		// cached verdict is reported straight away and costs no engine time.
		kept := make([]*adapters.TypedBid, 0, len(bids))
		changed := false
		for i, tb := range bids {
			if decision[i] == decideIgnore {
				kept = append(kept, tb)
				continue
			}
			if stripped, ok := withoutVerdict(tb); ok {
				tb = stripped
				changed = true
			}
			kept = append(kept, tb)
			if decision[i] != decideValidate {
				activity.Results = append(activity.Results, tag(payload.Bidder, tb.Bid, decision[i], nil))
				continue
			}
			var key []byte
			if m.cache != nil {
				key = cacheKey(docs[i], rk)
				if s, ok := m.cache.get(key); ok {
					activity.Results = append(activity.Results, m.cacheTag(tag(payload.Bidder, tb.Bid, actionChecked, &s), true))
					continue
				}
			}
			action := actionQueued
			if !m.async.submit(docs[i], cfg.Rules, key) {
				action = actionQueueFull
			}
			activity.Results = append(activity.Results, m.cacheTag(tag(payload.Bidder, tb.Bid, action, nil), false))
		}
		if len(activity.Results) > 0 {
			result.AnalyticsTags = hookanalytics.Analytics{Activities: []hookanalytics.Activity{activity}}
		}
		if changed {
			result.ChangeSet.RawBidderResponse().Bids().UpdateBids(kept)
		}
		return result, nil
	}

	// Phase 2: validate eligible bids, in parallel when there are several. A
	// cached verdict skips the engine. A panic inside the engine binding is
	// contained here and treated as an engine error: the bid is kept.
	outcomes := make([]outcome, len(bids))
	check := func(i int) {
		defer func() {
			if r := recover(); r != nil {
				outcomes[i] = outcome{err: fmt.Errorf("%w: panic: %v", vastlint.ErrEngine, r)}
			}
		}()
		if err := ctx.Err(); err != nil {
			// The budget is gone: do not spend hashing or engine time on a
			// result the framework will discard.
			outcomes[i] = outcome{err: err}
			return
		}
		adm := docs[i]
		var key []byte
		if m.cache != nil {
			key = cacheKey(adm, rk)
			if s, ok := m.cache.get(key); ok {
				outcomes[i] = outcome{summary: s, cached: true}
				return
			}
		}
		v, verr := m.engine.Validate(ctx, adm, vastlint.Options{RuleOverrides: map[string]string(cfg.Rules)})
		o := outcome{err: verr}
		if verr == nil {
			o.summary = summarize(v, m.engine.Version())
			if m.cache != nil {
				m.cache.put(key, o.summary)
			}
		}
		outcomes[i] = o
	}
	switch {
	case len(eligible) == 1:
		check(eligible[0])
	case len(eligible) > 1:
		var wg sync.WaitGroup
		sem := make(chan struct{}, m.parallel)
		for _, i := range eligible {
			wg.Add(1)
			sem <- struct{}{}
			go func(i int) {
				defer wg.Done()
				defer func() { <-sem }()
				check(i)
			}(i)
		}
		wg.Wait()
	}

	// Phase 3: assemble tags, annotations and drops in original order.
	kept := make([]*adapters.TypedBid, 0, len(bids))
	changed := false
	engineErrors := 0
	var firstErr error
	for i, tb := range bids {
		if decision[i] == decideIgnore {
			kept = append(kept, tb)
			continue
		}
		if decision[i] != decideValidate {
			if stripped, ok := withoutVerdict(tb); ok {
				tb = stripped
				changed = true
			}
			activity.Results = append(activity.Results, tag(payload.Bidder, tb.Bid, decision[i], nil))
			kept = append(kept, tb)
			continue
		}
		bid := tb.Bid
		o := outcomes[i]
		if o.err != nil {
			// Engine trouble never costs the host a bid.
			engineErrors++
			if firstErr == nil {
				firstErr = o.err
			}
			if stripped, ok := withoutVerdict(tb); ok {
				tb = stripped
				changed = true
			}
			r := tag(payload.Bidder, bid, actionEngineError, nil)
			r.Status = hookanalytics.ResultStatusError
			activity.Results = append(activity.Results, r)
			kept = append(kept, tb)
			continue
		}
		summary := o.summary
		drop := false
		if cfg.Mode == modeReject {
			for _, id := range summary.Rules {
				if cfg.rejects(id) {
					drop = true
					break
				}
			}
		}
		if drop {
			r := m.cacheTag(tag(payload.Bidder, bid, actionDropped, &summary), o.cached)
			r.Status = hookanalytics.ResultStatusBlock
			activity.Results = append(activity.Results, r)
			result.DebugMessages = append(result.DebugMessages, fmt.Sprintf("vastlint dropped bid %s from %s: %s", bid.ID, payload.Bidder, strings.Join(summary.Rules, ", ")))
			changed = true
			continue
		}
		r := m.cacheTag(tag(payload.Bidder, bid, actionChecked, &summary), o.cached)
		if *cfg.AnnotateBids {
			if annotated, ok := annotate(tb, summary, *cfg.AnnotateRules); ok {
				tb = annotated
				changed = true
				r.Status = hookanalytics.ResultStatusModify
			} else {
				result.DebugMessages = append(result.DebugMessages, fmt.Sprintf("vastlint: bid %s from %s not annotated: ext is not a JSON object", bid.ID, payload.Bidder))
			}
		} else if stripped, ok := withoutVerdict(tb); ok {
			tb = stripped
			changed = true
		}
		activity.Results = append(activity.Results, r)
		kept = append(kept, tb)
	}

	if engineErrors > 0 {
		// One line per response, not one per bid: the hook itself succeeded and
		// every bid was kept, so this is a warning in the trace, not an error.
		result.Warnings = append(result.Warnings, fmt.Sprintf("vastlint: %d of %d video bids from %s not validated: %v", engineErrors, len(eligible), payload.Bidder, firstErr))
		if engineErrors == len(eligible) {
			activity.Status = hookanalytics.ActivityStatusError
		}
	}
	if len(activity.Results) > 0 {
		result.AnalyticsTags = hookanalytics.Analytics{Activities: []hookanalytics.Activity{activity}}
	}
	if changed {
		result.ChangeSet.RawBidderResponse().Bids().UpdateBids(kept)
	}
	return result, nil
}

func isVideo(tb *adapters.TypedBid) bool {
	if tb.BidType == openrtb_ext.BidTypeVideo {
		return true
	}
	return tb.Bid != nil && tb.Bid.MType == openrtb2.MarkupVideo
}

// isURL reports whether the markup is a tag URL rather than a document.
func isURL(adm string) bool {
	if len(adm) < 8 {
		return false
	}
	head := strings.ToLower(adm[:8])
	return strings.HasPrefix(head, "http://") || strings.HasPrefix(head, "https://")
}

func summarize(v *vastlint.Result, engineVersion string) bidSummary {
	s := bidSummary{Valid: v.Valid, InputStatus: v.InputStatus, DocumentType: v.DocumentType, DocumentVersion: v.Version, Errors: v.Summary.Errors, Warnings: v.Summary.Warnings, Infos: v.Summary.Infos, EngineVersion: engineVersion}
	seen := map[string]bool{}
	for _, i := range v.Issues {
		if !seen[i.ID] {
			seen[i.ID] = true
			s.Rules = append(s.Rules, i.ID)
		}
	}
	return s
}

func tag(bidder string, bid *openrtb2.Bid, action string, s *bidSummary) hookanalytics.Result {
	values := map[string]interface{}{"action": action}
	if s != nil {
		values["valid"] = s.Valid
		values["input_status"] = s.InputStatus
		values["document_type"] = s.DocumentType
		values["document_version"] = s.DocumentVersion
		values["errors"] = s.Errors
		values["warnings"] = s.Warnings
		values["infos"] = s.Infos
		values["rules"] = append([]string{}, s.Rules...)
	}
	applied := hookanalytics.AppliedTo{Bidder: bidder, BidIds: []string{bid.ID}}
	if bid.ImpID != "" {
		applied.ImpIds = []string{bid.ImpID}
	}
	return hookanalytics.Result{Status: hookanalytics.ResultStatusAllow, Values: values, AppliedTo: applied}
}

// withoutVerdict returns a shallow copy of the typed bid with any vastlint
// key a bidder put in ext removed, so a verdict in the response can only come
// from this module. false means there was nothing to remove.
func withoutVerdict(tb *adapters.TypedBid) (*adapters.TypedBid, bool) {
	ext := tb.Bid.Ext
	if len(ext) == 0 || !bytes.Contains(ext, []byte(`"`+extKey+`"`)) || !gjson.GetBytes(ext, extKey).Exists() {
		return tb, false
	}
	cleaned, err := sjson.DeleteBytes(ext, extKey)
	if err != nil {
		return tb, false
	}
	bidCopy := *tb.Bid
	bidCopy.Ext = cleaned
	tbCopy := *tb
	tbCopy.Bid = &bidCopy
	return &tbCopy, true
}

// annotate returns a shallow copy of the typed bid whose bid carries the
// summary under ext.vastlint, leaving the bidder's other ext bytes as they
// were. The original objects are left untouched so the change flows through
// the hook change set. false means the existing ext is not a JSON object.
func annotate(tb *adapters.TypedBid, s bidSummary, withRules bool) (*adapters.TypedBid, bool) {
	if !withRules {
		s.Rules = nil
	}
	raw, err := jsonutil.Marshal(s)
	if err != nil {
		return tb, false
	}
	ext := tb.Bid.Ext
	if trimmed := bytes.TrimSpace(ext); len(trimmed) == 0 || string(trimmed) == "null" {
		ext = []byte("{}")
	} else if trimmed[0] != '{' || !jsonValid(ext) {
		return tb, false
	}
	merged, err := sjson.SetRawBytes(ext, extKey, raw)
	if err != nil || !jsonValid(merged) {
		return tb, false
	}
	bidCopy := *tb.Bid
	bidCopy.Ext = merged
	tbCopy := *tb
	tbCopy.Bid = &bidCopy
	return &tbCopy, true
}

// jsonValid reports whether b is a complete JSON value.
func jsonValid(b []byte) bool {
	var v interface{}
	return jsonutil.UnmarshalValid(b, &v) == nil
}
