# CTV VAST Enrichment — Scenariusze Testów Manualnych (Postman / curl)

## Jak uruchomić

```bash
cd /workspaces/prebid-server
go build .
./prebid-server -v 1 -logtostderr
```

Serwer startuje na **http://localhost:8000** (auction) i **:6060** (admin).

### Import do Postmana

Zaimportuj plik `sample/ctv_vast_enrichment_postman_collection.json` do Postmana.  
Zmienna `{{base_url}}` = `http://localhost:8000`.  
Kolekcja ma wbudowane asserty JS — kliknij "Run Collection" żeby odpalić wszystko naraz.

---

## Stworzone pliki danych

| Plik | Opis |
|------|------|
| `data/stored_responses/bid-vast-1.json` | VAST bez Pricing/Advertiser, price=1.50, adomain=www.advertiser.com (istniejący) |
| `data/stored_responses/bid-vast-2.json` | VAST **z** istniejącym Pricing=9.99 EUR i Advertiser=OriginalAdvertiser, price=2.75 (istniejący) |
| `data/stored_responses/bid-vast-no-pricing.json` | VAST bez Pricing/Advertiser, price=3.50, adomain=www.basic-advertiser.com |
| `data/stored_responses/bid-vast-empty-adm.json` | Bid z pustym `"adm": ""` |
| `data/stored_responses/bid-vast-banner.json` | Bid z banner HTML (`<img>`) zamiast VAST XML |
| `data/stored_responses/bid-vast-no-adomain.json` | VAST bez Pricing, price=5.00, `"adomain": []` (pusta tablica) |
| `data/stored_responses/bid-vast-multiple-cats.json` | VAST bez Pricing, price=7.25, cat=["IAB1","IAB3-1","IAB10"] |
| `data/stored_responses/bid-vast-zero-price.json` | VAST bez Pricing, price=0, adomain=www.free-advertiser.com |
| `stored_requests/data/by_id/accounts/ctv-test.json` | Konto testowe z `default_currency: "EUR"` (nadpisuje host config) |

### Konfiguracja modułu w `pbs.json`

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

## Scenariusze testowe

### Test 1: Podstawowe wzbogacanie — dodaje `<Pricing>` + `<Advertiser>`

**Stored response:** `bid-vast-no-pricing.json`  
**Co zawiera:** VAST XML bez `<Pricing>` i bez `<Advertiser>`, price=3.50, adomain=www.basic-advertiser.com

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

**Oczekiwane:**
- ✅ AdM zawiera `<Pricing model="CPM" currency="USD">3.5</Pricing>`
- ✅ AdM zawiera `<Advertiser>www.basic-advertiser.com</Advertiser>`
- ✅ Reszta VAST (AdSystem, Duration, MediaFiles) bez zmian

---

### Test 2: Polityka VAST_WINS — istniejące wartości NIE nadpisane

**Stored response:** `bid-vast-2.json`  
**Co zawiera:** VAST XML **z** istniejącym `<Pricing model="CPM" currency="EUR">9.99</Pricing>` i `<Advertiser>OriginalAdvertiser</Advertiser>`. Bid ma price=2.75 i adomain=www.different-advertiser.com

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

**Oczekiwane:**
- ✅ Pricing = **9.99 EUR** (oryginał z VAST, nie 2.75 USD z bida)
- ✅ Advertiser = **OriginalAdvertiser** (oryginał, nie www.different-advertiser.com)
- ❌ Moduł NIE nadpisuje istniejących wartości

---

### Test 3: Pusty AdM — moduł pomija bid

**Stored response:** `bid-vast-empty-adm.json`  
**Co zawiera:** Bid z `"adm": ""` (pusty string)

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

**Oczekiwane:**
- ✅ Bid przechodzi bez zmian
- ✅ Brak `<Pricing>` w odpowiedzi
- ✅ Brak błędu HTTP

---

### Test 4: Non-VAST AdM (banner HTML) — graceful skip

**Stored response:** `bid-vast-banner.json`  
**Co zawiera:** Bid z `"adm": "<img src=\"https://example.com/ad-banner.png\" />"` — HTML zamiast VAST XML

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

**Oczekiwane:**
- ✅ XML parsing fail → bid przechodzi bez zmian
- ✅ AdM nie zawiera `<Pricing>` ani `<Advertiser>`
- ✅ Oryginalny HTML nie jest zmieniony

---

### Test 5: Brak adomain — Pricing dodany, Advertiser NIE

**Stored response:** `bid-vast-no-adomain.json`  
**Co zawiera:** VAST XML, price=5.00, `"adomain": []` (pusta tablica)

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

**Oczekiwane:**
- ✅ `<Pricing model="CPM" currency="USD">5</Pricing>` — dodany (price > 0)
- ❌ Brak `<Advertiser>` — nie dodany (pusta adomain)

---

### Test 6: Price=0 — brak Pricing, ale Advertiser dodany

**Stored response:** `bid-vast-zero-price.json`  
**Co zawiera:** VAST XML, price=0, adomain=www.free-advertiser.com

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

**Oczekiwane:**
- ❌ Brak `<Pricing>` — nie dodany (price musi być > 0)
- ✅ `<Advertiser>www.free-advertiser.com</Advertiser>` — dodany (adomain istnieje)

---

### Test 7: Kategorie IAB

**Stored response:** `bid-vast-multiple-cats.json`  
**Co zawiera:** VAST XML, price=7.25, adomain=www.categorized-advertiser.com, cat=["IAB1","IAB3-1","IAB10"]

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

**Oczekiwane:**
- ✅ `<Pricing model="CPM" currency="USD">7.25</Pricing>` — dodany
- ✅ `<Advertiser>www.categorized-advertiser.com</Advertiser>` — dodany
- ✅ Kategorie IAB mogą pojawić się jako VAST Extensions

---

### Test 8: Nadpisanie konfiguracji z poziomu konta — waluta EUR

**Stored response:** `bid-vast-no-pricing.json` (ten sam co Test 1)  
**Konto:** `stored_requests/data/by_id/accounts/ctv-test.json` — ustawia `default_currency: "EUR"`  
**Klucz:** `"publisher": {"id": "ctv-test"}` w request body mapuje serwer na konto `ctv-test.json`

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

**Oczekiwane:**
- ✅ `<Pricing model="CPM" currency="EUR">3.5</Pricing>` — waluta **EUR** zamiast USD
- ✅ Konto nadpisuje konfigurację hosta

---

### Test 9: Podstawowy VAST (bid-vast-1)

**Stored response:** `bid-vast-1.json`  
**Co zawiera:** VAST XML bez Pricing/Advertiser, price=1.50, adomain=www.advertiser.com

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

**Oczekiwane:**
- ✅ `<Pricing model="CPM" currency="USD">1.5</Pricing>` — dodany
- ✅ `<Advertiser>www.advertiser.com</Advertiser>` — dodany
- ✅ Cena bida (1.50) nie zmieniona w odpowiedzi

---

### Test 10: Health check — serwer działa

```bash
curl -s http://localhost:8000/status
```

**Oczekiwane:**
- ✅ HTTP 200
- ✅ Odpowiedź potwierdza, że serwer działa

---

## Matryca pokrycia

| # | Scenariusz | Stored response | Pricing | Advertiser | Uwagi |
|---|-----------|----------------|---------|------------|-------|
| 1 | Podstawowe wzbogacanie | bid-vast-no-pricing | ✅ dodany | ✅ dodany | Happy path |
| 2 | VAST_WINS | bid-vast-2 | ❌ nie nadpisany | ❌ nie nadpisany | Collision policy |
| 3 | Pusty AdM | bid-vast-empty-adm | ❌ pominięty | ❌ pominięty | Edge case |
| 4 | Non-VAST (HTML) | bid-vast-banner | ❌ pominięty | ❌ pominięty | Graceful degradation |
| 5 | Brak adomain | bid-vast-no-adomain | ✅ dodany | ❌ brak danych | Partial enrichment |
| 6 | Price=0 | bid-vast-zero-price | ❌ price=0 | ✅ dodany | Partial enrichment |
| 7 | Kategorie IAB | bid-vast-multiple-cats | ✅ dodany | ✅ dodany | Extensions |
| 8 | Account override (EUR) | bid-vast-no-pricing + ctv-test account | ✅ EUR | ✅ dodany | Config layering |
| 9 | Podstawowy VAST | bid-vast-1 | ✅ dodany | ✅ dodany | Sanity check |
| 10 | Health check | — | — | — | Serwer dostępny |
