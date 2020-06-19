package httputil

import (
	"net"
	"net/http"
	"strings"

	"github.com/prebid/prebid-server/util/iputil"
)

var (
	trueClientIP    = http.CanonicalHeaderKey("True-Client-IP")
	xForwardedProto = http.CanonicalHeaderKey("X-Forwarded-Proto")
	xForwardedFor   = http.CanonicalHeaderKey("X-Forwarded-For")
	xRealIP         = http.CanonicalHeaderKey("X-Real-IP")
)

const (
	https = "https"
)

// IsSecure determines if a http request uses https.
func IsSecure(r *http.Request) bool {
	if strings.EqualFold(r.Header.Get(xForwardedProto), https) {
		return true
	}

	if strings.EqualFold(r.URL.Scheme, https) {
		return true
	}

	if r.TLS != nil {
		return true
	}

	return false
}

// FindIP returns the first ip address found in the http request matching the predicate m.
func FindIP(r *http.Request, m iputil.IPMatcher) (net.IP, iputil.IPVersion) {
	if ip, ver := findTrueClientIP(r, m); ip != nil {
		return ip, ver
	}

	if ip, ver := findForwardedFor(r, m); ip != nil {
		return ip, ver
	}

	if ip, ver := findRealIP(r, m); ip != nil {
		return ip, ver
	}

	if ip, ver := findRemoteAddr(r, m); ip != nil {
		return ip, ver
	}

	return nil, iputil.IPvUnknown
}

func findTrueClientIP(r *http.Request, m iputil.IPMatcher) (net.IP, iputil.IPVersion) {
	if v := r.Header.Get(trueClientIP); v != "" {
		v = strings.TrimSpace(v)
		if ip, ver := iputil.ParseIP(v); ip != nil && m.Match(ip, ver) {
			return ip, ver
		}
	}
	return nil, iputil.IPvUnknown
}

func findForwardedFor(r *http.Request, m iputil.IPMatcher) (net.IP, iputil.IPVersion) {
	if v := r.Header.Get(xForwardedFor); v != "" {
		for _, p := range strings.Split(v, ",") {
			p = strings.TrimSpace(p)
			if ip, ver := iputil.ParseIP(p); ip != nil && m.Match(ip, ver) {
				return ip, ver
			}
		}
	}
	return nil, iputil.IPvUnknown
}

func findRealIP(r *http.Request, m iputil.IPMatcher) (net.IP, iputil.IPVersion) {
	if v := r.Header.Get(xRealIP); v != "" {
		v = strings.TrimSpace(v)
		if ip, ver := iputil.ParseIP(v); ip != nil && m.Match(ip, ver) {
			return ip, ver
		}
	}
	return nil, iputil.IPvUnknown
}

func findRemoteAddr(r *http.Request, m iputil.IPMatcher) (net.IP, iputil.IPVersion) {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		if ip, ver := iputil.ParseIP(host); ip != nil && m.Match(ip, ver) {
			return ip, ver
		}
	}
	return nil, iputil.IPvUnknown
}
