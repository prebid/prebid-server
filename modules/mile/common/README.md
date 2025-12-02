# Common Module (`mile/common`)

The `common` module provides reusable functionality for resolving device, geo, and browser information from OpenRTB requests. This shared logic is used by multiple Mile modules (such as `trafficshaping` and `floors`) to ensure consistent identification across the platform.

## Features

- **Geo Resolution**: Extract country codes from `device.geo.country` with IP-based fallback
- **Device Detection**: Map OpenRTB device types to categories (mobile/tablet/desktop) with UA fallback
- **Browser Detection**: Parse user agent strings to identify browser type
- **Unified Resolver**: Combined resolver with fallback logic and analytics tracking

## Usage

```go
import "github.com/prebid/prebid-server/v3/modules/mile/common"

// Create a geo resolver (optional, for IP-based country fallback)
geoResolver, _ := common.NewHTTPGeoResolver(
    "http://geo-service.com/{ip}",
    time.Minute * 5,
    httpClient,
)

// Create a unified resolver
resolver := common.NewDefaultResolver(geoResolver)

// Resolve all information at once
info, activities, err := resolver.Resolve(ctx, wrapper)
// info.Country = "US"
// info.Device = "w" (web/desktop)
// info.Browser = "chrome"

// Or use individual functions
country, _ := common.ExtractCountry(wrapper)
device, _ := common.ExtractDeviceCategory(wrapper)
browser, _ := common.ExtractBrowser(wrapper)
```

## API Reference

### Geo Resolution

#### `GeoResolver` Interface

```go
type GeoResolver interface {
    Resolve(ctx context.Context, ip string) (string, error)
}
```

Resolves country codes from IP addresses. Returns ISO alpha-2 country code (e.g., "US", "CA", "GB").

#### `HTTPGeoResolver`

HTTP-based implementation with in-memory caching:

```go
func NewHTTPGeoResolver(endpoint string, ttl time.Duration, client *http.Client) (*HTTPGeoResolver, error)
```

- `endpoint`: URL template with `{ip}` placeholder (e.g., `"http://geo-service.com/{ip}"`)
- `ttl`: Cache TTL for geo lookups
- `client`: HTTP client (nil uses `http.DefaultClient`)

**Features**:
- In-memory caching with TTL-based expiry
- Supports multiple field names in response: `country`, `countryCode`, `country_code`, `iso_code`, `isoCode`
- Supports nested structures: `{location: {country: "US"}}`

#### `ExtractCountry(wrapper)`

Extract country from `device.geo.country`:

```go
country, err := common.ExtractCountry(wrapper)
```

- Returns: ISO alpha-2 country code (uppercase)
- Error: If `device.geo.country` is missing or invalid (not 2 letters)

#### `DeriveCountry(ctx, wrapper, geoResolver)`

Derive country from IP address using GeoResolver:

```go
country, err := common.DeriveCountry(ctx, wrapper, geoResolver)
```

- Uses `device.IP` or falls back to `device.IPv6`
- Returns: ISO alpha-2 country code
- Error: If GeoResolver is nil, device is missing, or resolution fails

### Device Detection

#### `ExtractDeviceCategory(wrapper)`

Extract device category from `device.devicetype`:

```go
device, err := common.ExtractDeviceCategory(wrapper)
```

**Returns**:
- `"m"` = Mobile/Phone
- `"t"` = Tablet/TV
- `"w"` = Web/Desktop

**Device Type Mapping** (OpenRTB 2.5):
- `1` (Mobile/Tablet) → `"m"` or `"t"` (checks UA for tablet keywords)
- `2` (Personal Computer) → `"w"`
- `3` (Connected TV) → `"t"`
- `4` (Phone) → `"m"`
- `5` (Tablet) → `"t"`
- `6` (Connected Device) → `"m"`
- `7` (Set Top Box) → `"t"`
- `0` (Unknown) → Error

**Tablet Detection**: For device type `1`, checks UA for: `ipad`, `tablet`, `kindle`

#### `DeriveDeviceCategory(wrapper)`

Fallback extraction when `device.devicetype` is unavailable:

```go
device := common.DeriveDeviceCategory(wrapper)
```

**Fallback Priority**:
1. **SUA (Structured User Agent)**:
   - `sua.Mobile == 1` → `"m"`
   - `sua.Mobile == 0` → `"w"`
   - `sua.Browsers` contains tablet keywords → `"t"`

2. **UA String Parsing**:
   - Tablet: `ipad`, `tablet`, `kindle`, `touch`, `nexus 7`, `xoom` → `"t"`
   - Mobile: `mobile`, `iphone`, `android`, `phone` → `"m"`
   - TV: `smart-tv`, `hbbtv`, `appletv`, `googletv`, `netcast.tv`, `firetv` → `"t"`
   - Default → `"w"`

**Returns**: Device category string or empty string if cannot be determined

### Browser Detection

#### `ExtractBrowser(wrapper)`

Parse user agent to detect browser:

```go
browser, err := common.ExtractBrowser(wrapper)
```

**Returns**:
- `"chrome"` = Chrome (includes CriOS for iOS)
- `"safari"` = Safari
- `"ff"` = Firefox (includes FxiOS for iOS)
- `"edge"` = Edge (Chromium or Legacy)
- `"opera"` = Opera
- Defaults to `"chrome"` for unknown browsers

**Detection Order** (important for accuracy):
1. Edge (`Edg/` or `Edge/`) - checked first since Edge contains "Chrome"
2. Opera (`OPR/` or `Opera/`) - checked before Chrome since Opera contains "Chrome"
3. Firefox (`Firefox/` or `FxiOS/`)
4. Chrome (`Chrome/` or `CriOS/`)
5. Safari (`Safari/`) - checked after Chrome since Chrome contains "Safari"

**Error**: If `device.ua` is missing or empty

### Unified Resolver

#### `Resolver` Interface

```go
type Resolver interface {
    Resolve(ctx context.Context, wrapper *openrtb_ext.RequestWrapper) (RequestInfo, []hookanalytics.Activity, error)
}
```

Resolves country, device, and browser with fallback logic.

#### `DefaultResolver`

Implementation with geo fallback support:

```go
resolver := common.NewDefaultResolver(geoResolver)
info, activities, err := resolver.Resolve(ctx, wrapper)
```

**RequestInfo** struct:
```go
type RequestInfo struct {
    Country string  // ISO alpha-2 country code
    Device  string  // "m", "t", or "w"
    Browser string  // Browser identifier
}
```

**Fallback Logic**:
- **Country**: `device.geo.country` → IP-based geo lookup (if GeoResolver provided)
- **Device**: `device.devicetype` → SUA → UA parsing
- **Browser**: UA parsing (no fallback, must be present)

**Analytics Activities**:
- `country_derived`: Country was resolved via IP fallback
- `devicetype_derived`: Device category was derived from SUA/UA

**Error Handling**:
- Returns error if country cannot be determined (even with fallback)
- Returns error if device category cannot be determined (even with fallback)
- Returns error if browser cannot be extracted (UA required)

## Examples

### Basic Usage

```go
import "github.com/prebid/prebid-server/v3/modules/mile/common"

// Extract individual fields
country, _ := common.ExtractCountry(wrapper)
device, _ := common.ExtractDeviceCategory(wrapper)
browser, _ := common.ExtractBrowser(wrapper)
```

### With Geo Fallback

```go
// Create geo resolver
geoResolver, _ := common.NewHTTPGeoResolver(
    "http://geo-service.com/{ip}",
    time.Minute * 5,
    httpClient,
)

// Use unified resolver
resolver := common.NewDefaultResolver(geoResolver)
info, activities, err := resolver.Resolve(ctx, wrapper)

if err != nil {
    // Handle error
    return
}

// Check if fallbacks were used
for _, activity := range activities {
    if activity.Name == "country_derived" {
        // Country was resolved via IP
    }
    if activity.Name == "devicetype_derived" {
        // Device was derived from SUA/UA
    }
}
```

### Custom Geo Resolver

```go
type MyGeoResolver struct{}

func (r *MyGeoResolver) Resolve(ctx context.Context, ip string) (string, error) {
    // Custom implementation
    return "US", nil
}

resolver := common.NewDefaultResolver(&MyGeoResolver{})
```

## Testing

Run tests with:

```bash
go test ./modules/mile/common/...
```

Run with race detection:

```bash
go test -race ./modules/mile/common/...
```

## Integration with Other Modules

This module is designed to be used by other Mile modules. Example integration:

```go
package mymodule

import "github.com/prebid/prebid-server/v3/modules/mile/common"

type Module struct {
    resolver *common.DefaultResolver
}

func NewModule(geoResolver common.GeoResolver) *Module {
    return &Module{
        resolver: common.NewDefaultResolver(geoResolver),
    }
}

func (m *Module) ProcessRequest(ctx context.Context, wrapper *openrtb_ext.RequestWrapper) {
    info, activities, err := m.resolver.Resolve(ctx, wrapper)
    if err != nil {
        // Handle error
        return
    }
    
    // Use info.Country, info.Device, info.Browser
    // Track activities for analytics
}
```

## Performance

- Geo lookups are cached with configurable TTL
- Device and browser detection are pure functions (no I/O)
- Typical overhead: < 10µs per request (without geo fallback)
- Geo fallback adds network latency (cached)

## Error Handling

All functions follow fail-fast principles:
- Missing required fields return errors immediately
- Fallbacks are attempted only when primary extraction fails
- Empty strings are returned (not errors) for optional fallbacks that cannot determine a value

## Support

For issues or questions, please refer to the Prebid Server documentation or open an issue on GitHub.

