# Mile Endpoint Module

Thin adapter-facing endpoint that validates Mile requests, enriches from Redis, and forwards in-process to `/openrtb2/auction`.

This module implements the Mile endpoint as a PBS module, allowing it to be configured alongside other modules in the hooks configuration.

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

## Redis Seeding

```bash
redis-cli SET mile:site:FKKJK '{
  "siteId":"FKKJK",
  "publisherId":"123455",
  "bidders":["appnexus","rubicon"],
  "placements":{
    "83954u44":{
      "placementId":"83954u44",
      "ad_unit":"banner_300x250",
      "sizes":[[300,250]],
      "floor":0.25,
      "bidders":["appnexus","rubicon"],
      "bidder_params":{"appnexus":{"placementId":123}}
    }
  },
  "siteConfig":{"page":"https://example.com/article/1"}
}'
```

## Request Format

```json
{
  "siteId": "FKKJK",
  "publisherId": "123455",
  "placementId": "83954u44",
  "customData": [{"settings": {}, "targeting": {"key1": "value1"}}]
}
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `siteId` | Yes | The site ID from Mile |
| `publisherId` | Yes | The publisher ID |
| `placementId` | Yes | The placement ID |
| `customData` | No | Optional targeting/settings data |

## Example Request

```bash
curl -X POST "http://localhost:8000/mile/v1/request" \
  -H "Content-Type: application/json" \
  -H "X-Mile-Token: your-auth-token" \
  -d '{"siteId":"FKKJK","publisherId":"123455","placementId":"83954u44","customData":[{"settings":{},"targeting":{"key1":"value1","key2":"value2"}}]}'
```

## Response

On success, returns the raw `/openrtb2/auction` response with status 200.

### Error Responses

| Status | Description |
|--------|-------------|
| 400 | Validation error, malformed request |
| 401 | Missing or invalid auth token |
| 404 | Site not found in Redis |
| 502 | Redis connection error |
| 503 | Auction handler not configured |

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
            ortb.Test = 1
            return ortb, nil
        },
        After: func(ctx context.Context, req mileModule.MileRequest, site *mileModule.SiteConfig, status int, body []byte) ([]byte, int, error) {
            // Modify response after auction
            return body, status, nil
        },
        OnException: func(ctx context.Context, req mileModule.MileRequest, err error) {
            // Handle errors
        },
    })
}
```

## Testing

```bash
# Run module tests
go test ./modules/mile/endpoint/...

# Run all tests
./validate.sh
```

## Architecture

This module:
1. Receives requests from MilePrebidAdapter
2. Validates required fields and optional auth token
3. Looks up site configuration from Redis
4. Builds an OpenRTB request from the Mile request + site config
5. Calls the auction handler in-process (no HTTP overhead)
6. Returns the auction response

The heavy lifting of the auction is handled by PBS core - this module is a thin orchestration layer.
