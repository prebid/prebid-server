package httputil

import (
	"crypto/tls"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSecure(t *testing.T) {
	testCases := []struct {
		description     string
		url             string
		xForwardedProto string
		tls             bool
		expectIsSecure  bool
	}{
		{
			description:    "HTTP",
			url:            "http://host.com",
			expectIsSecure: false,
		},
		{
			description:     "HTTPS - Forwarded Protocol",
			url:             "http://host.com",
			xForwardedProto: "https",
			expectIsSecure:  true,
		},
		{
			description:     "HTTPS - Forwarded Protocol - Case Insensitive",
			url:             "http://host.com",
			xForwardedProto: "HTTPS",
			expectIsSecure:  true,
		},
		{
			description:    "HTTPS - Protocol",
			url:            "https://host.com",
			expectIsSecure: true,
		},
		{
			description:    "HTTPS - Protocol - Case Insensitive",
			url:            "HTTPS://host.com",
			expectIsSecure: true,
		},
		{
			description:    "HTTPS - TLS",
			url:            "http://host.com",
			tls:            true,
			expectIsSecure: true,
		},
	}

	for _, test := range testCases {
		request, err := http.NewRequest("GET", test.url, nil)
		if err != nil {
			t.Fatalf("Unable to create test http request. Err: %v", err)
		}
		if test.xForwardedProto != "" {
			request.Header.Add("X-Forwarded-Proto", test.xForwardedProto)
		}
		if test.tls {
			request.TLS = &tls.ConnectionState{}
		}

		result := IsSecure(request)
		assert.Equal(t, test.expectIsSecure, result, test.description)
	}
}

func TestFindIP(t *testing.T) {
	alwaysTrue := func(net.IP) bool { return true }
	alwaysFalse := func(net.IP) bool { return false }

	testCases := []struct {
		description   string
		trueClientIP  string
		xForwardedFor string
		xRealIP       string
		remoteAddr    string
		matcher       IPAddressMatcher
		expected      string
	}{
		{
			description: "No Address",
			expected:    "",
		},
		{
			description:   "False Matcher",
			trueClientIP:  "1.1.1.1",
			xForwardedFor: "2.2.2.2, 3.3.3.3",
			xRealIP:       "4.4.4.4",
			remoteAddr:    "5.5.5.5:5",
			matcher:       alwaysFalse,
			expected:      "",
		},
		{
			description:   "Specific Matcher - IPv4 - X Forwarded For",
			xForwardedFor: "2.2.2.2, 3.3.3.3",
			matcher:       func(ip net.IP) bool { return net.IP.Equal(ip, net.ParseIP("3.3.3.3")) },
			expected:      "3.3.3.3",
		},
		{
			description:   "Specific Matcher - IPv6 - X Forwarded For",
			xForwardedFor: "2222::2222, 3333::3333",
			matcher:       func(ip net.IP) bool { return net.IP.Equal(ip, net.ParseIP("3333::3333")) },
			expected:      "3333::3333",
		},
		{
			description:   "True Matcher - True Client IP",
			trueClientIP:  "1.1.1.1",
			xForwardedFor: "2.2.2.2, 3.3.3.3",
			xRealIP:       "4.4.4.4",
			remoteAddr:    "5.5.5.5:5",
			matcher:       alwaysTrue,
			expected:      "1.1.1.1",
		},
		{
			description:   "True Matcher - True Client IP - Ignore Whitespace",
			trueClientIP:  "   1.1.1.1 ",
			xForwardedFor: "2.2.2.2, 3.3.3.3",
			xRealIP:       "4.4.4.4",
			remoteAddr:    "5.5.5.5:5",
			matcher:       alwaysTrue,
			expected:      "1.1.1.1",
		},
		{
			description:   "True Matcher - X Forwarded For",
			trueClientIP:  "",
			xForwardedFor: "2.2.2.2, 3.3.3.3",
			xRealIP:       "4.4.4.4",
			remoteAddr:    "5.5.5.5:5",
			matcher:       alwaysTrue,
			expected:      "2.2.2.2",
		},
		{
			description:   "True Matcher - X Forwarded For - Ignore Whitespace",
			trueClientIP:  "",
			xForwardedFor: "   2.2.2.2, 3.3.3.3 ",
			xRealIP:       "4.4.4.4",
			remoteAddr:    "5.5.5.5:5",
			matcher:       alwaysTrue,
			expected:      "2.2.2.2",
		},
		{
			description:   "True Matcher - X Real IP",
			trueClientIP:  "",
			xForwardedFor: "",
			xRealIP:       "4.4.4.4",
			remoteAddr:    "5.5.5.5:5",
			matcher:       alwaysTrue,
			expected:      "4.4.4.4",
		},
		{
			description:   "True Matcher - X Real IP - Ignore Whitespace",
			trueClientIP:  "",
			xForwardedFor: "",
			xRealIP:       "   4.4.4.4 ",
			remoteAddr:    "5.5.5.5:5",
			matcher:       alwaysTrue,
			expected:      "4.4.4.4",
		},
		{
			description:   "True Matcher - Remote Address IPv4",
			trueClientIP:  "",
			xForwardedFor: "",
			xRealIP:       "",
			remoteAddr:    "5.5.5.5:80",
			matcher:       alwaysTrue,
			expected:      "5.5.5.5",
		},
		{
			description:   "True Matcher - Remote Address IPv6",
			trueClientIP:  "",
			xForwardedFor: "",
			xRealIP:       "",
			remoteAddr:    "[5555::5555]:80",
			matcher:       alwaysTrue,
			expected:      "5555::5555",
		},
		{
			description:   "True Matcher - Malformed - All",
			trueClientIP:  "malformed",
			xForwardedFor: "malformed",
			xRealIP:       "malformed",
			remoteAddr:    "malformed",
			matcher:       alwaysTrue,
			expected:      "",
		},
		{
			description:   "True Matcher - Malformed - Some",
			trueClientIP:  "malformed",
			xForwardedFor: "malformed",
			xRealIP:       "4.4.4.4",
			remoteAddr:    "malformed",
			matcher:       alwaysTrue,
			expected:      "4.4.4.4",
		},
		{
			description:   "True Matcher - Malformed - X Forwarded For",
			trueClientIP:  "malformed",
			xForwardedFor: "malformed, 4.4.4.4, malformed",
			xRealIP:       "malformed",
			remoteAddr:    "malformed",
			matcher:       alwaysTrue,
			expected:      "4.4.4.4",
		},
	}

	for _, test := range testCases {
		// Build Request
		request, err := http.NewRequest("GET", "http://anyurl.com", nil)
		if err != nil {
			t.Fatalf("Unable to create test http request. Err: %v", err)
		}
		if test.trueClientIP != "" {
			request.Header.Add("True-Client-IP", test.trueClientIP)
		}
		if test.xForwardedFor != "" {
			request.Header.Add("X-Forwarded-For", test.xForwardedFor)
		}
		if test.xRealIP != "" {
			request.Header.Add("X-Real-IP", test.xRealIP)
		}
		request.RemoteAddr = test.remoteAddr

		// Run Test
		result := FindIP(request, test.matcher)

		// Assertions
		if test.expected == "" {
			assert.Empty(t, result, test.description+":result")
		} else {
			assert.Equal(t, net.ParseIP(test.expected), result, test.description+":result")
		}
	}
}
