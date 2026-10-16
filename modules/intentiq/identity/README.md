## Overview

The IntentIQ Identity module enriches an incoming OpenRTB request by adding resolved IDs to
`user.eids`. At the `processed_auction_request` stage it calls the IntentIQ Bid Enhancement S2S API
(`ProfilesEngineServlet`) and merges the eids from the response into `user.eids` before the request is
sent to bidders. Optionally, at the `auction_response` stage it reports each winning bid to the
IntentIQ impression API. Please contact your IntentIQ account manager to get a partner token.

This is the Go port of the prebid-server-java `extra/modules/intentiq-identity` module. See the
[S2S integration docs](https://s2s.documents.intentiq.com/) for the full API contract.

## Operation Details

The resolution request (`processed_auction_request`) sends the
`at=39`/`mi=10`/`pt=17`/`dpn=1`/`srvrReq=true`/`source=pbgo` constants plus `dpi` (= `partner_id`),
and — when present on the request — `ip`, `ipv6`, `uas`, `uh` (UA client hints built from
`device.sua`), `ref` (site domain/page or app bundle/name), `iiquid` (an existing `intentiq.com`
eid), `pcid`+`idtype` from `device.ifa` (`idtype 4` for MAID/AAID, `idtype 8` for CTV with the id
upper-cased; skipped when `device.lmt = 1`), and `gdpr`/`us_privacy`/`gpp`/`gpp_sid`. The TCF consent
string is sent as the `gdpr-consent` request header. The response `data.eids` are merged into
`user.eids`; on any failure the hook takes no action and the auction proceeds unchanged (fail-open).

## Setup

The module runs at two stages: `processed_auction_request` (enrich `user.eids`) and, optionally,
`auction_response` (report winning bids to `reports_endpoint`). Enable the module and add the hook(s)
to the execution plan.

### Execution Plan

```yaml
hooks:
  enabled: true
  host_execution_plan:
    endpoints:
      /openrtb2/auction:
        stages:
          processed_auction_request:
            groups:
              - timeout: 1000
                hook_sequence:
                  - module_code: "intentiq.identity"
                    hook_impl_code: "HandleProcessedAuctionHook"
          auction_response:
            groups:
              - timeout: 100
                hook_sequence:
                  - module_code: "intentiq.identity"
                    hook_impl_code: "HandleAuctionResponseHook"
```

### Global Config

```yaml
hooks:
  modules:
    intentiq:
      identity:
        enabled: true
        api_endpoint: https://be-api-s2s.intentiq.com/profiles_engine/ProfilesEngineServlet
        reports_endpoint: https://reports-s2s.intentiq.com/profiles_engine/ProfilesEngineServlet
        partner_id: "1234567890"
        timeout: 1000
        metrics_enabled: true
        trace_enabled: false       # opt-in per-request debug trace; see "Tracing"
        cache:
          enabled: true
          provider: redis          # redis | valkey | aerospike
          ttl_seconds: 43200
          max_keys: 10
          max_size: 33554432       # in-process (L1) byte budget; see "Caching"
          ttl_ceiling_first_party_seconds: 86400
          ttl_ceiling_third_party_seconds: 43200
          ttl_ceiling_device_seconds: 3600
          negative_ttl_seconds: 120
          in_progress_ttl_seconds: 1800
        
        # L2 backends — any of them may be present; cache.provider decides which is built.
        redis:
          host: localhost
          port: 6379
          # password: ""
        
        # valkey:
        #   host: localhost
        #   port: 6379
        #   password: ""
        
        # aerospike:
        #   host: localhost
        #   port: 3000
        #   namespace: prebid
        #   set: identity
        #   client_policy:            # all optional
        #     connection_queue_size: 1024
        #     min_connections_per_node: 3
        #     connect_timeout_ms: 30000
        #     idle_timeout_ms: 0
```

Use the region-specific `api_endpoint`: US `be-api-s2s.intentiq.com`, EU
`be-api-s2s-gdpr.intentiq.com`, APAC `be-api-s2s-apac.intentiq.com`. When `api_endpoint` is empty the
enrich hook is a no-op.

### Account-Level Config

Global config (above) provides defaults. Account-specific values can be set under the account's
`hooks.modules.intentiq.identity` config and are merged over the global defaults per request — so
`partner_id` and `timeout` can be tuned per account. `redis.*`, `aerospike.*`, `cache.provider`,
`cache.max_size`, and `metrics_enabled` are global-only.

> **`cache.*` TTLs are read once at startup.** The TTL policy is built in `Builder` from the global
> config, so an account-level `cache.ttl_seconds` or `ttl_ceiling_*` override is accepted by the
> config merge but never reaches the cache. Only `cache.enabled` and `cache.max_keys` vary per
> account. Non-positive TTLs fail startup rather than building a cache whose entries no lookup can
> return.

## Module Configuration Parameters

| Param                                  | Level   | Required | Type    | Default | Description                                                        |
|:---------------------------------------|:--------|:---------|:--------|:--------|:-------------------------------------------------------------------|
| `enabled`                              | global  | yes      | bool    | false   | Read by prebid-server, not the module; without `true` the module is never built |
| `api_endpoint`                         | global  | yes      | string  | none    | Bid Enhancement `ProfilesEngineServlet` URL (region-specific)      |
| `reports_endpoint`                     | global  | no       | string  | none    | Impression-reporting URL; blank disables the impression hook       |
| `partner_id`                           | account | yes      | string  | none    | Partner token from IntentIQ, sent as the `dpi` query parameter     |
| `timeout`                              | account | no       | int     | 1000    | HTTP timeout (ms) for the resolution/report calls                  |
| `cache.enabled`                        | account | no       | bool    | false   | Use the two-layer cache (requires `cache.provider` + its block)    |
| `cache.provider`                       | global  | cond.    | string  | none    | `redis`, `valkey` or `aerospike`, exactly; required when `cache.enabled` |
| `cache.ttl_seconds`                     | account | no       | int     | 43200   | Fallback positive TTL (s) when the response omits `cttl`           |
| `cache.max_keys`                       | account | no       | int     | 10      | Max alias keys derived per request                                 |
| `cache.max_size`                       | global  | no       | int     | 100000  | L1 (in-process) **byte** budget — see "Caching"                    |
| `cache.ttl_ceiling_first_party_seconds`| account | no       | int     | 86400   | Upper bound on TTL for first-party id keys                         |
| `cache.ttl_ceiling_third_party_seconds`| account | no       | int     | 43200   | Upper bound on TTL for third-party id keys (`intentiq.com`)        |
| `cache.ttl_ceiling_device_seconds`     | account | no       | int     | 3600    | Upper bound on TTL for the probabilistic device-composite key      |
| `cache.negative_ttl_seconds`           | account | no       | int     | 120     | TTL for the negative (unresolvable id) sentinel                    |
| `cache.in_progress_ttl_seconds`        | account | no       | int     | 1800    | TTL for the IN_PROGRESS marker that dedups concurrent resolutions  |
| `metrics_enabled`                      | global  | no       | bool    | true    | Record the module's Prometheus metrics on the server's `/metrics`; `false` to opt out |
| `trace_enabled`                        | account | no       | bool    | false   | Allow the per-request flow trace in `ext.trace.iiq-identity` — see "Tracing" |
| `redis.host`                           | global  | cond.    | string  | none    | Redis host (required when caching)                                 |
| `redis.port`                           | global  | cond.    | int     | none    | Redis port (required when caching)                                 |
| `redis.password`                       | global  | no       | string  | none    | Redis password                                                     |
| `redis.connect_timeout_ms`             | global  | no       | int     | 5000    | Bounds the startup connectivity check and connection dials         |
| `valkey.host`                          | global  | cond.    | string  | none    | Valkey host (required when using this backend)                     |
| `valkey.port`                          | global  | cond.    | int     | none    | Valkey port (usually 6379)                                         |
| `valkey.password`                      | global  | no       | string  | none    | Valkey password                                                    |
| `valkey.connect_timeout_ms`            | global  | no       | int     | 5000    | Bounds the startup connectivity check and connection dials         |
| `aerospike.host`                       | global  | cond.    | string  | none    | Aerospike seed host (required when using this backend)             |
| `aerospike.port`                       | global  | cond.    | int     | none    | Aerospike service port (usually 3000)                              |
| `aerospike.namespace`                  | global  | cond.    | string  | none    | Namespace the entries live in; must have `nsup-period > 0`         |
| `aerospike.set`                        | global  | cond.    | string  | none    | Set name within the namespace                                      |
| `aerospike.client_policy.connection_queue_size`   | global | no | int | 1024 | Max pooled connections per node                                 |
| `aerospike.client_policy.min_connections_per_node`| global | no | int | 3    | Connections kept warm per node                                  |
| `aerospike.client_policy.connect_timeout_ms`      | global | no | int | 30000| Connection/cluster-tending timeout                              |
| `aerospike.client_policy.idle_timeout_ms`         | global | no | int | 0    | Idle connection reap interval; 0 = never                        |

## Caching

When `cache.enabled` is true and a backend is configured, resolved eids are cached in two layers:
**L1** (in-process, [freecache](https://github.com/coocood/freecache)) backed by **L2**, a shared
store. L2 failures at request time are non-fatal — the hook falls through to a live API call.

### L2 backends

`cache.provider` names the backend to build, and is required whenever `cache.enabled` is true — it
is matched exactly (`redis`, `valkey` or `aerospike`, lowercase), never inferred from whichever block
happens to be filled in. Any number of blocks may live in one config file — handy for switching
environments or migrating — and only the named one is validated and connected.

Every backend verifies connectivity during `Builder`, so a configured-but-unreachable backend fails
startup instead of booting into a cache that misses on every read. A missing or unknown
`cache.provider`, or one naming a block that is absent or invalid, also fails startup: caching is
either working at startup or the module does not build.

| Backend | Config block |
|:--|:--|
| Redis | `redis.*` |
| Valkey | `valkey.*` |
| Aerospike | `aerospike.*` |

L2 capacity is not the module's concern: eviction is configured on the backend (Redis/Valkey
`maxmemory` + `maxmemory-policy`, Aerospike namespace high-water marks and `nsup-period`) and
observed through that backend's own exporter. Note that a stock Redis defaults to
`maxmemory-policy noeviction`, which rejects writes once full rather than evicting — the module
fails open, so the visible symptom is a rising `l2_put_error` and a collapsing hit rate.

Redis and Valkey are separate providers on separate clients ([go-redis](https://github.com/redis/go-redis)
and [valkey-go](https://github.com/valkey-io/valkey-go) respectively) — the wire protocol is shared,
so either block can point at either server, but `cache.provider` is what decides which client is
built. The Valkey client has its own client-side cache and command retries turned off: the module
already runs an L1, and a failed L2 read must surface immediately so the hook falls through to a live
API call instead of spending its timeout budget on retries.

Entries are stored identically by all of them: one key per alias, value is the serialized entry JSON. On
Aerospike that is a single bin named `v` in `(namespace, set, key)`, which is also the layout
IntentIQ's own prebid-server deployment writes, so both can share one cluster.

> **Aerospike namespaces need `nsup-period > 0`.** Per-record TTLs are only enforced when the
> namespace expiration thread runs. With `nsup-period 0` the module's `ttl_ceiling_*` values are
> written and then ignored, and entries accumulate indefinitely. Note the official
> `aerospike/aerospike-server` image emits `nsup-period` only when `DEFAULT_TTL` is non-zero, so a
> containerised namespace left at the default expires nothing.

### Behavior

- **Multi-key (alias) caching.** Every relevant first-party id on the request becomes a namespaced
  alias key, ordered by priority: `iiq:<id>` (`intentiq.com`), `pubcid:<id>`
  (`pubcid.org`/`sharedid.org`), `maid:<ifa>` (upper-cased for CTV, skipped when `device.lmt = 1`),
  `<source>:<id>` for any other eid, and a probabilistic `dev:<ifa_ua_ip>` composite (using a
  *normalized* UA, not the raw string) as last resort. Keys are de-duplicated and capped at
  `cache.max_keys`. On a lookup the highest-priority key with a live entry wins, and that entry is
  **back-filled** under every other key that missed, so the alias graph grows over time. Differing
  resolutions are never merged — only the single winning entry propagates.
- **TTL.** The response `cttl` (or `cache.ttl_seconds` when omitted) always wins, capped per id class by
  the `ttl_ceiling_*` values.
- **Negative caching.** When the API resolves no eids, a short-lived negative sentinel is written under
  all candidate keys so unresolvable ids do not re-hit the S2S API every request.
- **In-progress dedup.** On a full miss, an `IN_PROGRESS` marker is written under all candidate keys
  before the live call; a concurrent request for the same id reads it and skips a duplicate call. It is
  overwritten by the resolved/negative entry when the call completes, or expires otherwise.

> **`cache.max_size` is a byte budget, not an entry count.** The Java module bounds L1 by entry count
> (Caffeine); the Go L1 (freecache) is byte-budget bounded. Size it accordingly (e.g. `33554432` for
> 32 MiB); values below freecache's 512 KiB floor are bumped up.

## Impression Reporting

When `reports_endpoint` is set and the `auction_response` hook is in the execution plan, the module
reports each winning `seatbid[].bid[]` to the IntentIQ impression API — a fire-and-forget GET to
`<reports_endpoint>?at=45&rtype=1&source=pbgo&dpi=<partner_id>&rdata=<UTF-8 URL-encoded JSON>`. The
`rdata` carries `bidderCode`, `partnerId`, `cpm`, `currency`, `originalCpm`/`originalCurrency` (from
the bid ext), `placementId`, `biddingPlatformId=4`, `vrref`, `prebidAuctionId`, `partnerAuctionId`,
`abTestUuid`, `terminationCause`, `ip`, and `ua`. Because the Go `auction_response` payload exposes
only the bid response, the request-derived fields (`vrref`/`prebidAuctionId`/`ip`/`ua`) and the
`abTestUuid`/`terminationCause` from the resolution response are stashed by the enrich hook in the
module context and read here. With `reports_endpoint` blank the hook is a no-op. The bid response is
never modified.

## Tracing

Metrics show how often something happened. Tracing shows what happened on a single request and why.
It records the main flow through both hooks: request signals, cache result, IIQ API call, returned data, and whether anything was added to user.eids.
Tracing is enabled only when both conditions are true:

1. `trace_enabled: true` in the module config. It is off by default.
2. The request contains either e`xt.prebid.debug: true` or `ext.prebid.trace`.

Example:

```
[enrich] start — partner=1234567890, cache=on; request signals: eids=1, device.ifa=present, ip=1.2.3.4, consent=absent
[enrich] cache lookup over 2 candidate key(s): [abc(third_party), 9f3…(device)]
[enrich] cache MISS (keytype=third_party) in 412µs — in-progress marker set, calling IIQ
[enrich] → IIQ resolve GET (dpi=1234567890, ip=1.2.3.4, gdpr-consent=absent, timeout=1s)
[enrich]   url=https://be-api-s2s.intentiq.com/profiles_engine/ProfilesEngineServlet?at=39&…
[enrich] ← IIQ 200 in 84ms (GET latency) — eids=1, tc=none, cttl=43200, abTestUuid=a1b2c3d4e5f6…9x8y7z (36 chars)
[enrich] cached 1 eid(s) under all keys: [intentiq.com[xyz]]
[enrich] result: enriched user.eids += 1 eid(s) [intentiq.com[xyz]]; enrich hook took 85ms total
[response] start — flow latency (enrich → response) 132ms
[response] queued 2 impression report(s) → https://reports-s2s.intentiq.com/… (fire-and-forget); tc=none carried from enrich
[response] response hook took 91µs
```

## Metrics

The hook framework already emits per-module `call`/`success.*`/`failure`/`timeout`/`execution-error`/
`duration`. In addition this module records its own **Prometheus** series. Recording is on by
default; set `metrics_enabled: false` (global) to disable.

The collectors are registered into `moduledeps.ModuleDeps.MetricsRegisterer` during `Builder`, so
they are exported on prebid-server's own `/metrics` endpoint alongside the core series — the module
neither owns a registry nor opens a listener of its own. Nothing is recorded when
`metrics.prometheus.port` is `0`: with no Prometheus listener the host supplies no registerer, and
the module falls back to its no-op adapter rather than collecting series nobody can scrape.

The Prometheus dependency stays inside the `metrics/` subpackage, laid out like the sibling `cache/`
package — a port in one file, one file per backing implementation:

| File | Role |
|:--|:--|
| `metrics/metrics.go` | the `Metrics` port, plus `New` choosing an adapter for the host's registerer |
| `metrics/prometheus.go` | the Prometheus adapter (collector definitions, the series prefix) |
| `metrics/noop.go` | the `Noop` null-object adapter, used when metrics are disabled |

Keeping Prometheus behind that boundary leaves the hooks recording through `metrics.Metrics` only.

The partner id is a Prometheus `partner_id` **label** (the Java module used a `_<dpi>` name suffix).
All series are prefixed `iiq_identity_` (namespace `iiq`, subsystem `identity`):

| Metric (suffix)                    | Type      | Labels                          | Meaning                                                      |
|:-----------------------------------|:----------|:--------------------------------|:-------------------------------------------------------------|
| `requests_total`                   | counter   | partner_id                      | enrich-hook invocations (ingress QPS)                        |
| `cache_lookup_total`               | counter   | result, layer, partner_id       | `result=hit\|miss`; `layer=l1\|l2` on a hit, `none` on a miss |
| `api_success_total`                | counter   | partner_id                      | resolution API responded 2xx and parsed                      |
| `api_error_total`                  | counter   | reason, status_code, partner_id | `reason=timeout\|transport\|status\|body_read\|parse\|request`; `status_code` empty when no response arrived |
| `api_latency_seconds`              | histogram | partner_id                      | resolution API call duration                                 |
| `enriched_total`                   | counter   | partner_id                      | eids added to `user.eids`                                    |
| `not_enriched_total`               | counter   | reason, partner_id              | `reason=no_ids\|no_ids_cached\|in_progress\|no_endpoint`      |
| `impression_reported_total`        | counter   | partner_id                      | winning bid reported                                         |
| `impression_error_total`           | counter   | partner_id                      | impression report call failed                                |
| `l1_size` / `l1_eviction`          | gauge     | —                               | L1 entry count / cumulative evictions                        |
| `l1_put_error_total`               | counter   | —                               | L1 write did not land — chiefly freecache `ErrLargeEntry` (entry over `max_size/1024`) |
| `l2_get_latency_seconds` / `l2_put_latency_seconds` | histogram | —              | L2 GET/PUT duration                                          |
| `l2_requests_total`                | counter   | op, result                      | `op=get\|put`; `result=hit\|miss\|error` for get, `stored\|error` for put                  |

`cache_lookup_total` sums to total lookups, so `hit / sum` is the hit ratio. A negative-sentinel or
in-flight-marker lookup counts as a `miss` here (no entry was served) and is distinguished by the
`not_enriched_total` reason.

The L1/L2 health series are process-wide, so they carry no `partner_id` label.

## Running the demo

[`sample/002_intentiq_identity`](../../../sample/002_intentiq_identity) is a runnable configuration of
this module — the execution plan, the module config, a Valkey-backed cache, and an auction request to
post at it. Put your partner token in `partner_id` in its `app.yaml`, then from the `sample` directory:

```sh
docker-compose up --build 002_intentiq_identity

curl -s -X POST http://localhost:8000/openrtb2/auction \
  -H 'Content-Type: application/json' \
  -d @002_intentiq_identity/request.json \
  | jq '{eids: .ext.debug.resolvedrequest.user.eids, trace: .ext.trace["iiq-identity"]}'

curl -s http://localhost:9090/metrics | grep iiq_identity_
```

`eids` shows what was merged into `user.eids`, `trace` is the flow trace described under
[Tracing](#tracing), and the metrics are this module's series on prebid-server's own endpoint. Posting
the request twice shows the second one served from cache. See
[sample/README.md](../../../sample/README.md#002_intentiq_identity) for the annotated walkthrough, and
[Caching](#caching) to switch to Redis or Aerospike.

## Maintainer contacts

Any suggestions or questions can be directed to the IntentIQ team. Alternatively please open a new
[issue](https://github.com/prebid/prebid-server/issues/new) or
[pull request](https://github.com/prebid/prebid-server/pulls) in this repository.
