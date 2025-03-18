package httputil

import (
	"net"
	"net/http"
	"strings"

	"github.com/prebid/prebid-server/v3/util/iputil"
)

var (
	trueClientIP  = http.CanonicalHeaderKey("True-Client-IP")
	xForwardedFor = http.CanonicalHeaderKey("X-Forwarded-For")
	xRealIP       = http.CanonicalHeaderKey("X-Real-IP")
)

type ContentEncoding string

const (
	ContentEncodingGZIP ContentEncoding = "gzip"
)

func (k ContentEncoding) Normalize() ContentEncoding {
	return ContentEncoding(strings.ToLower(string(k)))
}

// FindIP returns the first ip address found in the http request matching the predicate v.
func FindIP(r *http.Request, v iputil.IPValidator) (net.IP, iputil.IPVersion) {
	if ip, ver := findTrueClientIP(r, v); ip != nil {
		return ip, ver
	}

	if ip, ver := findForwardedFor(r, v); ip != nil {
		return ip, ver
	}

	if ip, ver := findRealIP(r, v); ip != nil {
		return ip, ver
	}

	if ip, ver := findRemoteAddr(r, v); ip != nil {
		return ip, ver
	}

	return nil, iputil.IPvUnknown
}

func findTrueClientIP(r *http.Request, v iputil.IPValidator) (net.IP, iputil.IPVersion) {
	if value := r.Header.Get(trueClientIP); value != "" {
		value = strings.TrimSpace(value)
		if ip, ver := iputil.ParseIP(value); ip != nil && v.IsValid(ip, ver) {
			return ip, ver
		}
	}
	return nil, iputil.IPvUnknown
}

func findForwardedFor(r *http.Request, v iputil.IPValidator) (net.IP, iputil.IPVersion) {
	if value := r.Header.Get(xForwardedFor); value != "" {
		for _, p := range strings.Split(value, ",") {
			p = strings.TrimSpace(p)
			if ip, ver := iputil.ParseIP(p); ip != nil && v.IsValid(ip, ver) {
				return ip, ver
			}
		}
	}
	return nil, iputil.IPvUnknown
}

func findRealIP(r *http.Request, v iputil.IPValidator) (net.IP, iputil.IPVersion) {
	if value := r.Header.Get(xRealIP); value != "" {
		value = strings.TrimSpace(value)
		if ip, ver := iputil.ParseIP(value); ip != nil && v.IsValid(ip, ver) {
			return ip, ver
		}
	}
	return nil, iputil.IPvUnknown
}

func findRemoteAddr(r *http.Request, v iputil.IPValidator) (net.IP, iputil.IPVersion) {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		if ip, ver := iputil.ParseIP(host); ip != nil && v.IsValid(ip, ver) {
			return ip, ver
		}
	}
	return nil, iputil.IPvUnknown
}
