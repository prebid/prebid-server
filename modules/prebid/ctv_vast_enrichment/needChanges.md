# Analiza zmian wymaganych do zgodności z Tech Spec (issue #3726)

## Kontekst

Issue [#3726 — Support general GET interface](https://github.com/prebid/prebid-server/issues/3726) definiuje uogólniony interfejs GET dla Prebid Server, wspierający zarówno Audio jak i CTV. Techniczny dokument odpowiedzi (Tech Spec) określa architekturę składającą się z:

1. **GET Interface** — PBS Core obsługuje GET na `/openrtb2/auction`
2. **Profiles** — nowy feature core'owy (fragmenty ORTB per-account)
3. **Exitpoint hook stage** — nowy etap hooków do modyfikacji formatu odpowiedzi
4. **Moduły community** — HTTP Header, Ranking, VAST Response, Mapping, VAST Unwrapping

Moduł `ctv_vast_enrichment` musi zostać dostosowany do tej architektury.

---

## Stan obecny modułu

### Co jest zaimplementowane ✅

| Komponent | Status | Plik |
|-----------|--------|------|
| RawBidderResponse Hook | ✅ Działa | `module.go` |
| VAST Enrichment (Pricing, Advertiser, Categories, Duration) | ✅ Działa | `enrich/enrich.go` |
| Bid Selection (SINGLE, TOP_N, MAX_REVENUE) | ✅ Działa | `select/selector.go` |
| VAST Formatting (GAM SSU) | ✅ Działa | `format/format.go` |
| 3-warstwowa konfiguracja (host → account → profile) | ✅ Merge działa | `config.go` |
| Pipeline orchestration | ✅ Działa | `pipeline.go` |
| VAST XML parser + skeleton | ✅ Działa | `model/parser.go` |
| HTTP Handler (GET, builder pattern) | ⚠️ Częściowy | `handler.go` |
| Rejestracja w `modules/builder.go` | ✅ Zarejestrowany | `modules/builder.go` |

### Co NIE istnieje w PBS Core ❌

| Feature z Tech Spec | Status w PBS | Implikacja |
|---------------------|--------------|------------|
| GET na `/openrtb2/auction` | ❌ Tylko POST | Moduł nie może działać przez GET bez zmian w core |
| `ext.prebid.profiles` | ❌ Nie istnieje | Profiles nie są parsowane ani mergowane |
| `ext.prebid.of` (output format) | ❌ Nie istnieje | Brak mechanizmu sygnalizacji formatu odpowiedzi |
| `ext.prebid.rank` (ranking) | ❌ Nie istnieje | Brak standardowego rankingu bidów |
| `ext.prebid.outputmodule` | ❌ Nie istnieje | Brak mechanizmu wyboru modułu wyjściowego |
| `RequestMethod` w AuctionContext | ❌ Nie dostępne | Moduły nie wiedzą czy request przyszedł GET vs POST |
| Exitpoint hook stage | ✅ Istnieje | `hooks/hookstage/exitpoint.go` — payload `{Response any, W http.ResponseWriter}` |

---

## Analiza rozbieżności: Moduł vs Tech Spec

### 1. Endpoint: `/ctv/vast` vs GET na `/openrtb2/auction`

**Obecny plan (endpoint.md):** Dedykowany endpoint `/ctv/vast` w `handler.go`.

**Tech Spec mówi:** GET powinien działać na istniejącym `/openrtb2/auction`. Nie definiuje osobnego endpointu `/ctv/vast`. Format odpowiedzi zależy od `ext.prebid.of` (np. `vast3`, `vast4`). Moduł exitpoint sprawdza ten parametr i formatuje odpowiednio.

**Co trzeba zmienić:**
- ❌ **Porzucić** koncepcję osobnego endpointu `/ctv/vast`
- ✅ Handler HTTP (`handler.go`) staje się zbędny w obecnej formie
- ✅ Moduł powinien działać jako **exitpoint hook** formatujący VAST, NIE jako osobny endpoint
- ⚠️ Alternatywa: zachować `/ctv/vast` jako prosty redirector/convenience endpoint (ale nie jest to częścią spec)

### 2. Parsowanie parametrów query → PBS Core odpowiada za to

**Obecny plan:** `handler.go` → `buildBidRequest()` parsuje query params.

**Tech Spec mówi:** PBS Core parsuje ~60 parametrów GET (spreadsheet), buduje `BidRequest`, merguje stored requests i profiles. Moduł dostaje gotowy `BidRequest` jak przy POST.

**Co trzeba zmienić:**
- ❌ `buildBidRequest()` w handler.go jest zbędny
- ✅ Moduł nie musi parsować query params — PBS Core to robi
- ✅ Moduł dostaje standardowy `BidRequest` z hooków

### 3. Selekcja zwycięzcy → Ranking Module (osobny moduł)

**Obecny moduł:** `select/selector.go` implementuje selekcję bidów wewnętrznie.

**Tech Spec mówi:** Ranking to **osobny moduł** na etapie `all-processed-bid-responses`. Ustawia `seatbid.bid.ext.prebid.rank`. VAST response module czyta rank i formatuje pod.

**Co trzeba zmienić:**
- ⚠️ Selector w module jest **redundantny** z Ranking Module — ale ten moduł jeszcze nie istnieje
- ✅ Docelowo: moduł VAST powinien czytać `ext.prebid.rank` zamiast samodzielnie selekcjonować
- ✅ Na razie: zachować selector jako fallback dopóki ranking module nie powstanie
- ⚠️ Trzeba dodać logikę: "jeśli bidy mają `ext.prebid.rank`, użyj go; w przeciwnym razie użyj selektora"

### 4. Format odpowiedzi → Exitpoint Hook

**Obecny moduł:** Enrichment na `RawBidderResponse` + pipeline w `handler.go`.

**Tech Spec mówi:** VAST response module działa na **exitpoint stage**. Sprawdza:
1. Czy request przyszedł przez GET (`ext.prebid.server.requestmethod`)
2. Czy imp[] zawiera obsługiwane media types
3. Czy `ext.prebid.of` to format obsługiwany przez ten moduł

Następnie serializuje `BidResponse` do VAST XML.

**Co trzeba zmienić:**
- ✅ **Nowy hook:** Implementacja `HandleExitpointHook()` w `module.go`
- ✅ Hook exitpoint buduje VAST z `BidResponse` (pipeline.go już to umie)
- ✅ Hook sprawdza `ext.prebid.of` (vast3/vast4) i decyduje czy formatować
- ⚠️ `RawBidderResponse` hook **nadal jest potrzebny** do enrichmentu (dodawanie Pricing/Advertiser)
- ✅ Rozdzielenie odpowiedzialności:
  - `RawBidderResponse` → enrichment VAST w bidach
  - `Exitpoint` → formatowanie końcowej odpowiedzi jako VAST

### 5. Konfiguracja — `ext.prebid.of` i `ext.prebid.outputmodule`

**Obecny moduł:** Nie reaguje na te pola (nie istnieją w PBS).

**Tech Spec mówi:** `ext.prebid.of` = "vast3"/"vast4" sygnalizuje format. `ext.prebid.outputmodule` pozwala określić który moduł formatuje.

**Co trzeba zmienić:**
- ⚠️ **Blokowane przez PBS Core** — te pola muszą być najpierw dodane do `openrtb_ext.ExtRequestPrebid`
- ✅ Moduł powinien czytać je w exitpoint hook
- ✅ Jeśli `of` = "vast3"/"vast4" → moduł formatuje VAST
- ✅ Jeśli `of` jest puste lub "ortb2" → moduł nie interweniuje

### 6. Profiles — trzecie źródło konfiguracji

**Obecny moduł:** `MergeCTVVastConfig(host, account, profile)` — merge 3-warstwowy zaimplementowany, ale `profile` zawsze `nil`.

**Tech Spec mówi:** Profiles to ORTB fragments, nie module config. Ale koncepcja warstwowej konfiguracji jest zgodna.

**Co trzeba zmienić:**
- ⚠️ **Blokowane przez PBS Core** — profiles muszą być najpierw zaimplementowane w core
- ✅ Struktura `MergeCTVVastConfig()` jest gotowa na profiles
- ✅ Trzeba podłączyć pobieranie profile config z kontekstu hooka gdy feature będzie dostępny

---

## Plan implementacji — priorytety

### Faza 1: Dostosowanie modułu (bez zmian w core)

| # | Zadanie | Plik(i) | Priorytet |
|---|---------|---------|-----------|
| 1.1 | Implementacja `HandleExitpointHook()` | `module.go` | **P0** |
| 1.2 | Logika exitpoint: sprawdź `ext.prebid.of`, zbuduj VAST z BidResponse | `module.go` | **P0** |
| 1.3 | Reużycie `pipeline.go` w exitpoint hook | `pipeline.go` | **P0** |
| 1.4 | Dodanie exitpoint do `host_execution_plan` w `pbs.json` | `pbs.json` | **P0** |
| 1.5 | Testy unit dla exitpoint hook | `module_test.go` | **P0** |
| 1.6 | Fallback rankingu — czytaj `ext.prebid.rank` jeśli dostępny | `select/selector.go` | **P1** |
| 1.7 | Obsługa VAST version z konfiguracji | `format/format.go` | **P1** |

### Faza 2: Zmiany w PBS Core (wymagają review core team)

| # | Zadanie | Plik(i) | Priorytet |
|---|---------|---------|-----------|
| 2.1 | Dodanie GET na `/openrtb2/auction` | `router/router.go`, nowy handler | **P0** |
| 2.2 | Parser query parameters → BidRequest | nowy pakiet w `endpoints/` | **P0** |
| 2.3 | Dodanie `ext.prebid.of` do `ExtRequestPrebid` | `openrtb_ext/request.go` | **P0** |
| 2.4 | Dodanie `ext.prebid.outputmodule` do `ExtRequestPrebid` | `openrtb_ext/request.go` | **P1** |
| 2.5 | Dodanie `ext.prebid.profiles` do `ExtRequestPrebid` | `openrtb_ext/request.go` | **P1** |
| 2.6 | Implementacja Profile storage i merge | `stored_requests/`, `config/` | **P1** |
| 2.7 | Dodanie `RequestMethod` do `AuctionContext` / `ext.prebid.server` | `exchange/exchange.go` | **P1** |
| 2.8 | Stała `EndpointAuctionGET` w hookexecution | `hooks/hookexecution/executor.go` | **P1** |

### Faza 3: Moduły Community (osobne issue)

| # | Moduł | Hook Stage | Status |
|---|-------|------------|--------|
| 3.1 | HTTP Header Module | `RawAuctionRequest` | ❌ Nie istnieje |
| 3.2 | Bid Response Ranking Module | `AllProcessedBidResponses` | ❌ Nie istnieje |
| 3.3 | VAST Unwrapping & Validation Module | TBD | ❌ Nie istnieje |
| 3.4 | Category Mapping Module | `ProcessedAuctionRequest` | ❌ Nie istnieje |

---

## Architektura docelowa — przepływ

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
│  │      (dodaje Pricing, Advertiser, Categories) │
│  └─ AllProcessedBidResponses                    │
│      └─ Ranking Module: ustawia ext.prebid.rank │
└─────────────────┬──────────────────────────────┘
                  ▼
┌────────────────────────────────────────────────┐
│  Hook: AuctionResponse Stage                    │
└─────────────────┬──────────────────────────────┘
                  ▼
┌────────────────────────────────────────────────┐
│  Hook: Exitpoint Stage                          │
│  → ctv_vast_enrichment: VAST FORMATTING         │
│    1. Sprawdź ext.prebid.of == "vast4"?         │
│    2. Czytaj ext.prebid.rank z bidów            │
│    3. Wybierz bidy (rank lub selector fallback) │
│    4. Buduj VAST XML (pipeline.go)              │
│    5. Ustaw Content-Type: application/xml       │
│    6. Zwróć VAST zamiast JSON                   │
└─────────────────┬──────────────────────────────┘
                  ▼
        VAST XML Response
```

---

## Podsumowanie: co moduł robi dobrze a co wymaga zmiany

### ✅ Do zachowania (zgodne z Tech Spec)

1. **RawBidderResponse Hook** — enrichment VAST (Pricing, Advertiser, Categories, Duration) jest dokładnie tym czego wymaga Req8/Req9 z CTV/Audio
2. **Pipeline orchestration** — `BuildVastFromBidResponse()` dobrze komponuje VAST
3. **Konfiguracja 3-warstwowa** — gotowa na profiles
4. **VAST parser + skeleton** — solidna baza
5. **Enrich policy VAST_WINS** — zgodna z wymaganiami (nie nadpisuj istniejących wartości)

### ❌ Do zmiany

1. **Osobny endpoint `/ctv/vast`** → porzucić, użyć `/openrtb2/auction` GET + exitpoint
2. **Handler HTTP** → zastąpić exitpoint hookiem
3. **Wewnętrzna selekcja bidów** → adaptować na `ext.prebid.rank` (z fallbackiem)
4. **Brak exitpoint hook** → zaimplementować `HandleExitpointHook()`

### ⚠️ Blokery (czekają na PBS Core)

1. GET na `/openrtb2/auction` — **nie istnieje**
2. `ext.prebid.of` — **nie istnieje** w `ExtRequestPrebid`
3. `ext.prebid.profiles` — **nie istnieje** w core
4. `ext.prebid.rank` — **nie istnieje** (wymaga Ranking Module)
5. `ext.prebid.server.requestmethod` — **nie istnieje**

---

## Rekomendacja: co robić teraz

1. **Zaimplementować exitpoint hook** — jedyna zmiana, która nie wymaga modyfikacji PBS Core i jest zgodna z docelową architekturą
2. **Zachować RawBidderResponse hook** — enrichment to oddzielna odpowiedzialność od formatowania
3. **Zachować handler.go jako opcjonalny** — convenience endpoint na czas przejściowy
4. **Skontaktować się z core team** (jak sugeruje issue) w sprawie timeline dla GET interface i profiles
5. **Przygotować testy** dla exitpoint hook z mockami `ext.prebid.of`
