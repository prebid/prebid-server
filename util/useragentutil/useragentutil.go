package useragentutil

import (
	"strings"

	"github.com/mileusna/useragent"
)

// Device types enum
const (
	DEVICE_DESKTOP = iota
	DEVICE_MOBILE
	DEVICE_TABLET
)

// Browser types enum
const (
	BROWSER_CHROME = iota
	BROWSER_FIREFOX
	BROWSER_SAFARI
	BROWSER_EDGE
	BROWSER_INTERNET_EXPLORER
	BROWSER_OTHER
)

// OS types enum
const (
	OS_WINDOWS = iota
	OS_MAC
	OS_LINUX
	OS_UNIX
	OS_IOS
	OS_ANDROID
	OS_OTHER
)

// GetDeviceType determines the device type from the user agent
func GetDeviceType(userAgent string) int {
	ua := useragent.Parse(userAgent)
	if ua.Tablet {
		return DEVICE_TABLET
	} else if ua.Mobile {
		return DEVICE_MOBILE
	}
	return DEVICE_DESKTOP
}

// GetBrowser determines the browser type from the user agent
func GetBrowser(userAgent string) int {
	ua := useragent.Parse(userAgent)
	if ua.IsEdge() {
		return BROWSER_EDGE
	} else if ua.IsChrome() {
		return BROWSER_CHROME
	} else if ua.IsFirefox() {
		return BROWSER_FIREFOX
	} else if ua.IsSafari() {
		return BROWSER_SAFARI
	} else if ua.IsInternetExplorer() {
		return BROWSER_INTERNET_EXPLORER
	}
	return BROWSER_OTHER
}

// GetOS determines the OS from the user agent
func GetOS(userAgent string) int {
	ua := useragent.Parse(userAgent)
	if ua.IsAndroid() {
		return OS_ANDROID
	} else if ua.IsIOS() {
		return OS_IOS
	} else if ua.IsMacOS() {
		return OS_MAC
	} else if ua.IsWindows() {
		return OS_WINDOWS
	} else if strings.Contains(strings.ToLower(userAgent), "x11") {
		return OS_UNIX
	} else if ua.IsLinux() {
		return OS_LINUX
	}
	return OS_OTHER
}
