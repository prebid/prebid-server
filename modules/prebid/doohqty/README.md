# DOOH Qty Module

This module enriches Digital Out-of-Home OpenRTB requests with `imp.qty` values before bidder requests are created. Prebid Server does not own the display registration system; each publisher/account can point the module at the value source it owns.

The module supports two source modes:

- `csv_snapshot`: fetch a full publisher CSV snapshot asynchronously and serve auctions from the last successful in-memory snapshot.
- `request_lookup`: call the existing request-time bulk lookup API for uncached lookup keys.

`csv_snapshot` is preferred when a publisher can expose all display values in one file because endpoint latency does not block the auction path. `request_lookup` is useful for sparse or highly dynamic values, but cache misses wait on the endpoint up to `timeout_ms`.

The returned `multiplier`, `sourcetype`, and `vendor` fields are written directly to OpenRTB `imp.qty.multiplier`, `imp.qty.sourcetype`, and `imp.qty.vendor`.

For the exact external endpoint contract, see [API_CONTRACT.md](API_CONTRACT.md).

## Host Configuration

```yaml
hooks:
  enabled: true
  modules:
    prebid:
      doohqty:
        enabled: true
        source:
          type: csv_snapshot
          sync_rate_seconds: 300
        lookup_paths:
          - dooh.id
        overwrite_policy: missing_only
        timeout_ms: 100
  host_execution_plan:
    endpoints:
      /openrtb2/auction:
        stages:
          processed_auction_request:
            groups:
              - timeout: 150
                hook_sequence:
                  - module_code: prebid.doohqty
                    hook_impl_code: dooh-qty
```

`lookup_paths` defaults to `["dooh.id"]` and supports `dooh.id`, `dooh.name`, `dooh.publisher.id`, `imp.id`, and `imp.tagid`. The first non-empty path is used for each impression.

`overwrite_policy` defaults to `missing_only`. Set it to `always` only when the configured source should override an existing `imp.qty`.

`cache_ttl_seconds`, `negative_cache_ttl_seconds`, and `cache_size_bytes` apply to `request_lookup` caching. `csv_snapshot` uses one in-memory snapshot per publisher/source endpoint and refreshes based on `source.sync_rate_seconds`.

The host config should usually set defaults and hook execution only. Add `source.endpoint` here only if every publisher should use the same fallback value source.

## Where Publisher Config Lives

Publisher-specific source config is standard PBS account config. This module does not load account files directly; PBS loads the account record, extracts the `hooks.modules.prebid.doohqty` block, and passes that block to this module during hook execution.

For local filesystem accounts, enable account loading in the host config:

```yaml
accounts:
  filesystem:
    enabled: true
    directorypath: ./stored_requests/data/by_id
```

A DOOH request with this publisher:

```json
{
  "dooh": {
    "publisher": {
      "id": "publisher-a"
    }
  }
}
```

loads this account file:

```text
stored_requests/data/by_id/accounts/publisher-a.json
```

If `dooh.publisher.ext.prebid.parentAccount` is present, PBS uses that value instead of `dooh.publisher.id`, so the account filename must match the parent account.

## Publisher Configuration

Account-level module config overlays the host config for that publisher/account. The account ID is the normal PBS account context: for DOOH, `dooh.publisher.ext.prebid.parentAccount` first, otherwise `dooh.publisher.id`.

When an account changes `source.type` or `source.endpoint`, inherited source headers are cleared unless the account provides its own `source.headers`.

Example publisher CSV source:

```json
{
  "hooks": {
    "modules": {
      "prebid": {
        "doohqty": {
          "source": {
            "type": "csv_snapshot",
            "endpoint": "https://publisher.example.com/prebid/dooh-qty.csv",
            "sync_rate_seconds": 600,
            "headers": {
              "Authorization": "Bearer publisher-token"
            }
          }
        }
      }
    }
  }
}
```

Example publisher request-time source:

```json
{
  "hooks": {
    "modules": {
      "prebid": {
        "doohqty": {
          "source": {
            "type": "request_lookup",
            "endpoint": "https://publisher.example.com/prebid/dooh-qty"
          },
          "cache_ttl_seconds": 300,
          "negative_cache_ttl_seconds": 30
        }
      }
    }
  }
}
```

## CSV Snapshot API

For `csv_snapshot`, the module sends an asynchronous `GET` to `source.endpoint`. The endpoint should return the complete current value set for that publisher:

```csv
path,key,multiplier,sourcetype,vendor
dooh.id,screen-123,14.2,1,measurement.example
imp.tagid,tag-456,8.5,2,
```

Rows with invalid multipliers, unsupported lookup paths, unsupported source types, or missing `vendor` when `sourcetype` is `1` are skipped. A successful sync replaces the full publisher snapshot; missing rows are removed on the next successful sync. Failed refreshes keep the last good snapshot.

Cold publishers do not block auctions. The first matching request starts the async CSV fetch and leaves the request unchanged until a snapshot is available.

## Request Lookup API

For `request_lookup`, the module sends one bulk `POST` per auction for uncached lookup keys:

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

Missing values and API errors leave the request unchanged and add hook warnings.
