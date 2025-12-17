# Traffic Shaping Module

The Traffic Shaping module allows publishers to dynamically control which bidders and ad sizes are allowed for specific placements based on a remote configuration. This enables fine-grained traffic management and optimization.

**Note**: This module uses the [`mile/common`](../common/README.md) package for device, geo, and browser resolution.

## Features

- **GPID-based Shaping**: Filter bidders and banner sizes per Global Placement ID (GPID)
- **Dynamic URL Construction**: Automatically construct config URLs based on device geo, type, and browser (uses [`mile/common`](../common/README.md))
- **Whitelist Pre-filtering**: Skip shaping for site/geo/platform combinations not in the whitelist
- **Skip Rate Gating**: Deterministically skip shaping for a percentage of auctions
- **Country Gating**: Apply shaping only for specific countries
- **User ID Vendor Filtering**: Optionally prune user.ext.eids to allowed vendors
- **Account-level Overrides**: Override module configuration per account
- **Fail-open Behavior**: On configuration fetch failure, auctions proceed normally
- **Multi-config Caching**: Cache multiple configs with TTL-based expiry

## Configuration Modes

The module supports two configuration modes:

### 1. Dynamic Mode (Recommended)

Constructs the config URL dynamically per request based on device characteristics using the [`mile/common`](../common/README.md) module:

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
        geo_lookup_endpoint: "http://geo-service.com/{ip}"  # optional, for IP fallback
        geo_cache_ttl_ms: 300000  # optional, default: 300000ms
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

**URL Construction**: `{base_endpoint}{siteID}/{country}/{device}/{browser}/ts.json`

**Example**: `https://example.com/ts-server/test-site/US/w/chrome/ts.json`

**Path Components** (resolved using [`mile/common`](../common/README.md)):
- `siteID`: From `site.id` (required)
- `country`: ISO 3166-1 alpha-2 code from `device.geo.country` (e.g., "US", "IN", "GB")
  - Fallback: IP-based geo lookup if `geo_lookup_endpoint` is configured
- `device`: Device category from `device.devicetype`:
  - `w` = Desktop/PC (devicetype=2)
  - `m` = Mobile/Phone (devicetype=1,4,6)
  - `t` = Tablet/TV (devicetype=3,5,7, or devicetype=1 with iPad/Tablet UA)
  - Fallback: Derived from SUA or UA string if devicetype is missing
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
- `geo_lookup_endpoint` (optional, dynamic mode): HTTP endpoint for IP-based geo lookup fallback (supports `{ip}` placeholder)
- `geo_cache_ttl_ms` (optional, default: 300000): TTL for geo lookup cache in milliseconds (minimum: 1000)
- `geo_whitelist_endpoint` (optional): URL to fetch geo whitelist JSON (must be configured with `platform_whitelist_endpoint`)
- `platform_whitelist_endpoint` (optional): URL to fetch platform whitelist JSON (must be configured with `geo_whitelist_endpoint`)
- `whitelist_refresh_ms` (optional, default: 300000): Whitelist refresh interval in milliseconds (minimum: 1000)

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

In dynamic mode, the module constructs the config URL from request data using the [`mile/common`](../common/README.md) resolver. If any required field is missing, shaping is skipped entirely (fail-open behavior):

**Required fields**:
- `site.id` (required for URL construction)
- `device.geo.country` (2-letter ISO code) OR IP address (if `geo_lookup_endpoint` configured)
- `device.devicetype` (non-zero value) OR derivable from SUA/UA
- `device.ua` (user agent string)

**Fallback behavior** (via [`mile/common`](../common/README.md)):
- Country: If `device.geo.country` is missing, attempts IP-based lookup if `geo_lookup_endpoint` is configured
- Device: If `device.devicetype` is 0 or missing, derives from SUA or UA string
- Browser: No fallback (must be present in UA)

**Fail-open scenarios**:
- Missing or empty `site.id`
- Missing or empty `device.geo.country` AND no `geo_lookup_endpoint` configured
- Missing or empty `device.geo.country` AND IP-based lookup fails
- Invalid country code (not 2 letters)
- Missing or zero `device.devicetype` AND cannot derive from SUA/UA
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
- `sample = fnv1a32(salt + request.id) % 100`
- If `sample < skipRate`, shaping is skipped for the entire auction

This ensures consistent behavior across multiple pods/instances.

### Whitelist Pre-filtering

When both `geo_whitelist_endpoint` and `platform_whitelist_endpoint` are configured, the module performs early filtering:

**Geo Whitelist Format** (`ts-geos.json`):
```json
{
  "siteID1": ["US", "CA"],
  "siteID2": ["GB", "DE"]
}
```

**Platform Whitelist Format** (`ts-platforms.json`):
```json
{
  "siteID1": ["m-android/chrome", "m-ios/safari", "w/chrome"],
  "siteID2": ["w/safari", "w/edge"]
}
```

**Platform Key Format**: `{device-os}/{browser}`
- Device types: `m-android`, `m-ios`, `t-android`, `t-ios`, `w`
- Browsers: `chrome`, `safari`, `ff`, `edge`, `opera`, `google search`, `samsung internet for android`, `amazon silk`

**Filtering Logic**:
1. If whitelists are not loaded (fetch failed), allow all requests (fail-open)
2. If site is not in either whitelist, allow traffic shaping (fail-open for unknown sites)
3. If site is in both whitelists, both geo AND platform must match for shaping to proceed
4. If either geo or platform doesn't match, shaping is skipped

**Refresh Behavior**:
- Whitelists are fetched on module startup
- Background refresh every 5 minutes (configurable via `whitelist_refresh_ms`)
- Exponential backoff on fetch failures

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
- `country_derived`: Country was resolved via IP fallback (from [`mile/common`](../common/README.md))
- `devicetype_derived`: Device category was derived from SUA/UA (from [`mile/common`](../common/README.md))

## Error Handling

- **Configuration Fetch Failure**: Shaping is skipped, and a warning is added to the hook result
- **Invalid GPID**: Impression is not shaped
- **All Bidders Filtered**: Impression is left unchanged (fail-open)
- **All Sizes Filtered**: Original sizes are preserved (fail-open)
- **Geo Resolution Failure**: Falls back to fail-open if IP-based lookup fails

## Performance

- Configuration is cached in memory and refreshed in the background
- No network calls in the hot path (auction processing)
- Lock-free reads using atomic pointers
- Geo lookups are cached with configurable TTL
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

### Dynamic Mode with Geo Fallback

```yaml
hooks:
  enabled: true
  modules:
    mile:
      trafficshaping:
        enabled: true
        base_endpoint: "https://cdn.example.com/ts-server/"
        geo_lookup_endpoint: "http://geo-service.com/{ip}"
        geo_cache_ttl_ms: 300000
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

## Architecture

The Traffic Shaping module leverages the [`mile/common`](../common/README.md) package for device, geo, and browser resolution:

```
┌─────────────────────────────────────┐
│   Traffic Shaping Module            │
│   (trafficshaping/)                 │
│                                     │
│   ┌─────────────────────────────┐   │
│   │  URL Builder               │   │
│   │  - Uses common.DefaultResolver│ │
│   │  - Constructs config URL    │   │
│   └─────────────────────────────┘   │
│            │                         │
│            ▼                         │
│   ┌─────────────────────────────┐   │
│   │  Config Client              │   │
│   │  - Fetches & caches configs │   │
│   └─────────────────────────────┘   │
└────────────┬────────────────────────┘
             │
             ▼
┌─────────────────────────────────────┐
│   Common Module (mile/common)       │
│   See: ../common/README.md          │
└─────────────────────────────────────┘
```

## Limitations

- Only banner size filtering is supported (video and native are not filtered)
- Bidder aliases must use the same keys as in `imp.ext.prebid.bidder`
- EID pruning is best-effort and conservative
- Geo fallback requires `geo_lookup_endpoint` to be configured

## Support

For issues or questions, please refer to the Prebid Server documentation or open an issue on GitHub.
