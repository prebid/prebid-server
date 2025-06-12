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

  host_execution_plan:
    endpoints:
      /openrtb2/auction:
        stages:
          entrypoint:
            groups:
              - timeout: 5
                hook_sequence:
                  - module_code: "scope3.rtd"
                    hook_impl_code: "scope3-entrypoint"
          raw_auction_request:
            groups:
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
- Thread-safe segment storage during auction
- Configurable timeout and endpoint
- Graceful error handling (doesn't fail auctions on API errors)
- Integration with LiveRamp ATS for enhanced targeting

## Dependencies
This module can work with LiveRamp ATS to enhance targeting with RampID data. If you're using LiveRamp ATS, ensure it runs before the Scope3 module in your execution plan to populate user identifiers.

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
The module automatically detects and includes user identifiers (such as RampID from LiveRamp ATS) when available in the bid request. User identifiers are typically found in:
- `user.ext.eids[]` array with `source: "liveramp.com"`
- `user.ext.rampid` field

The complete bid request with all available user identifiers is sent to the Scope3 API for enhanced audience segmentation.
