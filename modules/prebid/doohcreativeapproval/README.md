# DOOH Creative Approval

`prebid.doohcreativeapproval` lets a publisher approve DOOH creatives before they can compete in an auction. The module runs only for DOOH requests and fails closed: a non-exempt bid is allowed through only when its creative approval status is `approved`.

PBS is not the durable approval system. It caches statuses in process to reduce endpoint calls and avoid repeated lookups after failures. The publisher approval service remains the source of truth.

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

Account config can override `enabled`, `platforms`, `endpoint`, `headers`, `timeout_ms`, status refresh TTLs, and `exempt_bidders`. `cache_size_bytes` is host-level because the cache is shared by the module instance.

## Behavior

- Exempt bidders bypass approval and do not call the publisher endpoint.
- `approved` creatives pass.
- `rejected`, `pending`, missing, malformed, timed-out, or unknown creatives are removed.
- When a cached status is due for refresh, PBS asks the publisher endpoint for a fresh status.
- If the refresh request fails and PBS still has a cached status, PBS keeps using that cached status and schedules another refresh attempt.
- If there is no cached status and the publisher endpoint cannot return a usable response, PBS treats the creative as `pending`.
- If PBS cannot write an approval status to the in-process cache, the current bid decision still uses the fresh endpoint response, but the hook returns a warning and the creative may be looked up again on a future request.

## Cache Behavior

The `*_ttl_seconds` settings control when a cached status is due for refresh. They do not delete the last-known status. If a refresh request fails because the publisher endpoint is down, timed out, or returned an unusable response, PBS continues using the last cached status.

`cache_size_bytes` is a memory cap, not a guarantee that every cached creative remains resident. If the cache reaches capacity, entries can be evicted. An evicted entry is treated as unknown: the next matching bid triggers a publisher endpoint lookup, and the bid is suppressed unless PBS gets a usable status.

PBS does not expose an admin or inspection API for this cache. The approval endpoint should keep the durable approval records.

## Limitations

- v1 supports only `platforms: ["dooh"]`. Site and app requests are intentionally ignored.
- v1 assumes `account_id + bidder + bid.crid` identifies the creative approval unit. It does not hash ad markup, media files, or preview URLs.
- Approval changes are picked up through refresh attempts, cache misses, or cache eviction, not through a push channel into PBS.
- Endpoint failures and malformed responses fail closed for creatives without cached status. Creatives with cached status keep using that status until a usable refresh succeeds or the cache entry is evicted.
- Fresh `pending`, `rejected`, and missing statuses fail closed.

See `API_CONTRACT.md` for the external approval API.
