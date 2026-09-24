# Propozycja: Endpoint GET `/ctv/vast` dla modułu CTV VAST Enrichment

## Stan obecny

Moduł `ctv_vast_enrichment` działa wyłącznie jako **hook PBS** na etapie `RawBidderResponse` — wzbogaca VAST XML w odpowiedziach bidderów podczas standardowej aukcji POST `/openrtb2/auction`.

Istnieje już plik `handler.go` z przygotowanym `Handler struct` implementującym `http.Handler` (metoda `ServeHTTP`), ale **nie ma mechanizmu rejestracji** tego handlera w routerze PBS. Handler jest "osierocony" — nie jest podłączony do żadnej trasy HTTP.

### Kluczowy problem

System modułów PBS (`modules.NewBuilder().Build()`) zwraca wyłącznie `hooks.HookRepository` — repozytorium hooków. **Moduły nie mają dostępu do routera HTTP** i nie mogą samodzielnie rejestrować endpointów. Wszystkie trasy HTTP są hardkodowane w `router/router.go` w funkcji `New()`.

## Proponowane podejście

### Opcja A: Rejestracja endpointu bezpośrednio w routerze (rekomendowana)

Tak samo jak robią to istniejące endpointy (`/openrtb2/amp`, `/openrtb2/video`, `/event` itp.) — endpoint jest tworzony w `router/router.go` i rejestrowany ręcznie.

#### Kroki implementacji

**1. Nowy interfejs `ModuleEndpointProvider` (opcjonalny, ale czyściejszy)**

Aby nie hardkodować wszystkiego w routerze, moduł może eksponować interfejs informujący o endpointach:

```go
// modules/moduledeps/endpoint.go
package moduledeps

import "net/http"

// ModuleEndpoint opisuje endpoint HTTP dostarczany przez moduł.
type ModuleEndpoint struct {
    Method  string       // "GET", "POST"
    Path    string       // np. "/ctv/vast"
    Handler http.Handler
}

// EndpointProvider to opcjonalny interfejs, który moduł może implementować,
// aby zarejestrować własne endpointy HTTP.
type EndpointProvider interface {
    Endpoints() []ModuleEndpoint
}
```

**2. Implementacja interfejsu w module**

```go
// modules/prebid/ctv_vast_enrichment/module.go

func (m Module) Endpoints() []moduledeps.ModuleEndpoint {
    handler := NewHandler().
        WithConfig(configToReceiverConfig(MergeCTVVastConfig(&m.hostConfig, nil, nil)))
    // Selector, Enricher, Formatter zostaną wstrzyknięte przez router
    // lub podłączone z domyślnymi implementacjami

    return []moduledeps.ModuleEndpoint{
        {
            Method:  "GET",
            Path:    "/ctv/vast",
            Handler: handler,
        },
    }
}
```

**3. Rejestracja w routerze**

Rozszerzenie `router/router.go` — po `Build()` modułów sprawdzamy, czy moduły implementują `EndpointProvider`:

```go
// router/router.go w funkcji New(), po modules.NewBuilder().Build()

// Rejestracja endpointów modułów
for id, module := range builtModules {
    if ep, ok := module.(moduledeps.EndpointProvider); ok {
        for _, endpoint := range ep.Endpoints() {
            logger.Infof("Registering module endpoint: %s %s (module: %s)", endpoint.Method, endpoint.Path, id)
            r.Handler(endpoint.Method, endpoint.Path, endpoint.Handler)
        }
    }
}
```

**4. Wstrzyknięcie AuctionFunc**

Handler potrzebuje `AuctionFunc` do wywoływania aukcji. To jest najważniejszy element — trzeba przekazać referencję do exchange'a:

```go
handler.WithAuctionFunc(func(ctx context.Context, req *openrtb2.BidRequest) (*openrtb2.BidResponse, error) {
    // Użyj theExchange.HoldAuction() lub dedykowaną metodę
    return theExchange.RunSimpleAuction(ctx, req)
})
```

#### Diagram przepływu (Opcja A)

```
GET /ctv/vast?pod_id=123&duration=30&max_ads=3
        │
        ▼
┌─────────────────────┐
│  router/router.go   │  ← rejestracja: r.Handler("GET", "/ctv/vast", handler)
│  httprouter         │
└─────────┬───────────┘
          ▼
┌─────────────────────┐
│  Handler.ServeHTTP  │  ← handler.go (już istnieje)
│                     │
│  1. Parse query     │  ← buildBidRequest() - TODO do implementacji
│  2. Build BidReq    │
│  3. AuctionFunc()   │  ← wstrzyknięty exchange
│  4. Pipeline VAST   │  ← BuildVastFromBidResponse() - już istnieje
│  5. Return XML      │
└─────────────────────┘
```

---

### Opcja B: Endpoint jako osobny pakiet w `endpoints/` (prostsze, bez nowego interfejsu)

Bez tworzenia nowego interfejsu modułowego — po prostu dodajemy endpoint w `endpoints/` tak jak inne:

#### Kroki

**1. Nowy plik `endpoints/ctv_vast.go`**

```go
package endpoints

import (
    "net/http"

    "github.com/julienschmidt/httprouter"
    ctv "github.com/prebid/prebid-server/v4/modules/prebid/ctv_vast_enrichment"
)

func NewCTVVastEndpoint(cfg ctv.ReceiverConfig, auctionFn ctv.AuctionFunc) httprouter.Handle {
    handler := ctv.NewHandler().
        WithConfig(cfg).
        WithSelector(selectpkg.NewDefaultSelector()).
        WithEnricher(enrichpkg.NewDefaultEnricher()).
        WithFormatter(formatpkg.NewGAMSSUFormatter()).
        WithAuctionFunc(auctionFn)

    return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
        handler.ServeHTTP(w, r)
    }
}
```

**2. Rejestracja w `router/router.go`**

```go
// W funkcji New(), obok istniejących endpointów:
if cfg.Hooks.Modules["prebid"]["ctv_vast_enrichment"] != nil {
    r.GET("/ctv/vast", endpoints.NewCTVVastEndpoint(ctvConfig, auctionFunc))
}
```

---

### Opcja C: Rejestracja przez Exitpoint Hook (bez zmian w routerze)

Zamiast nowego endpointu, moduł mógłby "przejąć" istniejący endpoint `/openrtb2/auction` przez hook `Exitpoint` i zmienić format odpowiedzi na VAST XML gdy wykryje specjalny parametr. **Nie rekomendowane** — to hack, nie czyste rozwiązanie.

---

## Rekomendacja

**Opcja A** (interfejs `EndpointProvider`) jest najczystsza architektonicznie:

| Kryterium | Opcja A | Opcja B | Opcja C |
|-----------|---------|---------|---------|
| Czystość architektury | ✅ Extensible | ⚠️ Hardcoded | ❌ Hack |
| Łatwość implementacji | ⚠️ Nowy interfejs | ✅ Proste | ✅ Proste |
| Reużywalność | ✅ Inne moduły też skorzystają | ❌ Jednorazowe | ❌ Jednorazowe |
| Zgodność z PBS | ⚠️ Wymaga zmiany w `modules/` | ✅ Istniejący wzorzec | ⚠️ Nadużycie hooków |
| Minimum zmian w core | ❌ 3 pliki core | ✅ 2 pliki core | ✅ 0 plików core |

**Dla szybkiego MVP: Opcja B** — najmniej zmian, zgodna z istniejącymi wzorcami PBS.

**Dla długoterminowej architektury: Opcja A** — tworzy mechanizm reużywalny dla przyszłych modułów.

---

## Co jest do zrobienia w każdym podejściu

Niezależnie od wybranej opcji, `handler.go` wymaga implementacji:

### 1. Parsowanie query parameters (`buildBidRequest`)

```go
func (h *Handler) buildBidRequest(r *http.Request) (*openrtb2.BidRequest, error) {
    q := r.URL.Query()

    podID := q.Get("pod_id")           // wymagany
    duration := q.Get("duration")       // max duration w sekundach
    maxAds := q.Get("max_ads")          // max reklam w podzie
    publisherID := q.Get("pub_id")      // ID publishera
    siteURL := q.Get("url")             // URL strony
    // ... dalsze parametry
}
```

### 2. Wstrzyknięcie Selector/Enricher/Formatter

Handler potrzebuje konkretnych implementacji interfejsów. Należy zdecydować:
- Czy tworzyć je w Builderze modułu?
- Czy w endpoincie w routerze?

### 3. Wstrzyknięcie AuctionFunc

Najważniejsza zależność — handler musi móc wywołać aukcję PBS. Wymaga dostępu do `exchange.Exchange`.

### 4. Testy

- Unit test `handler_test.go` z mockowanym `AuctionFunc`
- Integration test z pełnym pipeline (Postman collection częściowo istnieje)

---

## Podsumowanie ścieżki implementacji (Opcja B — MVP)

```
1. endpoints/ctv_vast.go          ← nowy plik: wrapper endpoint
2. router/router.go               ← dodanie r.GET("/ctv/vast", ...)
3. handler.go                     ← implementacja buildBidRequest()
4. handler_test.go                ← testy
5. config (opcjonalnie)           ← feature flag w pbs.json
```

Łączna estymacja: ~5 plików do zmiany/utworzenia.

---

## Jak moduły są uruchamiane przez endpoint GET — mechanizm Hook Executor

Moduły **nie są uruchamiane "przez endpoint" bezpośrednio** — są uruchamiane przez **hook executor**, który endpoint tworzy i wywołuje na odpowiednich etapach.

### Łańcuch wywołań (na przykładzie AMP)

```
GET /openrtb2/amp?tag_id=xyz
  │
  ▼
router.go: r.GET("/openrtb2/amp", ampEndpoint)
  │
  ▼
AmpAuction handler tworzy hook executor:
  hookExecutor := hookexecution.NewHookExecutor(planBuilder, "/openrtb2/amp", metrics)
  │
  ├─► hookExecutor.ExecuteEntrypointStage(r, nil)     ← moduły dostają raw *http.Request
  │
  ├─► exchange.HoldAuction(ctx, AuctionRequest{HookExecutor: hookExecutor, ...})
  │     wewnątrz exchange:
  │     ├─► ExecuteProcessedAuctionStage()
  │     ├─► ExecuteBidderRequestStage()        ← per bidder
  │     ├─► ExecuteRawBidderResponseStage()    ← TU działa ctv_vast_enrichment!
  │     └─► ExecuteAllProcessedBidResponsesStage()
  │
  ├─► hookExecutor.ExecuteAuctionResponseStage(response)
  └─► hookExecutor.ExecuteExitpointStage(ampResponse, w)
```

AMP uruchamia **7 z 8 etapów** — wszystkie oprócz `RawAuctionRequest` (bo GET nie ma body JSON).

### Kluczowy mechanizm: Execution Plan

Hook executor wie, które moduły uruchomić, bo **szuka endpointu w `host_execution_plan`** — to jest literalne wyszukiwanie w mapie (`hooks/plan.go`):

```go
cfg.Endpoints[endpoint].Stages[stage].Groups
```

Jeśli dany endpoint **nie jest kluczem** w mapie `endpoints` — **żadne hooki nie zostaną odpalone**, nawet jeśli moduł implementuje interfejs. Struktura konfiguracji (`config/hooks.go`):

```go
type HookExecutionPlan struct {
    Endpoints map[string]struct {      // klucz = "/openrtb2/auction", "/openrtb2/amp", itp.
        Stages map[string]struct {     // klucz = "entrypoint", "raw_bidder_response", itp.
            Groups []HookExecutionGroup
        }
    }
}
```

### Co to oznacza dla nowego endpointu `/ctv/vast`

Nowy endpoint GET musi zrobić **dokładnie to samo co AMP**:

**1. Zdefiniować stałą endpointu** w `hooks/hookexecution/executor.go`:

```go
const EndpointCtvVast = "/ctv/vast"
```

**2. Stworzyć hook executor** w handlerze:

```go
hookExecutor := hookexecution.NewHookExecutor(planBuilder, EndpointCtvVast, metrics)
```

**3. Wywołać etapy hooków** w odpowiednich momentach:

```go
// Na początku
hookExecutor.ExecuteEntrypointStage(r, nil)

// Przekazać executor do exchange
exchange.HoldAuction(ctx, AuctionRequest{HookExecutor: hookExecutor, ...})
// ↑ wewnątrz exchange automatycznie odpalą się:
//   ProcessedAuctionRequest, BidderRequest, RawBidderResponse, AllProcessedBidResponses

// Po aukcji
hookExecutor.ExecuteAuctionResponseStage(response)
hookExecutor.ExecuteExitpointStage(vastXML, w)
```

**4. Dodać endpoint do execution plan** w `pbs.json`:

```json
"host_execution_plan": {
  "endpoints": {
    "/openrtb2/auction": { ... },
    "/ctv/vast": {
      "stages": {
        "raw_bidder_response": {
          "groups": [
            {
              "timeout": 1000,
              "hook_sequence": [
                {
                  "module_code": "prebid.ctv_vast_enrichment",
                  "hook_impl_code": "code123"
                }
              ]
            }
          ]
        }
      }
    }
  }
}
```

### Podsumowanie mechanizmu

| Element | Rola |
|---------|------|
| `hookexecution.NewHookExecutor(planBuilder, endpoint, metrics)` | Tworzy executor powiązany z endpointem |
| `planBuilder.PlanForXxxStage(endpoint)` | Szuka hooków w execution plan dla danego endpointu |
| `host_execution_plan.endpoints["/ctv/vast"]` | **Konfiguracja decyduje**, które moduły się odpalą |
| `exchange.HoldAuction(req{HookExecutor})` | Wewnętrznie odpala BidderRequest, RawBidderResponse itd. |

Moduł `ctv_vast_enrichment` na nowym endpoincie GET zadziała **automatycznie** — o ile:
- handler tworzy `hookExecutor` i przekazuje go do `exchange.HoldAuction`
- endpoint jest dodany do `host_execution_plan` w konfiguracji

**Nie trzeba żadnego nowego interfejsu do samego uruchomienia hooków** — cały mechanizm już istnieje. Jedyne co trzeba zrobić to podłączyć handler do routera i dać mu dostęp do `exchange` + `planBuilder`.

### Zaktualizowana ścieżka implementacji

```
1. hooks/hookexecution/executor.go  ← nowa stała EndpointCtvVast
2. endpoints/ctv_vast.go            ← nowy handler z hookExecutor
3. router/router.go                 ← r.GET("/ctv/vast", ...)
4. handler.go                       ← implementacja buildBidRequest()
5. pbs.json                         ← dodanie /ctv/vast do execution plan
6. handler_test.go                  ← testy
```
