# DOOH Qty API Contract

This document defines the external value-source contracts used by `prebid.doohqty`.

The module supports two source modes:

- `csv_snapshot`: Prebid Server fetches a complete publisher/account value snapshot with `GET`.
- `request_lookup`: Prebid Server sends requested lookup keys with `POST` and receives matching values.

## Common Fields

| Field | Required | Description |
| --- | --- | --- |
| `path` | Yes | Lookup namespace. Supported values: `dooh.id`, `dooh.name`, `dooh.publisher.id`, `imp.id`, `imp.tagid`. |
| `key` | Yes | Lookup value from the OpenRTB request for the selected `path`. |
| `multiplier` | Yes | Positive finite number written to `imp.qty.multiplier`. |
| `sourcetype` | No | OpenRTB DOOH multiplier source type. Omitted or `0` means unknown. |
| `vendor` | Required when `sourcetype` is `1` | Vendor name written to `imp.qty.vendor`. |

Invalid values are skipped. The auction request is left unchanged when no valid value is available.

## CSV Snapshot Source

Use `source.type: csv_snapshot` when a publisher can expose all display values in one response.

### Request

```http
GET {source.endpoint}
Accept: text/csv
```

Configured `source.headers` are sent with the request.

### Response

Return `2xx` with CSV content. The CSV must include a header row.

Required columns:

- `path`
- `key`
- `multiplier`

Optional columns:

- `sourcetype`
- `vendor`

Example:

```csv
path,key,multiplier,sourcetype,vendor
dooh.id,screen-123,14.2,1,measurement.example
imp.tagid,tag-456,8.5,2,
```

### Semantics

- A successful fetch replaces the complete in-memory snapshot for that publisher/account and source endpoint.
- Rows missing from a later successful CSV response are removed from the active snapshot.
- CSV refresh happens asynchronously. The auction path does not wait for the endpoint.
- Cold cache requests are left unchanged while the first snapshot loads.
- Failed refreshes keep using the last successful snapshot.
- Duplicate `path,key` rows keep the first valid row and skip later duplicates with a warning.

## Request Lookup Source

Use `source.type: request_lookup` when values are sparse, highly dynamic, or only available through a request-time lookup API.

### Request

```http
POST {source.endpoint}
Content-Type: application/json
Accept: application/json
```

Configured `source.headers` are sent with the request.

Body:

```json
{
  "account_id": "acct",
  "lookups": [
    {
      "path": "dooh.id",
      "key": "screen-123"
    }
  ]
}
```

### Response

Return `2xx` with JSON content:

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

### Semantics

- Response values must match a requested `path,key`; unrequested values are skipped.
- Missing requested values are treated as misses.
- Valid hits are cached for `cache_ttl_seconds`.
- Misses and invalid values are cached for `negative_cache_ttl_seconds`.
- Cache misses wait on the endpoint up to `timeout_ms`.
- Non-2xx responses, timeouts, and invalid JSON leave the auction request unchanged and add hook warnings.

## Account And Publisher Scope

The module receives the normal PBS account ID for the request. For DOOH traffic, PBS resolves account ID from:

1. `dooh.publisher.ext.prebid.parentAccount`
2. `dooh.publisher.id`

Publisher-specific source configuration should be placed in that account's PBS account config under:

```json
{
  "hooks": {
    "modules": {
      "prebid": {
        "doohqty": {
          "source": {
            "type": "csv_snapshot",
            "endpoint": "https://publisher.example.com/prebid/dooh-qty.csv"
          }
        }
      }
    }
  }
}
```
