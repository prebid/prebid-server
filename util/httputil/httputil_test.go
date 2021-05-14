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
	alwaysTrue := hardcodedResponseIPValidator{response: true}
	alwaysFalse := hardcodedResponseIPValidator{response: false}

	testCases := []struct {
		description   string
		trueClientIP  string
		xForwardedFor string
		xRealIP       string
		remoteAddr    string
		validator     iputil.IPValidator
		expectedIP    net.IP
		expectedVer   iputil.IPVersion
	}{
		{
			description: "No Address",
			expectedIP:  nil,
			expectedVer: iputil.IPvUnknown,
		},
		{
			description:   "False Validator - IPv4",
			trueClientIP:  "1.1.1.1",
			xForwardedFor: "2.2.2.2, 3.3.3.3",
			xRealIP:       "4.4.4.4",
			remoteAddr:    "5.5.5.5:5",
			validator:     alwaysFalse,
			expectedIP:    nil,
			expectedVer:   iputil.IPvUnknown,
		},
		{
			description:   "False Validator - IPv6",
			trueClientIP:  "1111::",
			xForwardedFor: "2222::, 3333::",
			xRealIP:       "4444::",
			remoteAddr:    "[5555::]:5]",
			validator:     alwaysFalse,
			expectedIP:    nil,
			expectedVer:   iputil.IPvUnknown,
		},
		{
			description:   "True Validator - IPv4 - True Client IP",
			trueClientIP:  "1.1.1.1",
			xForwardedFor: "2.2.2.2, 3.3.3.3",
			xRealIP:       "4.4.4.4",
			remoteAddr:    "5.5.5.5:5",
			validator:     alwaysTrue,
			expectedIP:    net.ParseIP("1.1.1.1"),
			expectedVer:   iputil.IPv4,
		},
		{
			description:   "True Validator - IPv4 - True Client IP - Ignore Whitespace",
			trueClientIP:  "   1.1.1.1 ",
			xForwardedFor: "2.2.2.2, 3.3.3.3",
			xRealIP:       "4.4.4.4",
			remoteAddr:    "5.5.5.5:5",
			validator:     alwaysTrue,
			expectedIP:    net.ParseIP("1.1.1.1"),
			expectedVer:   iputil.IPv4,
		},
		{
			description:   "True Validator - IPv4 - X Forwarded For",
			trueClientIP:  "",
			xForwardedFor: "2.2.2.2, 3.3.3.3",
			xRealIP:       "4.4.4.4",
			remoteAddr:    "5.5.5.5:5",
			validator:     alwaysTrue,
			expectedIP:    net.ParseIP("2.2.2.2"),
			expectedVer:   iputil.IPv4,
		},
		{
			description:   "True Validator - IPv4 - X Forwarded For - Ignore Whitespace",
			trueClientIP:  "",
			xForwardedFor: "   2.2.2.2, 3.3.3.3 ",
			xRealIP:       "4.4.4.4",
			remoteAddr:    "5.5.5.5:5",
			validator:     alwaysTrue,
			expectedIP:    net.ParseIP("2.2.2.2"),
			expectedVer:   iputil.IPv4,
		},
		{
			description:   "True Validator - IPv4 - X Real IP",
			trueClientIP:  "",
			xForwardedFor: "",
			xRealIP:       "4.4.4.4",
			remoteAddr:    "5.5.5.5:5",
			validator:     alwaysTrue,
			expectedIP:    net.ParseIP("4.4.4.4"),
			expectedVer:   iputil.IPv4,
		},
		{
			description:   "True Validator - IPv4 - X Real IP - Ignore Whitespace",
			trueClientIP:  "",
			xForwardedFor: "",
			xRealIP:       "   4.4.4.4 ",
			remoteAddr:    "5.5.5.5:5",
			validator:     alwaysTrue,
			expectedIP:    net.ParseIP("4.4.4.4"),
			expectedVer:   iputil.IPv4,
		},
		{
			description:   "True Validator - IPv4 - Remote Address",
			trueClientIP:  "",
			xForwardedFor: "",
			xRealIP:       "",
			remoteAddr:    "5.5.5.5:80",
			validator:     alwaysTrue,
			expectedIP:    net.ParseIP("5.5.5.5"),
			expectedVer:   iputil.IPv4,
		},
		{
			description:   "True Validator - IPv6 - True Client IP",
			trueClientIP:  "1111::",
			xForwardedFor: "2222::, 3333::",
			xRealIP:       "4444::",
			remoteAddr:    "[5555::]:5",
			validator:     alwaysTrue,
			expectedIP:    net.ParseIP("1111::"),
			expectedVer:   iputil.IPv6,
		},
		{
			description:   "True Validator - IPv6 - True Client IP - Ignore Whitespace",
			trueClientIP:  "   1111:: ",
			xForwardedFor: "2222::, 3333::",
			xRealIP:       "4444::",
			remoteAddr:    "[5555::]:5",
			validator:     alwaysTrue,
			expectedIP:    net.ParseIP("1111::"),
			expectedVer:   iputil.IPv6,
		},
		{
			description:   "True Validator - IPv6 - X Forwarded For",
			trueClientIP:  "",
			xForwardedFor: "2222::, 3333::",
			xRealIP:       "4444::",
			remoteAddr:    "[5555::]:5",
			validator:     alwaysTrue,
			expectedIP:    net.ParseIP("2222::"),
			expectedVer:   iputil.IPv6,
		},
		{
			description:   "True Validator - IPv6 - X Forwarded For - Ignore Whitespace",
			trueClientIP:  "",
			xForwardedFor: "   2222::, 3333:: ",
			xRealIP:       "4444::",
			remoteAddr:    "[5555::]:5",
			validator:     alwaysTrue,
			expectedIP:    net.ParseIP("2222::"),
			expectedVer:   iputil.IPv6,
		},
		{
			description:   "True Validator - IPv6 - X Real IP",
			trueClientIP:  "",
			xForwardedFor: "",
			xRealIP:       "4444::",
			remoteAddr:    "[5555::]:5",
			validator:     alwaysTrue,
			expectedIP:    net.ParseIP("4444::"),
			expectedVer:   iputil.IPv6,
		},
		{
			description:   "True Validator - IPv6 - X Real IP - Ignore Whitespace",
			trueClientIP:  "",
			xForwardedFor: "",
			xRealIP:       "    4444:: ",
			remoteAddr:    "[5555::]:5",
			validator:     alwaysTrue,
			expectedIP:    net.ParseIP("4444::"),
			expectedVer:   iputil.IPv6,
		},
		{
			description:   "True Validator - IPv6 - Remote Address",
			trueClientIP:  "",
			xForwardedFor: "",
			xRealIP:       "",
			remoteAddr:    "[5555::]:5",
			validator:     alwaysTrue,
			expectedIP:    net.ParseIP("5555::"),
			expectedVer:   iputil.IPv6,
		},
		{
			description:   "True Validator - Malformed - All",
			trueClientIP:  "malformed",
			xForwardedFor: "malformed",
			xRealIP:       "malformed",
			remoteAddr:    "malformed",
			validator:     alwaysTrue,
			expectedIP:    nil,
			expectedVer:   iputil.IPvUnknown,
		},
		{
			description:   "True Validator - Malformed - Some",
			trueClientIP:  "malformed",
			xForwardedFor: "malformed",
			xRealIP:       "4.4.4.4",
			remoteAddr:    "malformed",
			validator:     alwaysTrue,
			expectedIP:    net.ParseIP("4.4.4.4"),
			expectedVer:   iputil.IPv4,
		},
		{
			description:   "True Validator - Malformed - X Forwarded For - IPv4",
			trueClientIP:  "malformed",
			xForwardedFor: "malformed, 4.4.4.4, 3333::, malformed",
			xRealIP:       "malformed",
			remoteAddr:    "malformed",
			validator:     alwaysTrue,
			expectedIP:    net.ParseIP("4.4.4.4"),
			expectedVer:   iputil.IPv4,
		},
		{
			description:   "True Validator - Malformed - X Forwarded For - IPv6",
			trueClientIP:  "malformed",
			xForwardedFor: "malformed, 3333::, 4.4.4.4, malformed",
			xRealIP:       "malformed",
			remoteAddr:    "malformed",
			validator:     alwaysTrue,
			expectedIP:    net.ParseIP("3333::"),
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
		ip, ver := FindIP(request, test.validator)

		// Assertions
		assert.Equal(t, test.expectedIP, ip, test.description+":ip")
		assert.Equal(t, test.expectedVer, ver, test.description+":ver")
	}
}

type hardcodedResponseIPValidator struct {
	response bool
}

func (v hardcodedResponseIPValidator) IsValid(net.IP, iputil.IPVersion) bool {
	return v.response
}
