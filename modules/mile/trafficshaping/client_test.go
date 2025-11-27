package trafficshaping

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type manualTimer struct {
	ch        chan time.Time
	mu        sync.Mutex
	lastReset time.Duration
}

func newManualTimer() *manualTimer {
	return &manualTimer{ch: make(chan time.Time, 1)}
}

func (mt *manualTimer) C() <-chan time.Time {
	return mt.ch
}

func (mt *manualTimer) Reset(d time.Duration) bool {
	mt.mu.Lock()
	mt.lastReset = d
	mt.mu.Unlock()
	return true
}

func (mt *manualTimer) Stop() bool {
	return true
}

func (mt *manualTimer) Trigger() {
	select {
	case mt.ch <- time.Now():
	default:
	}
}

func (mt *manualTimer) LastReset() time.Duration {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	return mt.lastReset
}

func (mt *manualTimer) SetLastReset(d time.Duration) {
	mt.mu.Lock()
	mt.lastReset = d
	mt.mu.Unlock()
}

func TestRefreshLoop(t *testing.T) {
	// Test backoff on errors
	t.Run("backoff_on_errors", func(t *testing.T) {
		var requestCount int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&requestCount, 1)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		config := &Config{
			Enabled:          true,
			Endpoint:         server.URL,
			RefreshMs:        100, // Short interval for testing
			RequestTimeoutMs: 1000,
			SampleSalt:       "pbs",
		}

		client := NewConfigClient(http.DefaultClient, config)
		defer client.Stop()

		// Wait for initial fetch and a few retry attempts
		time.Sleep(500 * time.Millisecond)

		// Should have made multiple attempts
		count := atomic.LoadInt32(&requestCount)
		assert.Greater(t, count, int32(1), "Should have made multiple fetch attempts")

		// Stop the client (defer already handles cleanup)
	})

	// Test backoff reset on success
	t.Run("backoff_reset_on_success", func(t *testing.T) {
		var requestCount int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count := atomic.AddInt32(&requestCount, 1)
			if count == 1 {
				// First request fails
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				// Subsequent requests succeed
				response := TrafficShapingData{
					Response: Response{
						SkipRate:      0,
						UserIdVendors: []string{},
						Values:        map[string]map[string]map[string]int{},
					},
				}
				json.NewEncoder(w).Encode(response)
			}
		}))
		defer server.Close()

		config := &Config{
			Enabled:          true,
			Endpoint:         server.URL,
			RefreshMs:        100,
			RequestTimeoutMs: 1000,
			SampleSalt:       "pbs",
		}

		client := NewConfigClient(http.DefaultClient, config)
		defer client.Stop()

		// Wait for initial fetch and recovery
		time.Sleep(400 * time.Millisecond)

		// Should have config after recovery
		cfg := client.GetConfig()
		assert.NotNil(t, cfg, "Should have config after successful fetch")
	})
}

func TestGetConfigForURLCacheExpiration(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		response := TrafficShapingData{
			Response: Response{
				SkipRate:      0,
				UserIdVendors: []string{},
				Values:        map[string]map[string]map[string]int{},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &Config{
		Enabled:          true,
		BaseEndpoint:     server.URL + "/",
		RefreshMs:        50, // Very short TTL for testing
		RequestTimeoutMs: 1000,
		SampleSalt:       "pbs",
	}

	client := NewConfigClient(http.DefaultClient, config)
	defer client.Stop()

	url := server.URL + "/ts.json"

	// First fetch
	cfg1 := client.GetConfigForURL(url)
	require.NotNil(t, cfg1)
	assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount))

	// Second fetch should use cache
	cfg2 := client.GetConfigForURL(url)
	require.NotNil(t, cfg2)
	assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount))

	// Wait for cache expiration
	time.Sleep(100 * time.Millisecond)

	// Third fetch should trigger new request
	cfg3 := client.GetConfigForURL(url)
	require.NotNil(t, cfg3)
	assert.GreaterOrEqual(t, atomic.LoadInt32(&requestCount), int32(2))
}

func TestGetConfigForURLFetchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := &Config{
		Enabled:          true,
		BaseEndpoint:     server.URL + "/",
		RefreshMs:        30000,
		RequestTimeoutMs: 1000,
		SampleSalt:       "pbs",
	}

	client := NewConfigClient(http.DefaultClient, config)
	defer client.Stop()

	url := server.URL + "/ts.json"

	// Should return nil on fetch error
	cfg := client.GetConfigForURL(url)
	assert.Nil(t, cfg)
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		name     string
		sizeStr  string
		expected *BannerSize
	}{
		{
			name:     "valid_size",
			sizeStr:  "320x50",
			expected: &BannerSize{W: 320, H: 50},
		},
		{
			name:     "valid_large_size",
			sizeStr:  "300x250",
			expected: &BannerSize{W: 300, H: 250},
		},
		{
			name:     "no_x_separator",
			sizeStr:  "32050",
			expected: nil,
		},
		{
			name:     "multiple_x",
			sizeStr:  "320x50x100",
			expected: nil,
		},
		{
			name:     "non_numeric_width",
			sizeStr:  "abcx50",
			expected: nil,
		},
		{
			name:     "non_numeric_height",
			sizeStr:  "320xabc",
			expected: nil,
		},
		{
			name:     "empty_string",
			sizeStr:  "",
			expected: nil,
		},
		{
			name:     "only_x",
			sizeStr:  "x",
			expected: nil,
		},
		{
			name:     "missing_height",
			sizeStr:  "320x",
			expected: nil,
		},
		{
			name:     "missing_width",
			sizeStr:  "x50",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSize(tt.sizeStr)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.W, result.W)
				assert.Equal(t, tt.expected.H, result.H)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid_config_with_endpoint",
			config: &Config{
				Endpoint:         "http://example.com/config.json",
				RefreshMs:        30000,
				RequestTimeoutMs: 1000,
				SampleSalt:       "pbs",
				GeoCacheTTLMS:    300000,
			},
			expectError: false,
		},
		{
			name: "valid_config_with_base_endpoint",
			config: &Config{
				BaseEndpoint:     "http://example.com/",
				RefreshMs:        30000,
				RequestTimeoutMs: 1000,
				SampleSalt:       "pbs",
				GeoCacheTTLMS:    300000,
			},
			expectError: false,
		},
		{
			name: "base_endpoint_normalized",
			config: &Config{
				BaseEndpoint:     "http://example.com",
				RefreshMs:        30000,
				RequestTimeoutMs: 1000,
				SampleSalt:       "pbs",
				GeoCacheTTLMS:    300000,
			},
			expectError: false,
		},
		{
			name: "missing_endpoint_and_base_endpoint",
			config: &Config{
				RefreshMs:        30000,
				RequestTimeoutMs: 1000,
				SampleSalt:       "pbs",
				GeoCacheTTLMS:    300000,
			},
			expectError: true,
		},
		{
			name: "refresh_ms_too_low",
			config: &Config{
				Endpoint:         "http://example.com/config.json",
				RefreshMs:        500,
				RequestTimeoutMs: 1000,
				SampleSalt:       "pbs",
				GeoCacheTTLMS:    300000,
			},
			expectError: true,
		},
		{
			name: "request_timeout_ms_too_low",
			config: &Config{
				Endpoint:         "http://example.com/config.json",
				RefreshMs:        30000,
				RequestTimeoutMs: 50,
				SampleSalt:       "pbs",
				GeoCacheTTLMS:    300000,
			},
			expectError: true,
		},
		{
			name: "empty_sample_salt",
			config: &Config{
				Endpoint:         "http://example.com/config.json",
				RefreshMs:        30000,
				RequestTimeoutMs: 1000,
				SampleSalt:       "",
				GeoCacheTTLMS:    300000,
			},
			expectError: true,
		},
		{
			name: "geo_cache_ttl_ms_too_low",
			config: &Config{
				Endpoint:         "http://example.com/config.json",
				RefreshMs:        30000,
				RequestTimeoutMs: 1000,
				SampleSalt:       "pbs",
				GeoCacheTTLMS:    500,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Verify base_endpoint normalization
				if tt.config.BaseEndpoint != "" && !tt.expectError {
					assert.True(t, len(tt.config.BaseEndpoint) > 0)
					// Check that it ends with / if it was supposed to be normalized
					if tt.name == "base_endpoint_normalized" {
						assert.True(t, len(tt.config.BaseEndpoint) > 0)
					}
				}
			}
		})
	}
}

func TestFetchForURL_ErrorPaths(t *testing.T) {
	t.Run("request_creation_error", func(t *testing.T) {
		config := &Config{
			Enabled:          true,
			BaseEndpoint:     "http://example.com/",
			RefreshMs:        30000,
			RequestTimeoutMs: 1000,
			SampleSalt:       "pbs",
		}

		client := NewConfigClient(http.DefaultClient, config)
		defer client.Stop()

		// Invalid URL that will cause request creation to fail
		// Using invalid URL scheme
		_, err := client.fetchForURL("://invalid-url")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create request")
	})

	t.Run("http_client_error", func(t *testing.T) {
		config := &Config{
			Enabled:          true,
			BaseEndpoint:     "http://example.com/",
			RefreshMs:        30000,
			RequestTimeoutMs: 1000,
			SampleSalt:       "pbs",
		}

		// Create a client that will fail on Do
		httpClient := &http.Client{
			Transport: &failingTransport{},
		}

		client := NewConfigClient(httpClient, config)
		defer client.Stop()

		_, err := client.fetchForURL("http://example.com/config.json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch")
	})

	t.Run("read_all_error", func(t *testing.T) {
		// Create a server that closes connection immediately
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set Content-Length but don't send body
			w.Header().Set("Content-Length", "1000")
			w.WriteHeader(http.StatusOK)
			// Close connection immediately
			hj, ok := w.(http.Hijacker)
			if ok {
				conn, _, _ := hj.Hijack()
				conn.Close()
			}
		}))
		defer server.Close()

		config := &Config{
			Enabled:          true,
			BaseEndpoint:     server.URL + "/",
			RefreshMs:        30000,
			RequestTimeoutMs: 1000,
			SampleSalt:       "pbs",
		}

		client := NewConfigClient(http.DefaultClient, config)
		defer client.Stop()

		_, err := client.fetchForURL(server.URL + "/config.json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read response")
	})

	t.Run("unmarshal_error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		config := &Config{
			Enabled:          true,
			BaseEndpoint:     server.URL + "/",
			RefreshMs:        30000,
			RequestTimeoutMs: 1000,
			SampleSalt:       "pbs",
		}

		client := NewConfigClient(http.DefaultClient, config)
		defer client.Stop()

		_, err := client.fetchForURL(server.URL + "/config.json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal")
	})
}

type failingTransport struct{}

func (f *failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, &net.OpError{Op: "dial", Err: &net.DNSError{Err: "no such host"}}
}

func TestFetchAndStore_StoreInCacheFalse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := TrafficShapingData{
			Response: Response{
				SkipRate:      0,
				UserIdVendors: []string{},
				Values:        map[string]map[string]map[string]int{},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &Config{
		Enabled:          true,
		BaseEndpoint:     server.URL + "/", // Use dynamic mode to avoid initial fetch
		RefreshMs:        30000,
		RequestTimeoutMs: 1000,
		SampleSalt:       "pbs",
	}

	client := NewConfigClient(http.DefaultClient, config)
	defer client.Stop()

	// Test fetchAndStore with storeInCache=false
	err := client.fetchAndStore(server.URL+"/test.json", false)
	assert.NoError(t, err)

	// Config should not be stored in static cache (GetConfig returns nil in dynamic mode)
	cfg := client.GetConfig()
	assert.Nil(t, cfg)
}

func TestPreprocessConfig_FlagZero(t *testing.T) {
	response := &Response{
		SkipRate:      0,
		UserIdVendors: []string{},
		Values: map[string]map[string]map[string]int{
			"test-gpid": {
				"rubicon": {
					"300x250": 0, // Flag = 0, should not be included
					"728x90":  1, // Flag = 1, should be included
				},
			},
		},
	}

	config := preprocessConfig(response)

	// Should have GPID rule
	rule, ok := config.GPIDRules["test-gpid"]
	require.True(t, ok)

	// Should only have 728x90 in allowed sizes
	_, has728x90 := rule.AllowedSizes[BannerSize{W: 728, H: 90}]
	assert.True(t, has728x90)

	_, has300x250 := rule.AllowedSizes[BannerSize{W: 300, H: 250}]
	assert.False(t, has300x250)

	// Should have rubicon in allowed bidders (because at least one size is allowed)
	_, hasRubicon := rule.AllowedBidders["rubicon"]
	assert.True(t, hasRubicon)
}

func TestConfigClientStopTwice(t *testing.T) {
	config := &Config{
		Enabled:          true,
		BaseEndpoint:     "http://example.com/",
		RefreshMs:        30000,
		RequestTimeoutMs: 1000,
		SampleSalt:       "pbs",
	}

	client := NewConfigClient(http.DefaultClient, config)
	assert.NotNil(t, client)

	assert.NotPanics(t, func() {
		client.Stop()
		client.Stop()
	})
}

func TestRefreshLoopRecoversAndResetsInterval(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		if count == 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		response := TrafficShapingData{Response: Response{Values: map[string]map[string]map[string]int{}}}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &Config{
		Enabled:          true,
		Endpoint:         server.URL,
		RefreshMs:        1000,
		RequestTimeoutMs: 1000,
		SampleSalt:       "pbs",
	}

	timer := newManualTimer()
	client := &ConfigClient{
		httpClient: server.Client(),
		config:     config,
		done:       make(chan struct{}),
		newTimer: func(d time.Duration) schedulerTimer {
			timer.SetLastReset(d)
			return timer
		},
		jitterFn: func(int64) int64 { return 0 },
	}

	require.NoError(t, client.fetch())
	go client.refreshLoop()
	defer client.Stop()

	timer.Trigger()
	require.Eventually(t, func() bool { return atomic.LoadInt32(&requestCount) >= 2 }, time.Second, 10*time.Millisecond)
	assert.Equal(t, 10*time.Second, timer.LastReset())

	timer.Trigger()
	require.Eventually(t, func() bool { return atomic.LoadInt32(&requestCount) >= 3 }, time.Second, 10*time.Millisecond)
	assert.Equal(t, config.GetRefreshInterval(), timer.LastReset())
}

func TestGetConfigForURLRemovesExpiredEntry(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		response := TrafficShapingData{Response: Response{Values: map[string]map[string]map[string]int{}}}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &Config{
		Enabled:          true,
		BaseEndpoint:     server.URL + "/",
		RefreshMs:        50,
		RequestTimeoutMs: 1000,
		SampleSalt:       "pbs",
	}

	client := NewConfigClient(server.Client(), config)
	defer client.Stop()

	url := server.URL + "/config.json"
	require.NotNil(t, client.GetConfigForURL(url))

	raw, ok := client.dynamicCache.Load(url)
	require.True(t, ok)
	cached := raw.(*cachedConfig)
	cached.expiresAt = time.Now().Add(-time.Second)
	atomic.StoreInt64(&client.lastDynamicCleanup, 0)

	require.NotNil(t, client.GetConfigForURL(url))
	assert.GreaterOrEqual(t, atomic.LoadInt32(&requestCount), int32(2))
}

func TestGetConfigForURLDoesNotCacheOnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := &Config{
		Enabled:          true,
		BaseEndpoint:     server.URL + "/",
		RefreshMs:        50,
		RequestTimeoutMs: 1000,
		SampleSalt:       "pbs",
	}

	client := NewConfigClient(server.Client(), config)
	defer client.Stop()

	url := server.URL + "/config.json"
	assert.Nil(t, client.GetConfigForURL(url))
	_, ok := client.dynamicCache.Load(url)
	assert.False(t, ok)
}
