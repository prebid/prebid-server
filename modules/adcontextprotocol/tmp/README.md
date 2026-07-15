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
        timeout_ms: 300
        # Set to a positive value to jitter the second of a provider's context /
        # identity outbound calls by a random [0, N] ms, breaking timing
        # correlation at a passive observer. Order of the two calls is always
        # randomized regardless.
        decorrelation_max_delay_ms: 0
        targeting_key: adcp
        add_to_targeting: false
        masking:
          enabled: true
          geo:
            preserve_metro: true
            preserve_zip: false
            preserve_city: false
            lat_long_precision: 2
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
          entrypoint:
            groups:
              - timeout: 5
                hook_sequence:
                  - module_code: "adcontextprotocol.tmp"
                    hook_impl_code: "HandleEntrypointHook"
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
| `providers[].name` | Human-readable label; used as the prefix on emitted targeting keys. |
| `providers[].identity_url` or `providers[].context_url` | At least one is required per provider. |

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

Merged signals are written to the auction response `ext` under the configured
`targeting_key` (default `adcp`):

```json
{
  "ext": {
    "adcp": {
      "segments": [
        "example_package=pkg-fall-2026",
        "example_segment=auto_intender"
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
