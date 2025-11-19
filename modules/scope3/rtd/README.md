# Scope3 RTD Module

This module integrates Scope3's Real-Time Data API to provide audience segments for targeting.

## Maintainer
- Email: bokelley@scope3.com
- Company: Scope3

## Configuration

### YAML Configuration
```yaml
hooks:
  enabled: true
  modules:
    scope3:
      rtd:
        enabled: true
        auth_key: ${SCOPE3_API_KEY}  # Set SCOPE3_API_KEY environment variable
        endpoint: https://rtdp.scope3.com/prebid/prebid
        timeout_ms: 1000
        cache_ttl_seconds: 60               # Cache segments for 60 seconds (default)
        add_to_targeting: false             # Set to true to add segments as individual targeting keys for GAM
        add_scope3_targeting_section: false # Also set targeting in dedicated scope3 section
        masking:                            # Optional privacy masking configuration
          enabled: true                     # Enable field masking before sending to Scope3
          geo:
            preserve_metro: true      # Preserve DMA code (default: true)
            preserve_zip: true        # Preserve postal code (default: true)
            preserve_city: false      # Preserve city name (default: false)
            lat_long_precision: 2     # Lat/long decimal places: 0-4 (default: 2)
          user:
            preserve_eids:            # EID sources to preserve (default list below)
              - "liveramp.com"        # RampID
              - "uidapi.com"          # UID2
              - "id5-sync.com"        # ID5
          device:
            preserve_mobile_ids: false # Keep mobile advertising IDs (default: false)

  host_execution_plan:
    endpoints:
      /openrtb2/auction:
        stages:
          entrypoint:
            groups:
              - timeout: 5
                hook_sequence:
                  - module_code: "scope3.rtd"
                    hook_impl_code: "HandleEntrypointHook"
          auction_processed:
            groups:
              - timeout: 2000
                hook_sequence:
                  - module_code: "scope3.rtd"
                    hook_impl_code: "HandleAuctionProcessedHook"
          auction_response:
            groups:
              - timeout: 5
                hook_sequence:
                  - module_code: "scope3.rtd"
                    hook_impl_code: "HandleAuctionResponseHook"
```

### JSON Configuration
```json
{
  "hooks": {
    "modules": {
      "scope3": {
        "rtd": {
          "enabled": true,
          "endpoint": "https://rtdp.scope3.com/prebid/prebid",
          "auth_key": "your-scope3-auth-key",
          "timeout_ms": 1000,
          "cache_ttl_seconds": 60,
          "add_to_targeting": false,
          "add_scope3_targeting_section": false,
          "masking": {
            "enabled": true,
            "geo": {
              "preserve_metro": true,
              "preserve_zip": true,
              "preserve_city": false,
              "lat_long_precision": 2
            },
            "user": {
              "preserve_eids": ["liveramp.com", "uidapi.com", "id5-sync.com"]
            },
            "device": {
              "preserve_mobile_ids": false
            }
          }
        }
      }
    }
  }
}
```

## Environment Variables
- `SCOPE3_API_KEY`: Your Scope3 API key for authentication

## Privacy Protection

This module includes comprehensive privacy masking to protect user data while preserving targeting capabilities. When masking is enabled, sensitive user information is removed or anonymized before being sent to the Scope3 API.

### Field Masking Categories

#### 🟢 ALWAYS SENT (Never Masked)
These fields are always passed through without modification as they are not considered sensitive:

**Device Information**
- `device.devicetype` - Device type (mobile, desktop, CTV, etc.)
- `device.os` - Operating system
- `device.osv` - OS version
- `device.make` - Device manufacturer
- `device.model` - Device model
- `device.ua` - User agent string
- `device.language` - Language preference
- `device.connectiontype` - Connection type (wifi, cellular, etc.)
- `device.js` - JavaScript support
- `device.h/w` - Screen dimensions
- `device.ppi` - Screen PPI
- `device.pxratio` - Pixel ratio

**Geographic Information (Coarse)**
- `geo.country` - Country code (e.g., "US")
- `geo.region` - State/region code (e.g., "CA")

**Context Information**
- `site.*` - All site fields (domain, page, ref, etc.)
- `app.*` - All app fields
- `imp.*` - All impression data (ad sizes, positions, etc.)

#### 🔴 NEVER SENT (Always Masked)
These fields are always removed for privacy protection:

**Personal Identifiers**
- `device.ip` - IPv4 address
- `device.ipv6` - IPv6 address
- `user.id` - Publisher's first-party user ID
- `user.buyeruid` - Exchange-specific user ID
- `user.yob` - Year of birth
- `user.gender` - Gender
- `user.data` - First-party data segments
- `user.keywords` - User interest keywords

**High-Precision Location**
- `geo.accuracy` - GPS accuracy radius

#### 🔧 CONFIGURABLE (With Defaults)
These fields can be configured to be preserved or masked:

**Geographic Information (Fine-Grained)**

| Field | Default | Options | Privacy Impact |
|-------|---------|---------|----------------|
| `geo.metro` | **Preserved** | preserve/remove | DMA code for regional targeting |
| `geo.zip` | **Preserved** | preserve/remove | Postal code for local targeting |
| `geo.city` | **Removed** | preserve/remove | City name |
| `geo.lat/lon` | **Truncated to 2 decimals** | 0-4 decimals or remove | See precision guide below |

**User Identifiers**

| Field | Default | Options | Notes |
|-------|---------|---------|-------|
| `user.eids` | **Filter to allowlist** | List of EID sources | Default: ["liveramp.com", "uidapi.com", "id5-sync.com"] |

**Device Identifiers**

| Field | Default | Options | Notes |
|-------|---------|---------|-------|
| `device.ifa` | **Removed** | preserve/remove | Mobile advertising ID |
| `device.dpidmd5` | **Removed** | preserve/remove | Hashed device ID |
| `device.dpidsha1` | **Removed** | preserve/remove | Hashed device ID |
| Other device IDs | **Removed** | preserve/remove | Various hashed identifiers |

### Geographic Precision Guide

When preserving latitude/longitude coordinates, the precision level has significant privacy implications:

| Decimals | Accuracy | Privacy Level | Use Case |
|----------|----------|---------------|----------|
| 0 | Removed | Maximum | No location data |
| 1 | ~11 km | Country/State | Regional campaigns |
| 2 | ~1.1 km | **Neighborhood (Default)** | Store radius matching |
| 3 | ~111 m | City block | Dense urban targeting |
| 4 | ~11 m | Building | Maximum allowed |

⚠️ **WARNING**: More than 4 decimal places can identify individuals and is not permitted by this module.

### Configuration Examples

#### Maximum Privacy (Strict Mode)
```yaml
masking:
  enabled: true
  geo:
    preserve_metro: false
    preserve_zip: false
    preserve_city: false
    lat_long_precision: 0
  user:
    preserve_eids: []
  device:
    preserve_mobile_ids: false
```

#### Balanced Privacy (Default)
```yaml
masking:
  enabled: true
  geo:
    preserve_metro: true
    preserve_zip: true
    preserve_city: false
    lat_long_precision: 2
  user:
    preserve_eids: ["liveramp.com", "uidapi.com", "id5-sync.com"]
  device:
    preserve_mobile_ids: false
```

#### Retail/Commerce Optimization
```yaml
masking:
  enabled: true
  geo:
    preserve_metro: true
    preserve_zip: true
    preserve_city: true
    lat_long_precision: 3  # Higher precision for store matching
  user:
    preserve_eids: ["liveramp.com", "retail-partner.com"]
  device:
    preserve_mobile_ids: true  # For in-store attribution
```

### Privacy Compliance

**GDPR Compliance**
- Removes direct identifiers by default
- Configurable to meet legitimate interest requirements
- Supports privacy-by-design principles

**CCPA Compliance**
- Removes sale-related identifiers
- Supports consumer privacy rights
- Configurable per jurisdiction requirements

**Industry Standards**
- Follows IAB OpenRTB privacy guidelines
- Compatible with TCF 2.0 consent frameworks
- Supports Privacy Sandbox initiatives

## Features
- **Privacy-First Design**: Comprehensive field masking to protect user data while preserving targeting
- **Flexible Masking**: Configurable privacy controls for different use cases and compliance requirements
- **Real-Time Segments**: Fetches audience segments from Scope3 API with intelligent caching
- **Thread-Safe Caching**: Handles high-frequency requests with concurrent access protection
- **Identity Integration**: Supports multiple identity providers (LiveRamp, UID2, ID5, etc.)
- **Geographic Privacy**: Truncates location data to configurable precision levels
- **Graceful Degradation**: Continues auction processing even when API errors occur
- **Performance Optimized**: HTTP/2, connection pooling, and compression for high-traffic scenarios

## Performance & Caching
This module implements intelligent caching and HTTP optimizations to handle high-frequency API requests:

- **Cache Key**: Generated from user identifiers, site domain, and page URL
- **Cache Duration**: Configurable via `cache_ttl_seconds` (default: 60 seconds)
- **Thread Safety**: Uses read-write mutexes for concurrent access
- **Memory Efficiency**: Stores only segment arrays, not full API responses
- **Frequency Cap Compatibility**: Short 60-second default ensures frequency-capped segments are refreshed quickly
- **HTTP Optimization**: Custom transport with connection pooling, HTTP/2, and compression for better performance

## User Identity Integration
This module automatically detects and forwards available user identifiers from the bid request including:

- LiveRamp identifiers (when available from publisher implementations or identity providers)
- Publisher first-party user IDs
- Device identifiers
- Encrypted identity envelopes


### Supported Identifier Types
1. **LiveRamp Identifiers** (when available):
   - `user.ext.eids[]` array with `source: "liveramp.com"`
   - `user.ext.rampid` field (alternative location)

2. **Encrypted Identity Envelopes**:
   - `user.ext.liveramp_idl` - ATS envelope location
   - `user.ext.ats_envelope` - Alternative envelope location
   - `user.ext.rampId_envelope` - Additional envelope location
   - `ext.liveramp_idl` - Request-level envelope

3. **Standard Identifiers**:
   - `user.id` - Publisher user ID
   - `device.ifa` - Device identifier
   - Other standard OpenRTB identifiers

### How It Works
The module forwards any fields that are not masked from the bid request to the Scope3 API (please see the [Privacy Protection](#privacy-protection) section for more details). Scope3's system can then utilize whatever identifiers are available for audience segmentation, whether they are resolved identifiers or encrypted envelopes that Scope3 may be able to process.

**Note**: The effectiveness of different identifier types depends on Scope3's integration capabilities and partnerships.

## Data Output & Integration

### Auction Response Data
The module adds audience segments to the auction response, giving publishers full control over how to use them:

1. **Publisher Flexibility**: Segments are returned in `ext.scope3.segments` when configured for the publisher to decide where to send
2. **Google Ad Manager (GAM)**: Individual targeting keys are added when `add_to_targeting: true` (e.g., `gmp_eligible=true`)
3. **Other Ad Servers**: Publisher can forward segments to any ad server or system
4. **Analytics**: Segment data is available for reporting and analysis

### Response Format Options
The module provides segments in two formats:

**When `add_scope3_targeting_section: true`:**
```json
{
  "ext": {
    "scope3": {
      "segments": ["gmp_eligible", "gmp_plus_eligible"]
    }
  }
}
```

**When `add_to_targeting: true`:**
```json
{
  "ext": {
    "prebid": {
      "targeting": {
        "gmp_eligible": "true",
        "gmp_plus_eligible": "true"
      }
    },
    "scope3": {
      "segments": ["gmp_eligible", "gmp_plus_eligible"]
    }
  }
}
```

This approach gives publishers maximum flexibility to:
- Send segments to GAM via targeting keys
- Forward to other ad servers or systems
- Use for analytics and reporting
- Control which segments go where based on their business logic
