# Mile Endpoint Design

## References
- Module implementation: `modules/mile/endpoint/`
- Module registration in router: `router/router.go:268-290`
- Canonical auction handler returned by `openrtb2.NewEndpoint`: `endpoints/openrtb2/auction.go:90-134`
- Module builder registration: `modules/builder.go`

## Goal
Expose `POST /mile/v1/request` as a thin adapter-facing shim. It validates Mile payloads, enriches them with site/placement configuration from Redis, translates into an OpenRTB 2.x request, and forwards in-process to PBS’s existing `/openrtb2/auction` handler (no extra HTTP hop). Before/after hooks allow custom logic without touching PBS core.

## Flow
1. Accept JSON (Content-Type: application/json), enforce configured body limit.
2. Validate required fields and optional header token.
3. Lookup `mile:site:{siteId}` in Redis; return 404 on miss, 502 on backend errors.
4. Merge placement + site config → OpenRTB request:
   - `site.id/publisher.id` from Redis (fallback to request.publisherId).
   - `imp.id/tagid` from placement; banner formats from `sizes`; `bidfloor` from config.
   - `imp.ext.prebid.bidder` per placement/site bidder list with bidder params.
   - `imp.ext.prebid.passthrough` carries `customData`; optional stored request id supported.
   - `req.ext.prebid.targeting` defaults to medium price granularity.
5. Run `Before` hook (may mutate OpenRTB request).
6. Call in-process `/openrtb2/auction` handler with cloned headers and shared context.
7. Run `After` hook (may rewrite body/status); return response.
8. Record metrics (request count/latency) and structured errors.

## Redis Schema
Key: `mile:site:{siteId}`  
Value example:
```json
{
  "siteId": "FKKJK",
  "publisherId": "123455",
  "bidders": ["appnexus", "rubicon"],
  "placements": {
    "83954u44": {
      "placementId": "83954u44",
      "ad_unit": "banner_300x250",
      "sizes": [[300,250]],
      "floor": 0.25,
      "bidders": ["appnexus", "rubicon"],
      "bidder_params": {
        "appnexus": {"placementId": 123}
      }
    }
  },
  "siteConfig": {"page": "https://example.com/article/1"}
}
```

## Hooks
```go
type Hooks struct {
  Validate(ctx, mileReq) error
  Before(ctx, mileReq, site, ortb) (*openrtb2.BidRequest, error)
  After(ctx, mileReq, site, status, body) ([]byte, int, error)
  OnException(ctx, mileReq, err)
}
```

## Error Handling & Timeouts
- 400 for validation/JSON errors; 401 if token mismatch; 404 when site missing; 502 for Redis failures; auction errors propagate.
- Request timeout configurable via `mile.request_timeout_ms`; Redis lookup timeout via `mile.redis_timeout_ms`.
- No retries on auction; optional retry handled externally via Redis HA.

## Observability & Security
- Metrics via `metricsEngine.RecordRequest/RecordRequestTime`; logs carry site/placement and failures.
- Optional header token `X-Mile-Token`; request size limit (`mile.max_request_size` fallback to global `max_request_size`).
- Context cancellation respected through in-process call into `/openrtb2/auction`.
