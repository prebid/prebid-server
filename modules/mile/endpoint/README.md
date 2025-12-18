# Mile Endpoint Module

Thin adapter-facing endpoint that validates Mile requests, enriches from Redis, and forwards in-process to `/openrtb2/auction`.

This module implements the Mile endpoint as a PBS module, allowing it to be configured alongside other modules in the hooks configuration.

## Features

- **Standard Mile Format**: Supports the original simplified Mile JSON request.
- **OpenRTB Compatibility**: Accepts standard OpenRTB 2.x `BidRequest` payloads.
- **Multi-Placement Support**: Processes multiple impressions/placements in parallel within a single request.
- **Context Preservation**: Automatically passes through client device, user, geo, and targeting information from the incoming request to the internal auction.
- **Redis Enrichment**: Dynamically injects bidder configurations and floor prices from Redis based on site and placement IDs.

## Configuration

Configure in your PBS config file under `hooks.modules`:

```yaml
hooks:
  modules:
    mile:
      endpoint:
        enabled: true
        endpoint: /mile/v1/request
        auth_token: "<optional-shared-token>"
        request_timeout_ms: 500
        redis_timeout_ms: 200
        max_request_size: 524288
        redis:
          addr: localhost:6379
          db: 0
          tls: false
```

## Redis Schema

The module looks for configurations in Redis using two key patterns:
1. **Primary**: `mile:site:{siteId}|plcmt:{placementId}` (Recommended for granular control)
2. **Legacy Fallback**: `mile:site:{siteId}` (Global configuration for the site)

### Redis Seeding Example

```bash
redis-cli SET "mile:site:FKKJK|plcmt:p1" '{
  "siteId":"FKKJK",
  "publisherId":123455,
  "placement":{
    "ad_unit":"banner_300x250",
    "sizes":[[300,250]],
    "floor":0.25,
    "bidders":[
      {"bidder":"appnexus","params":{"placementId":"123"}},
      {"bidder":"rubicon","params":{"siteId":443328,"zoneId":3590690,"accountId":16482}}
    ]
  }
}'
```

## Request Formats

The endpoint automatically detects the request format based on the JSON structure.

### 1. Simplified Mile Format

```json
{
  "siteId": "FKKJK",
  "publisherId": "123455",
  "placementIds": ["p1", "p2"],
  "customData": [{"targeting": {"key1": "value1"}}]
}
```

### 2. OpenRTB Format (Recommended)

Full OpenRTB 2.x requests are supported. The module extracts `SiteID` from `site.id` and looks for `placementId` inside `imp[i].ext.placementId` or `imp[i].tagid`.

```json
{
  "id": "req-123",
  "site": { "id": "FKKJK", "page": "https://example.com" },
  "device": { "ua": "...", "ip": "..." },
  "user": { "id": "..." },
  "imp": [
    {
      "id": "imp-1",
      "tagid": "p1",
      "banner": { "format": [{"w": 300, "h": 250}] }
    }
  ]
}
```

## Response

The endpoint returns a consolidated list of bids in a format optimized for the Mile Prebid.js adapter.

```json
{
  "bids": [
    {
      "requestId": "p1",
      "cpm": 0.5,
      "currency": "USD",
      "width": 300,
      "height": 250,
      "ad": "<html>...</html>",
      "ttl": 300,
      "creativeId": "cr-123",
      "netRevenue": true,
      "bidder": "appnexus",
      "mediaType": "banner"
    }
  ]
}
```

## Hooks

The module supports lifecycle hooks for custom logic:

```go
import mileModule "github.com/prebid/prebid-server/v3/modules/mile/endpoint"

// After building modules, configure hooks:
if module, ok := builtModules["mile.endpoint"].(*mileModule.Module); ok {
    module.SetHooks(mileModule.Hooks{
        Validate: func(ctx context.Context, req mileModule.MileRequest) error {
            // Custom validation
            return nil
        },
        Before: func(ctx context.Context, req mileModule.MileRequest, site *mileModule.SiteConfig, ortb *openrtb2.BidRequest) (*openrtb2.BidRequest, error) {
            // Modify request before auction
            return ortb, nil
        },
        After: func(ctx context.Context, req mileModule.MileRequest, site *mileModule.SiteConfig, status int, body []byte) ([]byte, int, error) {
            // Modify response after auction
            return body, status, nil
        },
    })
}
```

## Architecture

1. **Detection**: Determines if the payload is a Mile request or OpenRTB request.
2. **Identification**: Extracts Site and Placement IDs.
3. **Lookup**: Fetches configurations from Redis (Pipeline used for multi-placement requests).
4. **Context Merge**: Combines original request context (Device, User, Regs, etc.) with Redis-stored bidder settings.
5. **Auction**: Executes internal `/openrtb2/auction` calls in parallel.
6. **Consolidation**: Aggregates all winning bids into the final response.
