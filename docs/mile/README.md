# Mile Endpoint

Thin adapter-facing endpoint that validates Mile requests, enriches from Redis, and forwards in-process to `/openrtb2/auction`.

## Configuration
```yaml
mile:
  enabled: true
  endpoint: /mile/v1/request
  auth_token: "<optional-shared-token>"
  request_timeout_ms: 450
  redis_timeout_ms: 200
  max_request_size: 524288
  redis:
    addr: localhost:6379
    db: 0
    tls: false
```

## Redis seeding
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

## OpenAPI fragment
```yaml
post:
  summary: Mile adapter entrypoint
  operationId: mileRequest
  requestBody:
    required: true
    content:
      application/json:
        schema:
          type: object
          required: [siteId, publisherId, placementId]
          properties:
            siteId: {type: string}
            publisherId: {type: string}
            placementId: {type: string}
            customData:
              type: array
              items:
                type: object
                properties:
                  settings: {type: object}
                  targeting: {type: object}
  responses:
    "200":
      description: Auction response passthrough
      content:
        application/json: {}
    "4XX":
      description: Validation error
      content:
        application/json:
          schema:
            type: object
            properties: {error: {type: string}}
```

## Example request
```bash
curl -X POST "http://localhost:8000/mile/v1/request" \
  -H "Content-Type: application/json" \
  -d '{"siteId":"FKKJK","publisherId":"123455","placementId":"83954u44","customData":[{"settings":{},"targeting":{"key1":"value1","key2":"value2"}}]}'
```

## Hooks usage
```go
mileHooks := mile.Hooks{
  Validate: func(ctx context.Context, req mile.MileRequest) error { return nil },
  Before: func(ctx context.Context, req mile.MileRequest, site *mile.SiteConfig, ortb *openrtb2.BidRequest) (*openrtb2.BidRequest, error) {
    ortb.Test = 1
    return ortb, nil
  },
  After: func(ctx context.Context, req mile.MileRequest, site *mile.SiteConfig, status int, body []byte) ([]byte, int, error) {
    return body, status, nil
  },
}
```

## Testing
- Unit + integration: `go test ./endpoints/mile`
- Full suite: `./validate.sh` or `make test`

## Response examples
- Success: returns the raw `/openrtb2/auction` JSON body with status 200.
- Site missing: `404 {"error":"site not found: <siteId>"}`
