package rulesengine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/prebid/prebid-server/v3/util/timeutil"
	"github.com/stretchr/testify/assert"
)

func TestExpired(t *testing.T) {
	testCases := []struct {
		name           string
		inTime         timeutil.Time
		inTimestamp    time.Time
		expectedResult bool
	}{
		{
			name:           "expired",
			inTime:         mockTimeUtil{},
			inTimestamp:    mockTimeUtil{}.Now().Add(-time.Hour),
			expectedResult: true,
		},
		{
			name:           "not_expired",
			inTime:         mockTimeUtil{},
			inTimestamp:    mockTimeUtil{}.Now().Add(time.Hour),
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := expired(tc.inTime, tc.inTimestamp)
			assert.Equal(t, tc.expectedResult, res)
		})
	}
}

type mockTimeUtil struct{}

func (mt mockTimeUtil) Now() time.Time {
	return time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
}

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
		name           string
		inCacheEntry   *cacheEntry
		inJsonConfig   *json.RawMessage
		expectedResult bool
	}{
		{
			name: "non_expired_cache_entry_so_no_rebuild",
			inCacheEntry: &cacheEntry{
				timestamp: time.Date(2050, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expectedResult: false,
		},
		{
			name: "expired_entry_but_same_config_so_no_rebuild",
			inCacheEntry: &cacheEntry{
				timestamp:    time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				hashedConfig: "e21c19982a618f9dd3286fc2eb08dad62a1e9ee81d51ffa94b267ab2e3813964",
			},
			inJsonConfig:   &sampleJsonConfig,
			expectedResult: false,
		},
		{
			name: "expired_entry_and_different_config_so_rebuild",
			inCacheEntry: &cacheEntry{
				timestamp:    time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				hashedConfig: "oldHash",
			},
			inJsonConfig:   &sampleJsonConfig,
			expectedResult: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := rebuildTrees(tc.inCacheEntry, tc.inJsonConfig)
			assert.Equal(t, tc.expectedResult, res)
		})
	}
}
