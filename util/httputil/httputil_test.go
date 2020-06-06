package httputil

import (
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSecure(t *testing.T) {
	testCases := []struct {
		description    string
		url            string
		headers        map[string]string
		tls            bool
		expectIsSecure bool
	}{
		{
			description:    "HTTP",
			url:            "http://host.com",
			expectIsSecure: false,
		},
		{
			description:    "HTTPS - Forwarded Protocol",
			url:            "http://host.com",
			headers:        map[string]string{"X-Forwarded-Proto": "https"},
			expectIsSecure: true,
		},
		{
			description:    "HTTPS - Forwarded Protocol - Case Insensitive",
			url:            "http://host.com",
			headers:        map[string]string{"X-Forwarded-Proto": "HTTPS"},
			expectIsSecure: true,
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
		for k, v := range test.headers {
			request.Header.Add(k, v)
		}
		if test.tls {
			request.TLS = &tls.ConnectionState{}
		}

		result := IsSecure(request)
		assert.Equal(t, test.expectIsSecure, result, test.description)
	}
}
