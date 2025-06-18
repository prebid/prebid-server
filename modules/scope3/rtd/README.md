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
        endpoint: https://rtdp.scope3.com/amazonaps/rtii
        timeout_ms: 1000
        cache_ttl_seconds: 60   # Cache segments for 60 seconds (default)
        add_to_targeting: false # Set to true to add segments as individual targeting keys for GAM

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
          raw_auction_request:
            groups:
              - timeout: 2000
                hook_sequence:
                  - module_code: "scope3.rtd"
                    hook_impl_code: "HandleRawAuctionHook"
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
          "endpoint": "https://rtdp.scope3.com/amazonaps/rtii",
          "auth_key": "your-scope3-auth-key",
          "timeout_ms": 1000
        }
      }
    }
  }
}
```

## Environment Variables
- `SCOPE3_API_KEY`: Your Scope3 API key for authentication

## Features
- Fetches real-time audience segments from Scope3
- Adds segments to bid request targeting data
- Thread-safe segment caching to handle repeated requests
- Configurable timeout, endpoint, and cache TTL
- Graceful error handling (doesn't fail auctions on API errors)  
- Integration with various user identity systems (LiveRamp, publisher IDs, etc.)
- Efficient caching strategy for high-traffic scenarios

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
The module forwards the complete bid request with all available user identifiers to the Scope3 API. Scope3's system can then utilize whatever identifiers are available for audience segmentation, whether they are resolved identifiers or encrypted envelopes that Scope3 may be able to process.

**Note**: The effectiveness of different identifier types depends on Scope3's integration capabilities and partnerships.

## Data Output & Integration

### Auction Response Data
The module adds audience segments to the auction response, giving publishers full control over how to use them:

1. **Publisher Flexibility**: Segments are always returned in `ext.scope3.segments` for the publisher to decide where to send
2. **Google Ad Manager (GAM)**: Individual targeting keys are added when `add_to_targeting: true` (e.g., `gmp_eligible=true`)
3. **Other Ad Servers**: Publisher can forward segments to any ad server or system
4. **Analytics**: Segment data is available for reporting and analysis

### Response Format Options
The module provides segments in two formats:

**Always available:**
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
