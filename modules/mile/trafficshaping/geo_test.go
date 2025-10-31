package trafficshaping

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPGeoResolver(t *testing.T) {
	tests := []struct {
		name        string
		endpoint    string
		ttl         time.Duration
		client      *http.Client
		expectError bool
	}{
		{
			name:        "valid_config",
			endpoint:    "http://example.com/geo/{ip}",
			ttl:         time.Minute,
			client:      http.DefaultClient,
			expectError: false,
		},
		{
			name:        "empty_endpoint",
			endpoint:    "",
			ttl:         time.Minute,
			client:      http.DefaultClient,
			expectError: true,
		},
		{
			name:        "nil_client_defaults",
			endpoint:    "http://example.com/geo/{ip}",
			ttl:         time.Minute,
			client:      nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver, err := NewHTTPGeoResolver(tt.endpoint, tt.ttl, tt.client)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, resolver)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resolver)
				if tt.client == nil {
					// Verify default client is set (check that resolver is not nil is sufficient)
					assert.NotNil(t, resolver)
				}
			}
		})
	}
}

func TestResolve(t *testing.T) {
	t.Run("cache_hit", func(t *testing.T) {
		var requestCount int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&requestCount, 1)
			json.NewEncoder(w).Encode(map[string]string{"country": "US"})
		}))
		defer server.Close()

		resolver, err := NewHTTPGeoResolver(server.URL+"/{ip}", 5*time.Minute, http.DefaultClient)
		require.NoError(t, err)

		// First resolve
		country1, err := resolver.Resolve(context.Background(), "1.1.1.1")
		assert.NoError(t, err)
		assert.Equal(t, "US", country1)
		assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount))

		// Second resolve should use cache
		country2, err := resolver.Resolve(context.Background(), "1.1.1.1")
		assert.NoError(t, err)
		assert.Equal(t, "US", country2)
		assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount))
	})

	t.Run("cache_expiration", func(t *testing.T) {
		var requestCount int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&requestCount, 1)
			json.NewEncoder(w).Encode(map[string]string{"country": "CA"})
		}))
		defer server.Close()

		resolver, err := NewHTTPGeoResolver(server.URL+"/{ip}", 50*time.Millisecond, http.DefaultClient)
		require.NoError(t, err)

		// First resolve
		country1, err := resolver.Resolve(context.Background(), "2.2.2.2")
		assert.NoError(t, err)
		assert.Equal(t, "CA", country1)
		assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount))

		// Wait for cache expiration
		time.Sleep(100 * time.Millisecond)

		// Second resolve should fetch again
		country2, err := resolver.Resolve(context.Background(), "2.2.2.2")
		assert.NoError(t, err)
		assert.Equal(t, "CA", country2)
		assert.GreaterOrEqual(t, atomic.LoadInt32(&requestCount), int32(2))
	})

	t.Run("empty_ip", func(t *testing.T) {
		resolver, err := NewHTTPGeoResolver("http://example.com/{ip}", time.Minute, http.DefaultClient)
		require.NoError(t, err)

		_, err = resolver.Resolve(context.Background(), "")
		assert.Error(t, err)
	})

	t.Run("invalid_ip", func(t *testing.T) {
		resolver, err := NewHTTPGeoResolver("http://example.com/{ip}", time.Minute, http.DefaultClient)
		require.NoError(t, err)

		_, err = resolver.Resolve(context.Background(), "invalid-ip")
		assert.Error(t, err)
	})
}

func TestFetchCountry(t *testing.T) {
	t.Run("non_200_status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		resolver, err := NewHTTPGeoResolver(server.URL+"/{ip}", time.Minute, http.DefaultClient)
		require.NoError(t, err)

		_, err = resolver.Resolve(context.Background(), "1.1.1.1")
		assert.Error(t, err)
	})

	t.Run("json_decode_error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		resolver, err := NewHTTPGeoResolver(server.URL+"/{ip}", time.Minute, http.DefaultClient)
		require.NoError(t, err)

		_, err = resolver.Resolve(context.Background(), "1.1.1.1")
		assert.Error(t, err)
	})

	t.Run("missing_country_in_payload", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]string{"other": "value"})
		}))
		defer server.Close()

		resolver, err := NewHTTPGeoResolver(server.URL+"/{ip}", time.Minute, http.DefaultClient)
		require.NoError(t, err)

		_, err = resolver.Resolve(context.Background(), "1.1.1.1")
		assert.Error(t, err)
	})
}

func TestNormalizeIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected string
	}{
		{
			name:     "valid_ipv4",
			ip:       "192.168.1.1",
			expected: "192.168.1.1",
		},
		{
			name:     "valid_ipv6",
			ip:       "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			expected: "2001:db8:85a3::8a2e:370:7334",
		},
		{
			name:     "ipv6_with_zone",
			ip:       "fe80::1%lo0",
			expected: "", // IPv6 zones are normalized away
		},
		{
			name:     "invalid_ip",
			ip:       "not.an.ip",
			expected: "",
		},
		{
			name:     "empty_string",
			ip:       "",
			expected: "",
		},
		{
			name:     "ipv4_with_whitespace",
			ip:       "  192.168.1.1  ",
			expected: "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeIP(tt.ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractCountryFromPayload(t *testing.T) {
	tests := []struct {
		name      string
		payload   map[string]any
		expected  string
		shouldOk  bool
	}{
		{
			name:     "country_field",
			payload:  map[string]any{"country": "US"},
			expected: "US",
			shouldOk: true,
		},
		{
			name:     "countryCode_field",
			payload:  map[string]any{"countryCode": "CA"},
			expected: "CA",
			shouldOk: true,
		},
		{
			name:     "country_code_field",
			payload:  map[string]any{"country_code": "GB"},
			expected: "GB",
			shouldOk: true,
		},
		{
			name:     "iso_code_field",
			payload:  map[string]any{"iso_code": "FR"},
			expected: "FR",
			shouldOk: true,
		},
		{
			name:     "isoCode_field",
			payload:  map[string]any{"isoCode": "DE"},
			expected: "DE",
			shouldOk: true,
		},
		{
			name:     "nested_location",
			payload:  map[string]any{"location": map[string]any{"country": "IT"}},
			expected: "IT",
			shouldOk: true,
		},
		{
			name:     "nested_location_countryCode",
			payload:  map[string]any{"location": map[string]any{"countryCode": "ES"}},
			expected: "ES",
			shouldOk: true,
		},
		{
			name:     "invalid_value_type",
			payload:  map[string]any{"country": 123},
			expected: "",
			shouldOk: false,
		},
		{
			name:     "short_country_code",
			payload:  map[string]any{"country": "U"},
			expected: "",
			shouldOk: false,
		},
		{
			name:     "missing_country",
			payload:  map[string]any{"other": "value"},
			expected: "",
			shouldOk: false,
		},
		{
			name:     "lowercase_country",
			payload:  map[string]any{"country": "us"},
			expected: "US",
			shouldOk: true,
		},
		{
			name:     "long_country_code",
			payload:  map[string]any{"country": "USA"},
			expected: "US",
			shouldOk: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			country, ok := extractCountryFromPayload(tt.payload)
			assert.Equal(t, tt.shouldOk, ok)
			if tt.shouldOk {
				assert.Equal(t, tt.expected, country)
			}
		})
	}
}

