package httputil

import (
	"crypto/tls"
	"net"
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/util/iputil"
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
	alwaysTrue := hardcodedResponseIPMatcher{response: true}
	alwaysFalse := hardcodedResponseIPMatcher{response: false}

	testCases := []struct {
		description   string
		trueClientIP  string
		xForwardedFor string
		xRealIP       string
		remoteAddr    string
		matcher       iputil.IPMatcher
		expectedIP    net.IP
		expectedVer   iputil.IPVersion
	}{
		{
			description: "No Address",
			expectedIP:  nil,
			expectedVer: iputil.IPvUnknown,
		},
		{
			description:   "False Matcher - IPv4",
			trueClientIP:  "1.1.1.1",
			xForwardedFor: "2.2.2.2, 3.3.3.3",
			xRealIP:       "4.4.4.4",
			remoteAddr:    "5.5.5.5:5",
			matcher:       alwaysFalse,
			expectedIP:    nil,
			expectedVer:   iputil.IPvUnknown,
		},
		{
			description:   "False Matcher - IPv6",
			trueClientIP:  "0101::",
			xForwardedFor: "0202::, 0303::",
			xRealIP:       "0404::",
			remoteAddr:    "[0505::]:5]",
			matcher:       alwaysFalse,
			expectedIP:    nil,
			expectedVer:   iputil.IPvUnknown,
		},
		{
			description:   "True Matcher - IPv4 - True Client IP",
			trueClientIP:  "1.1.1.1",
			xForwardedFor: "2.2.2.2, 3.3.3.3",
			xRealIP:       "4.4.4.4",
			remoteAddr:    "5.5.5.5:5",
			matcher:       alwaysTrue,
			expectedIP:    net.ParseIP("1.1.1.1"),
			expectedVer:   iputil.IPv4,
		},
		{
			description:   "True Matcher - IPv4 - True Client IP - Ignore Whitespace",
			trueClientIP:  "   1.1.1.1 ",
			xForwardedFor: "2.2.2.2, 3.3.3.3",
			xRealIP:       "4.4.4.4",
			remoteAddr:    "5.5.5.5:5",
			matcher:       alwaysTrue,
			expectedIP:    net.ParseIP("1.1.1.1"),
			expectedVer:   iputil.IPv4,
		},
		{
			description:   "True Matcher - IPv4 - X Forwarded For",
			trueClientIP:  "",
			xForwardedFor: "2.2.2.2, 3.3.3.3",
			xRealIP:       "4.4.4.4",
			remoteAddr:    "5.5.5.5:5",
			matcher:       alwaysTrue,
			expectedIP:    net.ParseIP("2.2.2.2"),
			expectedVer:   iputil.IPv4,
		},
		{
			description:   "True Matcher - IPv4 - X Forwarded For - Ignore Whitespace",
			trueClientIP:  "",
			xForwardedFor: "   2.2.2.2, 3.3.3.3 ",
			xRealIP:       "4.4.4.4",
			remoteAddr:    "5.5.5.5:5",
			matcher:       alwaysTrue,
			expectedIP:    net.ParseIP("2.2.2.2"),
			expectedVer:   iputil.IPv4,
		},
		{
			description:   "True Matcher - IPv4 - X Real IP",
			trueClientIP:  "",
			xForwardedFor: "",
			xRealIP:       "4.4.4.4",
			remoteAddr:    "5.5.5.5:5",
			matcher:       alwaysTrue,
			expectedIP:    net.ParseIP("4.4.4.4"),
			expectedVer:   iputil.IPv4,
		},
		{
			description:   "True Matcher - IPv4 - X Real IP - Ignore Whitespace",
			trueClientIP:  "",
			xForwardedFor: "",
			xRealIP:       "   4.4.4.4 ",
			remoteAddr:    "5.5.5.5:5",
			matcher:       alwaysTrue,
			expectedIP:    net.ParseIP("4.4.4.4"),
			expectedVer:   iputil.IPv4,
		},
		{
			description:   "True Matcher - IPv4 - Remote Address",
			trueClientIP:  "",
			xForwardedFor: "",
			xRealIP:       "",
			remoteAddr:    "5.5.5.5:80",
			matcher:       alwaysTrue,
			expectedIP:    net.ParseIP("5.5.5.5"),
			expectedVer:   iputil.IPv4,
		},
		{
			description:   "True Matcher - IPv6 - True Client IP",
			trueClientIP:  "0101::",
			xForwardedFor: "0202::, 0303::",
			xRealIP:       "0404::",
			remoteAddr:    "[0505::]:5",
			matcher:       alwaysTrue,
			expectedIP:    net.ParseIP("0101::"),
			expectedVer:   iputil.IPv6,
		},
		{
			description:   "True Matcher - IPv6 - True Client IP - Ignore Whitespace",
			trueClientIP:  "   0101:: ",
			xForwardedFor: "0202::, 0303::",
			xRealIP:       "0404::",
			remoteAddr:    "[0505::]:5",
			matcher:       alwaysTrue,
			expectedIP:    net.ParseIP("0101::"),
			expectedVer:   iputil.IPv6,
		},
		{
			description:   "True Matcher - IPv6 - X Forwarded For",
			trueClientIP:  "",
			xForwardedFor: "0202::, 0303::",
			xRealIP:       "0404::",
			remoteAddr:    "[0505::]:5",
			matcher:       alwaysTrue,
			expectedIP:    net.ParseIP("0202::"),
			expectedVer:   iputil.IPv6,
		},
		{
			description:   "True Matcher - IPv6 - X Forwarded For - Ignore Whitespace",
			trueClientIP:  "",
			xForwardedFor: "   0202::, 0303:: ",
			xRealIP:       "0404::",
			remoteAddr:    "[0505::]:5",
			matcher:       alwaysTrue,
			expectedIP:    net.ParseIP("0202::"),
			expectedVer:   iputil.IPv6,
		},
		{
			description:   "True Matcher - IPv6 - X Real IP",
			trueClientIP:  "",
			xForwardedFor: "",
			xRealIP:       "0404::",
			remoteAddr:    "[0505::]:5",
			matcher:       alwaysTrue,
			expectedIP:    net.ParseIP("0404::"),
			expectedVer:   iputil.IPv6,
		},
		{
			description:   "True Matcher - IPv6 - X Real IP - Ignore Whitespace",
			trueClientIP:  "",
			xForwardedFor: "",
			xRealIP:       "   0404:: ",
			remoteAddr:    "[0505::]:5",
			matcher:       alwaysTrue,
			expectedIP:    net.ParseIP("0404::"),
			expectedVer:   iputil.IPv6,
		},
		{
			description:   "True Matcher - IPv6 - Remote Address",
			trueClientIP:  "",
			xForwardedFor: "",
			xRealIP:       "",
			remoteAddr:    "[0505::]:5",
			matcher:       alwaysTrue,
			expectedIP:    net.ParseIP("0505::"),
			expectedVer:   iputil.IPv6,
		},
		{
			description:   "True Matcher - Malformed - All",
			trueClientIP:  "malformed",
			xForwardedFor: "malformed",
			xRealIP:       "malformed",
			remoteAddr:    "malformed",
			matcher:       alwaysTrue,
			expectedIP:    nil,
			expectedVer:   iputil.IPvUnknown,
		},
		{
			description:   "True Matcher - Malformed - Some",
			trueClientIP:  "malformed",
			xForwardedFor: "malformed",
			xRealIP:       "4.4.4.4",
			remoteAddr:    "malformed",
			matcher:       alwaysTrue,
			expectedIP:    net.ParseIP("4.4.4.4"),
			expectedVer:   iputil.IPv4,
		},
		{
			description:   "True Matcher - Malformed - X Forwarded For - IPv4",
			trueClientIP:  "malformed",
			xForwardedFor: "malformed, 4.4.4.4, 0303::, malformed",
			xRealIP:       "malformed",
			remoteAddr:    "malformed",
			matcher:       alwaysTrue,
			expectedIP:    net.ParseIP("4.4.4.4"),
			expectedVer:   iputil.IPv4,
		},
		{
			description:   "True Matcher - Malformed - X Forwarded For - IPv6",
			trueClientIP:  "malformed",
			xForwardedFor: "malformed, 0303::, 4.4.4.4, malformed",
			xRealIP:       "malformed",
			remoteAddr:    "malformed",
			matcher:       alwaysTrue,
			expectedIP:    net.ParseIP("0303::"),
			expectedVer:   iputil.IPv6,
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
		ip, ver := FindIP(request, test.matcher)

		// Assertions
		assert.Equal(t, test.expectedIP, ip, test.description+":ip")
		assert.Equal(t, test.expectedVer, ver, test.description+":ver")
	}
}

type hardcodedResponseIPMatcher struct {
	response bool
}

func (m hardcodedResponseIPMatcher) Match(net.IP, iputil.IPVersion) bool {
	return m.response
}
