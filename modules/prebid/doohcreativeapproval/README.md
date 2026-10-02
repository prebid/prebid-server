# DOOH Creative Approval

`prebid.doohcreativeapproval` lets a publisher approve DOOH creatives before they can compete in an auction. The module runs only for DOOH requests. Once the module is active, a non-exempt bid is allowed through only when its last-known creative approval status is `approved`.

PBS is not the durable approval system. Each PBS process caches statuses locally and refreshes them in the background. The publisher approval service remains the source of truth.

## Terms And Scope

In this module, `account` means the PBS account/config scope. Account config controls the approval endpoint, status refresh TTLs, and exempt bidders. `publisher` means the business system or screen owner that reviews creatives. These are often the same operational boundary, but PBS does not require them to be the same identifier.

Creative approval state is scoped by PBS account, bidder, and `bid.crid`:

```text
creative_approval_id = "v1:" + sha256(account_id + "\x1f" + bidder + "\x1f" + bid.crid)
```

If the module is run without a PBS account, `account_id` is empty. Prefer account-level configuration when approvals need to be separated by publisher, tenant, or business owner.

## Hook Setup

The module must run in both stages:

```yaml
hooks:
  enabled: true
  modules:
    prebid:
      doohcreativeapproval:
        enabled: true
        platforms:
          - dooh
        timeout_ms: 100
        cache_size_bytes: 10485760
        max_concurrent_lookups: 8
        approved_ttl_seconds: 3600
        rejected_ttl_seconds: 300
        pending_ttl_seconds: 60
  host_execution_plan:
    endpoints:
      /openrtb2/auction:
        stages:
          processed_auction_request:
            groups:
              - timeout: 100
                hook_sequence:
                  - module_code: prebid.doohcreativeapproval
                    hook_impl_code: dooh-creative-approval
          all_processed_bid_responses:
            groups:
              - timeout: 100
                hook_sequence:
                  - module_code: prebid.doohcreativeapproval
                    hook_impl_code: dooh-creative-approval
```

The processed-auction hook only marks eligible DOOH auctions as active. The all-processed-bid-responses hook does the filtering. If the processed stage is omitted, the module intentionally does nothing at the filtering stage.

## Account Config

Publisher-specific endpoint config should live in account config:

```json
{
  "hooks": {
    "modules": {
      "prebid": {
        "doohcreativeapproval": {
          "endpoint": "https://publisher.example.com/creative-approval",
          "headers": {
            "Authorization": "Bearer token"
          },
          "exempt_bidders": ["house"]
        }
      }
    }
  }
}
```

Account config can override `enabled`, `platforms`, `endpoint`, `headers`, `timeout_ms`, status refresh TTLs, and `exempt_bidders`. `cache_size_bytes` and `max_concurrent_lookups` are host-level because the cache and refresh limit are shared by the module instance.

`timeout_ms` bounds each background HTTP request. It does not extend the auction or hook execution timeout.

## Behavior

- Exempt bidders bypass approval and do not call the publisher endpoint.
- A first-seen creative is treated as `pending` and removed from the current auction. PBS starts a background lookup for later auctions.
- Cached `approved` creatives pass. Cached `rejected` and `pending` creatives are removed.
- When a cached status is due for refresh, PBS keeps using that status while refreshing it in the background.
- Endpoint errors, timeouts, malformed responses, missing entries, unknown statuses, and duplicate entries do not replace an existing status. PBS retries after `pending_ttl_seconds`.
- If no prior status exists and the endpoint cannot return a usable status, the creative remains `pending`.
- Refreshes for the same creative are coalesced. At most `max_concurrent_lookups` bulk requests run in one PBS process.

## Cache Behavior

The `*_ttl_seconds` settings control when a cached status is due for refresh. They do not delete the last-known status. Refreshes happen outside the auction path, and an unusable refresh leaves the current status unchanged.

`cache_size_bytes` is a memory cap, not a guarantee that every cached creative remains resident. It must be at least 524288 bytes. If the cache reaches capacity, entries can be evicted. An evicted entry is treated as unknown, so its next bid is suppressed while PBS refreshes it.

PBS does not expose an admin or inspection API for this cache. The approval endpoint should keep the durable approval records.

## Limitations

- v1 supports only `platforms: ["dooh"]`. Site and app requests are intentionally ignored.
- v1 assumes `account_id + bidder + bid.crid` identifies the creative approval unit. It does not hash ad markup, media files, or preview URLs.
- Approval changes are picked up through background refreshes, cache misses, or cache eviction, not through a push channel into PBS.
- Cache contents and refresh work are local to each PBS process. Multiple PBS instances can refresh the same creative independently.
- A missing endpoint leaves the module inactive for that account. Invalid account config or a PBS hook execution failure can prevent filtering; these are configuration or host-execution failures rather than approval lookup results.
- The first auction for an uncached creative is always suppressed, even if the publisher endpoint would immediately approve it.

See `API_CONTRACT.md` for the external approval API.
