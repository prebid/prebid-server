package httputil

import (
	"net"
	"net/http"
	"strings"
)

var (
	xForwardedProto = http.CanonicalHeaderKey("X-Forwarded-Proto")
	xForwardedFor   = http.CanonicalHeaderKey("X-Forwarded-For")
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

// GetIPs returns all IP addresses found in the request.
func GetIPs(r *http.Request) []net.IP {
	result := getForwardedFor(r)

	if ip := getRealIP(r); ip != nil {
		result = append(result, ip)
	}

	if ip := getRemoteAddr(r); ip != nil {
		result = append(result, ip)
	}

	return result
}

func getForwardedFor(r *http.Request) []net.IP {
	var result []net.IP

	if xff := r.Header.Get(xForwardedFor); xff != "" {
		xffParts := strings.Split(xff, ",")
		for _, part := range xffParts {
			part = strings.TrimSpace(part)
			if ip := net.ParseIP(part); ip != nil {
				result = append(result, ip)
			}
		}
	}

	return result
}

func getRealIP(r *http.Request) net.IP {
	if xri := r.Header.Get(xRealIP); xri != "" {
		xri = strings.TrimSpace(xri)
		return net.ParseIP(xri)
	}

	return nil
}

func getRemoteAddr(r *http.Request) net.IP {
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return net.ParseIP(ip)
	}

	return nil
}
