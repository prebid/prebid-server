package trafficshaping

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWhitelistClient_IsAllowed(t *testing.T) {
	tests := []struct {
		name        string
		geoWL       *GeoWhitelist
		platformWL  *PlatformWhitelist
		siteID      string
		country     string
		platform    string
		expected    bool
		description string
	}{
		{
			name:        "nil_whitelists_fail_open",
			geoWL:       nil,
			platformWL:  nil,
			siteID:      "site1",
			country:     "US",
			platform:    "m-android|chrome",
			expected:    true,
			description: "Should allow when whitelists are not loaded",
		},
		{
			name:        "nil_geo_whitelist_fail_open",
			geoWL:       nil,
			platformWL:  &PlatformWhitelist{"site1": {"m-android|chrome": {}}},
			siteID:      "site1",
			country:     "US",
			platform:    "m-android|chrome",
			expected:    true,
			description: "Should allow when geo whitelist is not loaded",
		},
		{
			name:        "nil_platform_whitelist_fail_open",
			geoWL:       &GeoWhitelist{"site1": {"US": {}}},
			platformWL:  nil,
			siteID:      "site1",
			country:     "US",
			platform:    "m-android|chrome",
			expected:    true,
			description: "Should allow when platform whitelist is not loaded",
		},
		{
			name:        "site_not_in_geo_whitelist_skip_shaping",
			geoWL:       &GeoWhitelist{"other_site": {"US": {}}},
			platformWL:  &PlatformWhitelist{"other_site": {"m-android|chrome": {}}},
			siteID:      "site1",
			country:     "US",
			platform:    "m-android|chrome",
			expected:    false,
			description: "Should skip shaping when site not in geo whitelist",
		},
		{
			name:        "site_not_in_platform_whitelist_skip_shaping",
			geoWL:       &GeoWhitelist{"site1": {"US": {}}},
			platformWL:  &PlatformWhitelist{"other_site": {"m-android|chrome": {}}},
			siteID:      "site1",
			country:     "US",
			platform:    "m-android|chrome",
			expected:    false,
			description: "Should skip shaping when site not in platform whitelist",
		},
		{
			name:        "geo_and_platform_match",
			geoWL:       &GeoWhitelist{"site1": {"US": {}, "CA": {}}},
			platformWL:  &PlatformWhitelist{"site1": {"m-android|chrome": {}, "w|safari": {}}},
			siteID:      "site1",
			country:     "US",
			platform:    "m-android|chrome",
			expected:    true,
			description: "Should allow when both geo and platform match",
		},
		{
			name:        "geo_matches_platform_not",
			geoWL:       &GeoWhitelist{"site1": {"US": {}}},
			platformWL:  &PlatformWhitelist{"site1": {"w|safari": {}}},
			siteID:      "site1",
			country:     "US",
			platform:    "m-android|chrome",
			expected:    false,
			description: "Should deny when geo matches but platform doesn't",
		},
		{
			name:        "platform_matches_geo_not",
			geoWL:       &GeoWhitelist{"site1": {"CA": {}}},
			platformWL:  &PlatformWhitelist{"site1": {"m-android|chrome": {}}},
			siteID:      "site1",
			country:     "US",
			platform:    "m-android|chrome",
			expected:    false,
			description: "Should deny when platform matches but geo doesn't",
		},
		{
			name:        "neither_geo_nor_platform_match",
			geoWL:       &GeoWhitelist{"site1": {"CA": {}}},
			platformWL:  &PlatformWhitelist{"site1": {"w|safari": {}}},
			siteID:      "site1",
			country:     "US",
			platform:    "m-android|chrome",
			expected:    false,
			description: "Should deny when neither geo nor platform match",
		},
		{
			name:        "empty_site_id_fail_open",
			geoWL:       &GeoWhitelist{"site1": {"US": {}}},
			platformWL:  &PlatformWhitelist{"site1": {"m-android|chrome": {}}},
			siteID:      "",
			country:     "US",
			platform:    "m-android|chrome",
			expected:    true,
			description: "Should allow when site ID is empty",
		},
		{
			name:        "multiple_sites_correct_match",
			geoWL:       &GeoWhitelist{"site1": {"US": {}}, "site2": {"CA": {}}},
			platformWL:  &PlatformWhitelist{"site1": {"m-android|chrome": {}}, "site2": {"w|safari": {}}},
			siteID:      "site2",
			country:     "CA",
			platform:    "w|safari",
			expected:    true,
			description: "Should match correct site in multi-site whitelist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &WhitelistClient{}
			if tt.geoWL != nil {
				client.geoWhitelist.Store(tt.geoWL)
			}
			if tt.platformWL != nil {
				client.platformWhitelist.Store(tt.platformWL)
			}

			result := client.IsAllowed(tt.siteID, tt.country, tt.platform)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestWhitelistClient_FetchGeoWhitelist(t *testing.T) {
	t.Run("successful_fetch", func(t *testing.T) {
		geoData := map[string][]string{
			"site1": {"US", "CA"},
			"site2": {"GB"},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(geoData)
		}))
		defer server.Close()

		config := &Config{
			GeoWhitelistEndpoint:      server.URL,
			PlatformWhitelistEndpoint: server.URL,
			WhitelistRefreshMs:        300000,
			RequestTimeoutMs:          1000,
		}

		client := &WhitelistClient{
			httpClient: http.DefaultClient,
			config:     config,
		}

		err := client.fetchGeoWhitelist()
		require.NoError(t, err)

		geoWL := client.geoWhitelist.Load()
		require.NotNil(t, geoWL)

		// Check site1
		site1Geos := (*geoWL)["site1"]
		assert.Contains(t, site1Geos, "US")
		assert.Contains(t, site1Geos, "CA")

		// Check site2
		site2Geos := (*geoWL)["site2"]
		assert.Contains(t, site2Geos, "GB")
	})

	t.Run("fetch_error_non_200", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		config := &Config{
			GeoWhitelistEndpoint:      server.URL,
			PlatformWhitelistEndpoint: server.URL,
			WhitelistRefreshMs:        300000,
			RequestTimeoutMs:          1000,
		}

		client := &WhitelistClient{
			httpClient: http.DefaultClient,
			config:     config,
		}

		err := client.fetchGeoWhitelist()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected status code")
	})

	t.Run("fetch_error_invalid_json", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		config := &Config{
			GeoWhitelistEndpoint:      server.URL,
			PlatformWhitelistEndpoint: server.URL,
			WhitelistRefreshMs:        300000,
			RequestTimeoutMs:          1000,
		}

		client := &WhitelistClient{
			httpClient: http.DefaultClient,
			config:     config,
		}

		err := client.fetchGeoWhitelist()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal")
	})
}

func TestWhitelistClient_FetchPlatformWhitelist(t *testing.T) {
	t.Run("successful_fetch", func(t *testing.T) {
		platformData := map[string][]string{
			"site1": {"m-android|chrome", "w|safari"},
			"site2": {"m-ios|safari", "t-ios|chrome"},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(platformData)
		}))
		defer server.Close()

		config := &Config{
			GeoWhitelistEndpoint:      server.URL,
			PlatformWhitelistEndpoint: server.URL,
			WhitelistRefreshMs:        300000,
			RequestTimeoutMs:          1000,
		}

		client := &WhitelistClient{
			httpClient: http.DefaultClient,
			config:     config,
		}

		err := client.fetchPlatformWhitelist()
		require.NoError(t, err)

		platformWL := client.platformWhitelist.Load()
		require.NotNil(t, platformWL)

		// Check site1
		site1Platforms := (*platformWL)["site1"]
		assert.Contains(t, site1Platforms, "m-android|chrome")
		assert.Contains(t, site1Platforms, "w|safari")

		// Check site2
		site2Platforms := (*platformWL)["site2"]
		assert.Contains(t, site2Platforms, "m-ios|safari")
		assert.Contains(t, site2Platforms, "t-ios|chrome")
	})
}

func TestWhitelistClient_FetchAll(t *testing.T) {
	t.Run("both_succeed", func(t *testing.T) {
		geoData := map[string][]string{"site1": {"US"}}
		platformData := map[string][]string{"site1": {"w|chrome"}}

		geoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(geoData)
		}))
		defer geoServer.Close()

		platformServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(platformData)
		}))
		defer platformServer.Close()

		config := &Config{
			GeoWhitelistEndpoint:      geoServer.URL,
			PlatformWhitelistEndpoint: platformServer.URL,
			WhitelistRefreshMs:        300000,
			RequestTimeoutMs:          1000,
		}

		client := &WhitelistClient{
			httpClient: http.DefaultClient,
			config:     config,
		}

		err := client.fetchAll()
		assert.NoError(t, err)

		assert.NotNil(t, client.geoWhitelist.Load())
		assert.NotNil(t, client.platformWhitelist.Load())
	})

	t.Run("geo_fails_platform_succeeds", func(t *testing.T) {
		platformData := map[string][]string{"site1": {"w|chrome"}}

		geoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer geoServer.Close()

		platformServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(platformData)
		}))
		defer platformServer.Close()

		config := &Config{
			GeoWhitelistEndpoint:      geoServer.URL,
			PlatformWhitelistEndpoint: platformServer.URL,
			WhitelistRefreshMs:        300000,
			RequestTimeoutMs:          1000,
		}

		client := &WhitelistClient{
			httpClient: http.DefaultClient,
			config:     config,
		}

		err := client.fetchAll()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "geo whitelist fetch failed")
	})

	t.Run("both_fail", func(t *testing.T) {
		geoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer geoServer.Close()

		platformServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer platformServer.Close()

		config := &Config{
			GeoWhitelistEndpoint:      geoServer.URL,
			PlatformWhitelistEndpoint: platformServer.URL,
			WhitelistRefreshMs:        300000,
			RequestTimeoutMs:          1000,
		}

		client := &WhitelistClient{
			httpClient: http.DefaultClient,
			config:     config,
		}

		err := client.fetchAll()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "both fetches failed")
	})
}

func TestWhitelistClient_RefreshLoop(t *testing.T) {
	t.Run("refresh_on_interval", func(t *testing.T) {
		var fetchCount int32
		geoData := map[string][]string{"site1": {"US"}}
		platformData := map[string][]string{"site1": {"w|chrome"}}

		geoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&fetchCount, 1)
			json.NewEncoder(w).Encode(geoData)
		}))
		defer geoServer.Close()

		platformServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&fetchCount, 1)
			json.NewEncoder(w).Encode(platformData)
		}))
		defer platformServer.Close()

		config := &Config{
			GeoWhitelistEndpoint:      geoServer.URL,
			PlatformWhitelistEndpoint: platformServer.URL,
			WhitelistRefreshMs:        100, // Short interval for testing
			RequestTimeoutMs:          1000,
		}

		client := NewWhitelistClient(http.DefaultClient, config)
		defer client.Stop()

		// Wait for initial fetch + at least one refresh
		time.Sleep(300 * time.Millisecond)

		count := atomic.LoadInt32(&fetchCount)
		// Initial fetch (2 requests) + at least one refresh cycle (2 more)
		assert.GreaterOrEqual(t, count, int32(4), "Should have made multiple fetch attempts")
	})

	t.Run("stop_terminates_loop", func(t *testing.T) {
		geoData := map[string][]string{"site1": {"US"}}
		platformData := map[string][]string{"site1": {"w|chrome"}}

		geoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(geoData)
		}))
		defer geoServer.Close()

		platformServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(platformData)
		}))
		defer platformServer.Close()

		config := &Config{
			GeoWhitelistEndpoint:      geoServer.URL,
			PlatformWhitelistEndpoint: platformServer.URL,
			WhitelistRefreshMs:        100,
			RequestTimeoutMs:          1000,
		}

		client := NewWhitelistClient(http.DefaultClient, config)

		// Stop should not block
		done := make(chan struct{})
		go func() {
			client.Stop()
			close(done)
		}()

		select {
		case <-done:
			// Success
		case <-time.After(time.Second):
			t.Fatal("Stop() blocked for too long")
		}
	})
}

func TestNewWhitelistClient(t *testing.T) {
	t.Run("initial_fetch_on_creation", func(t *testing.T) {
		var fetchCount int32
		geoData := map[string][]string{"site1": {"US"}}
		platformData := map[string][]string{"site1": {"w|chrome"}}

		geoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&fetchCount, 1)
			json.NewEncoder(w).Encode(geoData)
		}))
		defer geoServer.Close()

		platformServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&fetchCount, 1)
			json.NewEncoder(w).Encode(platformData)
		}))
		defer platformServer.Close()

		config := &Config{
			GeoWhitelistEndpoint:      geoServer.URL,
			PlatformWhitelistEndpoint: platformServer.URL,
			WhitelistRefreshMs:        300000, // Long interval so refresh doesn't trigger
			RequestTimeoutMs:          1000,
		}

		client := NewWhitelistClient(http.DefaultClient, config)
		defer client.Stop()

		// Should have made initial fetch (2 requests: geo + platform)
		count := atomic.LoadInt32(&fetchCount)
		assert.Equal(t, int32(2), count, "Should have made exactly 2 fetch requests on init")

		// Verify data was loaded
		assert.NotNil(t, client.geoWhitelist.Load())
		assert.NotNil(t, client.platformWhitelist.Load())
	})

	t.Run("initial_fetch_failure_does_not_panic", func(t *testing.T) {
		config := &Config{
			GeoWhitelistEndpoint:      "http://invalid-host.local/geo",
			PlatformWhitelistEndpoint: "http://invalid-host.local/platform",
			WhitelistRefreshMs:        300000,
			RequestTimeoutMs:          100, // Short timeout
		}

		// Should not panic
		client := NewWhitelistClient(http.DefaultClient, config)
		defer client.Stop()

		// Whitelists should be nil (fail-open)
		assert.Nil(t, client.geoWhitelist.Load())
		assert.Nil(t, client.platformWhitelist.Load())

		// IsAllowed should still work (fail-open)
		assert.True(t, client.IsAllowed("site1", "US", "w|chrome"))
	})
}

