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

// GetIP returns the first IP address found in the request not matching a private network.
func GetIP(r *http.Request, privateNetworks []*net.IPNet) net.IP {
	if ip := getFirstForwardedFor(r, privateNetworks); ip != nil {
		return ip
	}

	if ip := getRealIP(r); ip != nil && !containsIP(ip, privateNetworks) {
		return ip
	}

	if ip := getRemoteAddr(r); ip != nil && !containsIP(ip, privateNetworks) {
		return ip
	}

	return nil
}

func containsIP(ip net.IP, privateNetworks []*net.IPNet) bool {
	for _, network := range privateNetworks {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

func getFirstForwardedFor(r *http.Request, privateNetworks []*net.IPNet) net.IP {
	if xff := r.Header.Get(xForwardedFor); xff != "" {
		xffParts := strings.Split(xff, ",")

		for _, part := range xffParts {
			part = strings.TrimSpace(part)

			if ip := net.ParseIP(part); ip != nil && !containsIP(ip, privateNetworks) {
				return ip
			}
		}
	}

	return nil
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
