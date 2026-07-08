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
`at=39`/`mi=10`/`pt=17`/`dpn=1`/`srvrReq=true`/`source=pbsgo` constants plus `dpi` (= `partner-id`),
and — when present on the request — `ip`, `ipv6`, `uas`, `uh` (UA client hints built from
`device.sua`), `ref` (site domain/page or app bundle/name), `iiquid` (an existing `intentiq.com`
eid), `pcid`+`idtype` from `device.ifa` (`idtype 4` for MAID/AAID, `idtype 8` for CTV with the id
upper-cased; skipped when `device.lmt = 1`), and `gdpr`/`us_privacy`/`gpp`/`gpp_sid`. The TCF consent
string is sent as the `gdpr-consent` request header. The response `data.eids` are merged into
`user.eids`; on any failure the hook takes no action and the auction proceeds unchanged (fail-open).

## Setup

The module runs at two stages: `processed_auction_request` (enrich `user.eids`) and, optionally,
`auction_response` (report winning bids to `reports-endpoint`). Enable the module and add the hook(s)
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
        api-endpoint: https://be-api-s2s.intentiq.com/profiles_engine/ProfilesEngineServlet
        reports-endpoint: https://reports-s2s.intentiq.com/profiles_engine/ProfilesEngineServlet
        partner-id: "1234567890"
        timeout: 1000
        cache-max-size: 33554432   # in-process (L1) byte budget; see "Caching"
        metrics-enabled: true
        cache:
          enabled: true
          ttlseconds: 43200
          max-keys: 10
          ttl-ceiling-first-party-seconds: 86400
          ttl-ceiling-third-party-seconds: 43200
          ttl-ceiling-device-seconds: 3600
          negative-ttl-seconds: 120
          in-progress-ttl-seconds: 1800
        redis:
          host: localhost
          port: 6379
          # password: ""
```

Use the region-specific `api-endpoint`: US `be-api-s2s.intentiq.com`, EU
`be-api-s2s-gdpr.intentiq.com`, APAC `be-api-s2s-apac.intentiq.com`. When `api-endpoint` is empty the
enrich hook is a no-op.

### Account-Level Config

Global config (above) provides defaults. Account-specific values can be set under the account's
`hooks.modules.intentiq.identity` config and are merged over the global defaults per request — so
`partner-id`, `timeout`, and `cache.*` can be tuned per account. `redis.*`, `cache-max-size`, and
`metrics-enabled` are global-only.

## Module Configuration Parameters

| Param                                  | Level   | Required | Type    | Default | Description                                                        |
|:---------------------------------------|:--------|:---------|:--------|:--------|:-------------------------------------------------------------------|
| `api-endpoint`                         | global  | yes      | string  | none    | Bid Enhancement `ProfilesEngineServlet` URL (region-specific)      |
| `reports-endpoint`                     | global  | no       | string  | none    | Impression-reporting URL; blank disables the impression hook       |
| `partner-id`                           | account | yes      | string  | none    | Partner token from IntentIQ, sent as the `dpi` query parameter     |
| `timeout`                              | account | no       | int     | 1000    | HTTP timeout (ms) for the resolution/report calls                  |
| `cache.enabled`                        | account | no       | bool    | false   | Use the two-layer cache (requires `redis.*`)                       |
| `cache.ttlseconds`                     | account | no       | int     | 43200   | Fallback positive TTL (s) when the response omits `cttl`           |
| `cache.max-keys`                       | account | no       | int     | 10      | Max alias keys derived per request                                 |
| `cache.ttl-ceiling-first-party-seconds`| account | no       | int     | 86400   | Upper bound on TTL for first-party id keys                         |
| `cache.ttl-ceiling-third-party-seconds`| account | no       | int     | 43200   | Upper bound on TTL for third-party id keys (`intentiq.com`)        |
| `cache.ttl-ceiling-device-seconds`     | account | no       | int     | 3600    | Upper bound on TTL for the probabilistic device-composite key      |
| `cache.negative-ttl-seconds`           | account | no       | int     | 120     | TTL for the negative (unresolvable id) sentinel                    |
| `cache.in-progress-ttl-seconds`        | account | no       | int     | 1800    | TTL for the IN_PROGRESS marker that dedups concurrent resolutions  |
| `cache-max-size`                       | global  | no       | int     | 100000  | L1 (in-process) **byte** budget — see "Caching"                    |
| `metrics-enabled`                      | global  | no       | bool    | true    | Emit the module's Prometheus metrics; `false` to opt out           |
| `redis.host`                           | global  | cond.    | string  | none    | Redis host (required when caching)                                 |
| `redis.port`                           | global  | cond.    | int     | none    | Redis port (required when caching)                                 |
| `redis.password`                       | global  | no       | string  | none    | Redis password                                                     |

## Caching

When `cache.enabled` is true and `redis.*` is configured, resolved eids are cached in two layers:
**L1** (in-process, [freecache](https://github.com/coocood/freecache)) backed by **L2** (shared, Redis
by default). L2 failures are non-fatal — the hook falls through to a live API call.

- **Multi-key (alias) caching.** Every relevant first-party id on the request becomes a namespaced
  alias key, ordered by priority: `iiq:<id>` (`intentiq.com`), `pubcid:<id>`
  (`pubcid.org`/`sharedid.org`), `maid:<ifa>` (upper-cased for CTV, skipped when `device.lmt = 1`),
  `<source>:<id>` for any other eid, and a probabilistic `dev:<ifa_ua_ip>` composite (using a
  *normalized* UA, not the raw string) as last resort. Keys are de-duplicated and capped at
  `cache.max-keys`. On a lookup the highest-priority key with a live entry wins, and that entry is
  **back-filled** under every other key that missed, so the alias graph grows over time. Differing
  resolutions are never merged — only the single winning entry propagates.
- **TTL.** The response `cttl` (or `cache.ttlseconds` when omitted) always wins, capped per id class by
  the `ttl-ceiling-*` values.
- **Negative caching.** When the API resolves no eids, a short-lived negative sentinel is written under
  all candidate keys so unresolvable ids do not re-hit the S2S API every request.
- **In-progress dedup.** On a full miss, an `IN_PROGRESS` marker is written under all candidate keys
  before the live call; a concurrent request for the same id reads it and skips a duplicate call. It is
  overwritten by the resolved/negative entry when the call completes, or expires otherwise.

> **`cache-max-size` is a byte budget, not an entry count.** The Java module bounds L1 by entry count
> (Caffeine); the Go L1 (freecache) is byte-budget bounded. Size it accordingly (e.g. `33554432` for
> 32 MiB); values below freecache's 512 KiB floor are bumped up.

## Impression Reporting

When `reports-endpoint` is set and the `auction_response` hook is in the execution plan, the module
reports each winning `seatbid[].bid[]` to the IntentIQ impression API — a fire-and-forget GET to
`<reports-endpoint>?at=45&rtype=1&source=pbsgo&dpi=<partner-id>&rdata=<UTF-8 URL-encoded JSON>`. The
`rdata` carries `bidderCode`, `partnerId`, `cpm`, `currency`, `originalCpm`/`originalCurrency` (from
the bid ext), `placementId`, `biddingPlatformId=4`, `vrref`, `prebidAuctionId`, `partnerAuctionId`,
`abTestUuid`, `terminationCause`, `ip`, and `ua`. Because the Go `auction_response` payload exposes
only the bid response, the request-derived fields (`vrref`/`prebidAuctionId`/`ip`/`ua`) and the
`abTestUuid`/`terminationCause` from the resolution response are stashed by the enrich hook in the
module context and read here. With `reports-endpoint` blank the hook is a no-op. The bid response is
never modified.

## Metrics

The hook framework already emits per-module `call`/`success.*`/`failure`/`timeout`/`execution-error`/
`duration`. In addition this module registers its own **Prometheus** collectors into the server's
scrape registry (threaded via `moduledeps.ModuleDeps.MetricsRegisterer`), exported on the server's
`/metrics` endpoint. Recording is on by default; set `metrics-enabled: false` (global) to disable.

The partner id is a Prometheus `partner` **label** (the Java module used a `_<dpi>` name suffix), and
`layer` (`l1`/`l2`) and `keytype` (`first_party`/`third_party`/`device`) are labels. All series are
prefixed `modules_module_intentiq_identity_custom_`:

| Metric (suffix)                    | Type        | Labels                    | Meaning                                              |
|:-----------------------------------|:------------|:--------------------------|:-----------------------------------------------------|
| `cache_hit_total`                  | counter     | layer, keytype, partner   | positive entry served from cache                     |
| `cache_miss_total`                 | counter     | keytype, partner          | full miss → API called                               |
| `cache_negative_hit_total`         | counter     | layer, keytype, partner   | negative sentinel hit (counted as miss, no API call) |
| `cache_in_progress_total`          | counter     | layer, keytype, partner   | in-flight marker hit; duplicate call skipped         |
| `api_success_total`                | counter     | partner                   | resolution API responded and parsed                  |
| `api_error_total`                  | counter     | partner                   | resolution API failed/timed out/unparseable          |
| `api_latency_seconds`              | histogram   | partner                   | resolution API call duration                         |
| `enriched_total`                   | counter     | partner                   | eids added to `user.eids`                            |
| `eids_none_total`                  | counter     | partner                   | resolution produced no eids                          |
| `skip_no_endpoint_total`           | counter     | partner                   | no `api-endpoint`; resolution skipped                |
| `termination_cause_total`          | counter     | tc, partner               | one series per termination-cause id (0–199)          |
| `flow_latency_seconds`             | histogram   | partner                   | whole-flow latency (enrich → bid release)            |
| `impression_reported_total`        | counter     | partner                   | winning bid reported                                 |
| `impression_error_total`           | counter     | partner                   | impression report call failed                        |
| `l1_size` / `l1_eviction`          | gauge       | —                         | L1 entry count / cumulative evictions                |
| `l2_size` / `l2_eviction`          | gauge       | —                         | Redis `DBSIZE` / `evicted_keys` (instance-wide)      |
| `l1_get_error` / `l1_put_error`    | counter     | —                         | L1 read/write threw (≈never)                         |
| `l2_get_latency_seconds` / `l2_put_latency_seconds` | histogram | —      | L2 GET/PUT duration                                  |
| `l2_get_error` / `l2_put_error`    | counter     | —                         | L2 GET/PUT failed (GET → fail-open live call)        |

The L1/L2 health series are process-wide, so they carry no `partner` label.

## Running the demo

1. Set `api-endpoint` and `partner-id` (and, for caching, a reachable `redis.*`) in
   `sample/configs/prebid-config-with-intentiq.yaml`.
2. Run the server with that config:
   `go run . -v 1 -logtostderr --config sample/configs/prebid-config-with-intentiq.yaml`
3. POST an OpenRTB request to `http://localhost:8000/openrtb2/auction` and observe `user.eids`
   enriched in `ext.debug.resolvedrequest` (send `"test": 1` in the request to get debug output).
4. Scrape the module's metrics at `http://localhost:9090/metrics`
   (series prefixed `modules_module_intentiq_identity_custom_*`).

## Maintainer contacts

Any suggestions or questions can be directed to the IntentIQ team. Alternatively please open a new
[issue](https://github.com/prebid/prebid-server/issues/new) or
[pull request](https://github.com/prebid/prebid-server/pulls) in this repository.
