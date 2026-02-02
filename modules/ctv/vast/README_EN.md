# CTV VAST Module

The CTV VAST module provides comprehensive VAST (Video Ad Serving Template) processing for Connected TV (CTV) ads in Prebid Server.

## Module Structure

```
modules/ctv/vast/
├── vast.go           # Main entry point and orchestration
├── handler.go        # HTTP handler for VAST requests
├── types.go          # Type definitions, interfaces and constants
├── config.go         # Configuration and layer merging (host/account/profile)
├── model/            # VAST XML data structures
│   ├── model.go      # High-level domain objects
│   ├── vast_xml.go   # XML structures for marshal/unmarshal
│   └── parser.go     # VAST XML parser
├── select/           # Bid selection logic
│   └── selector.go   # BidSelector implementations
├── enrich/           # VAST enrichment
│   └── enrich.go     # Enricher implementation (VAST_WINS)
└── format/           # VAST XML formatting
    └── format.go     # Formatter implementation (GAM_SSU)
```

## Components

### `vast.go` - Orchestration

Main entry point of the module. Contains:

- **`BuildVastFromBidResponse()`** - Main function orchestrating the entire pipeline:
  1. Bid selection from auction response
  2. VAST parsing from each bid's AdM (or skeleton creation)
  3. Enrichment of each ad with metadata
  4. Formatting to final XML

- **`Processor`** - Wrapper structure for the pipeline with injected dependencies
- **`DefaultConfig()`** - Default configuration for GAM SSU

### `handler.go` - HTTP Handler

HTTP request handling for CTV VAST ads:

- **`Handler`** - HTTP handler structure with configuration and dependencies
- **`ServeHTTP()`** - Handles GET requests, returns VAST XML
- **`buildBidRequest()`** - Builds OpenRTB BidRequest from HTTP parameters
- Builder methods: `WithConfig()`, `WithSelector()`, `WithEnricher()`, `WithFormatter()`, `WithAuctionFunc()`

### `types.go` - Types and Interfaces

Basic type definitions:

| Type | Description |
|------|-------------|
| `ReceiverType` | Receiver type (GAM_SSU, SPRINGSERVE, etc.) |
| `SelectionStrategy` | Bid selection strategy (SINGLE, TOP_N, MAX_REVENUE) |
| `CollisionPolicy` | Collision policy (VAST_WINS, BID_WINS, REJECT) |
| `PlacementLocation` | Element placement (VAST_PRICING, EXTENSION, etc.) |

**Interfaces:**

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

**Data Structures:**

- `CanonicalMeta` - Normalized bid metadata (BidID, Price, Currency, Adomain, etc.)
- `SelectedBid` - Selected bid with metadata and sequence number
- `EnrichedAd` - Enriched ad ready for formatting
- `VastResult` - Processing result (XML, warnings, errors)
- `ReceiverConfig` - VAST receiver configuration
- `PlacementRules` - Validation rules (pricing, advertiser, categories)

### `config.go` - Configuration

PBS-style layered configuration system:

- **`CTVVastConfig`** - Configuration structure with nullable fields
- **`MergeCTVVastConfig()`** - Layer merging: Host → Account → Profile
- **`ToReceiverConfig()`** - Conversion to ReceiverConfig

Layer priority (from lowest to highest):
1. Host (defaults)
2. Account (overrides host)
3. Profile (overrides everything)

### `model/` - VAST XML Structures

#### `vast_xml.go`

Go structures mapping VAST XML elements:

- `Vast` - Root element `<VAST>`
- `Ad` - Element `<Ad>` with id, sequence attributes
- `InLine` - Inline ad with full data
- `Wrapper` - Wrapper ad (redirect)
- `Creative`, `Linear`, `MediaFile` - Creative elements
- `Pricing`, `Impression`, `Extensions` - Metadata and tracking

Helper functions:
- `BuildNoAdVast()` - Creates empty VAST (no ads)
- `BuildSkeletonInlineVast()` - Creates minimal VAST skeleton
- `SecToHHMMSS()` - Converts seconds to HH:MM:SS format

#### `parser.go`

VAST XML parser:

- **`ParseVastAdm()`** - Parses AdM string to Vast structure
- **`ParseVastOrSkeleton()`** - Parses or creates skeleton if allowed
- **`ExtractFirstAd()`** - Extracts first ad from VAST
- **`ParseDurationToSeconds()`** - Parses duration "HH:MM:SS" to seconds

### `select/` - Bid Selection

Logic for selecting bids from auction response:

- **`PriceSelector`** - Price-based implementation:
  - Filters bids with price ≤ 0 or empty AdM
  - Sorts: deal > non-deal, then by price descending
  - Respects `MaxAdsInPod` for TOP_N strategy
  - Assigns sequence numbers (1-indexed)

- **`NewSelector(strategy)`** - Factory creating selector for strategy
- **`NewSingleSelector()`** - Returns only the best bid
- **`NewTopNSelector()`** - Returns top N bids

### `enrich/` - VAST Enrichment

Adding metadata to VAST ads:

- **`VastEnricher`** - Implementation with VAST_WINS policy:
  - Existing values in VAST are not overwritten
  - Adds missing: Pricing, Advertiser, Duration, Categories
  - Optional debug extensions with OpenRTB data

Enriched elements:
| Element | Source | Location |
|---------|--------|----------|
| Pricing | meta.Price | `<Pricing>` or Extension |
| Advertiser | meta.Adomain | `<Advertiser>` or Extension |
| Duration | meta.DurSec | `<Duration>` in Linear |
| Categories | meta.Cats | Extension (always) |
| Debug | all fields | Extension (when cfg.Debug=true) |

### `format/` - VAST Formatting

Building final VAST XML:

- **`VastFormatter`** - GAM SSU implementation:
  - Builds VAST document with list of `<Ad>` elements
  - Sets `id` from BidID
  - Sets `sequence` for pods (multiple ads)
  - Adds XML declaration and formatting

## Processing Flow

```
┌─────────────────┐
│  BidRequest     │
│  BidResponse    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   BidSelector   │  ← Filters and sorts bids
│   (select/)     │  ← Selects top N by strategy
└────────┬────────┘
         │ []SelectedBid
         ▼
┌─────────────────┐
│  ParseVast      │  ← Parses AdM to structure
│  (model/)       │  ← Or creates skeleton
└────────┬────────┘
         │ *model.Ad
         ▼
┌─────────────────┐
│   Enricher      │  ← Adds Pricing, Advertiser
│   (enrich/)     │  ← VAST_WINS policy
└────────┬────────┘
         │ EnrichedAd
         ▼
┌─────────────────┐
│   Formatter     │  ← Builds final XML
│   (format/)     │  ← Sets sequence, id
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   VastResult    │
│   (XML bytes)   │
└─────────────────┘
```

## Usage

### Basic Usage with Processor

```go
import (
    "github.com/prebid/prebid-server/v3/modules/ctv/vast"
    "github.com/prebid/prebid-server/v3/modules/ctv/vast/enrich"
    "github.com/prebid/prebid-server/v3/modules/ctv/vast/format"
    bidselect "github.com/prebid/prebid-server/v3/modules/ctv/vast/select"
)

// Configuration
cfg := vast.DefaultConfig()
cfg.MaxAdsInPod = 3
cfg.SelectionStrategy = vast.SelectionTopN

// Create components
selector := bidselect.NewSelector(cfg.SelectionStrategy)
enricher := enrich.NewEnricher()
formatter := format.NewFormatter()

// Create processor
processor := vast.NewProcessor(cfg, selector, enricher, formatter)

// Process
result := processor.Process(ctx, bidRequest, bidResponse)

if result.NoAd {
    // No ads available
}

// result.VastXML contains the ready XML
```

### HTTP Handler Usage

```go
handler := vast.NewHandler().
    WithConfig(cfg).
    WithSelector(selector).
    WithEnricher(enricher).
    WithFormatter(formatter).
    WithAuctionFunc(myAuctionFunc)

http.Handle("/vast", handler)
```

### Direct Invocation

```go
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

## Layer Configuration

```go
// Host configuration (defaults)
hostCfg := &vast.CTVVastConfig{
    Receiver:           vast.ReceiverGAMSSU,
    DefaultCurrency:    "USD",
    VastVersionDefault: "4.0",
}

// Account configuration (overrides host)
accountCfg := &vast.CTVVastConfig{
    MaxAdsInPod:       vast.IntPtr(5),
    SelectionStrategy: vast.SelectionTopN,
}

// Profile configuration (overrides everything)
profileCfg := &vast.CTVVastConfig{
    Debug: vast.BoolPtr(true),
}

// Merge layers
merged := vast.MergeCTVVastConfig(hostCfg, accountCfg, profileCfg)
receiverCfg := merged.ToReceiverConfig()
```

## Testing

Run all module tests:

```bash
go test ./modules/ctv/vast/... -v
```

Tests with coverage:

```bash
go test ./modules/ctv/vast/... -cover
```

## Extensions

### Adding a New Receiver

1. Add constant in `types.go`:
   ```go
   ReceiverMyReceiver ReceiverType = "MY_RECEIVER"
   ```

2. Implement `Formatter` for the new format in `format/`

3. Optionally: adjust `Enricher` if different enrichment is needed

### Adding a New Selection Strategy

1. Add constant in `types.go`:
   ```go
   SelectionMyStrategy SelectionStrategy = "MY_STRATEGY"
   ```

2. Implement `BidSelector` in `select/`

3. Update `NewSelector()` factory

## Dependencies

- `github.com/prebid/openrtb/v20/openrtb2` - OpenRTB types
- `encoding/xml` - XML parsing/serialization
- `net/http` - HTTP handler
