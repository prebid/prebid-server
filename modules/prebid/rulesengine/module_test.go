package rulesengine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/prebid/prebid-server/v3/modules/moduledeps"
	"github.com/prebid/prebid-server/v3/util/timeutil"
	"github.com/stretchr/testify/assert"
)

// TestBuilderWithWorkingDir tests the Builder function by changing the working directory
// to ensure the schema file can be found
func TestBuilderWithWorkingDir(t *testing.T) {
	// Save the current working directory
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	// Change to the project root directory
	err = os.Chdir(filepath.Join(origWd, "..", "..", ".."))
	if err != nil {
		t.Fatalf("Failed to change to project root directory: %v", err)
	}

	// Restore the original working directory when the test is done
	defer func() {
		err := os.Chdir(origWd)
		if err != nil {
			t.Fatalf("Failed to restore original working directory: %v", err)
		}
	}()

	testCases := []struct {
		name             string
		config           json.RawMessage
		deps             moduledeps.ModuleDeps
		expectError      bool
		expectedModuleOK bool
	}{
		{
			name:             "empty-config",
			config:           json.RawMessage(``),
			deps:             moduledeps.ModuleDeps{},
			expectError:      false,
			expectedModuleOK: true,
		},
		{
			name:             "nil-config",
			config:           nil,
			deps:             moduledeps.ModuleDeps{},
			expectError:      false,
			expectedModuleOK: true,
		},
		{
			name:   "with-geoscope-data",
			config: json.RawMessage(``),
			deps: moduledeps.ModuleDeps{
				Geoscope: map[string][]string{
					"bidder1": {"USA", "CAN"},
					"bidder2": {"GBR", "FRA"},
				},
			},
			expectError:      false,
			expectedModuleOK: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call the Builder function directly
			module, err := Builder(tc.config, tc.deps)

			// Check for expected errors
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// For non-error cases, verify the module structure
			if !tc.expectError && tc.expectedModuleOK {
				// Verify the module is not nil
				assert.NotNil(t, module)

				// Type assertion to Module
				m, ok := module.(Module)
				assert.True(t, ok, "Expected module to be of type Module")

				// Verify module components are initialized
				assert.NotNil(t, m.Cache)
				assert.NotNil(t, m.TreeManager)

				// Check BidderConfigRuleSet
				assert.NotNil(t, m.BidderConfigRuleSet)

				// For cases with geoscope data, check more details
				if len(tc.deps.Geoscope) > 0 {
					assert.Len(t, m.BidderConfigRuleSet, 1)
					assert.Equal(t, "Dynamic ruleset from geoscopes", m.BidderConfigRuleSet[0].name)
					assert.Len(t, m.BidderConfigRuleSet[0].modelGroups, 1)
					assert.Equal(t, 100, m.BidderConfigRuleSet[0].modelGroups[0].weight)
					assert.Equal(t, "1.0", m.BidderConfigRuleSet[0].modelGroups[0].version)
					assert.Equal(t, "bidderConfig", m.BidderConfigRuleSet[0].modelGroups[0].analyticsKey)
				}
			}
		})
	}
}

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
