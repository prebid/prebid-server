# AdContextProtocol TMP Module

This module implements the [Trusted Match Protocol (TMP)](https://github.com/adcontextprotocol/adcp)
router role inside Prebid Server:

- It converts each incoming OpenRTB bid request into a TMP `context_match_request`
  and, when identity tokens are present, a TMP `identity_match_request`.
- It fans out to one or more TMP providers in parallel, signing every outbound
  call with Ed25519 (`X-AdCP-Signature`, `X-AdCP-Key-Id`) per the TMP spec.
- It joins each provider's context offers with its identity eligibility set
  locally and surfaces the surviving package IDs plus response-level signals on
  the bid response.

TMP wire types, signing and URL canonicalization come from
[`github.com/adcontextprotocol/adcp-go`](https://github.com/adcontextprotocol/adcp-go);
this module builds the OpenRTB→TMP mapping and the property registry client
on top.

## Configuration

```yaml
hooks:
  enabled: true
  modules:
    adcontextprotocol:
      tmp:
        enabled: true
        seller_agent_url: https://seller.example.com
        signing:
          key_id: kid-1
          # PEM (PKCS#8) Ed25519 private key. Substitute from environment in
          # your deployment YAML.
          private_key_pem: ${ADCP_TMP_SIGNING_KEY_PEM}
        property_registry:
          endpoint: https://agenticadvertising.org/api/properties/resolve
          auth_bearer: ${ADCP_REGISTRY_TOKEN}   # optional
          cache_ttl_seconds: 3600
          negative_cache_ttl_seconds: 300
          cache_size: 4096
          timeout_ms: 500
        providers:
          - name: example
            identity_url: https://tmp.example.com/identity
            context_url: https://tmp.example.com/context
            timeout_ms: 200
            # `tmpx_slots` mirrors the provider's registered
            # `tmpx_slots` list from adcp provider-registration.json.
            # Order is significant: the module enforces the ordered-
            # prefix invariant (adcp#5971) on incoming responses —
            # any provider whose emitted `tmpx_chunks[].slot_id`
            # sequence deviates (reordered, sparse, unregistered,
            # over-cap) has its chunks dropped atomically. Required
            # only when the provider emits TMPX. Capped at 2 per
            # adcp v1.
            tmpx_slots:
              - primary
              - secondary
        # Publisher-owned deployment configuration that resolves each
        # provider's ordered TMPX chunks (provider-local {slot_id,
        # value} pairs, per adcp publisher-tmpx-config.json) to local
        # ad-server macro names on this Prebid Server surface. Outer
        # key MUST match one of the `providers[].name` above (used as
        # `provider_id` in the adcp spec); inner key MUST be a
        # `slot_id` the provider declared in `tmpx_slots`; value is
        # the publisher-local destination (GAM key, VAST URL macro,
        # DOOH play-log field). Providers absent from this map emit
        # no TMPX targeting. Chunks with an unmapped slot cause the
        # whole provider's chunks to be dropped for that impression
        # (fail-closed).
        tmpx_macro_mapping:
          example:
            primary: TMPX_1
            secondary: TMPX_2
        timeout_ms: 300
        # Set to a positive value to jitter the second of a provider's context /
        # identity outbound calls by a random [0, N] ms, breaking timing
        # correlation at a passive observer. Order of the two calls is always
        # randomized regardless.
        decorrelation_max_delay_ms: 0
        targeting_key: adcp
        add_to_targeting: false
        # Caps on the segment set surfaced onto the response ext. Guards
        # against a misbehaving or hostile provider bloating the bid
        # response.
        max_segments: 128
        max_segment_value_len: 256
        # Masking gates optional finer-grained fields into the context
        # payload (zip / city / lat-long) and controls which EID sources
        # / mobile IDs flow into the identity payload. Defaults are
        # strict: only country / region / metro on the context path;
        # a small hardcoded EID whitelist on the identity path unless
        # `enabled: true` and `preserve_eids` narrows or widens it.
        masking:
          enabled: true
          geo:
            preserve_metro: true
            preserve_zip: false
            preserve_city: false
            lat_long_precision: 0
          user:
            preserve_eids:
              - liveramp.com
              - uidapi.com
              - id5-sync.com
          device:
            preserve_mobile_ids: false

  host_execution_plan:
    endpoints:
      /openrtb2/auction:
        stages:
          auction_processed:
            groups:
              - timeout: 500
                hook_sequence:
                  - module_code: "adcontextprotocol.tmp"
                    hook_impl_code: "HandleProcessedAuctionHook"
          auction_response:
            groups:
              - timeout: 500
                hook_sequence:
                  - module_code: "adcontextprotocol.tmp"
                    hook_impl_code: "HandleAuctionResponseHook"
```

### Required fields

| Field | Notes |
|-------|-------|
| `seller_agent_url` | Publicly reachable URL identifying this Prebid Server deployment as a seller agent. Must appear as one of `authorized_agents[].url` in the publisher's `adagents.json` (compared under AdCP URL canonicalization). |
| `signing.key_id` | Sent in `X-AdCP-Key-Id`. Verifiers use it to look up the matching Ed25519 public key. |
| `signing.private_key_pem` | PEM-encoded PKCS#8 Ed25519 private key. |
| `property_registry.endpoint` | Resolves `site.domain` / `app.bundle` → `property_rid` via a `GET ?domain=…` call. |
| `providers[].name` | Stable provider identifier (adcp `provider_id`). Appears verbatim in logs, metrics, and as the outer key of `tmpx_macro_mapping`. Charset matches the adcp spec: `^[A-Za-z0-9_]{1,64}$`. |
| `providers[].identity_url` or `providers[].context_url` | At least one is required per provider. |
| `providers[].tmpx_slots` | Optional. Ordered list of `slot_id`s the provider registered in adcp `provider-registration.json`. Required when the provider emits TMPX. The module drops any provider response whose emitted slot sequence is not a non-empty ordered prefix of this list. |
| `tmpx_macro_mapping` | Optional. Publisher-owned map of `provider_id → slot_id → ad-server macro name` used to route each provider's TMPX chunks. Omit to disable TMPX targeting. Missing entries for a provider's registered slots produce a startup warning; unmapped slots seen at serve time fail closed. |

### Providers

Each entry describes one downstream TMP provider. A provider may expose only
an identity endpoint, only a context endpoint, or both:

- If only `context_url` is set, no identity match is performed for that
  provider and all offers pass through unfiltered.
- If only `identity_url` is set, no offers are produced (eligibility with no
  context is not useful on its own — the module drops that combination).
- If both are set, offers are intersected with the identity eligibility set.

Providers are called in parallel; per-provider `timeout_ms` overrides the
module-level `timeout_ms`.

### Property registry

`site.domain` (or `app.bundle` when no site is present) is resolved to a
`property_rid` via the configured registry endpoint. Successful and negative
answers are cached in an in-memory LRU (`cache_size`, `cache_ttl_seconds`,
`negative_cache_ttl_seconds`). The first request from a cold domain may miss
its auction's timeout budget — that is expected; subsequent requests hit the
cache.

## Response surface

Merged targeting is written to the auction response `ext` under the configured
`targeting_key` (default `adcp`) as a flat list of `key=value` strings. Four
surfaces are covered per the adcp TMP spec:

- **Package IDs** eligible under identity, comma-joined under
  `package_targeting_key` (default `adcp_package_id`).
- **Response-level context signals** — the identity-agent-neutral
  `ContextMatchResponse.signals` map, one `key=value` per scalar entry.
- **Per-offer creative macros** — `Offer.macros` for offers that survived
  the identity eligibility gate.
- **Identity TMPX chunks** resolved through `tmpx_macro_mapping`. Providers
  emit `{slot_id, value}` pairs against their registered `tmpx_slots`; the
  publisher's mapping decides the ad-server destination for each pair on
  this surface. Chunks with unmapped slots are dropped atomically for that
  provider (fail-closed).

```json
{
  "ext": {
    "adcp": {
      "segments": [
        "adcp_package_id=pkg-fall-2026,pkg-holiday",
        "iab_cat=IAB1",
        "brand=Acme",
        "TMPX_1=opaque-chunk-value"
      ]
    }
  }
}
```

When `add_to_targeting: true`, each `key=value` pair is also mirrored into
`ext.prebid.targeting` so downstream ad servers (e.g. Google Ad Manager) can
consume them without a custom bridge.

## Privacy

- The TMP wire is decorrelated by design: context requests carry no identity
  tokens, identity requests carry no page context. This module never mixes the
  two payloads.
- Identity token count is capped at three, matching the TMP HPKE budget.
- Masking is applied to the context path (geo coarsening, EID allowlist)
  before requests leave the process. Identity requests never carry the masked
  fields to begin with.

## References

- TMP spec: [`adcontextprotocol/adcp`](https://github.com/adcontextprotocol/adcp) — `docs/trusted-match/specification.mdx`
- Go SDK: [`adcontextprotocol/adcp-go`](https://github.com/adcontextprotocol/adcp-go) — `tmproto`, `urlcanon`
- Property registry: [agenticadvertising.org](https://agenticadvertising.org)
