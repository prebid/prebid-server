# Gap Analysis: CTV VAST Enrichment Module vs Tech Spec (issue #3726)

## Context

Issue [#3726 — Support general GET interface](https://github.com/prebid/prebid-server/issues/3726) defines a generalized GET interface for Prebid Server supporting both Audio and CTV use cases. The Technical Response document specifies an architecture consisting of:

1. **GET Interface** — PBS Core handles GET on `/openrtb2/auction`
2. **Profiles** — new core feature (per-account ORTB fragments)
3. **Exitpoint hook stage** — new hook stage for response format modification
4. **Community Modules** — HTTP Header, Ranking, VAST Response, Mapping, VAST Unwrapping

The `ctv_vast_enrichment` module must be aligned with this architecture.

---

## Current Module State

### What Is Implemented ✅

| Component | Status | File |
|-----------|--------|------|
| RawBidderResponse Hook | ✅ Working | `module.go` |
| VAST Enrichment (Pricing, Advertiser, Categories, Duration) | ✅ Working | `enrich/enrich.go` |
| Bid Selection (SINGLE, TOP_N, MAX_REVENUE) | ✅ Working | `select/selector.go` |
| VAST Formatting (GAM SSU) | ✅ Working | `format/format.go` |
| 3-layer config merge (host → account → profile) | ✅ Merge works | `config.go` |
| Pipeline orchestration | ✅ Working | `pipeline.go` |
| VAST XML parser + skeleton | ✅ Working | `model/parser.go` |
| HTTP Handler (GET, builder pattern) | ⚠️ Partial | `handler.go` |
| Registration in `modules/builder.go` | ✅ Registered | `modules/builder.go` |

### What Does NOT Exist in PBS Core ❌

| Feature from Tech Spec | Status in PBS | Implication |
|------------------------|---------------|-------------|
| GET on `/openrtb2/auction` | ❌ POST only | Module cannot serve GET without core changes |
| `ext.prebid.profiles` | ❌ Does not exist | Profiles are not parsed or merged |
| `ext.prebid.of` (output format) | ❌ Does not exist | No mechanism to signal response format |
| `ext.prebid.rank` (ranking) | ❌ Does not exist | No standardized bid ranking |
| `ext.prebid.outputmodule` | ❌ Does not exist | No mechanism to select output module |
| `RequestMethod` in AuctionContext | ❌ Not available | Modules don't know GET vs POST |
| Exitpoint hook stage | ✅ Exists | `hooks/hookstage/exitpoint.go` — payload `{Response any, W http.ResponseWriter}` |

---

## Gap Analysis: Module vs Tech Spec

### 1. Endpoint: `/ctv/vast` vs GET on `/openrtb2/auction`

**Current plan (endpoint.md):** Dedicated `/ctv/vast` endpoint in `handler.go`.

**Tech Spec says:** GET should work on the existing `/openrtb2/auction`. No separate `/ctv/vast` endpoint is defined. Response format depends on `ext.prebid.of` (e.g., `vast3`, `vast4`). An exitpoint module checks this parameter and formats accordingly.

**Required changes:**
- ❌ **Abandon** the separate `/ctv/vast` endpoint concept
- ✅ HTTP handler (`handler.go`) becomes unnecessary in its current form
- ✅ Module should act as an **exitpoint hook** formatting VAST, NOT as a separate endpoint
- ⚠️ Alternative: keep `/ctv/vast` as a simple convenience redirect (but this is not part of the spec)

### 2. Query Parameter Parsing → PBS Core's Responsibility

**Current plan:** `handler.go` → `buildBidRequest()` parses query params.

**Tech Spec says:** PBS Core parses ~60 GET parameters (see spreadsheet), builds `BidRequest`, merges stored requests and profiles. The module receives a ready-made `BidRequest` just like with POST.

**Required changes:**
- ❌ `buildBidRequest()` in handler.go is redundant
- ✅ Module doesn't need to parse query params — PBS Core does it
- ✅ Module receives standard `BidRequest` from hooks

### 3. Winner Selection → Ranking Module (Separate Module)

**Current module:** `select/selector.go` implements bid selection internally.

**Tech Spec says:** Ranking is a **separate module** at the `all-processed-bid-responses` stage. It sets `seatbid.bid.ext.prebid.rank`. The VAST response module reads rank and formats the pod.

**Required changes:**
- ⚠️ Internal selector is **redundant** with the Ranking Module — but that module doesn't exist yet
- ✅ Target: VAST module should read `ext.prebid.rank` instead of self-selecting
- ✅ For now: keep selector as a fallback until ranking module is built
- ⚠️ Need to add logic: "if bids have `ext.prebid.rank`, use it; otherwise use internal selector"

### 4. Response Formatting → Exitpoint Hook

**Current module:** Enrichment at `RawBidderResponse` + pipeline in `handler.go`.

**Tech Spec says:** VAST response module operates at the **exitpoint stage**. It checks:
1. Whether the request came via GET (`ext.prebid.server.requestmethod`)
2. Whether imp[] contains only media types it handles
3. Whether `ext.prebid.of` is a format this module handles

Then it serializes `BidResponse` to VAST XML.

**Required changes:**
- ✅ **New hook:** Implement `HandleExitpointHook()` in `module.go`
- ✅ Exitpoint hook builds VAST from `BidResponse` (pipeline.go already does this)
- ✅ Hook checks `ext.prebid.of` (vast3/vast4) to decide whether to format
- ⚠️ `RawBidderResponse` hook is **still needed** for enrichment (adding Pricing/Advertiser)
- ✅ Separation of concerns:
  - `RawBidderResponse` → VAST enrichment in individual bids
  - `Exitpoint` → formatting the final response as VAST

### 5. Configuration — `ext.prebid.of` and `ext.prebid.outputmodule`

**Current module:** Does not react to these fields (they don't exist in PBS).

**Tech Spec says:** `ext.prebid.of` = "vast3"/"vast4" signals format. `ext.prebid.outputmodule` allows specifying which module formats.

**Required changes:**
- ⚠️ **Blocked by PBS Core** — these fields must first be added to `openrtb_ext.ExtRequestPrebid`
- ✅ Module should read them in the exitpoint hook
- ✅ If `of` = "vast3"/"vast4" → module formats VAST
- ✅ If `of` is empty or "ortb2" → module does not intervene

### 6. Profiles — Third Configuration Source

**Current module:** `MergeCTVVastConfig(host, account, profile)` — 3-layer merge implemented, but `profile` always `nil`.

**Tech Spec says:** Profiles are ORTB fragments, not module config. But the layered configuration concept is compatible.

**Required changes:**
- ⚠️ **Blocked by PBS Core** — profiles must be implemented in core first
- ✅ `MergeCTVVastConfig()` structure is already ready for profiles
- ✅ Need to connect profile config retrieval from hook context when the feature becomes available

---

## Implementation Plan — Priorities

### Phase 1: Module Adaptation (No Core Changes)

| # | Task | File(s) | Priority |
|---|------|---------|----------|
| 1.1 | Implement `HandleExitpointHook()` | `module.go` | **P0** |
| 1.2 | Exitpoint logic: check `ext.prebid.of`, build VAST from BidResponse | `module.go` | **P0** |
| 1.3 | Reuse `pipeline.go` in exitpoint hook | `pipeline.go` | **P0** |
| 1.4 | Add exitpoint to `host_execution_plan` in `pbs.json` | `pbs.json` | **P0** |
| 1.5 | Unit tests for exitpoint hook | `module_test.go` | **P0** |
| 1.6 | Ranking fallback — read `ext.prebid.rank` if available | `select/selector.go` | **P1** |
| 1.7 | VAST version handling from config | `format/format.go` | **P1** |

### Phase 2: PBS Core Changes (Require Core Team Review)

| # | Task | File(s) | Priority |
|---|------|---------|----------|
| 2.1 | Add GET support to `/openrtb2/auction` | `router/router.go`, new handler | **P0** |
| 2.2 | Query parameter parser → BidRequest | new package in `endpoints/` | **P0** |
| 2.3 | Add `ext.prebid.of` to `ExtRequestPrebid` | `openrtb_ext/request.go` | **P0** |
| 2.4 | Add `ext.prebid.outputmodule` to `ExtRequestPrebid` | `openrtb_ext/request.go` | **P1** |
| 2.5 | Add `ext.prebid.profiles` to `ExtRequestPrebid` | `openrtb_ext/request.go` | **P1** |
| 2.6 | Implement Profile storage and merge | `stored_requests/`, `config/` | **P1** |
| 2.7 | Add `RequestMethod` to AuctionContext / `ext.prebid.server` | `exchange/exchange.go` | **P1** |
| 2.8 | Add `EndpointAuctionGET` constant in hookexecution | `hooks/hookexecution/executor.go` | **P1** |

### Phase 3: Community Modules (Separate Issues)

| # | Module | Hook Stage | Status |
|---|--------|------------|--------|
| 3.1 | HTTP Header Module | `RawAuctionRequest` | ❌ Does not exist |
| 3.2 | Bid Response Ranking Module | `AllProcessedBidResponses` | ❌ Does not exist |
| 3.3 | VAST Unwrapping & Validation Module | TBD | ❌ Does not exist |
| 3.4 | Category Mapping Module | `ProcessedAuctionRequest` | ❌ Does not exist |

---

## Target Architecture — Request Flow

```
GET /openrtb2/auction?srid=my-stored-req&of=vast4&pubid=pub-1&mindur=15&maxdur=60
    │
    ▼
┌────────────────────────────────────────────────┐
│  PBS Core: GET Handler                          │
│  1. Parse query params → partial BidRequest     │
│  2. Load stored request (srid)                  │
│  3. Load & merge profiles (rprof, iprof)        │
│  4. Merge all layers                            │
│  5. Set ext.prebid.of = "vast4"                 │
│  6. Set ext.prebid.server.requestmethod = "GET" │
└─────────────────┬──────────────────────────────┘
                  ▼
┌────────────────────────────────────────────────┐
│  Hook: Entrypoint Stage                         │
│  → HTTP Header Module (X-Device-IP, etc.)       │
└─────────────────┬──────────────────────────────┘
                  ▼
┌────────────────────────────────────────────────┐
│  exchange.HoldAuction()                         │
│  ├─ ProcessedAuction → Mapping Module           │
│  ├─ BidderRequest    → per bidder               │
│  ├─ RawBidderResponse                           │
│  │   └─ ctv_vast_enrichment: ENRICHMENT         │
│  │      (adds Pricing, Advertiser, Categories)  │
│  └─ AllProcessedBidResponses                    │
│      └─ Ranking Module: sets ext.prebid.rank    │
└─────────────────┬──────────────────────────────┘
                  ▼
┌────────────────────────────────────────────────┐
│  Hook: AuctionResponse Stage                    │
└─────────────────┬──────────────────────────────┘
                  ▼
┌────────────────────────────────────────────────┐
│  Hook: Exitpoint Stage                          │
│  → ctv_vast_enrichment: VAST FORMATTING         │
│    1. Check ext.prebid.of == "vast4"?           │
│    2. Read ext.prebid.rank from bids            │
│    3. Select bids (rank or selector fallback)   │
│    4. Build VAST XML (pipeline.go)              │
│    5. Set Content-Type: application/xml         │
│    6. Return VAST instead of JSON               │
└─────────────────┬──────────────────────────────┘
                  ▼
        VAST XML Response
```

---

## Summary: What the Module Does Well vs What Needs Change

### ✅ Keep (Aligned with Tech Spec)

1. **RawBidderResponse Hook** — VAST enrichment (Pricing, Advertiser, Categories, Duration) is exactly what CTV/Audio Req8/Req9 require
2. **Pipeline orchestration** — `BuildVastFromBidResponse()` composes VAST well
3. **3-layer config merge** — ready for profiles
4. **VAST parser + skeleton** — solid foundation
5. **VAST_WINS enrichment policy** — aligned with requirements (don't overwrite existing values)

### ❌ Needs Change

1. **Separate `/ctv/vast` endpoint** → abandon, use `/openrtb2/auction` GET + exitpoint
2. **HTTP handler** → replace with exitpoint hook
3. **Internal bid selection** → adapt to read `ext.prebid.rank` (with fallback)
4. **No exitpoint hook** → implement `HandleExitpointHook()`

### ⚠️ Blockers (Waiting on PBS Core)

1. GET on `/openrtb2/auction` — **does not exist**
2. `ext.prebid.of` — **does not exist** in `ExtRequestPrebid`
3. `ext.prebid.profiles` — **does not exist** in core
4. `ext.prebid.rank` — **does not exist** (requires Ranking Module)
5. `ext.prebid.server.requestmethod` — **does not exist**

---

## Recommendation: What To Do Now

1. **Implement exitpoint hook** — the only change that doesn't require PBS Core modifications and aligns with the target architecture
2. **Keep RawBidderResponse hook** — enrichment is a separate responsibility from formatting
3. **Keep handler.go as optional** — convenience endpoint during transition period
4. **Contact core team** (as suggested in the issue) regarding timeline for GET interface and profiles
5. **Prepare tests** for the exitpoint hook with mocked `ext.prebid.of`
