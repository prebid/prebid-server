# Scope3 TMP Module

This module integrates the AdCP **Trusted Match Protocol** (TMP) for cross-publisher
frequency capping with publisher-side privacy join. It calls a Scope3-operated TMP
router (e.g., `https://tmp.interchange.io`) and enriches the bid response with
eligible package IDs, the TMPX exposure token, and buyer-defined targeting
key-value pairs.

## Maintainer

- Email: bokelley@scope3.com
- Company: Scope3

## Spec references

- TMP overview: https://docs.adcontextprotocol.org/docs/trusted-match
- TMP specification: https://docs.adcontextprotocol.org/docs/trusted-match/specification
- Universal Macros (`{TMPX}`): https://docs.adcontextprotocol.org/docs/creative/universal-macros

## Configuration

### Module config (`pbs.yaml`)

```yaml
hooks:
  enabled: true
  modules:
    scope3:
      tmp:
        enabled: true
        router_url: https://tmp.interchange.io
        seller_agent_url: https://prebid.example.com/scope3
        auth_key: ${SCOPE3_TMP_AUTH_KEY}
        timeout_ms: 200
        cache_ttl_seconds: 60
        cache_size: 10485760
        add_to_targeting: false
        masking:
          enabled: true
          geo:
            preserve_metro: true
            preserve_zip: true
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
                  - module_code: "scope3.tmp"
                    hook_impl_code: "HandleEntrypointHook"
          processed_auction_request:
            groups:
              - timeout: 5
                hook_sequence:
                  - module_code: "scope3.tmp"
                    hook_impl_code: "HandleProcessedAuctionHook"
          auction_response:
            groups:
              - timeout: 250
                hook_sequence:
                  - module_code: "scope3.tmp"
                    hook_impl_code: "HandleAuctionResponseHook"
```

### Account config (per-publisher stored config)

```json
{
  "scope3": {
    "tmp": {
      "property_rid": "01916f3a-9c4e-7000-8000-000000000010",
      "property_type": "website",
      "placements": {
        "div-gpt-ad-header":  "header_728x90",
        "div-gpt-ad-sidebar": "sidebar_300x250",
        "div-gpt-ad-video":   "preroll_video"
      }
    }
  }
}
```

`property_rid` and `property_type` are required per-account. Without them the
module silently skips the auction (no enrichment). `seller_agent_url` defaults
to the module-level config; account config can override.

### Per-request override (testing only)

```json
{
  "ext": {
    "prebid": {
      "modules": {
        "scope3": {
          "tmp": {
            "property_rid": "01916f3a-...",
            "placement_id": "test_slot"
          }
        }
      }
    }
  }
}
```

## Output shape

```json
{
  "ext": {
    "scope3": { "tmp": { "tmpx": "k1.dG1weC1leGFtcGxl..." } }
  },
  "seatbid": [{
    "bid": [{
      "impid": "imp-header",
      "ext": {
        "scope3": {
          "tmp": {
            "placement_id": "header_728x90",
            "eligible_packages": ["pkg_abc"],
            "segments": ["news_intender"]
          }
        },
        "prebid": {
          "targeting": {
            "TMPX": "k1.dG1weC1leGFtcGxl...",
            "buyer_kv_key": "buyer_kv_value"
          }
        }
      }
    }]
  }]
}
```

`ext.scope3.tmp.tmpx` is auction-level (user-scoped, identical across all bids).
Per-bid `ext.scope3.tmp` carries the placement-specific enrichment.

When `add_to_targeting: true`, `TMPX` is set as a `prebid.targeting` key (matching
the AdCP Universal Macros convention) and `signals.targeting_kvs` from the buyer
flow verbatim into the same `prebid.targeting` namespace.

## Privacy

Field masking follows the same model as `modules/scope3/rtd/`. The TMP wire
guarantees an extra structural separation:

- Context Match contains page context but no user identity.
- Identity Match contains opaque user tokens but no page context.
- `package_ids` is omitted from Identity Match per spec (correlation prevention).
- Identity tokens are capped at 3 per the TMP spec hard limit (`maxItems: 3`).
- `request_id`s on the two calls are independent UUIDs.

## Multi-imp behavior

Each imp's placement is resolved from `account.scope3.tmp.placements[imp.tagid]`.
Unique placements deduplicate to one Context Match each. Identity Match is one
call per auction (page-context-free). Per-imp results are scoped onto each
`seatbid[].bid[]` via `bid.impid` → `imp.tagid` → `placement_id` lookup.

## Coexistence with scope3.rtd

If both `scope3.rtd` and `scope3.tmp` are enabled, the TMP module overwrites
the RTD module's writes to `ext.scope3.*`. Operators should configure the
execution plan with TMP listed last in the `auction_response` stage; the
overwrite is also defensive (independent of plan order).
