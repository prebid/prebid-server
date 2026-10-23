# IntentIQ Identity Module

The module enriches OpenRTB requests with EIDs before bidder requests are
created and optionally reports returned bids to IntentIQ. Reusable enrichment,
cache, HTTP, metrics, and reporting behavior comes from the
github.com/Intent-IQ/identity-go library.

## Hooks

Register `intentiq.identity` at the `processed_auction_request` stage and,
when impression reporting is required, at the `auction_response` stage.
Enrichment is fail-open: a timeout or upstream error leaves the auction
unchanged.

```yaml
hooks:
  enabled: true
  host_execution_plan:
    endpoints:
      "/openrtb2/auction":
        stages:
          processed_auction_request:
            groups:
              - timeout: 300
                hook_sequence:
                  - module_code: intentiq.identity
                    hook_impl_code: intentiq-identity-processed-auction-request-hook
          auction_response:
            groups:
              - timeout: 10
                hook_sequence:
                  - module_code: intentiq.identity
                    hook_impl_code: intentiq-identity-auction-response-hook
```

The `processed_auction_request` group timeout bounds how long the auction
waits for enrichment, so set it above `wait_timeout` (or above `timeout` when
running synchronously).

## Configuration

```yaml
hooks:
  modules:
    intentiq:
      identity:
        enabled: true
        partner_id: publisher-id
        api_endpoint: https://be-api-s2s.intentiq.com/profiles_engine/ProfilesEngineServlet
        reports_endpoint: https://reports-s2s.intentiq.com/profiles_engine/ProfilesEngineServlet
        timeout: 1000
        wait_timeout: 200
        max_concurrent_calls: 100
        metrics_enabled: true
        trace_enabled: false
        reporting:
          max_concurrent_calls: 16
          shutdown_timeout: 5000
        cache:
          enabled: true
          provider: redis
          ttl_seconds: 43200
          max_keys: 10
          max_size: 100000
          ttl_ceiling_first_party_seconds: 86400
          ttl_ceiling_third_party_seconds: 43200
          ttl_ceiling_device_seconds: 3600
          negative_ttl_seconds: 120
        redis:
          host: localhost
          port: 6379
          connect_timeout_ms: 5000
```

The provider may be `redis`, `valkey`, or `aerospike`; the matching provider
configuration is required. When caching is disabled, omit the provider config.

`timeout` and `wait_timeout` are milliseconds. An omitted `wait_timeout`, or a
value greater than or equal to `timeout`, runs synchronously. A smaller value
allows the auction to stop waiting while admitted enrichment work continues and
warms the cache. This mode requires caching and a positive
`max_concurrent_calls`.

Account configuration may override request-specific values such as partner ID,
endpoints, timeouts, and `trace_enabled`. Cache resources, concurrency, and metrics remain
process-level settings.

Impression reporting is non-blocking and bounded by `max_concurrent_calls`.
`shutdown_timeout` is the graceful-shutdown budget in milliseconds. Reporting
settings are process-level and cannot be overridden by account configuration.

When Prebid Prometheus metrics are enabled, the module exports the
`iiq_identity_*` collectors through the existing `/metrics` endpoint.

## Tracing

Tracing records the flow of a single request through both hooks: request
signals, the enrichment outcome and where it came from, the EIDs added to
`user.eids`, and the impression reports queued. It is collected only when both
conditions hold:

1. `trace_enabled: true` in the module or account configuration (off by default).
2. The request contains `ext.prebid.debug: true` or a non-empty `ext.prebid.trace`.

The messages are returned as hook debug messages and written to
`ext.trace.iiq-identity` in the auction response:

```
[enrich] start — partner=1234567890, cache=on; request signals: eids=1, device.ifa=present, ip=1.2.3.4, consent=absent
[enrich] outcome=enriched via IIQ S2S — eids=1, tc=none, abTestUuid=a1b2c3d4e5f6…9x8y7z (36 chars)
[enrich] result: enriched user.eids += 1 eid(s) [intentiq.com[xyz]]; enrich hook took 85ms total
[response] start — flow latency (enrich → response) 132ms
[response] queued 2 impression report(s) → https://reports-s2s.intentiq.com/… (fire-and-forget), 0 rejected; tc=none carried from enrich
[response] response hook took 91µs
```

The source is `IIQ S2S`, `cache l1`, `cache l2`, or `none` when the auction did
not wait for a result.
