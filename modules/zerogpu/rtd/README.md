# ZeroGPU Real Time Data Module

## Overview

The ZeroGPU RTD module enriches an incoming OpenRTB request with IAB content
categories derived from the publisher's domain.

On each auction the module resolves the domain from the bid request, classifies
it with ZeroGPU's `zlm-v1-iab-domain-classifier` model, and appends the
resulting categories to `{site,app,dooh}.content.data` as Seller-Defined
contextual segments under `ext.segtax: 6` (IAB Content Taxonomy 2.2). Because
the segments are written to standard First Party Data fields, every bidder in
the auction can read them - no bidder-specific integration is required.

**The auction never waits on ZeroGPU.** The hook reads an in-process cache and
nothing else. When a domain is not yet cached, the auction proceeds unenriched
and the classification is fetched in the background, so subsequent auctions on
that domain are enriched from memory. Measured against the live API, the hook
costs ~15µs whether the domain is cached or not, while the classification call
it avoids takes ~0.9s.

Every failure mode is fail-open: if the ZeroGPU API is slow, unreachable, or
returns an error, auctions simply go unenriched. The module never rejects a
request, never delays one, and never creates bids.

## Prerequisites

A ZeroGPU API key is required. Sign in at
<https://platform.zerogpu.ai/dashboard>, or start from <https://zerogpu.ai/>
and click **Start Building**.

API reference:

- <https://docs.zerogpu.ai/api-reference/models/zlm-v1-iab-domain-classifier>
- <https://docs.zerogpu.ai/docs/text-classification#zlm-v1-iab-domain-classifier>

## Configuration

The module is disabled unless `enabled` is set. `api_key` is the only required
parameter; everything else has a working default.

```yaml
hooks:
  enabled: true
  modules:
    zerogpu:
      rtd:
        enabled: true
        api_key: ${ZEROGPU_API_KEY}
  host_execution_plan: >
    {
      "endpoints": {
        "/openrtb2/auction": {
          "stages": {
            "processed_auction_request": {
              "groups": [{
                "timeout": 10,
                "hook_sequence": [{
                  "module_code": "zerogpu.rtd",
                  "hook_impl_code": "zerogpu-rtd-processed-auction-request"
                }]
              }]
            }
          }
        }
      }
    }
```

A complete host configuration is in [sample/pbs_example.json](sample/pbs_example.json).

The group `timeout` only has to cover an in-memory cache read, so a small value
is correct. It is unrelated to `timeout_ms`, which bounds the background
warm-up and never applies to the auction path.

### Parameters

| Parameter | Type | Required | Default | Description |
| --- | --- | --- | --- | --- |
| `api_key` | string | yes | - | ZeroGPU API key, sent as the `x-api-key` header. |
| `endpoint` | string | no | `https://api.zerogpu.ai/v1/responses` | Classification endpoint. Overridable for a different region or a proxy; the Responses API request and response shape is assumed. |
| `model` | string | no | `zlm-v1-iab-domain-classifier` | Model to classify with. |
| `timeout_ms` | int | no | `2000` | HTTP timeout for a background classification call. Never applies to the auction path. |
| `cache_ttl_seconds` | int | no | `86400` | How long a successful classification is cached. |
| `negative_cache_ttl_seconds` | int | no | `300` | How long an empty result or a stable failure (400/401/403/420) is cached. |
| `retry_cache_ttl_seconds` | int | no | `30` | How long a transient failure (timeout, 5xx) is cached before retrying. |
| `cache_size` | int | no | `10485760` | Cache size in bytes. Minimum 524288. |
| `min_score` | float | no | `0.5` | Minimum confidence a category must have to be emitted. |
| `max_segments` | int | no | `0` | Maximum segments per taxonomy. `0` means unlimited. |
| `data_provider_name` | string | no | `zerogpu.ai` | Value written to the `name` field of each injected data object. |
| `enrich_content_1_0` | bool | no | `false` | Also emit IAB Content Taxonomy 1.0 codes under `ext.segtax: 1`. |
| `enrich_user_audience` | bool | no | `false` | Also emit IAB Audience Taxonomy 1.1 segments to `user.data` under `ext.segtax: 4`. See [Privacy](#privacy). |
| `account_filter.allow_list` | []string | no | `[]` | Account IDs permitted to use the module. Empty means all accounts. |

## Enrichment

The module runs at the `processed_auction_request` stage - the last point at
which the request is still shared by every bidder, after stored requests have
been merged. Running at `bidder_request` would warm the same domain once per
bidder.

### Cache warming

On each auction the module looks the domain up in its local cache:

* **Hit** - the segments are attached. No I/O.
* **Miss** - the auction returns unenriched, and a background warm-up fetches
  the classification and caches it for `cache_ttl_seconds` (24h by default).

Concurrent auctions for the same uncached domain collapse onto a single
outbound request, so a burst of traffic on a new domain does not produce a
burst of API calls.

Warm-ups deliberately use the module's own lifetime context rather than the
hook context. The hook context is cancelled the moment the execution plan's
group timeout elapses, which would abort the warm-up and leave the domain
permanently cold.

The practical cost of this design is that the first impressions on a
newly-seen domain go unenriched - roughly the duration of one classification
call. After that the domain is cached for 24 hours.

### Domain resolution

The domain is resolved from the first usable value of `site.domain`,
`site.page`, `site.publisher.domain`, `app.domain`, `app.bundle`,
`app.publisher.domain`, `dooh.domain`, `dooh.publisher.domain`. Values that
carry no domain signal - `localhost`, bare IP addresses, numeric iOS store IDs -
are skipped. If nothing resolves, the module does nothing.

The value is then normalized so that every spelling of a site shares one cache
entry: the scheme, path, query, port and any trailing dot are dropped, case is
folded, and a leading `www.`, `m.` or `amp.` is removed. So
`https://AMP.Example.com/article?x=1` and `example.com` are one entry, not two.

Other subdomains are preserved - `blog.example.com` is classified separately
from `example.com`, because they host different content. A variant prefix is
only stripped when at least two labels remain, so `amp.dev` stays `amp.dev`.

### Injected data

Given a classification for `coursera.com`, the module appends:

```json
{
  "site": {
    "content": {
      "data": [{
        "name": "zerogpu.ai",
        "ext": { "segtax": 6 },
        "segment": [{ "id": "132" }, { "id": "148" }]
      }]
    }
  }
}
```

Existing `content.data` entries are preserved. Enrichment is idempotent: if a
data object from the same provider already exists for a taxonomy, nothing is
appended.

Taxonomy identifiers follow the
[IAB segtax registry](https://github.com/InteractiveAdvertisingBureau/openrtb/blob/main/extensions/community_extensions/segtax.md):
`1` for Content Taxonomy 1.0, `4` for Audience Taxonomy 1.1, `6` for Content
Taxonomy 2.2.

## Privacy

The default configuration sends only a domain to ZeroGPU and writes only
contextual data. No user identifiers, device data, or geographic information
leave Prebid Server, and nothing is written to user-scoped ORTB fields.

`enrich_user_audience` is off by default and should stay off unless the host has
made a deliberate decision. When enabled, IAB Audience Taxonomy segments are
written to `user.data`. Two caveats:

1. The segments are inferred from the domain, not observed from the user.
   Publishing them under `user.data` presents contextual inference as
   audience data.
2. Prebid requires a module supplying user-level data to check the `enrichUfpd`
   [Activity Control](https://docs.prebid.org/prebid-server/features/pbs-activitycontrols.html).
   PBS-Go does not currently expose activity controls to modules, so the module
   cannot perform that check on the host's behalf.

The module does not create bids and does not add pixels to creatives.

## Analytics Tags

Each invocation emits one activity named
`zerogpu-rtd-domain-classification`:

| Outcome | Activity status | Result status | Values |
| --- | --- | --- | --- |
| Segments injected | `success` | `modify` | `domain`, `content_2_2_count`, `content_1_0_count`, `audience_count` |
| Nothing to inject | `success` | `allow` | `reason` |

Because classification happens off the auction path, a failed warm-up is not
attributable to any one auction. Warm-up failures are logged instead: `warn` for
a rejected domain, `error` for an authentication or quota problem, and `info`
for transient failures.

## Maintainer

<prebid@zerogpu.ai>
