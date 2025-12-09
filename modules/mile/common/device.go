package common

import (
	"errors"
	"fmt"
	"strings"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// ExtractDeviceCategory maps OpenRTB device.devicetype to device category code
// OpenRTB 2.5 device types: 1=Mobile/Tablet, 2=Personal Computer, 3=Connected TV, 4=Phone, 5=Tablet, 6=Connected Device, 7=Set Top Box
// Returns: "m" for mobile, "t" for tablet/TV, "w" for web/desktop
func ExtractDeviceCategory(wrapper *openrtb_ext.RequestWrapper) (string, error) {
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

// DeriveDeviceCategory derives device category from SUA or UA string when device.devicetype is unavailable
// Returns: "m" for mobile, "t" for tablet/TV, "w" for web/desktop, or empty string if cannot be determined
func DeriveDeviceCategory(wrapper *openrtb_ext.RequestWrapper) string {
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

// isTablet checks if the UA string indicates a tablet device
func isTablet(ua string) bool {
	uaLower := strings.ToLower(ua)
	return strings.Contains(uaLower, "ipad") ||
		strings.Contains(uaLower, "tablet") ||
		strings.Contains(uaLower, "kindle")
}
