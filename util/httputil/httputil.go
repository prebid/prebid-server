package httputil

import (
	"net"
	"net/http"
	"strings"
)

var (
	xForwardedProto = http.CanonicalHeaderKey("X-Forwarded-Proto")
	xForwardedFor   = http.CanonicalHeaderKey("X-Forwarded-For")
	xTrueClientIP   = http.CanonicalHeaderKey("True-Client-IP")
	xRealIP         = http.CanonicalHeaderKey("X-Real-IP")
)

const (
	https = "https"
)

// IsSecure determines if the request uses https.
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

// IPAddressMatcher should return true when a desired IP address is found.
type IPAddressMatcher func(net.IP) bool

// FindIP returns the first IP address found in the request matching the predicate f.
func FindIP(r *http.Request, m IPAddressMatcher) net.IP {
	if ip := getTrueClientIP(r, m); ip != nil {
		return ip
	}

	if ip := getFirstForwardedFor(r, m); ip != nil {
		return ip
	}

	if ip := getRealIP(r, m); ip != nil {
		return ip
	}

	if ip := getRemoteAddr(r, m); ip != nil {
		return ip
	}

	return nil
}

func getTrueClientIP(r *http.Request, m IPAddressMatcher) net.IP {
	if tci := r.Header.Get(xTrueClientIP); tci != "" {
		tci = strings.TrimSpace(tci)

		if ip := net.ParseIP(tci); ip != nil && m(ip) {
			return ip
		}
	}

	return nil
}

func getFirstForwardedFor(r *http.Request, m IPAddressMatcher) net.IP {
	if xff := r.Header.Get(xForwardedFor); xff != "" {
		xffParts := strings.Split(xff, ",")

		for _, part := range xffParts {
			part = strings.TrimSpace(part)

			if ip := net.ParseIP(part); ip != nil && m(ip) {
				return ip
			}
		}
	}

	return nil
}

func getRealIP(r *http.Request, m IPAddressMatcher) net.IP {
	if xri := r.Header.Get(xRealIP); xri != "" {
		xri = strings.TrimSpace(xri)

		if ip := net.ParseIP(xri); ip != nil && m(ip) {
			return ip
		}
	}

	return nil
}

func getRemoteAddr(r *http.Request, m IPAddressMatcher) net.IP {
	if ipRaw, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		if ip := net.ParseIP(ipRaw); ip != nil && m(ip) {
			return ip
		}
	}

	return nil
}
