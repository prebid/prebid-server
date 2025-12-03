package common

import (
	"strings"
)

// ClassifyDevicePlatform analyzes the device User Agent string and classifies it
// into a platform string in the format "{device}|{browser}"
// Examples: "m-android|chrome", "m-ios|safari", "t-ios|chrome", "w|safari"
// Returns empty string if UA is empty
func ClassifyDevicePlatform(ua string) string {
	if ua == "" {
		return ""
	}

	deviceType := detectDeviceType(ua)
	browser := detectBrowser(ua)

	if deviceType == "" || browser == "" {
		return ""
	}

	return deviceType + "|" + browser
}

// detectDeviceType detects the device type and OS from user agent
// Returns: "m-android", "m-ios", "t-android", "t-ios", "w"
func detectDeviceType(ua string) string {
	uaLower := strings.ToLower(ua)

	// Check for iOS devices first
	isIOS := strings.Contains(uaLower, "iphone") ||
		strings.Contains(uaLower, "ipod") ||
		strings.Contains(uaLower, "ipad")

	if isIOS {
		// iPad is tablet
		if strings.Contains(uaLower, "ipad") {
			return "t-ios"
		}
		// iPhone/iPod are mobile
		return "m-ios"
	}

	// Check for Android devices
	isAndroid := strings.Contains(uaLower, "android")
	if isAndroid {
		// Android tablet detection
		isTablet := strings.Contains(uaLower, "tablet") ||
			(!strings.Contains(uaLower, "mobile") && strings.Contains(uaLower, "android"))

		if isTablet {
			return "t-android"
		}
		// Android mobile
		return "m-android"
	}

	// Everything else is web/desktop
	return "w"
}

// detectBrowser detects the browser from user agent
// Returns: "chrome", "safari", "edge", "google search", "ff" (firefox), "opera", "samsung internet for android", "amazon silk"
func detectBrowser(ua string) string {
	// Check for Google Search App (GSA)
	if strings.Contains(ua, "GSA/") {
		return "google search"
	}

	// Check for Samsung Internet (must be before Chrome since it contains "Chrome")
	if strings.Contains(ua, "SamsungBrowser/") {
		return "samsung internet for android"
	}

	// Check for Amazon Silk (must be before Chrome since it may contain "Chrome")
	if strings.Contains(ua, "Silk/") {
		return "amazon silk"
	}

	// Check for Edge (must be before Chrome since Edge also contains "Chrome")
	if strings.Contains(ua, "Edg/") || strings.Contains(ua, "Edge/") || strings.Contains(ua, "EdgiOS/") {
		return "edge"
	}

	// Check for Opera (must be before Chrome since Opera also contains "Chrome")
	if strings.Contains(ua, "OPR/") || strings.Contains(ua, "Opera/") {
		return "opera"
	}

	// Check for Firefox
	if strings.Contains(ua, "Firefox/") || strings.Contains(ua, "FxiOS/") {
		return "ff"
	}

	// Check for Chrome (CriOS for iOS)
	if strings.Contains(ua, "Chrome/") || strings.Contains(ua, "CriOS/") {
		return "chrome"
	}

	// Check for Safari (must be after Chrome since Chrome also contains "Safari")
	if strings.Contains(ua, "Safari/") {
		return "safari"
	}

	// Default to chrome for unknown browsers
	return "chrome"
}
