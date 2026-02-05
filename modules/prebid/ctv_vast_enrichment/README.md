# Moduł CTV VAST Enrichment

Moduł CTV VAST Enrichment to moduł hook Prebid Server, który wzbogaca odpowiedzi VAST (Video Ad Serving Template) o dodatkowe metadane dla reklam Connected TV (CTV).

## Struktura Modułu

```
modules/prebid/ctv_vast_enrichment/
├── module.go         # Punkt wejścia modułu PBS (Builder + HandleRawBidderResponseHook)
├── module_test.go    # Testy modułu
├── pipeline.go       # Samodzielny pipeline przetwarzania VAST
├── pipeline_test.go  # Testy pipeline
├── handler.go        # Handler HTTP dla bezpośrednich żądań VAST
├── types.go          # Definicje typów, interfejsy i stałe
├── config.go         # Konfiguracja i mergowanie warstw (host/account/profile)
├── config_test.go    # Testy konfiguracji
├── model/            # Struktury danych VAST XML
│   ├── model.go      # Obiekty domenowe wysokiego poziomu
│   ├── vast_xml.go   # Struktury XML do marshal/unmarshal
│   ├── parser.go     # Parser VAST XML
│   └── *_test.go     # Testy
├── select/           # Logika selekcji bidów
│   ├── selector.go   # Implementacje BidSelector
│   └── *_test.go     # Testy
├── enrich/           # Wzbogacanie VAST
│   ├── enrich.go     # Implementacja Enricher (VAST_WINS)
│   └── *_test.go     # Testy
└── format/           # Formatowanie VAST XML
    ├── format.go     # Implementacja Formatter (GAM_SSU)
    └── *_test.go     # Testy
```

## Integracja z PBS

Moduł jest zgodny ze standardowym wzorcem modułów Prebid Server:

### `module.go` - Główny Punkt Wejścia

```go
// Builder tworzy nową instancję modułu CTV VAST enrichment.
func Builder(cfg json.RawMessage, deps moduledeps.ModuleDeps) (interface{}, error)

// Module implementuje funkcjonalność wzbogacania CTV VAST jako moduł hook PBS.
type Module struct {
    hostConfig CTVVastConfig
}

// HandleRawBidderResponseHook przetwarza odpowiedzi bidderów, wzbogacając VAST XML.
func (m Module) HandleRawBidderResponseHook(
    ctx context.Context,
    miCtx hookstage.ModuleInvocationContext,
    payload hookstage.RawBidderResponsePayload,
) (hookstage.HookResult[hookstage.RawBidderResponsePayload], error)
```

### Hook Stage

Moduł działa na etapie hooka **RawBidderResponse**, przetwarzając odpowiedź każdego biddera przed agregacją. Dla każdego bida zawierającego VAST XML:

1. Parsuje VAST XML z pola `AdM` bida
2. Wzbogaca VAST o pricing, advertiser i metadane kategorii
3. Aktualizuje pole `AdM` bida wzbogaconym VAST XML

### Konfiguracja

Moduł używa warstwowej konfiguracji w stylu PBS:

```json
{
  "modules": {
    "prebid": {
      "ctv_vast_enrichment": {
        "enabled": true,
        "receiver": "GAM_SSU",
        "default_currency": "USD",
        "vast_version_default": "3.0",
        "max_ads_in_pod": 10
      }
    }
  }
}
```

Konfiguracja na poziomie konta nadpisuje ustawienia na poziomie hosta.

## Komponenty

### `module.go` - Moduł PBS

Główny punkt wejścia zgodny z konwencjami modułów PBS:

- **`Builder()`** - Tworzy instancję modułu z konfiguracji JSON
- **`Module`** - Struktura przechowująca konfigurację na poziomie hosta
- **`HandleRawBidderResponseHook()`** - Implementacja hooka:
  - Parsuje konfigurację na poziomie konta
  - Merguje konfiguracje hosta i konta
  - Wzbogaca VAST w każdym bidzie video

### `pipeline.go` - Samodzielny Pipeline

Alternatywny punkt wejścia do bezpośredniego wywołania (używany przez handler.go):

- **`BuildVastFromBidResponse()`** - Orkiestruje pełny pipeline:
  1. Selekcja bidów z odpowiedzi aukcji
  2. Parsowanie VAST z AdM każdego bida
  3. Wzbogacanie metadanymi
  4. Formatowanie do końcowego XML

- **`Processor`** - Wrapper z wstrzykniętymi zależnościami
- **`DefaultConfig()`** - Domyślna konfiguracja dla GAM SSU

### `handler.go` - Handler HTTP

Obsługa żądań HTTP dla reklam CTV VAST (opcjonalny endpoint):

- **`Handler`** - Handler HTTP z konfiguracją i zależnościami
- **`ServeHTTP()`** - Obsługuje żądania GET, zwraca VAST XML
- Metody buildera: `WithConfig()`, `WithSelector()`, itp.

### `types.go` - Typy i Interfejsy

| Typ | Opis |
|-----|------|
| `ReceiverType` | Typ odbiorcy (GAM_SSU, GENERIC) |
| `SelectionStrategy` | Strategia selekcji bidów (SINGLE, TOP_N, MAX_REVENUE) |
| `CollisionPolicy` | Polityka kolizji (reject, warn, ignore) |

**Interfejsy:**

```go
type BidSelector interface {
    Select(req, resp, cfg) ([]SelectedBid, []string, error)
}

type Enricher interface {
    Enrich(ad *model.Ad, meta CanonicalMeta, cfg ReceiverConfig) ([]string, error)
}

type Formatter interface {
    Format(ads []EnrichedAd, cfg ReceiverConfig) ([]byte, []string, error)
}
```

**Struktury Danych:**

- `CanonicalMeta` - Znormalizowane metadane bida (BidID, Price, Currency, Adomain, itp.)
- `SelectedBid` - Wybrany bid z metadanymi i numerem sekwencji
- `EnrichedAd` - Wzbogacona reklama gotowa do formatowania
- `VastResult` - Wynik przetwarzania (XML, ostrzeżenia, błędy)
- `ReceiverConfig` - Konfiguracja odbiorcy VAST
- `PlacementRules` - Reguły walidacji (pricing, advertiser, categories)

### `config.go` - Konfiguracja

Warstwowy system konfiguracji w stylu PBS:

- **`CTVVastConfig`** - Struktura konfiguracji z polami nullable
- **`MergeCTVVastConfig()`** - Mergowanie warstw: Host → Account → Profile

Priorytet warstw (od najniższego do najwyższego):
1. Host (domyślne)
2. Account (nadpisuje host)
3. Profile (nadpisuje wszystko)

### `model/` - Struktury VAST XML

#### `vast_xml.go`

Struktury Go mapujące elementy VAST XML:

- `Vast` - Element główny `<VAST>`
- `Ad` - Element `<Ad>` z atrybutami id, sequence
- `InLine` - Reklama inline z pełnymi danymi
- `Wrapper` - Reklama wrapper (przekierowanie)
- `Creative`, `Linear`, `MediaFile` - Elementy kreacji
- `Pricing`, `Impression`, `Extensions` - Metadane i tracking

Funkcje pomocnicze:
- `BuildNoAdVast()` - Tworzy pusty VAST (brak reklam)
- `BuildSkeletonInlineVast()` - Tworzy minimalny szkielet VAST
- `Marshal()` / `MarshalCompact()` - Serializacja do XML

#### `parser.go`

Parser VAST XML:

- **`ParseVastAdm()`** - Parsuje string AdM do struktury Vast
- **`ParseVastOrSkeleton()`** - Parsuje lub tworzy szkielet jeśli dozwolone
- **`ExtractFirstAd()`** - Wyciąga pierwszą reklamę z VAST

### `select/` - Selekcja Bidów

Logika wyboru bidów z odpowiedzi aukcji:

- **`PriceSelector`** - Implementacja oparta na cenie:
  - Filtruje bidy z ceną ≤ 0 lub pustym AdM
  - Sortuje: deal > non-deal, potem po cenie malejąco
  - Respektuje `MaxAdsInPod` dla strategii TOP_N
  - Przypisuje numery sekwencji (1-indexed)

- **`NewSelector(strategy)`** - Fabryka tworząca selektor dla strategii

### `enrich/` - Wzbogacanie VAST

Dodawanie metadanych do reklam VAST:

- **`VastEnricher`** - Implementacja z polityką VAST_WINS:
  - Istniejące wartości w VAST nie są nadpisywane
  - Dodaje brakujące: Pricing, Advertiser, Duration, Categories

Wzbogacane elementy:
| Element | Źródło | Lokalizacja |
|---------|--------|-------------|
| Pricing | meta.Price | `<Pricing>` lub Extension |
| Advertiser | meta.Adomain | `<Advertiser>` lub Extension |
| Duration | meta.DurSec | `<Duration>` w Linear |
| Categories | meta.Cats | Extension (zawsze) |

### `format/` - Formatowanie VAST

Budowanie końcowego VAST XML:

- **`VastFormatter`** - Implementacja GAM SSU:
  - Buduje dokument VAST z listą elementów `<Ad>`
  - Ustawia `id` z BidID
  - Ustawia `sequence` dla podów (wiele reklam)

## Przepływ Przetwarzania

```
┌─────────────────────────────────────────────────────┐
│              PBS Auction Pipeline                    │
└─────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────┐
│          RawBidderResponse Hook Stage               │
│  ┌───────────────────────────────────────────────┐  │
│  │   HandleRawBidderResponseHook()               │  │
│  │   Dla każdego bida z VAST w AdM:              │  │
│  │   1. Parsuje VAST XML                         │  │
│  │   2. Wzbogaca o pricing/advertiser            │  │
│  │   3. Aktualizuje bid.AdM                      │  │
│  └───────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────┐
│              Wzbogacona BidderResponse               │
│              (VAST z <Pricing>, itp.)               │
└─────────────────────────────────────────────────────┘
```

## Użycie

### Jako Moduł PBS (Rekomendowane)

Moduł jest automatycznie wywoływany podczas pipeline aukcji gdy włączony w konfiguracji:

```yaml
# Konfiguracja PBS
hooks:
  enabled_modules:
    - prebid.ctv_vast_enrichment

modules:
  prebid:
    ctv_vast_enrichment:
      enabled: true
      default_currency: "USD"
      receiver: "GAM_SSU"
```

Nadpisanie na poziomie konta:
```json
{
  "hooks": {
    "modules": {
      "prebid.ctv_vast_enrichment": {
        "enabled": true,
        "default_currency": "EUR"
      }
    }
  }
}
```

### Samodzielny Pipeline (dla handlera HTTP)

```go
import (
    vast "github.com/prebid/prebid-server/v3/modules/prebid/ctv_vast_enrichment"
    "github.com/prebid/prebid-server/v3/modules/prebid/ctv_vast_enrichment/enrich"
    "github.com/prebid/prebid-server/v3/modules/prebid/ctv_vast_enrichment/format"
    bidselect "github.com/prebid/prebid-server/v3/modules/prebid/ctv_vast_enrichment/select"
)

// Konfiguracja
cfg := vast.DefaultConfig()
cfg.MaxAdsInPod = 3

// Tworzenie komponentów
selector := bidselect.NewSelector(cfg.SelectionStrategy)
enricher := enrich.NewEnricher()
formatter := format.NewFormatter()

// Bezpośrednie wywołanie
result, err := vast.BuildVastFromBidResponse(
    ctx,
    bidRequest,
    bidResponse,
    cfg,
    selector,
    enricher,
    formatter,
)
```

### Handler HTTP

```go
handler := vast.NewHandler().
    WithConfig(cfg).
    WithSelector(selector).
    WithEnricher(enricher).
    WithFormatter(formatter).
    WithAuctionFunc(myAuctionFunc)

http.Handle("/vast", handler)
```

## Konfiguracja Warstwowa

```go
// Konfiguracja hosta (domyślne)
hostCfg := &vast.CTVVastConfig{
    Receiver:           "GAM_SSU",
    DefaultCurrency:    "USD",
    VastVersionDefault: "4.0",
}

// Konfiguracja konta (nadpisuje host)
accountCfg := &vast.CTVVastConfig{
    MaxAdsInPod: 5,
}

// Merge warstw
merged := vast.MergeCTVVastConfig(hostCfg, accountCfg, nil)
```

## Testowanie

Uruchom wszystkie testy modułu:

```bash
go test ./modules/prebid/ctv_vast_enrichment/... -v
```

Testy z pokryciem:

```bash
go test ./modules/prebid/ctv_vast_enrichment/... -cover
```

Uruchom tylko testy module.go:

```bash
go test ./modules/prebid/ctv_vast_enrichment -run TestBuilder -v
go test ./modules/prebid/ctv_vast_enrichment -run TestHandleRawBidderResponseHook -v
```

## Rozszerzenia

### Dodawanie Nowego Odbiorcy

1. Dodaj stałą w `types.go`:
   ```go
   ReceiverMyReceiver ReceiverType = "MY_RECEIVER"
   ```

2. Zaimplementuj `Formatter` dla nowego formatu w `format/`

3. Zaktualizuj `configToReceiverConfig()` w `module.go`

### Dodawanie Nowej Strategii Selekcji

1. Dodaj stałą w `types.go`:
   ```go
   SelectionMyStrategy SelectionStrategy = "MY_STRATEGY"
   ```

2. Zaimplementuj `BidSelector` w `select/`

3. Zaktualizuj fabrykę `NewSelector()`

## Zależności

- `github.com/prebid/prebid-server/v3/hooks/hookstage` - Interfejsy hooków PBS
- `github.com/prebid/prebid-server/v3/modules/moduledeps` - Zależności modułów
- `github.com/prebid/openrtb/v20/openrtb2` - Typy OpenRTB
- `encoding/xml` - Parsowanie/serializacja XML
