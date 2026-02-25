package useragentutil

import (
	"strings"

	"github.com/mileusna/useragent"
)

// Device types enum
const (
	DeviceDesktop = iota
	DeviceMobile
	DeviceTablet
)

// Browser types enum
const (
	BrowserChrome = iota
	BrowserFirefox
	BrowserSafari
	BrowserEdge
	BrowserInternetExplorer
	BrowserOther
)

// OS types enum
const (
	OsWindows = iota
	OsMac
	OsLinux
	OsUnix
	OsIOS
	OsAndroid
	OsOther
)

// GetDeviceType determines the device type from the user agent
func GetDeviceType(userAgent string) int {
	ua := useragent.Parse(userAgent)
	if ua.Tablet {
		return DeviceTablet
	} else if ua.Mobile {
		return DeviceMobile
	}
	return DeviceDesktop
}

// GetBrowser determines the browser type from the user agent
func GetBrowser(userAgent string) int {
	ua := useragent.Parse(userAgent)
	if ua.IsEdge() {
		return BrowserEdge
	} else if ua.IsChrome() {
		return BrowserChrome
	} else if ua.IsFirefox() {
		return BrowserFirefox
	} else if ua.IsSafari() {
		return BrowserSafari
	} else if ua.IsInternetExplorer() {
		return BrowserInternetExplorer
	}
	return BrowserOther
}

// GetOS determines the OS from the user agent
func GetOS(userAgent string) int {
	ua := useragent.Parse(userAgent)
	if ua.IsAndroid() {
		return OsAndroid
	} else if ua.IsIOS() {
		return OsIOS
	} else if ua.IsMacOS() {
		return OsMac
	} else if ua.IsWindows() {
		return OsWindows
	} else if strings.Contains(strings.ToLower(userAgent), "x11") {
		return OsUnix
	} else if ua.IsLinux() {
		return OsLinux
	}
	return OsOther
}
