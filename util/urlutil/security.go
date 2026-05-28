package urlutil

import (
	"regexp"
	"strings"
)

var safeHostPattern = regexp.MustCompile(`^[a-zA-Z0-9.-]+(:[0-9]+)?$`)

// IsSafeHost returns true for bare hostnames with an optional port.
// It intentionally rejects URL control characters such as '/', '?', '#', and '@'
// so user-supplied host values cannot rewrite the outbound request URL.
func IsSafeHost(host string) bool {
	return safeHostPattern.MatchString(host)
}

// IsSafePath returns true for non-empty relative paths that cannot alter the
// query string or fragment of a URL when appended to a fixed endpoint.
func IsSafePath(path string) bool {
	return path != "" && !strings.ContainsAny(path, "?#") && !strings.Contains(path, "://") && !strings.HasPrefix(path, "//")
}

// IsSafePathSegment returns true for a single path segment that cannot inject
// path, query, or fragment separators.
func IsSafePathSegment(segment string) bool {
	return segment != "" && !strings.ContainsAny(segment, "/?#")
}
