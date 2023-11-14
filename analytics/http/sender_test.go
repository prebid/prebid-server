package http

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prebid/prebid-server/v2/config"
	"github.com/stretchr/testify/assert"
)

func TestCreateHttpSender(t *testing.T) {
	testCases := []struct {
		name        string
		endpoint    config.AnalyticsHttpEndpoint
		wantHeaders http.Header
		wantErr     bool
	}{
		{
			name: "Test with invalid/empty URL",
			endpoint: config.AnalyticsHttpEndpoint{
				Url:     "%%2815197306101420000%29",
				Timeout: "1s",
				Gzip:    false,
			},
			wantErr: true,
		},
		{
			name: "Test with timeout",
			endpoint: config.AnalyticsHttpEndpoint{
				Url:     "http://localhost:8080",
				Timeout: "2x", // Very short timeout
				Gzip:    false,
			},
			wantErr: true,
		},
		{
			name: "Test with Gzip true",
			endpoint: config.AnalyticsHttpEndpoint{
				Url:               "http://localhost:8080",
				Timeout:           "1s",
				Gzip:              true,
				AdditionalHeaders: map[string]string{"X-Test-Header": "Test-Value"},
			},
			wantHeaders: http.Header{
				"X-Test-Header":    []string{"Test-Value"},
				"Content-Encoding": []string{"gzip"},
				"Content-Type":     []string{"application/json"},
			},
			wantErr: false,
		},
		{
			name: "Test with Gzip false",
			endpoint: config.AnalyticsHttpEndpoint{
				Url:     "http://localhost:8080",
				Timeout: "1s",
				Gzip:    false,
			},
			wantHeaders: http.Header{
				"Content-Type": []string{"application/json"},
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testBody := []byte("[{ \"type\": \"test\" }]")
			// Create a test server
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check the headers
				for name, wantValues := range tc.wantHeaders {
					assert.Equal(t, wantValues, r.Header[name], "Expected header '%s' to be '%v', got '%v'", name, wantValues, r.Header[name])
				}
				defer r.Body.Close()
				var reader io.ReadCloser
				var err error
				if tc.endpoint.Gzip {
					reader, err = gzip.NewReader(r.Body)
					assert.NoError(t, err)
					defer reader.Close()
				} else {
					reader = r.Body
				}

				decompressedData, err := io.ReadAll(reader)
				assert.NoError(t, err)

				assert.Equal(t, string(testBody), string(decompressedData))

				w.WriteHeader(http.StatusOK)
			}))
			defer ts.Close()

			// Update the URL of the endpoint to the URL of the test server
			if !tc.wantErr {
				tc.endpoint.Url = ts.URL
			}

			// Create a test client
			client := &http.Client{}

			// Test the createHttpSender function
			sender, err := createHttpSender(client, tc.endpoint)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// Test the returned HttpSender function
			err = sender([]byte(testBody))
			assert.NoError(t, err)
		})
	}
}
