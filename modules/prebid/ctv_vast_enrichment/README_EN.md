# CTV VAST Enrichment Module

The CTV VAST Enrichment module is a Prebid Server hook module that enriches VAST (Video Ad Serving Template) XML responses with additional metadata for Connected TV (CTV) ads.

## Module Structure

```
modules/prebid/ctv_vast_enrichment/
├── module.go         # PBS module entry point (Builder + HandleRawBidderResponseHook)
├── module_test.go    # Module tests
├── pipeline.go       # Standalone VAST processing pipeline
├── pipeline_test.go  # Pipeline tests
├── handler.go        # HTTP handler for direct VAST requests
├── types.go          # Type definitions, interfaces and constants
├── config.go         # Configuration and layer merging (host/account/profile)
├── config_test.go    # Configuration tests
├── model/            # VAST XML data structures
│   ├── model.go      # High-level domain objects
│   ├── vast_xml.go   # XML structures for marshal/unmarshal
│   ├── parser.go     # VAST XML parser
│   └── *_test.go     # Tests
├── select/           # Bid selection logic
│   ├── selector.go   # BidSelector implementations
│   └── *_test.go     # Tests
├── enrich/           # VAST enrichment
│   ├── enrich.go     # Enricher implementation (VAST_WINS)
│   └── *_test.go     # Tests
└── format/           # VAST XML formatting
    ├── format.go     # Formatter implementation (GAM_SSU)
    └── *_test.go     # Tests
```

## PBS Module Integration

This module follows the standard Prebid Server module pattern.

### Registration in `modules/builder.go`

The module must be registered in `modules/builder.go`, which is the central registry of all PBS modules:

```go
import (
    prebidCtvVastEnrichment "github.com/prebid/prebid-server/v3/modules/prebid/ctv_vast_enrichment"
)

var newModuleBuilders = map[string]map[string]interface{}{
    "prebid": {
        "ctv_vast_enrichment": prebidCtvVastEnrichment.Builder,
    },
}
```

> **Note:** The Go package name is `ctv_vast_enrichment`, but subpackages (enrich, select, format) use the `vast` alias when importing the parent package:
> ```go
> import vast "github.com/prebid/prebid-server/v3/modules/prebid/ctv_vast_enrichment"
> ```

### \`module.go\` - Main Entry Point

\`\`\`go
// Builder creates a new CTV VAST enrichment module instance.
func Builder(cfg json.RawMessage, deps moduledeps.ModuleDeps) (interface{}, error)

// Module implements the CTV VAST enrichment functionality as a PBS hook module.
type Module struct {
    hostConfig CTVVastConfig
}

// HandleRawBidderResponseHook processes bidder responses to enrich VAST XML.
func (m Module) HandleRawBidderResponseHook(
    ctx context.Context,
    miCtx hookstage.ModuleInvocationContext,
    payload hookstage.RawBidderResponsePayload,
) (hookstage.HookResult[hookstage.RawBidderResponsePayload], error)
\`\`\`

### Hook Stage

The module runs at the **RawBidderResponse** hook stage, processing each bidder's response before aggregation. For each bid containing VAST XML:

1. Parses the VAST XML from the bid's \`AdM\` field
2. Enriches the VAST with pricing, advertiser, and category metadata
3. Creates a new `*adapters.TypedBid` with a new `*openrtb2.Bid` containing the enriched AdM
4. Returns the mutation via `changeSet.RawBidderResponse().Bids().UpdateBids(modifiedBids)`

> **ChangeSet Pattern:** The hook does not modify the payload directly. Instead, it builds a new `[]adapters.TypedBid` slice and registers the mutation via `UpdateBids()`. PBS applies the mutation after the hook returns — following the pattern from the `ortb2blocking` module.

### Configuration

The module uses PBS-style layered configuration:

\`\`\`json
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
\`\`\`

Account-level configuration overrides host-level settings.

## Components

### \`module.go\` - PBS Module

Main entry point following PBS module conventions:

- **\`Builder()\`** - Creates module instance from JSON config
- **\`Module\`** - Struct holding host-level configuration
- **\`HandleRawBidderResponseHook()\`** - Hook implementation that:
  - Parses account-level config
  - Merges host and account configs
  - Enriches VAST in each video bid

### \`pipeline.go\` - Standalone Pipeline

Alternative entry point for direct invocation (used by handler.go):

- **\`BuildVastFromBidResponse()\`** - Orchestrates the full pipeline:
  1. Bid selection from auction response
  2. VAST parsing from each bid's AdM
  3. Enrichment with metadata
  4. Formatting to final XML

- **\`Processor\`** - Wrapper with injected dependencies
- **\`DefaultConfig()\`** - Default configuration for GAM SSU

### \`handler.go\` - HTTP Handler

HTTP request handling for CTV VAST ads (optional endpoint):

- **\`Handler\`** - HTTP handler with configuration and dependencies
- **\`ServeHTTP()\`** - Handles GET requests, returns VAST XML
- Builder methods: \`WithConfig()\`, \`WithSelector()\`, etc.

### \`types.go\` - Types and Interfaces

| Type | Description |
|------|-------------|
| \`ReceiverType\` | Receiver type (GAM_SSU, GENERIC) |
| \`SelectionStrategy\` | Bid selection strategy (SINGLE, TOP_N, MAX_REVENUE) |
| \`CollisionPolicy\` | Collision policy (reject, warn, ignore) |

**Interfaces:**

\`\`\`go
type BidSelector interface {
    Select(req, resp, cfg) ([]SelectedBid, []string, error)
}

type Enricher interface {
    Enrich(ad *model.Ad, meta CanonicalMeta, cfg ReceiverConfig) ([]string, error)
}

type Formatter interface {
    Format(ads []EnrichedAd, cfg ReceiverConfig) ([]byte, []string, error)
}
\`\`\`

**Data Structures:**

- \`CanonicalMeta\` - Normalized bid metadata (BidID, Price, Currency, Adomain, etc.)
- \`SelectedBid\` - Selected bid with metadata and sequence number
- \`EnrichedAd\` - Enriched ad ready for formatting
- \`VastResult\` - Processing result (XML, warnings, errors)
- \`ReceiverConfig\` - VAST receiver configuration
- \`PlacementRules\` - Validation rules (pricing, advertiser, categories)

### \`config.go\` - Configuration

PBS-style layered configuration system:

- **\`CTVVastConfig\`** - Configuration structure with nullable fields
- **\`MergeCTVVastConfig()\`** - Layer merging: Host → Account → Profile

Layer priority (from lowest to highest):
1. Host (defaults)
2. Account (overrides host)
3. Profile (overrides everything)

### \`model/\` - VAST XML Structures

#### \`vast_xml.go\`

Go structures mapping VAST XML elements:

- \`Vast\` - Root element \`<VAST>\`
- \`Ad\` - Element \`<Ad>\` with id, sequence attributes
- \`InLine\` - Inline ad with full data
- \`Wrapper\` - Wrapper ad (redirect)
- \`Creative\`, \`Linear\`, \`MediaFile\` - Creative elements
- \`Pricing\`, \`Impression\`, \`Extensions\` - Metadata and tracking

Helper functions:
- `BuildNoAdVast()` - Creates empty VAST (no ads)
- `BuildSkeletonInlineVast()` - Creates minimal VAST skeleton
- `Marshal()` / `MarshalCompact()` - Serialize to XML
- `clearInnerXML()` - Clears `InnerXML` fields before serialization (prevents element duplication)

> **XML Fix:** VAST structures use the `,innerxml` tag to preserve raw XML during parsing. Before `Marshal()`, `clearInnerXML()` is called to zero out `InnerXML` fields on `Ad`, `InLine`, `Wrapper`, `Creative`, and `Linear` structs, preventing duplicate elements in the output XML.

#### `parser.go`

VAST XML parser:

- **\`ParseVastAdm()\`** - Parses AdM string to Vast structure
- **\`ParseVastOrSkeleton()\`** - Parses or creates skeleton if allowed
- **\`ExtractFirstAd()\`** - Extracts first ad from VAST

### \`select/\` - Bid Selection

Logic for selecting bids from auction response:

- **\`PriceSelector\`** - Price-based implementation:
  - Filters bids with price ≤ 0 or empty AdM
  - Sorts: deal > non-deal, then by price descending
  - Respects \`MaxAdsInPod\` for TOP_N strategy
  - Assigns sequence numbers (1-indexed)

- **\`NewSelector(strategy)\`** - Factory creating selector for strategy

### \`enrich/\` - VAST Enrichment

Adding metadata to VAST ads:

- **\`VastEnricher\`** - Implementation with VAST_WINS policy:
  - Existing values in VAST are not overwritten
  - Adds missing: Pricing, Advertiser, Duration, Categories

Enriched elements:
| Element | Source | Location |
|---------|--------|----------|
| Pricing | meta.Price | \`<Pricing>\` or Extension |
| Advertiser | meta.Adomain | \`<Advertiser>\` or Extension |
| Duration | meta.DurSec | \`<Duration>\` in Linear |
| Categories | meta.Cats | Extension (always) |

### \`format/\` - VAST Formatting

Building final VAST XML:

- **\`VastFormatter\`** - GAM SSU implementation:
  - Builds VAST document with list of \`<Ad>\` elements
  - Sets \`id\` from BidID
  - Sets \`sequence\` for pods (multiple ads)

## Processing Flow

\`\`\`
┌─────────────────────────────────────────────────────┐
│              PBS Auction Pipeline                    │
└─────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────┐
│          RawBidderResponse Hook Stage               │
│  ┌───────────────────────────────────────────────┐  │
│  │   HandleRawBidderResponseHook()               │  │
│  │   For each bid with VAST in AdM:              │  │
│  │   1. Parse VAST XML                           │  │
│  │   2. Enrich with pricing/advertiser           │  │
│  │   3. Update bid.AdM                           │  │
│  └───────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────┐
│              Enriched BidderResponse                 │
│              (VAST with <Pricing>, etc.)            │
└─────────────────────────────────────────────────────┘
\`\`\`

## Usage

### As PBS Module (Recommended)

The module is automatically invoked during the auction pipeline when enabled in configuration.

**1. Ensure the module is registered in `modules/builder.go`** (see "Registration" section above).

**2. Add hooks configuration to `pbs.json`:**

```json
{
  "hooks": {
    "enabled": true,
    "modules": {
      "prebid": {
        "ctv_vast_enrichment": {
          "enabled": true,
          "default_currency": "USD",
          "receiver": "GAM_SSU"
        }
      }
    },
    "host_execution_plan": {
      "endpoints": {
        "/openrtb2/auction": {
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
  }
}
```

**3. Account-level override** (optional):
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

### Standalone Pipeline (for HTTP handler)

\`\`\`go
import (
    vast "github.com/prebid/prebid-server/v3/modules/prebid/ctv_vast_enrichment"
    "github.com/prebid/prebid-server/v3/modules/prebid/ctv_vast_enrichment/enrich"
    "github.com/prebid/prebid-server/v3/modules/prebid/ctv_vast_enrichment/format"
    bidselect "github.com/prebid/prebid-server/v3/modules/prebid/ctv_vast_enrichment/select"
)

// Configuration
cfg := vast.DefaultConfig()
cfg.MaxAdsInPod = 3

// Create components
selector := bidselect.NewSelector(cfg.SelectionStrategy)
enricher := enrich.NewEnricher()
formatter := format.NewFormatter()

// Direct invocation
result, err := vast.BuildVastFromBidResponse(
    ctx,
    bidRequest,
    bidResponse,
    cfg,
    selector,
    enricher,
    formatter,
)
\`\`\`

### HTTP Handler

\`\`\`go
handler := vast.NewHandler().
    WithConfig(cfg).
    WithSelector(selector).
    WithEnricher(enricher).
    WithFormatter(formatter).
    WithAuctionFunc(myAuctionFunc)

http.Handle("/vast", handler)
\`\`\`

## Layer Configuration

\`\`\`go
// Host configuration (defaults)
hostCfg := &vast.CTVVastConfig{
    Receiver:           "GAM_SSU",
    DefaultCurrency:    "USD",
    VastVersionDefault: "4.0",
}

// Account configuration (overrides host)
accountCfg := &vast.CTVVastConfig{
    MaxAdsInPod: 5,
}

// Merge layers
merged := vast.MergeCTVVastConfig(hostCfg, accountCfg, nil)
\`\`\`

## Testing

Run all module tests:

\`\`\`bash
go test ./modules/prebid/ctv_vast_enrichment/... -v
\`\`\`

Tests with coverage:

\`\`\`bash
go test ./modules/prebid/ctv_vast_enrichment/... -cover
\`\`\`

Run only module.go tests:

\`\`\`bash
go test ./modules/prebid/ctv_vast_enrichment -run TestBuilder -v
go test ./modules/prebid/ctv_vast_enrichment -run TestHandleRawBidderResponseHook -v
\`\`\`

## Extensions

### Adding a New Receiver

1. Add constant in \`types.go\`:
   \`\`\`go
   ReceiverMyReceiver ReceiverType = "MY_RECEIVER"
   \`\`\`

2. Implement \`Formatter\` for the new format in \`format/\`

3. Update \`configToReceiverConfig()\` in \`module.go\`

### Adding a New Selection Strategy

1. Add constant in \`types.go\`:
   \`\`\`go
   SelectionMyStrategy SelectionStrategy = "MY_STRATEGY"
   \`\`\`

2. Implement \`BidSelector\` in \`select/\`

3. Update \`NewSelector()\` factory

## Dependencies

- \`github.com/prebid/prebid-server/v3/hooks/hookstage\` - PBS hook interfaces
- \`github.com/prebid/prebid-server/v3/modules/moduledeps\` - Module dependencies
- \`github.com/prebid/openrtb/v20/openrtb2\` - OpenRTB types
- \`encoding/xml\` - XML parsing/serialization
