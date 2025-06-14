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
        bid_meta_data: false    # Set to true to include segments in bid.meta (future enhancement)

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
          processed_auction_request:
            groups:
              - timeout: 5
                hook_sequence:
                  - module_code: "scope3.rtd"
                    hook_impl_code: "HandleProcessedAuctionHook"
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
This module implements intelligent caching to handle scenarios with hundreds of identical requests per user session:

- **Cache Key**: Generated from user identifiers, site domain, and page URL
- **Cache Duration**: Configurable via `cache_ttl_seconds` (default: 60 seconds)
- **Thread Safety**: Uses read-write mutexes for concurrent access
- **Memory Efficiency**: Stores only segment arrays, not full API responses
- **Frequency Cap Compatibility**: Short 60-second default ensures frequency-capped segments are refreshed quickly

## Dependencies
This module can integrate with various user identity systems. It automatically detects and forwards available user identifiers including:

- LiveRamp identifiers (when available from other sources)
- Publisher first-party user IDs
- Device identifiers
- Encrypted identity envelopes

### Integration with LiveRamp ATS
When using both LiveRamp ATS and Scope3 RTD modules, configure execution order so LiveRamp runs first:

```yaml
host_execution_plan:
  endpoints:
    /openrtb2/auction:
      stages:
        raw_auction_request:
          groups:
            # Group 1: LiveRamp ATS runs first to populate RampID
            - timeout: 1000
              hook_sequence:
                - module_code: "liveramp.ats"
                  hook_impl_code: "liveramp-fetch"
            # Group 2: Scope3 runs after LiveRamp, can use RampID
            - timeout: 2000
              hook_sequence:
                - module_code: "scope3.rtd"
                  hook_impl_code: "scope3-fetch"
        processed_auction_request:
          groups:
            - timeout: 5
              hook_sequence:
                - module_code: "scope3.rtd"
                  hook_impl_code: "scope3-targeting"
```

Alternative configuration (same group, sequential execution):
```yaml
raw_auction_request:
  groups:
    - timeout: 3000
      hook_sequence:
        # LiveRamp runs first
        - module_code: "liveramp.ats"
          hook_impl_code: "liveramp-fetch"
        # Scope3 runs second, can access RampID
        - module_code: "scope3.rtd"
          hook_impl_code: "scope3-fetch"
```

## User Identifier Integration
The module automatically detects and includes user identifiers when available in the bid request. This includes support for:

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

### Targeting Data Approach
The module adds audience segments as targeting data in the bid request, making them available to:

1. **Google Ad Manager (GAM)**: Segments appear as `hb_scope3_segments` targeting key
2. **Bidders**: Segments are available in the request context for bid decisioning
3. **Analytics**: Targeting data is logged for reporting purposes

### Targeting vs Bid.Meta
This implementation uses **targeting data** rather than **bid.meta** for the following reasons:

- **RTD Module Pattern**: Prebid Server RTD modules typically operate on requests, not responses
- **Universal Availability**: Targeting data is available to all bidders and ad servers
- **Performance**: No need to process individual bid responses
- **Flexibility**: Publishers can choose where to send the targeting data

The current hook architecture processes auction requests rather than individual bid responses, making targeting data the appropriate integration method for RTD modules.

### Example Targeting Output
```json
{
  "ext": {
    "targeting": {
      "hb_scope3_segments": "gmp_eligible,gmp_plus_eligible"
    }
  }
}
```
