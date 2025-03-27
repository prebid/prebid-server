package silverpush

import (
	"regexp"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
)

var isValidMobile = regexp.MustCompile(`(ios|ipod|ipad|iphone|android)`)
var isCtv = regexp.MustCompile(`(smart[-]?tv|hbbtv|appletv|googletv|hdmi|netcast\.tv|viera|nettv|roku|\bdtv\b|sonydtv|inettvbrowser|\btv\b)`)
var isIos = regexp.MustCompile(`(iPhone|iPod|iPad)`)

func isMobile(ua string) bool {
	return isValidMobile.MatchString(strings.ToLower(ua))
}

func isCTV(ua string) bool {
	return isCtv.MatchString(strings.ToLower(ua))
}

// isValidEids checks for valid eids.
func isValidEids(eids []openrtb2.EID) bool {
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
	} else if isIos.MatchString(ua) {
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
