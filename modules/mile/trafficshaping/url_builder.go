package trafficshaping

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/prebid/prebid-server/v3/hooks/hookanalytics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// buildConfigURL constructs the traffic shaping config URL from request data
// Format: {baseEndpoint}{country}/{device}/{browser}/ts.json
// Returns error if required fields are missing (triggers fail-open behavior)
func buildConfigURL(baseEndpoint string, wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if wrapper == nil || wrapper.BidRequest == nil {
		return "", errors.New("invalid request wrapper")
	}

	// Extract country from device.geo.country
	country, err := extractCountry(wrapper)
	if err != nil {
		return "", err
	}

	// Map device.devicetype to device category
	deviceCategory, err := extractDeviceCategory(wrapper)
	if err != nil {
		return "", err
	}

	// Parse device.ua to detect browser
	browser, err := extractBrowser(wrapper)
	if err != nil {
		return "", err
	}

	// Construct URL: {baseEndpoint}{country}/{device}/{browser}/ts.json
	url := fmt.Sprintf("%s%s/%s/%s/ts.json", baseEndpoint, country, deviceCategory, browser)
	return url, nil
}

// extractCountry extracts and normalizes the country code from device.geo.country
func extractCountry(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if wrapper.Device == nil || wrapper.Device.Geo == nil || wrapper.Device.Geo.Country == "" {
		return "", errors.New("missing device.geo.country")
	}

	country := strings.ToUpper(strings.TrimSpace(wrapper.Device.Geo.Country))
	if len(country) != 2 {
		return "", fmt.Errorf("invalid country code: %s (expected 2-letter ISO code)", country)
	}

	return country, nil
}

// extractDeviceCategory maps OpenRTB device.devicetype to device category code
// OpenRTB 2.5 device types: 1=Mobile/Tablet, 2=Personal Computer, 3=Connected TV, 4=Phone, 5=Tablet, 6=Connected Device, 7=Set Top Box
func extractDeviceCategory(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if wrapper.Device == nil {
		return "", errors.New("missing device")
	}

	deviceType := wrapper.Device.DeviceType

	// If deviceType is 0 (default/unknown), return error for fail-open
	if deviceType == 0 {
		return "", errors.New("missing or unknown device type")
	}

	switch deviceType {
	case 1: // Mobile/Tablet (need to check UA or default to mobile)
		// Try to detect tablet from UA
		if wrapper.Device.UA != "" && isTablet(wrapper.Device.UA) {
			return "t", nil
		}
		return "m", nil
	case 2: // Personal Computer
		return "w", nil
	case 3: // Connected TV
		return "t", nil // Using 't' for TV
	case 4: // Phone
		return "m", nil
	case 5: // Tablet
		return "t", nil
	case 6: // Connected Device
		return "m", nil
	case 7: // Set Top Box
		return "t", nil
	default:
		return "", fmt.Errorf("unknown device type: %d", deviceType)
	}
}

// isTablet checks if the UA string indicates a tablet device
func isTablet(ua string) bool {
	uaLower := strings.ToLower(ua)
	return strings.Contains(uaLower, "ipad") ||
		strings.Contains(uaLower, "tablet") ||
		strings.Contains(uaLower, "kindle")
}

// extractBrowser parses the user agent to detect the browser
func extractBrowser(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if wrapper.Device == nil || wrapper.Device.UA == "" {
		return "", errors.New("missing device.ua")
	}

	ua := wrapper.Device.UA

	// Check for Edge (must be before Chrome since Edge also contains "Chrome")
	if strings.Contains(ua, "Edg/") || strings.Contains(ua, "Edge/") {
		return "edge", nil
	}

	// Check for Opera (must be before Chrome since Opera also contains "Chrome")
	if strings.Contains(ua, "OPR/") || strings.Contains(ua, "Opera/") {
		return "opera", nil
	}

	// Check for Firefox
	if strings.Contains(ua, "Firefox/") || strings.Contains(ua, "FxiOS/") {
		return "ff", nil
	}

	// Check for Chrome (CriOS for iOS)
	if strings.Contains(ua, "Chrome/") || strings.Contains(ua, "CriOS/") {
		return "chrome", nil
	}

	// Check for Safari (must be after Chrome since Chrome also contains "Safari")
	if strings.Contains(ua, "Safari/") {
		return "safari", nil
	}

	// Default to chrome for unknown browsers
	return "chrome", nil
}

// extractSiteID extracts the site ID from the bid request
func extractSiteID(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if wrapper.BidRequest == nil || wrapper.BidRequest.Site == nil || wrapper.BidRequest.Site.ID == "" {
		return "", errors.New("missing site.id")
	}
	return wrapper.BidRequest.Site.ID, nil
}

// buildConfigURLWithFallback resolves missing country/device using fallbacks.
func buildConfigURLWithFallback(ctx context.Context, baseEndpoint string, wrapper *openrtb_ext.RequestWrapper, geoResolver GeoResolver) (string, []hookanalytics.Activity, error) {
	activities := make([]hookanalytics.Activity, 0, 3)

	if wrapper == nil || wrapper.BidRequest == nil {
		return "", activities, errors.New("invalid request wrapper")
	}

	siteID, err := extractSiteID(wrapper)
	if err != nil {
		return "", activities, errors.New("site.id unavailable")
	}

	country, err := extractCountry(wrapper)
	if err != nil || country == "" {
		derivedCountry, derr := deriveCountry(ctx, wrapper, geoResolver)
		if derr == nil && derivedCountry != "" {
			country = derivedCountry
			activities = append(activities, hookanalytics.Activity{Name: "country_derived", Status: hookanalytics.ActivityStatusSuccess})
		} else {
			return "", activities, errors.New("country unavailable")
		}
	}

	deviceCategory, err := extractDeviceCategory(wrapper)
	if err != nil || deviceCategory == "" {
		derivedDevice := deriveDeviceCategory(wrapper)
		if derivedDevice == "" {
			return "", activities, errors.New("device category unavailable")
		}
		deviceCategory = derivedDevice
		activities = append(activities, hookanalytics.Activity{Name: "devicetype_derived", Status: hookanalytics.ActivityStatusSuccess})
	}

	browser, err := extractBrowser(wrapper)
	if err != nil {
		return "", activities, err
	}

	url := fmt.Sprintf("%s%s/%s/%s/%s/ts.json", baseEndpoint, siteID, country, deviceCategory, browser)
	return url, activities, nil
}

func deriveCountry(ctx context.Context, wrapper *openrtb_ext.RequestWrapper, geoResolver GeoResolver) (string, error) {
	if geoResolver == nil {
		return "", errors.New("geo resolver not configured")
	}
	if wrapper.Device == nil {
		return "", errors.New("device missing")
	}
	ip := strings.TrimSpace(wrapper.Device.IP)
	if ip == "" {
		ip = strings.TrimSpace(wrapper.Device.IPv6)
	}
	// Attempt resolution even if IP is empty; concrete resolvers may still error.
	country, err := geoResolver.Resolve(ctx, ip)
	if err != nil || country == "" {
		return "", errors.New("country resolve failed")
	}
	return country, nil
}

func deriveDeviceCategory(wrapper *openrtb_ext.RequestWrapper) string {
	if wrapper.Device == nil {
		return ""
	}

	// Prefer SUA information if available
	if sua := wrapper.Device.SUA; sua != nil {
		if sua.Mobile != nil {
			if *sua.Mobile == 1 {
				return "m"
			}
			if *sua.Mobile == 0 {
				return "w"
			}
		}
		if len(sua.Browsers) > 0 {
			for _, browser := range sua.Browsers {
				if browser.Brand != "" {
					brand := strings.ToLower(browser.Brand)
					if strings.Contains(brand, "tablet") || strings.Contains(brand, "ipad") || strings.Contains(brand, "surface") {
						return "t"
					}
				}
			}
		}
	}

	ua := wrapper.Device.UA
	if ua == "" {
		return ""
	}
	uaLower := strings.ToLower(ua)
	switch {
	case strings.Contains(uaLower, "ipad"), strings.Contains(uaLower, "tablet"), strings.Contains(uaLower, "kindle"), strings.Contains(uaLower, "touch"), strings.Contains(uaLower, "nexus 7"), strings.Contains(uaLower, "xoom"):
		return "t"
	case strings.Contains(uaLower, "mobile"), strings.Contains(uaLower, "iphone"), strings.Contains(uaLower, "android"), strings.Contains(uaLower, "phone"):
		return "m"
	case strings.Contains(uaLower, "smart-tv"), strings.Contains(uaLower, "hbbtv"), strings.Contains(uaLower, "appletv"), strings.Contains(uaLower, "googletv"), strings.Contains(uaLower, "netcast.tv"), strings.Contains(uaLower, "firetv"):
		return "t"
	default:
		return "w"
	}
}
