# GET Interface — Reviewer Guide

> Branch: `GetInterface` | Base: `origin/master`

This document is written for code reviewers. It explains **what changed, why, and how to read each file** in this PR.

---

## Background

Prebid Server currently only accepts auction requests via HTTP **POST** (`/openrtb2/auction`) with a JSON body.  
CTV and Audio use cases require a **GET** endpoint — devices like smart TVs and set-top boxes often cannot send a POST body, and ad tags are typically URL-based.

This PR implements the first phase of the GET Interface as described in the technical spec:
*"Prebid Server Technical Response to Audio and CTV Requirements"*.

---

## What Changed — 10 Files

### Quick map

```
endpoints/openrtb2/
  get_auction.go          ← NEW  — core of this PR: query string → OpenRTB
  get_auction_test.go     ← NEW  — 14 test functions for the above
  auction.go              ← MOD  — wires GET into the existing POST flow
  auction_test.go         ← MOD  — fixes after exitpoint guard change

openrtb_ext/
  request.go              ← MOD  — 3 new fields on ExtRequestPrebid
  imp.go                  ← MOD  — 1 new field on ExtImpPrebid
  request_profiles_test.go ← NEW — serialization tests for new fields

router/router.go          ← MOD  — registers GET route

stored_requests/
  profiles.go             ← NEW  — ProfileFetcher interface + MergeProfiles
  profiles_test.go        ← NEW  — 7 tests for merge logic
```

---

## File-by-file Walkthrough

### 1. `endpoints/openrtb2/get_auction.go` ← **Start here**

This is the heart of the PR. The single exported entry point is:

```go
func parseGETRequest(r *http.Request) ([]byte, error)
```

It reads the HTTP GET query string and produces a **JSON-serialized `openrtb2.BidRequest`** that is then fed back into the existing `parseRequest` flow — exactly as if it were a POST body. No auction logic is duplicated.

**`srid` is the only required parameter.** Without a stored request ID the server cannot know the basic auction structure (channel, media type, bidders). Everything else is optional and overlays on top.

**Parameter precedence (lowest → highest priority):**

| Layer | Source |
|---|---|
| 1 | Stored request (loaded later via existing `processStoredRequests`) |
| 2 | Request profiles (`rprof`) — declared in `ext.prebid.profiles` |
| 3 | Individual query params mapped to OpenRTB fields |
| 4 | HTTP headers (`Referer`, `User-Agent`, `X-Forwarded-For` — handled by existing code) |

**Media type routing** (`mtype` param):

| `mtype` value | Result |
|---|---|
| `1` or absent | `imp[0].banner` populated if `w`/`h` given |
| `2` or `vid` | `imp[0].video` populated with video params |
| `3` or `aud` | `imp[0].audio` populated with audio params |

**Key parameter mappings** (subset, see full list in `PBS GET Parameters - Query Params`):

| Query param | OpenRTB field |
|---|---|
| `srid` | `ext.prebid.storedrequest.id` |
| `slot` | `imp[0].tagid` |
| `pubid` | `site.publisher.id` |
| `rprof` / `req_profiles` | `ext.prebid.profiles[]` |
| `iprof` / `imp_profiles` | `imp[0].ext.prebid.profiles[]` |
| `of` | `ext.prebid.of` (output format, e.g. `vast4`) |
| `om` | `ext.prebid.om` (output module) |
| `tmax` | `tmax` (ignored if < 100ms) |
| `gdpr` | `regs.gdpr` |
| `gdpr_consent` | `user.consent` |
| `gppc` | `regs.gpp` |
| `coppa` | `regs.coppa` |
| `cgenre`, `ctitle`, `clang` | `site.content.genre/title/language` |
| `mindur`, `maxdur` | `imp[0].video.minduration/maxduration` |
| `proto` | `imp[0].video.protocols` (comma-separated) |

Helper functions at the bottom of the file (`qFirst`, `qCSV`, `qInts`) are private utilities for reading query values with alias support (e.g. `gdpr_consent` and `tcfc` both map to the same field).

---

### 2. `endpoints/openrtb2/get_auction_test.go`

Table-driven tests covering all major parameter groups. Each test function is self-contained and uses `httptest.NewRequest` to build a fake GET request.

If you want to verify a specific mapping, search for the test name:

| Test | What it verifies |
|---|---|
| `TestParseGETRequest_RequiresSrid` | error when `srid` missing |
| `TestParseGETRequest_SridInStoredRequest` | `srid` → `ext.prebid.storedrequest.id` |
| `TestParseGETRequest_Tmax` | tmax validation (≥100 required) |
| `TestParseGETRequest_Profiles` | `rprof` and `iprof` wiring |
| `TestParseGETRequest_OutputFormat` | `of` and `om` fields |
| `TestParseGETRequest_VideoParams` | video imp params |
| `TestParseGETRequest_AudioParams` | audio imp params |
| `TestParseGETRequest_Privacy` | GDPR, GPP, COPPA |
| `TestParseGETRequest_ContentParams` | site.content fields |
| `TestParseGETRequest_CSVParams` | comma-separated arrays (`proto`, `api`) |

---

### 3. `endpoints/openrtb2/auction.go`

**Two changes only:**

**Change A** — GET request injection (around line 433):
```go
if httpRequest.Method == http.MethodGet {
    getBody, getErr := parseGETRequest(httpRequest)
    if getErr != nil { ... return }
    httpRequest.Body = io.NopCloser(strings.NewReader(string(getBody)))
}
```
This runs at the very top of `parseRequest`, before the body is read. It replaces the empty GET body with the JSON produced by `parseGETRequest`. From this point forward the existing POST path handles everything — stored request merge, validation, auction, hooks — unchanged.

**Change B** — exitpoint guard in `sendAuctionResponse`:
```go
if !hasErrors {
    finalResponse = hookExecutor.ExecuteExitpointStage(response, w)
}
```
Per spec the exitpoint stage must only run when there are **no errors** during auction processing. Previously it ran on rejection paths too. The `hasErrors bool` parameter was added to `sendAuctionResponse`; `rejectAuctionRequest` passes `true`.

---

### 4. `router/router.go`

Single line addition:
```go
r.POST("/openrtb2/auction", openrtbEndpoint)  // existing
r.GET("/openrtb2/auction", openrtbEndpoint)   // new
```
Both methods share the same handler. The GET-specific logic is entirely inside `parseGETRequest`.

---

### 5. `openrtb_ext/request.go`

Three new fields added to `ExtRequestPrebid`:

```go
Profiles      []string `json:"profiles,omitempty"`  // ext.prebid.profiles
OutputFormat  string   `json:"of,omitempty"`         // ext.prebid.of
OutputModule  string   `json:"om,omitempty"`         // ext.prebid.om
```

And one new field on `ExtRequestPrebidServer`:
```go
RequestMethod string `json:"requestmethod,omitempty"`  // "GET" or "POST"
```

`RequestMethod` is set to `"GET"` by `parseGETRequest` so that exit-point modules can detect the channel and decide whether to activate (e.g. a VAST formatter module should only run on GET CTV requests, not regular POST auctions).

---

### 6. `openrtb_ext/imp.go`

One new field on `ExtImpPrebid`:
```go
Profiles []string `json:"profiles,omitempty"`  // imp[].ext.prebid.profiles
```
This carries impression-level profile IDs passed via the `iprof` query param.

---

### 7. `stored_requests/profiles.go`

Defines the **Profiles** concept: small named OpenRTB fragments stored server-side, merged into the request to avoid sending large repeated payloads on every URL.

```go
type ProfileFetcher interface {
    FetchProfiles(ctx context.Context, accountID string, profileIDs []string) (map[string]json.RawMessage, []error)
}
```

`MergeProfiles` applies profiles sequentially using **RFC 7396 JSON Merge Patch** (deep merge, not shallow replace):

```go
func MergeProfiles(baseJSON []byte, profileIDs []string, profileData map[string]json.RawMessage) ([]byte, []error)
```

`NoopProfileFetcher` is the default stub — returns empty maps. The actual storage backend (DB / file / HTTP) is **out of scope for this PR** and will be implemented separately.

---

### 8. `openrtb_ext/request_profiles_test.go` and `stored_requests/profiles_test.go`

Serialization and merge logic tests. Notable cases in `profiles_test.go`:

- `TestMergeProfiles_MultipleProfiles_OrderMatters` — verifies that later profiles win on conflict
- `TestMergeProfiles_DeepMerge` — verifies RFC 7396 (nested fields are merged, not replaced)
- `TestMergeProfiles_MissingProfileSkipped` — missing profile is silently skipped, no fatal error

---

## What Is NOT in This PR (Planned Next)

| Item | Notes |
|---|---|
| Profile storage backend | DB/file/HTTP fetcher for named profiles |
| `MergeProfiles` call in `processStoredRequests` | Wiring profiles into the auction pipeline |
| `auction.profiles.limit` config | Per-account profile count limit |
| Single impression enforcement | Discard imp[1..n] with sampled log warning |
| VAST response module (exitpoint) | Separate PR — `ctv_vast_enrichment` branch |

---

## How to Test Locally

```bash
# Run new tests only
go test -mod=mod ./endpoints/openrtb2/ -run "TestParseGET" -v
go test -mod=mod ./stored_requests/ -run "TestMerge|TestNoop" -v
go test -mod=mod ./openrtb_ext/ -run "TestExt.*Profile|TestExtRequestPrebidServer" -v

# Run all affected packages
go test -mod=mod ./endpoints/openrtb2/ ./stored_requests/ ./openrtb_ext/ ./router/
```

**Manual smoke test** (requires a running PBS with a stored request `test-sr-1` defined):
```
GET /openrtb2/auction?srid=test-sr-1&mtype=2&slot=my-slot&mindur=5&maxdur=30&pubid=pub-123&gdpr=0
```

Expected: valid OpenRTB auction response (same as equivalent POST).

---

## Review Checklist

- [ ] `parseGETRequest` — parameter mappings match the spec document
- [ ] `srid` required, all others optional — confirmed
- [ ] GET and POST share the same auction pipeline — no logic duplication
- [ ] Exitpoint only fires on success path (`!hasErrors`) — confirmed
- [ ] `RequestMethod` set correctly so future modules can detect GET channel
- [ ] `MergeProfiles` uses deep merge (RFC 7396), not shallow replace
- [ ] All new fields use `omitempty` — no impact on existing POST requests
- [ ] Tests cover error paths (missing srid, invalid tmax, missing profiles)
