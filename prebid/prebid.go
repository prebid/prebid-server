package prebid

import (
	"net"
	"net/http"
	"strings"
)

var xForwardedFor = http.CanonicalHeaderKey("X-Forwarded-For")
var xRealIP = http.CanonicalHeaderKey("X-Real-IP")
var xForwardedProto = http.CanonicalHeaderKey("X-Forwarded-Proto")

// IsSecure attempts to detect whether the request is https
func IsSecure(r *http.Request) bool {
	// lowercase for case-insensitive match for X-Forwarded-Proto header
	if strings.ToLower(r.Header.Get(xForwardedProto)) == "https" {
		return true
	}
	// ensure that URL.Scheme is lowercase (it should be "https")
	if strings.ToLower(r.URL.Scheme) == "https" {
		return true
	}
	// use strings.HasPrefix because a valid example is "HTTP/1.0"
	if strings.HasPrefix(r.Proto, "HTTPS") {
		return true
	}
	// check if TLS is not-nil as a final fallback
	if r.TLS != nil {
		return true
	}
	return false
}

// GetIP will attempt to get the IP Address by first checking headers
// and then falling back on the RemoteAddr
func GetIP(r *http.Request) string {
	// first check headers
	if ip := GetForwardedIP(r); ip != "" {
		return ip
	}
	// next try to parse the RemoteAddr.
	// if err is not nil then weird hosts might appear as the ip: https://github.com/golang/go/issues/14827
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}
	return ""
}

// GetForwardedIP will return back X-Forwarded-For or X-Real-IP (if set)
func GetForwardedIP(r *http.Request) string {
	// first attempt to parse X-Forwarded-For
	if ip := getForwardedFor(r); ip != "" {
		return ip
	}
	// if we don't have X-Forwarded-For then try X-Real-IP
	if ip := getRealIP(r); ip != "" {
		return ip
	}
	return ""
}

// getForwardedFor will attempt to parse the X-Forwarded-For header
func getForwardedFor(r *http.Request) string {
	if xff := r.Header.Get(xForwardedFor); xff != "" {
		// X-Forwarded-For: client1, proxy1, proxy2
		i := strings.Index(xff, ", ")
		if i == -1 {
			i = len(xff)
		}
		return xff[:i]
	}
	return ""
}

// getRealIP will attempt to parse the X-Real-IP header
// Header.Get is case-insensitive
func getRealIP(r *http.Request) string {
	if xrip := r.Header.Get(xRealIP); xrip != "" {
		return xrip
	}
	return ""
}
