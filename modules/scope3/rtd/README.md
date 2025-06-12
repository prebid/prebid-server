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
