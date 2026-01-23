# CTV VAST Endpoint for Prebid Server

## Overview

This implementation adds a new MVP CTV (Connected TV) endpoint to Prebid Server that returns VAST XML for GAM Server-Side Unified (SSU) ad insertion and other CTV receivers. The endpoint accepts GET requests with query parameters, executes a standard Prebid auction, and returns VAST XML with enriched metadata.

## Architecture

### Package Structure

```
endpoints/ctv/
├── ctv.go                    # Main endpoint handler
├── ctv_test.go              # Endpoint tests
└── vast/
    ├── model/               # VAST XML data structures
    │   ├── vast.go          # VAST structs with XML marshal/unmarshal
    │   └── vast_test.go
    ├── selector/            # Bid selection logic (single/pod)
    │   ├── selector.go
    │   └── selector_test.go
    ├── enricher/            # OpenRTB → VAST field mapping
    │   ├── enricher.go
    │   └── enricher_test.go
    ├── formatter/           # Receiver-specific formatters
    │   ├── formatter.go
    │   ├── formatter_test.go
    │   └── golden_test.go
    └── testdata/            # Golden XML test files
        ├── single_bid_enriched.xml
        ├── pod_three_ads.xml
        └── empty_vast.xml

config/
├── ctv_vast.go              # CTV VAST configuration
└── ctv_vast_test.go

metrics/
└── metrics.go               # Added CTV metrics types
```

## Features

### Core Functionality

1. **GET Endpoint**: `/ctv/vast`
   - Query parameter parsing (publisher_id, dimensions, duration, macros)
   - OpenRTB BidRequest construction from query params
   - Auction execution via existing PBS pipeline
   - VAST XML response (always 200 status, even for no-bid)

2. **Bid Selection**
   - **Single Strategy**: Selects highest-priced bid
   - **Top-N Strategy**: Selects top N bids for ad pods
   - Price-based sorting with stable secondary sorting by bid ID

3. **VAST Enrichment**
   - Maps OpenRTB fields to VAST elements:
     - Price → `<Pricing model="CPM" currency="...">`
     - Advertiser → `<Advertiser>`
     - Categories → `<Category>` or Extensions
     - Duration → `<Duration>HH:MM:SS</Duration>`
     - IDs/Debug → Extensions (configurable)
   - Collision policies:
     - `VAST_WINS`: Don't overwrite existing non-empty VAST fields
     - `OPENRTB_WINS`: Always overwrite with OpenRTB data
   - Configurable placement rules (INLINE, EXTENSIONS, SKIP)

4. **Receiver Formatting**
   - **GAM SSU**: VAST 3.0 with GAM-specific requirements
     - Auto-generates missing IDs
     - Ensures required fields (AdSystem, AdTitle, Impression)
     - Proper sequence numbering for pods
   - **Generic**: Standard VAST 4.0 output

5. **Configuration Layering**
   - Host defaults
   - Account-level overrides
   - Profile-level overrides (highest priority)
   - Merge function: `MergeCTVVastConfig(host, account, profile)`

## Configuration

### Example Configuration (YAML)

```yaml
ctv_vast:
  enabled: true
  receiver: "GAM_SSU"  # or "GENERIC"
  vast_version_default: "3.0"
  default_currency: "USD"
  max_ads_in_pod: 3
  selection_strategy: "TOP_N"  # or "SINGLE"
  collision_policy: "VAST_WINS"  # or "OPENRTB_WINS"
  placement_rules:
    price: "INLINE"
    currency: "INLINE"
    advertiser: "INLINE"
    categories: "EXTENSIONS"
    duration: "INLINE"
    ids: "EXTENSIONS"
    deal_id: "EXTENSIONS"
  macro_config:
    enabled: true
    unknown_macro_policy: "KEEP"  # or "REMOVE", "ERROR"
    mappings:
      CORRELATOR:
        source: "query"
        key: "correlator"
      DEVICE_IP:
        source: "header"
        key: "X-Forwarded-For"
  include_debug_ids: false
  stored_requests_enabled: true
```

### Configuration Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | false | Enable/disable CTV endpoint |
| `receiver` | string | "GAM_SSU" | Target receiver profile |
| `vast_version_default` | string | "3.0" | Default VAST version |
| `default_currency` | string | "USD" | Currency when not in bid response |
| `max_ads_in_pod` | int | 1 | Maximum ads in pod (TOP_N strategy) |
| `selection_strategy` | string | "SINGLE" | Bid selection strategy |
| `collision_policy` | string | "VAST_WINS" | How to handle existing VAST fields |
| `include_debug_ids` | bool | false | Include bid/imp IDs in extensions |
| `stored_requests_enabled` | bool | true | Enable stored request support |

## API Usage

### Request Examples

#### Single Ad Request
```
GET /ctv/vast?publisher_id=pub123&width=1920&height=1080&min_duration=15&max_duration=30
```

#### Ad Pod Request
```
GET /ctv/vast?publisher_id=pub123&width=1920&height=1080&stored_request_id=pod-profile
```

#### With Debug
```
GET /ctv/vast?publisher_id=pub123&debug=1
```

### Query Parameters

| Parameter | Required | Type | Description |
|-----------|----------|------|-------------|
| `publisher_id` | Yes | string | Publisher/account identifier |
| `stored_request_id` | No | string | Stored request ID to merge |
| `width` | No | int | Video width |
| `height` | No | int | Video height |
| `min_duration` | No | int | Minimum ad duration (seconds) |
| `max_duration` | No | int | Maximum ad duration (seconds) |
| `debug` | No | bool | Enable debug mode (1=true) |
| `*` | No | string | Custom macros (non-reserved params) |

### Response

**Success (200 OK)**
- Content-Type: `application/xml; charset=utf-8`
- Body: VAST XML (even for no-bid scenarios)

**Example Response:**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<VAST version="3.0">
  <Ad id="bid123" sequence="1">
    <InLine>
      <AdSystem>prebidserver</AdSystem>
      <AdTitle>Ad bid123</AdTitle>
      <Impression><![CDATA[http://example.com/imp]]></Impression>
      <Advertiser>example.com</Advertiser>
      <Pricing model="CPM" currency="USD">5.50</Pricing>
      <Creatives>
        <Creative id="creative-bid123">
          <Linear>
            <Duration>00:00:30</Duration>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4">
                <![CDATA[http://example.com/video.mp4]]>
              </MediaFile>
            </MediaFiles>
          </Linear>
        </Creative>
      </Creatives>
      <Extensions>
        <Extension type="prebid">
          {"categories":["IAB1-1"],"deal_id":"deal456"}
        </Extension>
      </Extensions>
    </InLine>
  </Ad>
</VAST>
```

**No Bid Response:**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<VAST version="3.0"></VAST>
```

## Integration with Prebid Server

### PBS-Native Patterns Used

1. **OpenRTB Types**: Uses `github.com/prebid/openrtb/v20/openrtb2`
2. **Auction Execution**: Calls `exchange.HoldAuction()` - same as other endpoints
3. **Logging**: Uses PBS logger from context
4. **Metrics**: 
   - Added `DemandCTV`, `ReqTypeCTV`, `EndpointCTV` to metrics
   - Records standard request/time metrics
5. **Configuration**: Follows PBS config layering pattern
6. **Error Handling**: Always returns 200 with empty VAST (CTV convention)

### Metrics Recorded

- `requests_total{source="ctv", type="ctv"}`
- `request_time{source="ctv", type="ctv"}`
- `request_status{source="ctv", type="ctv", status="ok|nobid|err"}`

## Testing

### Unit Tests

Run all tests:
```bash
go test ./endpoints/ctv/...
```

Run specific package:
```bash
go test ./endpoints/ctv/vast/model
go test ./endpoints/ctv/vast/selector
go test ./endpoints/ctv/vast/enricher
go test ./endpoints/ctv/vast/formatter
go test ./endpoints/ctv
```

### Golden File Tests

Golden files are stored in `endpoints/ctv/vast/testdata/`. To update golden files:

```bash
UPDATE_GOLDEN=1 go test ./endpoints/ctv/vast/formatter -run TestGoldenFiles
```

### Integration Tests

The `ctv_test.go` file includes httptest-based integration tests:
- Valid request with bids → 200 + VAST XML
- No bids → 200 + empty VAST
- Auction error → 200 + empty VAST
- Query parameter parsing
- BidRequest construction

## Extension Points

### Adding New Receiver Profiles

1. Create new formatter in `formatter/formatter.go`:
```go
type MyReceiverFormatter struct {
    config Config
}

func (f *MyReceiverFormatter) Format(vast *model.VAST) ([]byte, error) {
    // Apply receiver-specific transformations
    return vast.Marshal()
}
```

2. Add to factory:
```go
case ReceiverMyReceiver:
    return NewMyReceiverFormatter(config)
```

### Adding Custom Enrichment

Implement custom enricher:
```go
type CustomEnricher struct {
    base *DefaultEnricher
}

func (e *CustomEnricher) Enrich(vast *model.VAST, bid *openrtb2.Bid, ...) error {
    // Custom enrichment logic
    return e.base.Enrich(vast, bid, ...)
}
```

### Macro Expansion (Future)

The `MacroConfig` structure supports macro expansion:
```go
type MacroMapping struct {
    Source       string // "query", "header", "context", "default"
    Key          string
    DefaultValue string
}
```

MVP keeps macros as raw strings; production would expand based on config.

## Performance Considerations

1. **No External Calls**: VAST unwinding/fetching is NOT implemented in MVP
2. **Memory Efficient**: Streaming XML marshaling where possible
3. **Reuses Auction Logic**: No adapter reimplementation - uses existing PBS exchange
4. **Configurable**: Can disable endpoint per account

## Future Enhancements

- [ ] Stored request fetching and merging
- [ ] Macro expansion engine
- [ ] VAST wrapper unwinding
- [ ] Account-level configuration fetching
- [ ] Pod-specific optimizations (CPM normalization, etc.)
- [ ] VAST validation strictness levels
- [ ] Caching of formatted VAST
- [ ] Request/response hooks integration
- [ ] Analytics events

## Troubleshooting

### Enable Debug Logging
Add `debug=1` query parameter to see debug output (when implemented).

### Common Issues

**Empty VAST returned:**
- Check if endpoint is enabled: `ctv_vast.enabled: true`
- Verify publisher_id is valid
- Check bidder configuration

**Missing enrichment fields:**
- Review placement_rules configuration
- Check collision_policy setting
- Verify OpenRTB fields are present in bid response

**Wrong VAST version:**
- Set `vast_version_default` in config
- Verify receiver profile (GAM SSU uses 3.0 by default)

## Standards Compliance

- **VAST 3.0**: IAB Digital Video Ad Serving Template 3.0
- **VAST 4.0**: IAB Digital Video Ad Serving Template 4.0
- **OpenRTB 2.5**: OpenRTB API Specification 2.5
- **GAM SSU**: Google Ad Manager Server-Side Unified requirements

## License

This code is part of Prebid Server and follows the same Apache 2.0 license.
