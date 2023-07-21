package silverpush

import (
	"regexp"
	"strings"

	"github.com/prebid/openrtb/v19/openrtb2"
)

func isMobile(ua string) bool {
	isValidMobile := regexp.MustCompile(`(ios|ipod|ipad|iphone|android)`)
	return isValidMobile.MatchString(strings.ToLower(ua))
}

func isCTV(ua string) bool {
	isCtv := regexp.MustCompile(`(smart[-]?tv|hbbtv|appletv|googletv|hdmi|netcast\.tv|viera|nettv|roku|\bdtv\b|sonydtv|inettvbrowser|\btv\b)`)
	return isCtv.MatchString(strings.ToLower(ua))
}

// Check valid Eids
func IsValidEids(eids []openrtb2.EID) bool {
	for i := 0; i < len(eids); i++ {
		if len(eids[i].UIDs) > 0 && eids[i].UIDs[0].ID != "" {
			return true
		}
	}
	return false
}

func getOS(ua string) string {
	if strings.Contains(ua, "Windows") {
		return "Windows"
	} else if regexp.MustCompile(`(iPhone|iPod|iPad)`).MatchString(ua) {
		return "iOS"
	} else if strings.Contains(ua, "Mac OS X") {
		return "macOS"
	} else if strings.Contains(ua, "Android") {
		return "Android"
	} else if strings.Contains(ua, "Linux") {
		return "Linux"
	} else {
		return "Unknown"
	}
}
