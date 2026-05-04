# DOOH Impression Value Module

This module enriches Digital Out-of-Home OpenRTB requests with `imp.qty` values before bidder requests are created. It is generic: Prebid Server does not own the registration CRUD service. Instead, this module calls a configured external bulk lookup API and caches returned values.

## Configuration

```yaml
hooks:
  enabled: true
  modules:
    prebid:
      doohimpressionvalue:
        enabled: true
        endpoint: "https://values.example.com/prebid/dooh-impression-values"
        lookup_paths:
          - dooh.id
        overwrite_policy: missing_only
        timeout_ms: 100
        cache_ttl_seconds: 300
        negative_cache_ttl_seconds: 30
        cache_size_bytes: 10485760
        headers:
          Authorization: "Bearer ${DOOH_VALUE_TOKEN}"
  host_execution_plan:
    endpoints:
      /openrtb2/auction:
        stages:
          processed_auction_request:
            groups:
              - timeout: 150
                hook_sequence:
                  - module_code: prebid.doohimpressionvalue
                    hook_impl_code: dooh-impression-value
```

`endpoint` is required. `lookup_paths` defaults to `["dooh.id"]` and supports `dooh.id`, `dooh.name`, `dooh.publisher.id`, `imp.id`, and `imp.tagid`. The first non-empty path is used for each impression.

`overwrite_policy` defaults to `missing_only`. Set it to `always` only when the external service should override an existing `imp.qty`.

## Lookup API

The module sends one bulk `POST` per auction for uncached lookup keys:

```json
{
  "account_id": "acct",
  "lookups": [
    {"path": "dooh.id", "key": "screen-123"}
  ]
}
```

The service should return matching values:

```json
{
  "values": [
    {
      "path": "dooh.id",
      "key": "screen-123",
      "multiplier": 14.2,
      "sourcetype": 1,
      "vendor": "measurement.example"
    }
  ]
}
```

Values with invalid multipliers, unsupported source types, or missing `vendor` when `sourcetype` is `1` are skipped. Missing values and API errors leave the request unchanged and add hook warnings.

## Manual Test

Run a local value service that accepts the bulk request and returns the response above, then start Prebid Server with the module config and execution plan. Send a DOOH auction request:

```sh
curl -X POST http://localhost:8000/openrtb2/auction \
  -H 'Content-Type: application/json' \
  -d '{
    "id": "auction-1",
    "dooh": {"id": "screen-123", "publisher": {"id": "acct"}},
    "imp": [{"id": "imp-1", "banner": {"format": [{"w": 1920, "h": 1080}]}, "ext": {"appnexus": {"placementId": 12883451}}}],
    "tmax": 500
  }'
```

With request debugging enabled, bidder HTTP debug output should show `imp[0].qty.multiplier` set to the returned value before the bidder call.
