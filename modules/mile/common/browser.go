package common

import (
	"errors"
	"strings"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// ExtractBrowser parses the user agent to detect the browser
// Returns: "chrome", "safari", "ff" (Firefox), "edge", "opera", or error if UA is missing
func ExtractBrowser(wrapper *openrtb_ext.RequestWrapper) (string, error) {
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

