package rulesengine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var sampleJsonConfig json.RawMessage = json.RawMessage(`{"enabled": true, "ruleSets": []}`)

func TestConfigChanged(t *testing.T) {

	testCases := []struct {
		name           string
		inOldHash      hash
		inData         *json.RawMessage
		expectedResult bool
	}{
		{
			name:           "nil_data",
			inOldHash:      "oldHash",
			inData:         nil,
			expectedResult: false,
		},
		{
			name:           "config_changed",
			inOldHash:      "oldHash",
			inData:         &sampleJsonConfig,
			expectedResult: true,
		},
		{
			name:           "config_did_not change",
			inOldHash:      "e21c19982a618f9dd3286fc2eb08dad62a1e9ee81d51ffa94b267ab2e3813964",
			inData:         &sampleJsonConfig,
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := configChanged(tc.inOldHash, tc.inData)
			assert.Equal(t, tc.expectedResult, res)
		})
	}
}

func TestRebuildTrees(t *testing.T) {
	testCases := []struct {
		name               string
		inCacheEntry       *cacheEntry
		inJsonConfig       *json.RawMessage
		refreshRateSeconds int
		expectedResult     bool
	}{
		{
			name: "non_expired_cache_entry_so_no_rebuild",
			inCacheEntry: &cacheEntry{
				timestamp: time.Date(2050, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			inJsonConfig:       &sampleJsonConfig,
			refreshRateSeconds: 1,
			expectedResult:     false,
		},
		{
			name: "expired_entry_but_same_config_and_default_no_update_so_no_rebuild",
			inCacheEntry: &cacheEntry{
				timestamp:    time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				hashedConfig: "e21c19982a618f9dd3286fc2eb08dad62a1e9ee81d51ffa94b267ab2e3813964",
			},
			inJsonConfig:       &sampleJsonConfig,
			refreshRateSeconds: 1,
			expectedResult:     false,
		},
		{
			name: "expired_entry_but_same_config_and_zero_minutes_update_so_no_rebuild",
			inCacheEntry: &cacheEntry{
				timestamp:    time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				hashedConfig: "e21c19982a618f9dd3286fc2eb08dad62a1e9ee81d51ffa94b267ab2e3813964",
			},
			inJsonConfig:       &sampleJsonConfig,
			refreshRateSeconds: 0,
			expectedResult:     false,
		},
		{
			name: "expired_entry_and_different_config_so_rebuild",
			inCacheEntry: &cacheEntry{
				timestamp:    time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				hashedConfig: "oldHash",
			},
			inJsonConfig:       &sampleJsonConfig,
			refreshRateSeconds: 1,
			expectedResult:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			refreshFreq := time.Duration(tc.refreshRateSeconds) * time.Second
			var c cacher = &cache{
				refreshFrequency: refreshFreq,
			}
			res := rebuildTrees(tc.inCacheEntry, tc.inJsonConfig, c)
			assert.Equal(t, tc.expectedResult, res)
		})
	}
}

func TestGetRefreshRate(t *testing.T) {

	testCases := []struct {
		name                string
		inData              json.RawMessage
		expectedRefreshRate int
		expectError         bool
	}{
		{
			name:                "nil_data",
			inData:              nil,
			expectedRefreshRate: 0,
		},
		{
			name:                "valid_config",
			inData:              json.RawMessage(`{"enabled": true, "refreshrateseconds": 10}`),
			expectedRefreshRate: 10,
		},
		{
			name:                "valid_config_negative_refresh_rate",
			inData:              json.RawMessage(`{"enabled": true, "refreshrateseconds": -10}`),
			expectedRefreshRate: -10,
		},
		{
			name:                "valid_config_no_refresh_rate",
			inData:              json.RawMessage(`{"enabled": true}`),
			expectedRefreshRate: 0,
		},
		{
			name:                "invalid_config",
			inData:              json.RawMessage(`{"enabled": true, "refreshrateseconds": "test"}`),
			expectedRefreshRate: 0,
			expectError:         true,
		},
		{
			name:                "path_not_foud",
			inData:              json.RawMessage(`{"enabled": true, "test": 10}`),
			expectedRefreshRate: 0,
			expectError:         false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := getRefreshRate(tc.inData)
			if tc.expectError {
				assert.Error(t, err, "Expected an error but got none")
			} else {
				assert.NoError(t, err, "Expected no error but got one")
			}
			assert.Equal(t, tc.expectedRefreshRate, res)
		})
	}
}
