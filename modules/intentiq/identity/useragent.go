package identity

import (
	"regexp"
	"strings"
)

// normalizeUA reduces a raw User-Agent string to a compact, deterministic cache-key segment such as
// "iOS17_MobileSafari17_iPhone": coarse OS (family+major), browser (family+major) and device (family)
// tokens joined by "_". Empty tokens are dropped; a blank or wholly unrecognized UA returns "".
//
// This is an intentional lightweight adaptation of the Java DeviceUserAgent.normalize, which uses the
// ua_parser library. We deliberately do NOT pull in that dependency: the segment's only job is to be a
// stable, low-cardinality cache-key fragment, so exact ua_parser parity is not required. The single
// hard requirement is determinism — the same input must always yield the same output — which a fixed
// ordered heuristic table guarantees. Rendering engine and app-vs-browser distinctions are omitted,
// matching the Java version's own omissions.
func normalizeUA(ua string) string {
	ua = strings.TrimSpace(ua)
	if ua == "" {
		return ""
	}

	var parts []string
	if t := osToken(ua); t != "" {
		parts = append(parts, t)
	}
	if t := browserToken(ua); t != "" {
		parts = append(parts, t)
	}
	if t := deviceToken(ua); t != "" {
		parts = append(parts, t)
	}
	return strings.Join(parts, "_")
}

// uaRule pairs a detection regex with the family name emitted on a match. When the regex has a capture
// group its first submatch is used as the major version; otherwise no version is appended.
type uaRule struct {
	re      *regexp.Regexp
	family  string
	version bool // whether re captures a major-version group at index 1
}

// osRules are evaluated in order; the first match wins. iOS is checked before Mac OS X because iPhone
// UAs contain the "like Mac OS X" token.
var osRules = []uaRule{
	{regexp.MustCompile(`(?i)(?:iphone os|cpu os)\s+(\d+)`), "iOS", true},
	{regexp.MustCompile(`(?i)mac os x\s+(\d+)`), "MacOSX", true},
	{regexp.MustCompile(`(?i)android[\s/]+(\d+)`), "Android", true},
	{regexp.MustCompile(`(?i)windows nt\s+(\d+)`), "Windows", true},
	{regexp.MustCompile(`(?i)cros`), "ChromeOS", false},
	{regexp.MustCompile(`(?i)linux`), "Linux", false},
}

// browserRules are evaluated in order; the first match wins. Wrapper browsers (Edge/Opera/Samsung)
// must precede Chrome, and Chrome must precede Safari, because each later UA embeds the earlier
// token.
var browserRules = []uaRule{
	{regexp.MustCompile(`(?i)edg(?:e|ios|a)?/(\d+)`), "Edge", true},
	{regexp.MustCompile(`(?i)(?:opr|opera)/(\d+)`), "Opera", true},
	{regexp.MustCompile(`(?i)samsungbrowser/(\d+)`), "SamsungBrowser", true},
	{regexp.MustCompile(`(?i)(?:firefox|fxios)/(\d+)`), "Firefox", true},
	{regexp.MustCompile(`(?i)(?:crios|chromium|chrome)/(\d+)`), "Chrome", true},
	{regexp.MustCompile(`(?i)version/(\d+).*safari`), "Safari", true},
}

// deviceRules are evaluated in order; the first match wins.
var deviceRules = []uaRule{
	{regexp.MustCompile(`(?i)ipad`), "iPad", false},
	{regexp.MustCompile(`(?i)ipod`), "iPod", false},
	{regexp.MustCompile(`(?i)iphone`), "iPhone", false},
	{regexp.MustCompile(`(?i)pixel\s+(\d+)`), "Pixel", true},
	{regexp.MustCompile(`(?i)nexus\s+(\d+)`), "Nexus", true},
	{regexp.MustCompile(`(?i)macintosh`), "Mac", false},
}

func osToken(ua string) string { return matchRules(ua, osRules) }

func deviceToken(ua string) string { return matchRules(ua, deviceRules) }

// browserToken mirrors ua_parser's mobile-aware families: Chrome and Safari become their "Mobile"
// variants when the UA advertises "Mobile".
func browserToken(ua string) string {
	mobile := strings.Contains(strings.ToLower(ua), "mobile")
	for _, r := range browserRules {
		m := r.re.FindStringSubmatch(ua)
		if m == nil {
			continue
		}
		family := r.family
		if mobile && (family == "Chrome" || family == "Safari") {
			family = "Mobile" + family
		}
		version := ""
		if r.version && len(m) > 1 {
			version = m[1]
		}
		return token(family, version)
	}
	return ""
}

func matchRules(ua string, rules []uaRule) string {
	for _, r := range rules {
		m := r.re.FindStringSubmatch(ua)
		if m == nil {
			continue
		}
		version := ""
		if r.version && len(m) > 1 {
			version = m[1]
		}
		return token(r.family, version)
	}
	return ""
}

// whitespace collapses any run of whitespace so a family like "Mobile Safari" becomes "MobileSafari".
var whitespace = regexp.MustCompile(`\s+`)

// token joins a family with an optional major version, stripping all whitespace. It returns "" for a
// blank family so callers can drop it.
func token(family, version string) string {
	value := whitespace.ReplaceAllString(family+version, "")
	return value
}
