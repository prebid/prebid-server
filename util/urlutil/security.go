package urlutil

import "regexp"

var safeHostPattern = regexp.MustCompile(`^[a-zA-Z0-9.-]+(:[0-9]+)?$`)

// IsSafeHost returns true for bare hostnames with an optional port.
// It intentionally rejects URL control characters such as '/', '?', '#', and '@'
// so user-supplied host values cannot rewrite the outbound request URL.
func IsSafeHost(host string) bool {
	return safeHostPattern.MatchString(host)
}
