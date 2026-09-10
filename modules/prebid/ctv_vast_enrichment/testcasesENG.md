# CTV VAST Enrichment — Manual Test Scenarios (Postman / curl)

## How to Run

```bash
cd /workspaces/prebid-server
go build .
./prebid-server -v 1 -logtostderr
```

Server starts on **http://localhost:8000** (auction) and **:6060** (admin).

### Import into Postman

Import the file `sample/ctv_vast_enrichment_postman_collection.json` into Postman.  
Variable `{{base_url}}` = `http://localhost:8000`.  
The collection has built-in JS test assertions — click "Run Collection" to execute all tests at once.

---

## Created Data Files

| File | Description |
|------|-------------|
| `data/stored_responses/bid-vast-1.json` | VAST without Pricing/Advertiser, price=1.50, adomain=www.advertiser.com (pre-existing) |
| `data/stored_responses/bid-vast-2.json` | VAST **with** existing Pricing=9.99 EUR and Advertiser=OriginalAdvertiser, price=2.75 (pre-existing) |
| `data/stored_responses/bid-vast-no-pricing.json` | VAST without Pricing/Advertiser, price=3.50, adomain=www.basic-advertiser.com |
| `data/stored_responses/bid-vast-empty-adm.json` | Bid with empty `"adm": ""` |
| `data/stored_responses/bid-vast-banner.json` | Bid with banner HTML (`<img>`) instead of VAST XML |
| `data/stored_responses/bid-vast-no-adomain.json` | VAST without Pricing, price=5.00, `"adomain": []` (empty array) |
| `data/stored_responses/bid-vast-multiple-cats.json` | VAST without Pricing, price=7.25, cat=["IAB1","IAB3-1","IAB10"] |
| `data/stored_responses/bid-vast-zero-price.json` | VAST without Pricing, price=0, adomain=www.free-advertiser.com |
| `stored_requests/data/by_id/accounts/ctv-test.json` | Test account with `default_currency: "EUR"` (overrides host config) |

### Module Configuration in `pbs.json`

```json
{
  "hooks": {
    "enabled": true,
    "modules": {
      "prebid": {
        "ctv_vast_enrichment": {
          "enabled": true,
          "receiver": "GAM_SSU",
          "default_currency": "USD",
          "vast_version_default": "3.0",
          "max_ads_in_pod": 5,
          "selection_strategy": "max_revenue"
        }
      }
    },
    "host_execution_plan": {
      "endpoints": {
        "/openrtb2/auction": {
          "stages": {
            "raw_bidder_response": {
              "groups": [{
                "timeout": 1000,
                "hook_sequence": [{
                  "module_code": "prebid.ctv_vast_enrichment",
                  "hook_impl_code": "code123"
                }]
              }]
            }
          }
        }
      }
    }
  }
}
```

---

## Test Scenarios

### Test 1: Basic Enrichment — Adds `<Pricing>` + `<Advertiser>`

**Stored response:** `bid-vast-no-pricing.json`  
**Contents:** VAST XML without `<Pricing>` or `<Advertiser>`, price=3.50, adomain=www.basic-advertiser.com

```bash
curl -s -X POST http://localhost:8000/openrtb2/auction \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-1",
    "imp": [{
      "id": "test-div-1",
      "video": {"mimes": ["video/mp4"], "protocols": [1,2,5], "w": 640, "h": 360},
      "ext": {"prebid": {
        "bidder": {"appnexus": {"placementId": 12345}},
        "storedbidresponse": [{"bidder": "appnexus", "id": "bid-vast-no-pricing"}]
      }}
    }],
    "site": {"page": "https://example.com", "publisher": {"id": "pub-1"}},
    "regs": {"ext": {"gdpr": 0}}
  }' | python3 -m json.tool
```

**Expected:**
- ✅ AdM contains `<Pricing model="CPM" currency="USD">3.5</Pricing>`
- ✅ AdM contains `<Advertiser>www.basic-advertiser.com</Advertiser>`
- ✅ Rest of VAST (AdSystem, Duration, MediaFiles) unchanged

---

### Test 2: VAST_WINS Policy — Existing Values NOT Overwritten

**Stored response:** `bid-vast-2.json`  
**Contents:** VAST XML **with** existing `<Pricing model="CPM" currency="EUR">9.99</Pricing>` and `<Advertiser>OriginalAdvertiser</Advertiser>`. Bid has price=2.75 and adomain=www.different-advertiser.com

```bash
curl -s -X POST http://localhost:8000/openrtb2/auction \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-2",
    "imp": [{
      "id": "test-div-1",
      "video": {"mimes": ["video/mp4"], "protocols": [1,2,5], "w": 1920, "h": 1080},
      "ext": {"prebid": {
        "bidder": {"appnexus": {"placementId": 12345}},
        "storedbidresponse": [{"bidder": "appnexus", "id": "bid-vast-2"}]
      }}
    }],
    "site": {"page": "https://example.com", "publisher": {"id": "pub-1"}},
    "regs": {"ext": {"gdpr": 0}}
  }' | python3 -m json.tool
```

**Expected:**
- ✅ Pricing = **9.99 EUR** (original from VAST, not 2.75 USD from bid)
- ✅ Advertiser = **OriginalAdvertiser** (original, not www.different-advertiser.com)
- ❌ Module does NOT overwrite existing values

---

### Test 3: Empty AdM — Module Skips Bid

**Stored response:** `bid-vast-empty-adm.json`  
**Contents:** Bid with `"adm": ""` (empty string)

```bash
curl -s -X POST http://localhost:8000/openrtb2/auction \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-3",
    "imp": [{
      "id": "test-div-1",
      "video": {"mimes": ["video/mp4"], "protocols": [1,2,5], "w": 640, "h": 360},
      "ext": {"prebid": {
        "bidder": {"appnexus": {"placementId": 12345}},
        "storedbidresponse": [{"bidder": "appnexus", "id": "bid-vast-empty-adm"}]
      }}
    }],
    "site": {"page": "https://example.com", "publisher": {"id": "pub-1"}},
    "regs": {"ext": {"gdpr": 0}}
  }' | python3 -m json.tool
```

**Expected:**
- ✅ Bid passes through unchanged
- ✅ No `<Pricing>` in response
- ✅ No HTTP error

---

### Test 4: Non-VAST AdM (Banner HTML) — Graceful Skip

**Stored response:** `bid-vast-banner.json`  
**Contents:** Bid with `"adm": "<img src=\"https://example.com/ad-banner.png\" />"` — HTML instead of VAST XML

```bash
curl -s -X POST http://localhost:8000/openrtb2/auction \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-4",
    "imp": [{
      "id": "test-div-1",
      "video": {"mimes": ["video/mp4"], "protocols": [1,2,5], "w": 640, "h": 360},
      "ext": {"prebid": {
        "bidder": {"appnexus": {"placementId": 12345}},
        "storedbidresponse": [{"bidder": "appnexus", "id": "bid-vast-banner"}]
      }}
    }],
    "site": {"page": "https://example.com", "publisher": {"id": "pub-1"}},
    "regs": {"ext": {"gdpr": 0}}
  }' | python3 -m json.tool
```

**Expected:**
- ✅ XML parsing fails → bid passes through unchanged
- ✅ AdM does not contain `<Pricing>` or `<Advertiser>`
- ✅ Original HTML is not modified

---

### Test 5: Missing Adomain — Pricing Added, No Advertiser

**Stored response:** `bid-vast-no-adomain.json`  
**Contents:** VAST XML, price=5.00, `"adomain": []` (empty array)

```bash
curl -s -X POST http://localhost:8000/openrtb2/auction \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-5",
    "imp": [{
      "id": "test-div-1",
      "video": {"mimes": ["video/mp4"], "protocols": [1,2,5], "w": 640, "h": 360},
      "ext": {"prebid": {
        "bidder": {"appnexus": {"placementId": 12345}},
        "storedbidresponse": [{"bidder": "appnexus", "id": "bid-vast-no-adomain"}]
      }}
    }],
    "site": {"page": "https://example.com", "publisher": {"id": "pub-1"}},
    "regs": {"ext": {"gdpr": 0}}
  }' | python3 -m json.tool
```

**Expected:**
- ✅ `<Pricing model="CPM" currency="USD">5</Pricing>` — added (price > 0)
- ❌ No `<Advertiser>` — not added (empty adomain)

---

### Test 6: Zero Price — No Pricing, But Advertiser Added

**Stored response:** `bid-vast-zero-price.json`  
**Contents:** VAST XML, price=0, adomain=www.free-advertiser.com

```bash
curl -s -X POST http://localhost:8000/openrtb2/auction \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-6",
    "imp": [{
      "id": "test-div-1",
      "video": {"mimes": ["video/mp4"], "protocols": [1,2,5], "w": 640, "h": 360},
      "ext": {"prebid": {
        "bidder": {"appnexus": {"placementId": 12345}},
        "storedbidresponse": [{"bidder": "appnexus", "id": "bid-vast-zero-price"}]
      }}
    }],
    "site": {"page": "https://example.com", "publisher": {"id": "pub-1"}},
    "regs": {"ext": {"gdpr": 0}}
  }' | python3 -m json.tool
```

**Expected:**
- ❌ No `<Pricing>` — not added (price must be > 0)
- ✅ `<Advertiser>www.free-advertiser.com</Advertiser>` — added (adomain exists)

---

### Test 7: IAB Categories

**Stored response:** `bid-vast-multiple-cats.json`  
**Contents:** VAST XML, price=7.25, adomain=www.categorized-advertiser.com, cat=["IAB1","IAB3-1","IAB10"]

```bash
curl -s -X POST http://localhost:8000/openrtb2/auction \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-7",
    "imp": [{
      "id": "test-div-1",
      "video": {"mimes": ["video/mp4"], "protocols": [1,2,5], "w": 1280, "h": 720},
      "ext": {"prebid": {
        "bidder": {"appnexus": {"placementId": 12345}},
        "storedbidresponse": [{"bidder": "appnexus", "id": "bid-vast-multiple-cats"}]
      }}
    }],
    "site": {"page": "https://example.com", "publisher": {"id": "pub-1"}},
    "regs": {"ext": {"gdpr": 0}}
  }' | python3 -m json.tool
```

**Expected:**
- ✅ `<Pricing model="CPM" currency="USD">7.25</Pricing>` — added
- ✅ `<Advertiser>www.categorized-advertiser.com</Advertiser>` — added
- ✅ IAB categories may appear as VAST Extensions

---

### Test 8: Account-Level Config Override — EUR Currency

**Stored response:** `bid-vast-no-pricing.json` (same as Test 1)  
**Account:** `stored_requests/data/by_id/accounts/ctv-test.json` — sets `default_currency: "EUR"`  
**Key:** `"publisher": {"id": "ctv-test"}` in request body maps the server to the `ctv-test.json` account

```bash
curl -s -X POST http://localhost:8000/openrtb2/auction \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-8",
    "imp": [{
      "id": "test-div-1",
      "video": {"mimes": ["video/mp4"], "protocols": [1,2,5], "w": 640, "h": 360},
      "ext": {"prebid": {
        "bidder": {"appnexus": {"placementId": 12345}},
        "storedbidresponse": [{"bidder": "appnexus", "id": "bid-vast-no-pricing"}]
      }}
    }],
    "site": {"page": "https://example.com", "publisher": {"id": "ctv-test"}},
    "regs": {"ext": {"gdpr": 0}}
  }' | python3 -m json.tool
```

**Expected:**
- ✅ `<Pricing model="CPM" currency="EUR">3.5</Pricing>` — currency is **EUR** instead of USD
- ✅ Account config overrides host config

---

### Test 9: Basic VAST (bid-vast-1)

**Stored response:** `bid-vast-1.json`  
**Contents:** VAST XML without Pricing/Advertiser, price=1.50, adomain=www.advertiser.com

```bash
curl -s -X POST http://localhost:8000/openrtb2/auction \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-9",
    "imp": [{
      "id": "test-div-1",
      "video": {"mimes": ["video/mp4"], "protocols": [1,2,5], "w": 640, "h": 360},
      "ext": {"prebid": {
        "bidder": {"appnexus": {"placementId": 12345}},
        "storedbidresponse": [{"bidder": "appnexus", "id": "bid-vast-1"}]
      }}
    }],
    "site": {"page": "https://example.com", "publisher": {"id": "pub-1"}},
    "regs": {"ext": {"gdpr": 0}}
  }' | python3 -m json.tool
```

**Expected:**
- ✅ `<Pricing model="CPM" currency="USD">1.5</Pricing>` — added
- ✅ `<Advertiser>www.advertiser.com</Advertiser>` — added
- ✅ Bid price (1.50) unchanged in response

---

### Test 10: Health Check — Server Running

```bash
curl -s http://localhost:8000/status
```

**Expected:**
- ✅ HTTP 200
- ✅ Response confirms server is running

---

## Coverage Matrix

| # | Scenario | Stored Response | Pricing | Advertiser | Notes |
|---|----------|----------------|---------|------------|-------|
| 1 | Basic enrichment | bid-vast-no-pricing | ✅ added | ✅ added | Happy path |
| 2 | VAST_WINS policy | bid-vast-2 | ❌ not overwritten | ❌ not overwritten | Collision policy |
| 3 | Empty AdM | bid-vast-empty-adm | ❌ skipped | ❌ skipped | Edge case |
| 4 | Non-VAST (HTML) | bid-vast-banner | ❌ skipped | ❌ skipped | Graceful degradation |
| 5 | Missing adomain | bid-vast-no-adomain | ✅ added | ❌ no data | Partial enrichment |
| 6 | Zero price | bid-vast-zero-price | ❌ price=0 | ✅ added | Partial enrichment |
| 7 | IAB categories | bid-vast-multiple-cats | ✅ added | ✅ added | Extensions |
| 8 | Account override (EUR) | bid-vast-no-pricing + ctv-test account | ✅ EUR | ✅ added | Config layering |
| 9 | Basic VAST | bid-vast-1 | ✅ added | ✅ added | Sanity check |
| 10 | Health check | — | — | — | Server available |
