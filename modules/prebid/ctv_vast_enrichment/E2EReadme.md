# CTV VAST Enrichment — End-to-End Test Suite

> **File:** `modules/prebid/ctv_vast_enrichment/module_e2e_test.go`
> **Package:** `ctv_vast_enrichment_test`
> **Total tests:** 25

---

## Overview

The end-to-end test suite exercises the **full hook path** of the `ctv_vast_enrichment` module using real sub-package implementations (`enrich.NewEnricher`, `format.NewFormatter`, `select.NewSelector`) instead of mocks.

This means regressions in integration points — config merging, VAST parsing, enrichment, marshaling — are caught at the boundary where all components work together, not just in isolation.

The suite is divided into four groups:

| Group | Prefix | Focus |
|-------|--------|-------|
| A | `TestE2E_A*` | Hook path correctness (HandleRawBidderResponseHook) |
| B | `TestE2E_B*` | Configuration merging correctness |
| C | `TestE2E_C*` | Pipeline end-to-end (BuildVastFromBidResponse) |
| D | `TestE2E_D*` | Regression tests — bugs documented in `ctv-bugs-and-resolve.md` |

---

## Group A — Hook Path Correctness

These tests call `HandleRawBidderResponseHook` directly and verify the resulting VAST XML after applying all ChangeSet mutations.

### A1 — Video bid enriched

**Scenario:** A single video bid with a valid VAST is submitted.

**Expectation:**
- `<Pricing model="CPM" currency="USD">` is injected into the VAST
- `<Advertiser>` is populated with the first entry from `ADomain`

---

### A2 — Banner bid passes through untouched

**Scenario:** A bid with `BidType = banner` is submitted with HTML content in `AdM`.

**Expectation:**
- The `AdM` field is identical before and after the hook
- No VAST parsing is attempted (BidType guard fires first)

---

### A3 — Native bid passes through untouched

**Scenario:** A bid with `BidType = native` is submitted with JSON content in `AdM`.

**Expectation:**
- The `AdM` field is identical before and after the hook
- No VAST parsing is attempted

---

### A4 — BidMeta preserved on enriched TypedBid

**Scenario:** A video bid carries `BidMeta` with `NetworkID = 42`, `AdvertiserID = 99`, `BrandID = 7`.

**Expectation:**
- After enrichment the `TypedBid.BidMeta` pointer is not nil
- All three fields retain their original values

---

### A5 — BidderResponse.Currency used in `<Pricing>`

**Scenario:** Host config sets `DefaultCurrency = "USD"`. The DSP's `BidderResponse.Currency` is `"EUR"`.

**Expectation:**
- `<Pricing currency="EUR">` appears in the VAST
- `currency="USD"` does **not** appear — DSP currency takes precedence over host default

---

### A6 — Fallback to DefaultCurrency when BidderResponse.Currency is empty

**Scenario:** The DSP omits `Currency` in its response. Host config sets `DefaultCurrency = "GBP"`.

**Expectation:**
- `<Pricing currency="GBP">` appears in the VAST
- The fallback chain is: `BidderResponse.Currency` → `DefaultCurrency` → `"USD"`

---

### A7 — VAST_WINS: existing `<Pricing>` not overwritten

**Scenario:** The VAST already contains `<Pricing model="CPM" currency="GBP">3.00</Pricing>`. The bid price is 9.99.

**Expectation:**
- Original `GBP` and `3.00` are preserved
- `9.99` does not appear — VAST_WINS collision policy protects existing data

---

### A8 — DSP-specific VAST extensions preserved after marshal

**Scenario:** The VAST contains a `<Extension type="dsp_custom">` block with a custom tracking URL.

**Expectation:**
- `type="dsp_custom"` survives the parse → enrich → marshal round-trip
- The DSP tracker URL survives unchanged

---

### A9 — Mixed bid types: only video bids enriched

**Scenario:** Two bids in one response — one `banner`, one `video`.

**Expectation:**
- Banner `AdM` is byte-identical before and after the hook
- Video `AdM` contains `<Pricing>`

---

## Group B — Configuration Correctness

### B1 — VAST_WINS collision policy round-trips through account config

**Scenario:** No host-level `collision_policy`. Account config sets `collision_policy = "VAST_WINS"`. VAST already has pricing.

**Expectation:**
- Original pricing is preserved (VAST_WINS applied correctly)
- Bidder price does not replace existing VAST pricing
- Verifies BUG 2: `VAST_WINS` was previously silently converted to `CollisionReject`

---

### B2 — Account config overrides host config

**Scenario:** Host config sets `default_currency = "USD"`. Account config sets `default_currency = "EUR"`. DSP omits currency.

**Expectation:**
- `currency="EUR"` appears in the VAST — account-level config wins over host-level

---

## Group C — Pipeline End-to-End

These tests call `BuildVastFromBidResponse` directly with real selector, enricher, and formatter components (no mocks).

### C1 — Single video bid → enriched VAST

**Scenario:** One bid at price 5.00 USD from seat "rubicon" with domain "brand.example.com".

**Expectation:**
- `result.NoAd` is false
- VAST contains `<Pricing>` with value 5
- VAST contains `<Advertiser>brand.example.com</Advertiser>`

---

### C2 — Ad pod: sequence attributes set correctly

**Scenario:** Three video bids with prices 10.0, 8.0, 6.0 using `SelectionTopN` with `MaxAdsInPod = 3`.

**Expectation:**
- All three bids are selected
- VAST output contains `sequence="1"`, `sequence="2"`, `sequence="3"` on each `<Ad>`

---

### C3 — No bids → NoAd VAST returned

**Scenario:** Empty `BidResponse` with no seat bids.

**Expectation:**
- `result.NoAd` is true
- A valid empty `<VAST>` document is returned (not an error)

---

### C4 — Invalid VAST, skeleton disabled → NoAd

**Scenario:** Bid has `AdM = "not-xml-at-all"`. `AllowSkeletonVast = false`.

**Expectation:**
- `result.NoAd` is true
- No panic, no error returned at the function level

---

### C5 — Invalid VAST, skeleton enabled → VAST with warning

**Scenario:** Bid has `AdM = "not-xml-at-all"`. `AllowSkeletonVast = true`.

**Expectation:**
- `result.NoAd` is false — a skeleton VAST is generated
- `result.Warnings` is non-empty (warning about invalid VAST parsing)

---

### C6 — Duration from metadata injected into `<Linear><Duration>`

**Scenario:** A custom selector injects `DurSec = 45` into `CanonicalMeta`. The VAST has no `<Duration>` element.

**Expectation:**
- Output VAST contains `<Duration>00:00:45</Duration>`

---

### C7 — IAB categories injected as VAST extension

**Scenario:** Bid has `Cat = ["IAB1", "IAB2-3"]`.

**Expectation:**
- Output VAST contains `IAB1` and `IAB2-3`
- Extension is typed `iab_category`

---

### C8 — Debug extension includes BidID and Seat

**Scenario:** `ReceiverConfig.Debug = true`. Bid ID is `"debug-bid-123"`, seat is `"rubicon"`.

**Expectation:**
- Output VAST contains `<BidID>debug-bid-123</BidID>`
- Output VAST contains `<Seat>rubicon</Seat>`
- Both wrapped in `<Extension type="openrtb">`

---

## Group D — Regression Tests

Each test in this group directly targets a specific bug documented in [`ctv-bugs-and-resolve.md`](ctv-bugs-and-resolve.md).

### D1 — Non-USD DSP currency preserved (BUG 1)

**Scenario:** Host config: `USD`. DSP responses in `EUR`, `JPY`, `BRL`, `AUD` (parametrized sub-tests).

**Expectation:**
- Each currency appears in `<Pricing currency="...">` unchanged
- `USD` does not appear in any output

**Bug fixed:** `BidderResponse.Currency` was ignored; `DefaultCurrency` from host config was always used, producing wrong labels like `<Pricing currency="USD">0.85</Pricing>` for a EUR DSP.

---

### D2 — VAST_WINS not silently converted to Reject (BUG 2)

**Scenario:** Host config sets `collision_policy = "VAST_WINS"`. VAST has existing pricing.

**Expectation:**
- Existing pricing preserved (VAST_WINS applied)

**Bug fixed:** The `switch` statement in `configToReceiverConfig` was missing the `VAST_WINS` case, causing it to fall through to the zero value `CollisionReject`. Publishers setting `VAST_WINS` got the opposite behavior with no error.

---

### D3 — Hook uses enrich subpackage (BUG 3)

**Scenario:** Account config sets `debug = true`. A video bid is submitted.

**Expectation:**
- Output VAST contains `<Extension type="openrtb">` with `<BidID>`
- This extension is only added by `enrich.VastEnricher`, not the fallback `hookEnricher`

**Bug fixed:** The hook was calling a private `enrichVastDocument()` function that only handled `Pricing` and `Advertiser`. The subpackages `enrich/`, `format/`, `select/` were completely bypassed. Config fields like `debug`, `selection_strategy`, `placement` had no effect in production.

---

### D4 — MediaFiles preserved after clearInnerXML + marshal (BUG 4)

**Scenario:** A VAST with a `<MediaFile>` containing a video URL is enriched.

**Expectation:**
- After enrichment and marshal, `<MediaFile>` element still exists
- The video URL is unchanged
- The `type="video/mp4"` attribute is preserved

**Bug fixed:** `clearInnerXML()` was zeroing all `,innerxml` fields recursively, including `Creative.InnerXML` and `Linear.InnerXML`. This silently dropped `<MediaFiles>`, `<TrackingEvents>`, and any DSP-specific extensions.

---

### D5 — BidMeta fields survive the hook mutation (BUG 6)

**Scenario:** A `TypedBid` carries `BidMeta` with `NetworkID`, `AdvertiserID`, `BrandID`, `PrimaryCategoryID`.

**Expectation:**
- After the hook, all four fields retain their original values

**Bug fixed:** When constructing the enriched `TypedBid`, `BidMeta` was not copied. Analytics and targeting systems downstream received `nil` instead of the original metadata.

---

### D6 — Only first ADomain used in `<Advertiser>` (BUG 7)

**Scenario:** Bid has `ADomain = ["primary.com", "secondary.com", "tertiary.com"]`.

**Expectation:**
- `<Advertiser>primary.com</Advertiser>` — exactly the first domain
- The string `"primary.com,secondary.com"` does not appear anywhere

**Bug fixed:** `strings.Join(bid.ADomain, ",")` was producing `"primary.com,secondary.com,tertiary.com"` as the advertiser value. VAST `<Advertiser>` is a single human-readable string — joining multiple domains is non-standard and breaks ad server parsing.

---

## Running the tests

```bash
# Run only the E2E suite
go test ./modules/prebid/ctv_vast_enrichment/... -run TestE2E -v

# Run the full module test suite
go test ./modules/prebid/ctv_vast_enrichment/... -v

# Run with race detector
go test ./modules/prebid/ctv_vast_enrichment/... -race -v
```

---

## Test fixtures used

| Constant | Description |
|----------|-------------|
| `minimalVAST` | Well-formed VAST 3.0 with one InLine ad, `<Duration>`, and `<MediaFile>`. Baseline for most tests. |
| `vastWithPricing` | VAST with existing `<Pricing model="CPM" currency="GBP">3.00</Pricing>`. Used for VAST_WINS tests. |
| `vastWithExtensions` | VAST with a `<Extension type="dsp_custom">` block. Used for clearInnerXML regression test. |

---

## Related files

| File | Description |
|------|-------------|
| [`ctv-bugs-and-resolve.md`](ctv-bugs-and-resolve.md) | Full bug descriptions, root cause analysis, and fix specifications |
| [`module.go`](module.go) | Hook implementation — entry point for all Group A and B tests |
| [`pipeline.go`](pipeline.go) | `BuildVastFromBidResponse` — entry point for all Group C tests |
| [`enrich/enrich.go`](enrich/enrich.go) | `VastEnricher` — enriches `<Pricing>`, `<Advertiser>`, `<Duration>`, categories, debug |
| [`model/vast_xml.go`](model/vast_xml.go) | VAST data model and `clearInnerXML` — relevant to D4 |
