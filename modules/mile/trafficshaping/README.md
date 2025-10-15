# Traffic Shaping Module

The Traffic Shaping module allows publishers to dynamically control which bidders and ad sizes are allowed for specific placements based on a remote configuration. This enables fine-grained traffic management and optimization.

## Features

- **GPID-based Shaping**: Filter bidders and banner sizes per Global Placement ID (GPID)
- **Dynamic URL Construction**: Automatically construct config URLs based on device geo, type, and browser
- **Skip Rate Gating**: Deterministically skip shaping for a percentage of auctions
- **Country Gating**: Apply shaping only for specific countries
- **User ID Vendor Filtering**: Optionally prune user.ext.eids to allowed vendors
- **Account-level Overrides**: Override module configuration per account
- **Fail-open Behavior**: On configuration fetch failure, auctions proceed normally
- **Multi-config Caching**: Cache multiple configs with TTL-based expiry

## Configuration Modes

The module supports two configuration modes:

### 1. Dynamic Mode (Recommended)

Constructs the config URL dynamically per request based on device characteristics:

```yaml
hooks:
  enabled: true
  modules:
    mile:
      trafficshaping:
        enabled: true
        base_endpoint: "https://example.com/ts-server/"
        refresh_ms: 30000
        request_timeout_ms: 1000
        prune_user_ids: false
        sample_salt: "pbs"
  default_account_execution_plan:
    endpoints:
      /openrtb2/auction:
        stages:
          processed_auction_request:
            groups:
              - timeout: 50
                hook_sequence:
                  - module_code: mile.trafficshaping
                    hook_impl_code: default
```

**URL Construction**: `{base_endpoint}{country}/{device}/{browser}/ts.json`

**Example**: `https://example.com/ts-server/US/w/chrome/ts.json`

**Path Components**:
- `country`: ISO 3166-1 alpha-2 code from `device.geo.country` (e.g., "US", "IN", "GB")
- `device`: Device category from `device.devicetype`:
  - `w` = Desktop/PC (devicetype=2)
  - `m` = Mobile/Phone (devicetype=1,4,6)
  - `t` = Tablet/TV (devicetype=3,5,7, or devicetype=1 with iPad/Tablet UA)
- `browser`: Browser detected from `device.ua`:
  - `chrome` = Chrome (includes CriOS for iOS)
  - `safari` = Safari
  - `ff` = Firefox (includes FxiOS for iOS)
  - `edge` = Edge (Chromium or Legacy)
  - `opera` = Opera
  - Defaults to `chrome` for unknown browsers

**Caching**: Each unique URL is cached with TTL (refresh_ms). Configs are fetched on-demand.

### 2. Static Mode (Legacy)

Uses a single static endpoint with background refresh:

```yaml
hooks:
  enabled: true
  modules:
    mile:
      trafficshaping:
        enabled: true
        endpoint: "https://example.com/traffic-shaping-config.json"
        refresh_ms: 30000
        request_timeout_ms: 1000
        prune_user_ids: false
        sample_salt: "pbs"
        allowed_countries: ["US", "CA"]  # optional
  default_account_execution_plan:
    endpoints:
      /openrtb2/auction:
        stages:
          processed_auction_request:
            groups:
              - timeout: 50
                hook_sequence:
                  - module_code: mile.trafficshaping
                    hook_impl_code: default
```

### Configuration Parameters

- `enabled` (required): Enable/disable the module
- `base_endpoint` (dynamic mode): Base URL for dynamic config fetching (must end with `/`)
- `endpoint` (static mode): URL of the remote traffic shaping configuration
- `refresh_ms` (optional, default: 30000): Configuration refresh interval (static mode) or cache TTL (dynamic mode) in milliseconds (minimum: 1000)
- `request_timeout_ms` (optional, default: 1000): HTTP request timeout in milliseconds (minimum: 100)
- `prune_user_ids` (optional, default: false): Enable user ID vendor filtering
- `sample_salt` (optional, default: "pbs"): Salt for deterministic sampling
- `allowed_countries` (optional, static mode only): List of allowed countries for shaping (ISO 3166-1 alpha-2 codes)

### Account-level Configuration

Account-specific overrides can be configured via stored requests:

```json
{
  "hooks": {
    "modules": {
      "mile": {
        "trafficshaping": {
          "endpoint": "https://account-specific.example.com/config.json",
          "prune_user_ids": true,
          "allowed_countries": ["US"]
        }
      }
    }
  }
}
```

## Remote Configuration Format

The module fetches a JSON configuration from the specified endpoint:

```json
{
  "meta": {
    "createdAt": 1234567890
  },
  "response": {
    "schema": {
      "fields": ["gpID"]
    },
    "skipRate": 10,
    "userIdVendors": ["uid2", "pubcid", "tdid"],
    "values": {
      "placement-123": {
        "rubicon": {
          "300x250": 1,
          "728x90": 1
        },
        "appnexus": {
          "300x250": 1
        }
      },
      "placement-456": {
        "pubmatic": {
          "320x50": 1
        }
      }
    }
  }
}
```

### Configuration Fields

- `skipRate`: Percentage (0-100) of auctions to skip shaping
- `userIdVendors`: List of allowed user ID vendors (when `prune_user_ids` is enabled)
- `values`: Map of GPID to allowed bidders and sizes
  - Key: GPID (from `imp.ext.gpid` or fallback to `imp.ext.data.adserver.adslot`)
  - Value: Map of bidder names to allowed sizes
    - Size format: "WxH" (e.g., "300x250")

## Behavior

### Dynamic Mode URL Construction (Fail-Open)

In dynamic mode, the module constructs the config URL from request data. If any required field is missing, shaping is skipped entirely (fail-open behavior):

**Required fields**:
- `device.geo.country` (2-letter ISO code)
- `device.devicetype` (non-zero value)
- `device.ua` (user agent string)

**Fail-open scenarios**:
- Missing or empty `device.geo.country`
- Invalid country code (not 2 letters)
- Missing or zero `device.devicetype`
- Missing or empty `device.ua`
- Config fetch fails for the constructed URL

When shaping is skipped, the auction proceeds normally with all configured bidders.

### GPID Resolution

The module attempts to extract GPID in the following order:
1. `imp.ext.gpid`
2. `imp.ext.data.adserver.adslot` (fallback)

If no GPID is found, the impression is not shaped.

### Skip Rate

The module uses deterministic sampling based on the request ID:
- `sample = fnv1a32(hex(salt + request.id)) % 100`
- If `sample < skipRate`, shaping is skipped for the entire auction

This ensures consistent behavior across multiple pods/instances.

### Country Gating (Static Mode Only)

When `allowed_countries` is configured in static mode:
- Shaping is applied only if `device.geo.country` is in the allowed list
- If country is missing or not allowed, shaping is skipped (fail-open)

### Bidder Filtering

For each impression with a matching GPID:
- Only bidders present in the configuration are allowed
- Bidders not in the allowlist are filtered out via the mutation API

### Banner Size Filtering

For banner impressions:
- If `imp.banner.format` is present, only allowed sizes are kept
- If all sizes would be filtered out, the original formats are preserved (fail-open)
- If `imp.banner.w` and `imp.banner.h` are set but not allowed, they are converted to allowed formats

### User ID Vendor Filtering

When `prune_user_ids` is enabled:
- `user.ext.eids` are filtered to only allowed vendors from `userIdVendors`
- Matching is conservative (substring matching on source)
- Ambiguous sources are kept (fail-open)

Vendor mappings:
- `uid2` → `uidapi.com`
- `pubcid` → `pubcid.org`
- `tdid` → `adserver.org` (checks `rtiPartner=TDID`)
- `criteoId` → `criteo.com`
- And more...

## Analytics Tags

The module emits the following analytics activities:

- `applied`: Shaping was successfully applied
- `shaped`: Alias for applied (for easier downstream reporting)
- `skipped_by_skiprate`: Skipped due to skipRate sampling
- `skipped_no_config`: Skipped because configuration is unavailable
- `fetch_failed`: Remote configuration fetch failed/unavailable
- `skipped_country`: Skipped due to country gating
- `missing_gpid`: One or more impressions had no GPID
- `skipped`: General skip marker added alongside specific skip reason

## Error Handling

- **Configuration Fetch Failure**: Shaping is skipped, and a warning is added to the hook result
- **Invalid GPID**: Impression is not shaped
- **All Bidders Filtered**: Impression is left unchanged (fail-open)
- **All Sizes Filtered**: Original sizes are preserved (fail-open)

## Performance

- Configuration is cached in memory and refreshed in the background
- No network calls in the hot path (auction processing)
- Lock-free reads using atomic pointers
- Typical overhead: < 50µs per impression

## Testing

Run tests with:

```bash
go test ./modules/mile/trafficshaping/...
```

Run with race detection:

```bash
./validate.sh --race 1
```

## Examples

### Basic Setup

```yaml
hooks:
  enabled: true
  modules:
    mile:
      trafficshaping:
        enabled: true
        endpoint: "https://cdn.example.com/shaping.json"
```

### With Country Gating

```yaml
hooks:
  enabled: true
  modules:
    mile:
      trafficshaping:
        enabled: true
        endpoint: "https://cdn.example.com/shaping.json"
        allowed_countries: ["US", "CA", "GB"]
```

### With User ID Filtering

```yaml
hooks:
  enabled: true
  modules:
    mile:
      trafficshaping:
        enabled: true
        endpoint: "https://cdn.example.com/shaping.json"
        prune_user_ids: true
```

## Limitations

- Only banner size filtering is supported (video and native are not filtered)
- Bidder aliases must use the same keys as in `imp.ext.prebid.bidder`
- EID pruning is best-effort and conservative

## Support

For issues or questions, please refer to the Prebid Server documentation or open an issue on GitHub.

