# Scope3 TMP Prebid Server Module — Design

**Date**: 2026-05-27
**Author**: brainstormed with @bhuo
**Status**: design (pre-implementation)

## Context

Scope3 currently ships `modules/scope3/rtd/` — a Prebid Server hook module that fetches audience segments from Scope3's RTDP endpoint (`rtdp.scope3.com/prebid/prebid`) and writes them into the bid response for downstream targeting (GAM, other ad servers, analytics).

The AdCP **Trusted Match Protocol (TMP)** is a privacy-by-architecture alternative. It splits the page-context match and the user-identity match into two structurally separated calls so that no single party sees both, enabling cross-publisher frequency capping without exposing a user×page graph. The protocol is described at https://docs.adcontextprotocol.org/docs/trusted-match.

This document specifies a new Prebid Server module `modules/scope3/tmp/` that integrates Scope3-operated TMP routers and serves as the migration target away from RTD.

## Goals

1. Implement a Prebid Server hook module that calls a Scope3-operated TMP router (`https://tmp.interchange.io`, planned) and enriches the bid response with TMP-derived eligibility data.
2. Conform to the AdCP TMP wire format as defined in https://github.com/adcontextprotocol/adcp-go `tmproto/` package.
3. Ship as an additive module — the existing `scope3.rtd` continues to compile and operate. RTD is deprecated via its README, not by removal.
4. Match privacy and observability standards of the existing RTD module.

## Non-goals

- v0 does **not** implement creative-side `{TMPX}` substitution (the publisher's serve-side does this).
- v0 does **not** run an embedded TMP router inside Prebid Server. The module is a client of an externally-operated router.
- v0 does **not** consume `aee_signals.bidders` or any per-bidder routing logic.
- v0 does **not** share code with `modules/scope3/rtd/`. Shared identity-extraction helpers are explicitly deferred to a follow-up.

## Reference dependencies

- AdCP TMP specification: https://docs.adcontextprotocol.org/docs/trusted-match and `.../specification`
- AdCP Universal Macros: https://docs.adcontextprotocol.org/docs/creative/universal-macros
- AdCP Go SDK: https://github.com/adcontextprotocol/adcp-go (specifically `tmproto/types_gen.go` — types vendored into this module; see Approach decision below)

## Decisions log (from brainstorming)

| Decision | Value | Rationale |
|---|---|---|
| Output target | Segments in `ext.scope3.tmp.*`; async goroutine | Mirrors RTD lifecycle so downstream Scope3 conventions stay familiar |
| Identity source | Forward existing OpenRTB EIDs (RampID, UID2, ID5); router/buyer hash per spec | No new identity model; reuse RTD-style extraction |
| Coexistence with RTD | Two modules in the tree; RTD README gains deprecation pointer | "Alternative" implies parallel for at least one release |
| `property_rid` / `placement_id` / `property_type` source | **Account config only** (per-publisher stored config); per-request ext override allowed for testing; **no module-level fallback** | A module-level default risks cross-publisher contamination |
| `router_url` source | Module config (account-config override allowed) | Stable per deployment |
| `seller_agent_url` source | Module config (account-config override allowed) | Static per deployment; the open-source TMP router does not inject this field, so the module must write it onto outbound JSON |
| Latency budget | `timeout_ms` configurable, default **200ms** | Router's internal budget is 50ms; 200ms gives headroom without hiding tail issues |
| Auth to router | Optional `auth_key` header (`x-scope3-auth`), mirrors RTD | Router itself is unauthenticated; CDN/WAF in front provides bearer in production |
| SDK dependency strategy | **Vendor `tmproto` types into `modules/scope3/tmp/proto.go`** with provenance comment | adcp-go's `go.mod` declares `go 1.25.0`; prebid-server is `go 1.23.0`. Importing directly forces a Go-version bump |
| Output collision with RTD on `ext.scope3.*` | TMP overwrites — TMP wins | "We trust TMP more" |
| Multi-imp handling | **Fan-out per unique `placement_id`**; one Identity Match per auction (identity is page-context-free) | Spec mandates one placement per Context Match; Identity Match `MUST NOT contain page context` |
| Partial-success handling | **Strict — both calls must succeed for any enrichment** | TMP's value is the privacy join; lenient mode silently degrades the differentiator |
| Identity cap | **Configurable allowlist validated to ≤ 3 at Builder time** (I1) | Spec hard limit `maxItems: 3` driven by TMPX HPKE plaintext budget; operator picks the 3 |
| TTL handling | `min(cfg.CacheTTLSeconds, ContextMatchResponse.CacheTTL, IdentityMatchResponse.TTLSec)` per cached entry; `0` from server bypasses cache | Honors per-response TTL overrides per spec |
| Test coverage gate | **90% line, 100% branch on `intersect`, `accountResolver`, per-imp result mapping** | Stricter than RTD's bar; covers the privacy-sensitive paths |

## Architecture

### Module placement

```
modules/scope3/tmp/
  module.go         — Builder, Module struct, Config, three hook handlers
  async_request.go  — AsyncRequest state, two-call orchestration, intersection
  account.go        — Resolve property/placement/seller_agent_url from AccountConfig + ext override
  masking.go        — Adapted from RTD; identity selection capped at 3 (I1)
  proto.go          — Vendored copy of tmproto types we use, with provenance stamp
  module_test.go    — Hook tests + mock HTTP integration tests
  testdata/         — JSON fixtures
  README.md         — Configuration, deployment, migration notes
```

Registered in `modules/builder.go` alongside `scope3.rtd`:

```go
scope3Tmp "github.com/prebid/prebid-server/v4/modules/scope3/tmp"
...
"scope3": {
    "rtd": scope3Rtd.Builder,
    "tmp": scope3Tmp.Builder,
},
```

### Three-stage hook lifecycle

| Stage | Purpose |
|---|---|
| `Entrypoint` | Initialize `AsyncRequest` in `ModuleContext` under `scope3.tmp.AsyncRequest`. No I/O. |
| `ProcessedAuctionRequest` | Resolve identifiers, kick off the asymmetric `N+1` parallel HTTP fan-out (N Context Match + 1 Identity Match) via `errgroup.Group`. Return immediately. |
| `AuctionResponse` | Wait on the goroutine's `Done` channel (bounded by hook context). Write the per-imp enrichment into `BidResponse.Ext` and `seatbid[].bid[].ext`. |

The three-stage shape mirrors the RTD module's, with the goroutine internally orchestrating multiple calls instead of one.

### Relationship to RTD

- Coexist; no code shared in v0.
- RTD's README gains a one-paragraph pointer to this module as the preferred path.
- Both modules can be configured simultaneously. If both write to `ext.scope3.*`, TMP runs second (operator orders the execution plan accordingly) **and** TMP's `sjson.SetBytes` overwrites unconditionally, so TMP wins regardless.
- Startup log line: `INFO scope3.tmp module enabled — if scope3.rtd is also enabled, scope3.tmp will overwrite RTD segments in the response`.

## Configuration

### Module config (`pbs.yaml` under `hooks.modules.scope3.tmp`)

```yaml
hooks:
  enabled: true
  modules:
    scope3:
      tmp:
        enabled: true
        router_url: https://tmp.interchange.io
        seller_agent_url: https://prebid.example.com/scope3   # our seller-agent URL
        auth_key: ${SCOPE3_TMP_AUTH_KEY}                       # optional
        timeout_ms: 200
        cache_ttl_seconds: 60
        cache_size: 10485760                                   # 10 MB
        add_to_targeting: false
        masking:
          enabled: true
          geo:
            preserve_metro: true
            preserve_zip: true
            preserve_city: false
            lat_long_precision: 2
          user:
            preserve_eids:                                     # capped at 3 by spec
              - liveramp.com
              - uidapi.com
              - id5-sync.com
          device:
            preserve_mobile_ids: false
```

`Builder()` validates at startup and fails the load if:
- `router_url` empty
- `seller_agent_url` empty
- `len(masking.user.preserve_eids) > 3` (spec hard cap)
- `masking.geo.lat_long_precision` outside `[0, 4]`
- `timeout_ms <= 0`

### Account config (per-publisher, in Prebid Server account stored config)

```json
{
  "scope3": {
    "tmp": {
      "property_rid": "01916f3a-9c4e-7000-8000-000000000010",
      "property_type": "website",
      "placements": {
        "div-gpt-ad-header":  "header_728x90",
        "div-gpt-ad-sidebar": "sidebar_300x250",
        "div-gpt-ad-video":   "preroll_video"
      },
      "seller_agent_url": "https://altprebid.example.com/scope3"
    }
  }
}
```

`seller_agent_url` here overrides the module-level value for this account only. `property_rid` and `property_type` have no module-level fallback — if absent from account config, the module no-ops for that auction.

### Per-request ext override

```json
{
  "ext": {
    "prebid": {
      "modules": {
        "scope3": {
          "tmp": {
            "property_rid": "01916f3a-...",   // overrides account
            "placement_id": "test_slot"        // overrides per-imp account map; single value for whole auction
          }
        }
      }
    }
  }
}
```

Intended for testing and for cases where a publisher with multiple properties on one account wants to disambiguate per-request. Account config wins under normal operation.

### Identifier resolution precedence

| Field | Resolution order |
|---|---|
| `property_rid` | ext override → account config → no-op |
| `property_type` | account config → no-op |
| `placement_id` (per imp) | ext override (single value) → `account.scope3.tmp.placements[imp.tagid]` → imp skipped for TMP |
| `router_url` | account override (if present) → module config |
| `seller_agent_url` | account override (if present) → module config |

## Data flow

### Lifecycle

```
1. POST /openrtb2/auction
   │
   ▼
2. HandleEntrypointHook
   • Create AsyncRequest{ctx, cancel}; stash in ModuleContext
   │
   ▼
3. (Prebid parses OpenRTB, stored requests, GDPR, etc.)
   │
   ▼
4. HandleProcessedAuctionHook
   • Retrieve AsyncRequest
   • asyncRequest.fetchAsync(bidRequest, accountConfig)
   • Returns immediately (goroutine started)
   │
   ▼  (goroutine runs in parallel with bidder fan-out)
   Goroutine:
     a) Resolve property_rid, property_type, seller_agent_url
        from accountConfig (with ext override).
        If property_rid/property_type/seller_agent_url missing → mark auction skipped.

     b) For each imp:
          placement_id ← ext override OR accountConfig.placements[imp.tagid]
          If missing → mark that imp as skipped (other imps continue).
        uniquePlacements ← dedupe(resolved placement_ids).

     c) Per uniquePlacement:
          cacheKey_ctx[i] = sha256(property_rid + placement_id +
                                   site.domain + site.page +
                                   privacy-safe identifiers)
          contextCache[i] = cache.Get(cacheKey_ctx[i])
        cacheKey_id = sha256(seller_agent_url + privacy-safe identifiers + country)
        identityCache  = cache.Get(cacheKey_id)

     d) Apply privacy masking (masking.go).

     e) Generate independent request_ids (UUIDs).
        ContextMatchRequest{ type, request_id_ctx[i], property_rid, property_id?,
                              property_type, placement_id, artifact_refs?[site.page] }
        IdentityMatchRequest{ type, request_id_id,
                              seller_agent_url, identities[<=3 by I1], country_alpha2 }
        // NOTE: package_ids OMITTED on IdentityMatch per privacy mandate.

     f) errgroup.Group fan-out (skipping any call satisfied by cache):
          for placement in uniquePlacements (cache miss):
            g.Go(POST router_url + "/tmp/context"  → ContextMatchResponse[placement])
          if identityCache empty:
            g.Go(POST router_url + "/tmp/identity" → IdentityMatchResponse)
          g.Wait()

     g) Intersection (P1 strict — both sides must succeed):
          identityPkgs = set(idResp.EligiblePackageIDs)
          for placement, ctxResp in contextResults:
            packages[placement] = {o.PackageID for o in ctxResp.Offers} ∩ identityPkgs
            kvs[placement]      = ctxResp.Signals.targeting_kvs
            segs[placement]     = ctxResp.Signals.segments

     h) Cache: store per-placement context result with
          ttl = min(cfg.CacheTTLSeconds, ctxResp.CacheTTL)
        Store identity result with
          ttl = min(cfg.CacheTTLSeconds, idResp.TTLSec)
        Any 0 server-side TTL bypasses caching for that entry.

     i) Build AsyncResult{ PerPlacement: {placement_id → {packages, kvs, segs}},
                          PerImp: {imp_id → placement_id},
                          TMPX: idResp.Tmpx }
        Close Done.
   │
   ▼
5. HandleAuctionResponseHook
   • <-asyncRequest.Done OR <-ctx.Done() (graceful)
   • If P1 strict failure → no mutation, analytics tag emitted
   • Mutate via ChangeSet:
       - ext.scope3.tmp.tmpx = result.TMPX  (auction-level — user-scoped)
       - For each seatbid[].bid[]:
           bid.ext.scope3.tmp.placement_id      = lookup(bid.impid)
           bid.ext.scope3.tmp.eligible_packages = result.PerPlacement[...].packages
           bid.ext.scope3.tmp.segments          = result.PerPlacement[...].segs
           if cfg.AddToTargeting:
             bid.ext.prebid.targeting.TMPX = result.TMPX
             for kv in result.PerPlacement[...].kvs:
               bid.ext.prebid.targeting[kv.key] = kv.value
   • defer asyncRequest.Cancel()
```

### Output shape (response ext)

```json
{
  "ext": {
    "scope3": { "tmp": { "tmpx": "k1.dG1weC1leGFtcGxl..." } }
  },
  "seatbid": [{
    "bid": [{
      "impid": "imp-header",
      "ext": {
        "scope3": {
          "tmp": {
            "placement_id": "header_728x90",
            "eligible_packages": ["pkg_abc"],
            "segments": ["news_intender"]
          }
        },
        "prebid": {
          "targeting": {
            "TMPX": "k1.dG1weC1leGFtcGxl...",
            "<buyer_kv_key>": "<buyer_kv_value>"
          }
        }
      }
    }]
  }]
}
```

`TMPX` is auction-level on `ext.scope3.tmp` and per-bid on `ext.prebid.targeting` when `add_to_targeting: true`. `eligible_packages`, `segments`, and `targeting_kvs` are per-imp.

## Components

### `Config` (`module.go`)

```go
type Config struct {
    RouterURL       string        `json:"router_url"`
    SellerAgentURL  string        `json:"seller_agent_url"`
    AuthKey         string        `json:"auth_key"`
    TimeoutMs       int           `json:"timeout_ms"`         // default 200
    CacheTTLSeconds int           `json:"cache_ttl_seconds"`  // default 60
    CacheSize       int           `json:"cache_size"`         // default 10 MB
    AddToTargeting  bool          `json:"add_to_targeting"`
    Masking         MaskingConfig `json:"masking"`            // same shape as RTD
}
```

### `Module` (`module.go`)

```go
type Module struct {
    cfg        Config
    httpClient *http.Client
    cache      *freecache.Cache
    sha256Pool *sync.Pool
}
```

Implements `hookstage.Entrypoint`, `hookstage.ProcessedAuctionRequest`, `hookstage.AuctionResponse`.

### `AsyncRequest` (`async_request.go`)

```go
type AsyncRequest struct {
    *Module
    Context context.Context
    Cancel  context.CancelFunc
    Done    chan struct{}
    Result  *AsyncResult        // nil on error
    Err     error
}

type AsyncResult struct {
    PerPlacement map[string]PlacementResult // placement_id → result
    ImpToPlacement map[string]string         // imp.id → placement_id (for bid scoping)
    TMPX           string
}

type PlacementResult struct {
    EligiblePackages []string
    TargetingKVs     []KeyValuePair
    Segments         []string
}
```

### `accountResolver` (`account.go`)

Pure function. Reads from `miCtx.AccountConfig`, `request.Ext`, and module `Config`. Returns the resolved identifiers or a structured "what's missing" error. No state.

### `masking.go`

Adapted from `modules/scope3/rtd/masking.go`. Differences:
- Identity extractor produces the `Identities []IdentityToken` array used by `IdentityMatchRequest`, capped at 3 by spec hard limit (validated at `Builder()`).
- Country code converter: OpenRTB `device.geo.country` is ISO 3166-1 alpha-3; TMP `IdentityMatchRequest.Country` is alpha-2. Needs a static lookup table covering the ISO 3166-1 spec (the implementer can use any open-source alpha3↔alpha2 map; the conversion is value-stable and doesn't change).

### `proto.go`

Vendored copy of `tmproto/types_gen.go` from a specific upstream commit. Header comment:

```go
// Types in this file are copied from
//   github.com/adcontextprotocol/adcp-go/tmproto/types_gen.go
// at upstream commit <SHA>.
// Re-sync manually when the TMP wire schema changes.
```

Only the types this module actually uses are vendored: `ContextMatchRequest`, `ContextMatchResponse`, `IdentityMatchRequest`, `IdentityMatchResponse`, `Offer`, `Identity`, `IdentityToken`, `Signals`, `KeyValuePair`, `Artifact`, `ArtifactRef`, `PropertyType` (+ constants), `ErrorCode` (+ constants we handle), `ErrorResponse`.

## Error handling and observability

All errors are non-fatal to the auction. The module never blocks bidder fan-out or rejects a bid response.

### Failure matrix

| Category | Trigger | Action | Log level | Analytics tag | Metric |
|---|---|---|---|---|---|
| `config_missing` | Required identifier absent after resolution | Skip enrichment | `WARN` once/auction | `Status=ActivityStatusError`, `Name="HandleProcessedAuctionHook.config"` | `scope3_tmp_requests_total{status="config_missing"}` |
| `network_error` | DNS / dial / TLS / connection-reset | Skip enrichment | `ERROR` | `error.cause="network"` | `scope3_tmp_requests_total{status="network_error", endpoint=...}` |
| `timeout` | Per-request `TimeoutMs` fires | Skip enrichment | `WARN` | `error.cause="timeout"` | `scope3_tmp_requests_total{status="timeout", endpoint=...}` |
| `http_4xx` | Router rejected (400/401/403/etc.) | Skip enrichment | `ERROR` | `error.cause="client", error.code=<status>` | `scope3_tmp_requests_total{status="http_4xx"}` |
| `http_5xx` | Transient router/provider failure | Skip enrichment | `WARN` | `error.cause="server"` | `scope3_tmp_requests_total{status="http_5xx"}` |
| `decode_error` | JSON unmarshal failed | Skip enrichment | `ERROR` | `error.cause="decode"` | `scope3_tmp_requests_total{status="decode_error"}` |
| `partial_success` | One endpoint succeeded, the other failed | **P1 strict** — no enrichment (no packages, no TMPX, no KVs) | `WARN` | `error.cause="partial", failed_endpoint=...` | `scope3_tmp_requests_total{status="partial"}` |
| `panic` | Goroutine panicked | `recover()` + log stack | `ERROR` | `error.cause="panic"` | `scope3_tmp_requests_total{status="panic"}` |
| `success_tmpx_only` | Both succeeded; intersection empty for every placement; Identity returned a TMPX | Emit `ext.scope3.tmp.tmpx` only; per-bid ext gets empty `eligible_packages` plus any `targeting_kvs` from successful Context calls | `DEBUG` | None | `scope3_tmp_requests_total{status="success_tmpx_only"}` |
| `success_empty` | Both succeeded; no eligible packages on any imp; no TMPX | No mutation | `DEBUG` | None | `scope3_tmp_requests_total{status="success_empty"}` |
| `success` | At least one imp has non-empty eligible packages, or TMPX present alongside non-empty results | Mutate per the output shape | None | None | `scope3_tmp_requests_total{status="success"}` |

### Logging

Structured via Prebid `logger.Errorf/Warnf/Infof`. One log line per failure per auction. Required context: `auction_id`, `property_rid`, `placement_id`, `endpoint`, `request_id`, `http_status` (if applicable), error chain (never lost). **No raw user data in logs ever.**

### Metrics

| Metric | Type | Labels |
|---|---|---|
| `scope3_tmp_requests_total` | Counter | `status`, `endpoint` ("context" \| "identity") |
| `scope3_tmp_request_duration_seconds` | Histogram | `endpoint`, `status` |
| `scope3_tmp_cache_total` | Counter | `result` ("hit" \| "miss" \| "skip_zero_ttl") |
| `scope3_tmp_eligible_packages_count` | Histogram | none |
| `scope3_tmp_identities_sent` | Histogram | none |
| `scope3_tmp_identities_truncated_total` | Counter | none |
| `scope3_tmp_unique_placements_per_auction` | Histogram | none |

**Deliberately not labels**: `property_rid`, `placement_id`, `seller_agent_url`, `auction_id` — cardinality explosion. Property-level breakdowns deferred to v0.1 with measurement.

### Analytics tags

Same `hookanalytics.Analytics` pattern as RTD `module.go:208-218`. Values: `endpoint`, `duration_ms`, `eligible_packages` (count, not contents), `cache_hit`, `error` (string, omit on success). No request/response bodies. No PII.

## Testing

### Coverage targets

- **90% line coverage** module-wide.
- **100% branch coverage** on `intersect()`, `accountResolver`, and per-imp result mapping.
- Trivial getters/setters and obvious `if err != nil { return err }` paths exempted via `//nolint:gocover` so we don't write low-value tests on plumbing.

### Layered tests

1. **Unit tests** — `account_test.go`, `async_request_test.go`, `masking_test.go`, Builder validation in `module_test.go`. Pure logic, no I/O.
2. **Hook-level tests** — `module_test.go` with `httptest.NewServer()` mock router. Covers single-imp, multi-imp three-placement, multi-imp shared placement, cache hit, cache TTL=0, partial-success (P1), missing config, 4xx, timeout, unknown placement, Builder rejection of >3 EIDs, `add_to_targeting` true.
3. **Wire-shape tests** — capture outbound JSON, assert: distinct `request_id`s, no `package_ids` on Identity Match, `Identities` ≤ 3, ISO alpha-2 country, no IP/IFA/user.id in either body.
4. **Optional fuzz** on `intersect`. Local sanity, not CI.

### Fixtures

```
testdata/
  bid_request_single_imp.json
  bid_request_multi_imp_three_placements.json
  bid_request_multi_imp_shared_placement.json
  bid_request_no_user.json
  bid_request_no_eids.json
  account_config_basic.json
  account_config_with_placement_override.json
  context_response_two_offers.json
  context_response_empty.json
  context_response_with_signals.json
  identity_response_two_packages.json
  identity_response_empty_with_tmpx.json
  error_response_invalid_request.json
```

### CI gate

Prebid Server's `.github/workflows/adapter-code-coverage.yml` runs against modules. Coverage target is set in module-local config; PR fails if below threshold.

## Migration plan

| Phase | What |
|---|---|
| Ship v0 | This module lands behind a default-disabled flag. RTD untouched. README on `modules/scope3/rtd/` gets a one-paragraph pointer. |
| Internal pilot | Scope3-operated test publisher runs both modules side-by-side; compare segment overlap and downstream targeting outcomes. |
| Customer migration | Each publisher enables `scope3.tmp` after registering their property in the AdCP registry and configuring per-account `property_rid` and placement maps. Publishers can run both modules during cutover; TMP wins on the `ext.scope3.*` collision. |
| RTD sunset | After all active publishers are on TMP, RTD is deleted in a follow-up PR. |

## Open follow-ups (explicitly out of v0 scope)

- v0.1 — per-account override of `preserve_eids` order (I2).
- v0.1 — extract identity-extraction and masking helpers into a shared `modules/scope3/internal/` package once both modules survive past internal pilot.
- v0.1 — `property_rid` label on `scope3_tmp_requests_total` metric, conditional on deployment size impact measurement.
- v0.1 — active health-check goroutine against router `/health`.
- v0.2 — AdCP registry sync (resolve `site.domain → property_rid` automatically via `adcp-go/registry`) so per-account config becomes optional.
- v0.2 — propagate `aee_signals.bidders`-style per-bidder routing if AdCP adds it.

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| `tmp.interchange.io` not deployed at module-ship time | Module loads but every auction is a `network_error` | Default disabled until router available; staging-only flag |
| TMP wire schema drift after we vendor `tmproto` types | Decode failures appear as `decode_error` | Re-sync `proto.go` manually; integration test asserts current schema; review on every `adcp-go` upstream tag |
| Multi-imp fan-out at high QPS overloads router | Tail latency degrades | Per-call timeout enforces upper bound; dedupe by placement_id reduces actual N |
| Identity cap of 3 drops the user's "best" identity | Reduced eligibility / cap accuracy | I1 makes the choice operator-controlled and explicit; `scope3_tmp_identities_sent` histogram surfaces truncation in real traffic |
| Prebid OSS maintainers reject vendored types | PR blocked | Fallback to hand-written types restricted to fields we use (smaller `proto.go`) |
| RTD and TMP both write `ext.scope3.*` differently and confuse downstream consumers | Targeting line items misfire | TMP wins by overwrite; documented; startup log line warns |
