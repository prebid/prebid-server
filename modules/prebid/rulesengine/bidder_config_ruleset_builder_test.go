package rulesengine

import (
	"testing"

	"github.com/prebid/prebid-server/v3/rules"
	"github.com/stretchr/testify/assert"
)

func TestMergeCountryGroups(t *testing.T) {
	defaultCountryGroups := CountryGroups{
		"EEA":           []string{"FRA", "DEU", "ITA"},
		"NORTH_AMERICA": []string{"USA", "CAN", "MEX"},
		"NORDIC":        []string{"FIN", "NOR", "SWE"},
	}

	testCases := []struct {
		name           string
		defaults       CountryGroups
		setDefinitions CountryGroups
		expected       CountryGroups
	}{
		{
			name:           "nil-set-definitions",
			defaults:       defaultCountryGroups,
			setDefinitions: nil,
			expected: CountryGroups{
				"EEA":           []string{"FRA", "DEU", "ITA"},
				"NORTH_AMERICA": []string{"USA", "CAN", "MEX"},
				"NORDIC":        []string{"FIN", "NOR", "SWE"},
			},
		},
		{
			name:           "empty-set-definitions",
			defaults:       defaultCountryGroups,
			setDefinitions: CountryGroups{},
			expected: CountryGroups{
				"EEA":           []string{"FRA", "DEU", "ITA"},
				"NORTH_AMERICA": []string{"USA", "CAN", "MEX"},
				"NORDIC":        []string{"FIN", "NOR", "SWE"},
			},
		},
		{
			name:     "override-existing-group",
			defaults: defaultCountryGroups,
			setDefinitions: CountryGroups{
				"EEA": []string{"ESP", "PRT", "GRC"},
			},
			expected: CountryGroups{
				"EEA":           []string{"ESP", "PRT", "GRC"},
				"NORTH_AMERICA": []string{"USA", "CAN", "MEX"},
				"NORDIC":        []string{"FIN", "NOR", "SWE"},
			},
		},
		{
			name:     "add-new-group",
			defaults: defaultCountryGroups,
			setDefinitions: CountryGroups{
				"ASIA": []string{"JPN", "KOR", "CHN"},
			},
			expected: CountryGroups{
				"EEA":           []string{"FRA", "DEU", "ITA"},
				"NORTH_AMERICA": []string{"USA", "CAN", "MEX"},
				"NORDIC":        []string{"FIN", "NOR", "SWE"},
				"ASIA":          []string{"JPN", "KOR", "CHN"},
			},
		},
		{
			name:     "empty-defaults",
			defaults: CountryGroups{},
			setDefinitions: CountryGroups{
				"ASIA": []string{"JPN", "KOR", "CHN"},
			},
			expected: CountryGroups{
				"ASIA": []string{"JPN", "KOR", "CHN"},
			},
		},
		{
			name:     "multiple-operations",
			defaults: defaultCountryGroups,
			setDefinitions: CountryGroups{
				"EEA":    []string{"ESP", "PRT"},
				"ASIA":   []string{"JPN", "KOR"},
				"AFRICA": []string{"ZAF", "EGY", "NGA"},
			},
			expected: CountryGroups{
				"EEA":           []string{"ESP", "PRT"},
				"NORTH_AMERICA": []string{"USA", "CAN", "MEX"},
				"NORDIC":        []string{"FIN", "NOR", "SWE"},
				"ASIA":          []string{"JPN", "KOR"},
				"AFRICA":        []string{"ZAF", "EGY", "NGA"},
			},
		},
		{
			name:     "empty-slice-in-set-definitions",
			defaults: defaultCountryGroups,
			setDefinitions: CountryGroups{
				"EMPTY_GROUP": []string{},
			},
			expected: CountryGroups{
				"EEA":           []string{"FRA", "DEU", "ITA"},
				"NORTH_AMERICA": []string{"USA", "CAN", "MEX"},
				"NORDIC":        []string{"FIN", "NOR", "SWE"},
				"EMPTY_GROUP":   []string{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mergeCountryGroups(tc.defaults, tc.setDefinitions)

			assert.Len(t, result, len(tc.expected), "Result should have the expected number of groups")

			for groupName, expectedCountries := range tc.expected {
				actualCountries, exists := result[groupName]
				assert.True(t, exists, "Group %s should exist in result", groupName)
				assert.ElementsMatch(t, expectedCountries, actualCountries)
			}
		})
	}
}

func TestBuild(t *testing.T) {
	// Define test groups
	testCountryGroups := CountryGroups{
		"EEA":           []string{"FRA", "DEU", "ITA"},
		"NORTH_AMERICA": []string{"USA", "CAN", "MEX"},
	}

	testCases := []struct {
		name           string
		geoscopeConfig map[string][]string
		countryGroups  CountryGroups
		nilRoot        bool
		expectError    bool
		expectedKeys   map[string][]string
	}{
		{
			name:           "empty-geoscope-config",
			geoscopeConfig: map[string][]string{},
			countryGroups:  testCountryGroups,
			nilRoot:        false,
			expectError:    false,
			expectedKeys:   nil,
		},
		{
			name:           "nil-root-node",
			geoscopeConfig: map[string][]string{"bidder1": {"USA"}},
			countryGroups:  testCountryGroups,
			nilRoot:        true,
			expectError:    false,
			expectedKeys:   nil,
		},
		{
			name: "global-directive-only",
			geoscopeConfig: map[string][]string{
				"bidder1": {"GLOBAL"},
			},
			countryGroups: testCountryGroups,
			nilRoot:       false,
			expectError:   false,
			expectedKeys: map[string][]string{
				"*": {},
			},
		},
		{
			name: "allow-list-with-groups",
			geoscopeConfig: map[string][]string{
				"bidder1": {"NORTH_AMERICA"},
			},
			countryGroups: testCountryGroups,
			nilRoot:       false,
			expectError:   false,
			expectedKeys: map[string][]string{
				"*":   {"bidder1"},
				"USA": {},
				"CAN": {},
				"MEX": {},
			},
		},
		{
			name: "block-list-with-groups",
			geoscopeConfig: map[string][]string{
				"bidder1": {"!NORTH_AMERICA"},
			},
			countryGroups: testCountryGroups,
			nilRoot:       false,
			expectError:   false,
			expectedKeys: map[string][]string{
				"*":   {},
				"USA": {"bidder1"},
				"CAN": {"bidder1"},
				"MEX": {"bidder1"},
			},
		},
		{
			name: "allow-list-with-specific-countries",
			geoscopeConfig: map[string][]string{
				"bidder1": {"ESP", "GBR"},
			},
			countryGroups: testCountryGroups,
			nilRoot:       false,
			expectError:   false,
			expectedKeys: map[string][]string{
				"*":   {"bidder1"},
				"ESP": {},
				"GBR": {},
			},
		},
		{
			name: "block-list-with-specific-countries",
			geoscopeConfig: map[string][]string{
				"bidder1": {"!ESP", "!GBR"},
			},
			countryGroups: testCountryGroups,
			nilRoot:       false,
			expectError:   false,
			expectedKeys: map[string][]string{
				"*":   {},
				"ESP": {"bidder1"},
				"GBR": {"bidder1"},
			},
		},
		{
			name: "multiple-bidders-with-different-scopes",
			geoscopeConfig: map[string][]string{
				"bidder1": {"NORTH_AMERICA"},
				"bidder2": {"!EEA"},
				"bidder3": {"ESP", "GBR"},
				"bidder4": {"!JPN"},
			},
			countryGroups: testCountryGroups,
			nilRoot:       false,
			expectError:   false,
			expectedKeys: map[string][]string{
				"*":   {"bidder1", "bidder3"},
				"USA": {"bidder3"},
				"CAN": {"bidder3"},
				"MEX": {"bidder3"},
				"FRA": {"bidder1", "bidder2", "bidder3"},
				"DEU": {"bidder1", "bidder2", "bidder3"},
				"ITA": {"bidder1", "bidder2", "bidder3"},
				"ESP": {"bidder1"},
				"GBR": {"bidder1"},
				"JPN": {"bidder1", "bidder3", "bidder4"},
			},
		},
		// {
		// 	name: "invalid-geoscope-returns-error",
		// 	geoscopeConfig: map[string][]string{
		// 		"bidder1": {"INVALID_FORMAT"},
		// 	},
		// 	countryGroups:     testCountryGroups,
		// 	nilRoot:           false,
		// 	expectError:       true,
		// },
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a tree
			tree := &rules.Tree[RequestWrapper, ProcessedAuctionHookResult]{
				Root: nil,
			}

			if !tc.nilRoot {
				tree.Root = &rules.Node[RequestWrapper, ProcessedAuctionHookResult]{
					Children: make(map[string]*rules.Node[RequestWrapper, ProcessedAuctionHookResult]),
				}
			}

			// Create the builder
			builder := &bidderConfigRuleSetBuilder[RequestWrapper, ProcessedAuctionHookResult]{
				countryGroups:  tc.countryGroups,
				geoscopeConfig: tc.geoscopeConfig,
			}

			// Call the Build function
			err := builder.Build(tree)

			// Check for expected error
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tc.nilRoot {
				assert.Nil(t, tree.Root)
			} else if len(tc.geoscopeConfig) == 0 {
				assert.Nil(t, tree.Root.SchemaFunction)
				assert.Len(t, tree.Root.Children, 0)
			} else {
				assert.NotNil(t, tree.Root.SchemaFunction)
				assert.Equal(t, len(tc.expectedKeys), len(tree.Root.Children))

				for k, v := range tc.expectedKeys {
					childNode, ok := tree.Root.Children[k]
					assert.True(t, ok, "Expected child node %q to be present", k)
					assert.Len(t, childNode.ResultFunctions, 1)
					assert.IsType(t, &ExcludeBidders{}, childNode.ResultFunctions[0])
					assert.ElementsMatch(t, v, childNode.ResultFunctions[0].(*ExcludeBidders).Args.Bidders)
				}
			}
		})
	}
}

func TestInitializeCountryExclusions(t *testing.T) {
	// Define test groups
	testCountryGroups := CountryGroups{
		"TEST_GROUP1": []string{"USA", "CAN", "MEX"},
		"TEST_GROUP2": []string{"FRA", "DEU", "ITA"},
		"EMPTY_GROUP": []string{},
	}

	testCases := []struct {
		name           string
		countryGroups  CountryGroups
		geoscopeConfig map[string][]string
		expected       CountryExclusions
	}{
		{
			name:           "empty-geoscope-config",
			countryGroups:  testCountryGroups,
			geoscopeConfig: map[string][]string{},
			expected: CountryExclusions{
				"*": []string{},
			},
		},
		{
			name:           "nil-geoscope-config",
			countryGroups:  testCountryGroups,
			geoscopeConfig: nil,
			expected: CountryExclusions{
				"*": []string{},
			},
		},
		{
			name:          "single-country-code",
			countryGroups: testCountryGroups,
			geoscopeConfig: map[string][]string{
				"bidder1": {"USA"},
			},
			expected: CountryExclusions{
				"USA": []string{},
				"*":   []string{},
			},
		},
		{
			name:          "multiple-country-codes",
			countryGroups: testCountryGroups,
			geoscopeConfig: map[string][]string{
				"bidder1": {"USA", "CAN", "MEX"},
			},
			expected: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"MEX": []string{},
				"*":   []string{},
			},
		},
		{
			name:          "country-group-expands-to-countries",
			countryGroups: testCountryGroups,
			geoscopeConfig: map[string][]string{
				"bidder1": {"TEST_GROUP1"},
			},
			expected: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"MEX": []string{},
				"*":   []string{},
			},
		},
		{
			name:          "excluded-country-code",
			countryGroups: testCountryGroups,
			geoscopeConfig: map[string][]string{
				"bidder1": {"!USA"},
			},
			expected: CountryExclusions{
				"USA": []string{},
				"*":   []string{},
			},
		},
		{
			name:          "excluded-country-group-expands-to-countries",
			countryGroups: testCountryGroups,
			geoscopeConfig: map[string][]string{
				"bidder1": {"!TEST_GROUP1"},
			},
			expected: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"MEX": []string{},
				"*":   []string{},
			},
		},
		{
			name:          "mix-of-countries-and-groups",
			countryGroups: testCountryGroups,
			geoscopeConfig: map[string][]string{
				"bidder1": {"TEST_GROUP1", "FRA"},
				"bidder2": {"!TEST_GROUP2", "USA"},
				"bidder3": {"ESP", "JAP"},
			},
			expected: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"MEX": []string{},
				"FRA": []string{},
				"DEU": []string{},
				"ITA": []string{},
				"ESP": []string{},
				"JAP": []string{},
				"*":   []string{},
			},
		},
		{
			name:          "global-directive-is-skipped",
			countryGroups: testCountryGroups,
			geoscopeConfig: map[string][]string{
				"bidder1": {"GLOBAL", "USA"},
			},
			expected: CountryExclusions{
				"USA": []string{},
				"*":   []string{},
			},
		},
		{
			name:          "empty-group-not-expanded",
			countryGroups: testCountryGroups,
			geoscopeConfig: map[string][]string{
				"bidder1": {"EMPTY_GROUP"},
			},
			expected: CountryExclusions{
				"*": []string{},
			},
		},
		{
			name:          "multiple-bidders-same-countries",
			countryGroups: testCountryGroups,
			geoscopeConfig: map[string][]string{
				"bidder1": {"USA", "CAN"},
				"bidder2": {"USA", "MEX"},
			},
			expected: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"MEX": []string{},
				"*":   []string{},
			},
		},
		{
			name:          "group-only-expanded-once-when-used-multiple-times",
			countryGroups: testCountryGroups,
			geoscopeConfig: map[string][]string{
				"bidder1": {"TEST_GROUP1"},
				"bidder2": {"TEST_GROUP1"},
			},
			expected: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"MEX": []string{},
				"*":   []string{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := &bidderConfigRuleSetBuilder[RequestWrapper, ProcessedAuctionHookResult]{
				countryGroups:  tc.countryGroups,
				geoscopeConfig: tc.geoscopeConfig,
			}

			result := builder.initializeCountryExclusions()

			// Check keys are the same (regardless of order)
			expectedKeys := make([]string, 0, len(tc.expected))
			for k := range tc.expected {
				expectedKeys = append(expectedKeys, k)
			}

			resultKeys := make([]string, 0, len(result))
			for k := range result {
				resultKeys = append(resultKeys, k)
			}

			assert.ElementsMatch(t, expectedKeys, resultKeys)

			// Check all expected keys have empty exclusion lists
			for country, excludedBidders := range result {
				assert.Empty(t, excludedBidders, "Country %s should have empty exclusion list", country)
			}
		})
	}
}

func TestParseBidderGeoscopes(t *testing.T) {
	// Define test groups
	testCountryGroups := CountryGroups{
		"EEA":           []string{"FRA", "DEU", "ITA"},
		"BALTIC":        []string{"EST", "LVA", "LTU"},
		"NORDIC":        []string{"FIN", "NOR", "SWE"},
		"NORTH_AMERICA": []string{"USA", "CAN", "MEX"},
	}

	testCases := []struct {
		name        string
		bidder      string
		geoscopes   []string
		expected    parsedConfig
		expectError bool
	}{
		{
			name:      "nil-geoscopes",
			bidder:    "bidder1",
			geoscopes: nil,
			expected: parsedConfig{
				bidder:                  "bidder1",
				globalIncluded:          false,
				countryGroupsIncluded:   nil,
				countryGroupsExcluded:   nil,
				singleCountriesIncluded: nil,
				singleCountriesExcluded: nil,
			},
			expectError: false,
		},
		{
			name:      "empty-geoscopes",
			bidder:    "bidder1",
			geoscopes: []string{},
			expected: parsedConfig{
				bidder:                  "bidder1",
				globalIncluded:          false,
				countryGroupsIncluded:   nil,
				countryGroupsExcluded:   nil,
				singleCountriesIncluded: nil,
				singleCountriesExcluded: nil,
			},
			expectError: false,
		},
		{
			name:      "global-scope",
			bidder:    "bidder1",
			geoscopes: []string{"GLOBAL"},
			expected: parsedConfig{
				bidder:                  "bidder1",
				globalIncluded:          true,
				countryGroupsIncluded:   nil,
				countryGroupsExcluded:   nil,
				singleCountriesIncluded: nil,
				singleCountriesExcluded: nil,
			},
			expectError: false,
		},
		{
			name:      "included-country-group",
			bidder:    "bidder1",
			geoscopes: []string{"NORTH_AMERICA"},
			expected: parsedConfig{
				bidder:                  "bidder1",
				globalIncluded:          false,
				countryGroupsIncluded:   []string{"NORTH_AMERICA"},
				countryGroupsExcluded:   nil,
				singleCountriesIncluded: nil,
				singleCountriesExcluded: nil,
			},
			expectError: false,
		},
		{
			name:      "excluded-country-group",
			bidder:    "bidder1",
			geoscopes: []string{"!NORTH_AMERICA"},
			expected: parsedConfig{
				bidder:                  "bidder1",
				globalIncluded:          false,
				countryGroupsIncluded:   nil,
				countryGroupsExcluded:   []string{"NORTH_AMERICA"},
				singleCountriesIncluded: nil,
				singleCountriesExcluded: nil,
			},
			expectError: false,
		},
		{
			name:      "included-single-country",
			bidder:    "bidder1",
			geoscopes: []string{"ESP"},
			expected: parsedConfig{
				bidder:                  "bidder1",
				globalIncluded:          false,
				countryGroupsIncluded:   nil,
				countryGroupsExcluded:   nil,
				singleCountriesIncluded: []string{"ESP"},
				singleCountriesExcluded: nil,
			},
			expectError: false,
		},
		{
			name:      "excluded-single-country",
			bidder:    "bidder1",
			geoscopes: []string{"!ESP"},
			expected: parsedConfig{
				bidder:                  "bidder1",
				globalIncluded:          false,
				countryGroupsIncluded:   nil,
				countryGroupsExcluded:   nil,
				singleCountriesIncluded: nil,
				singleCountriesExcluded: []string{"ESP"},
			},
			expectError: false,
		},
		{
			name:      "mixed-multiple-geoscopes",
			bidder:    "bidder1",
			geoscopes: []string{"GLOBAL", "NORTH_AMERICA", "NORDIC", "!BALTIC", "!EEA", "ESP", "!FRA", "BRA", "!ARG"},
			expected: parsedConfig{
				bidder:                  "bidder1",
				globalIncluded:          true,
				countryGroupsIncluded:   []string{"NORTH_AMERICA", "NORDIC"},
				countryGroupsExcluded:   []string{"BALTIC", "EEA"},
				singleCountriesIncluded: []string{"ESP", "BRA"},
				singleCountriesExcluded: []string{"FRA", "ARG"},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := &bidderConfigRuleSetBuilder[RequestWrapper, ProcessedAuctionHookResult]{
				countryGroups: testCountryGroups,
			}

			result, err := builder.parseBidderGeoscopes(tc.bidder, tc.geoscopes)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				assert.Equal(t, tc.expected.bidder, result.bidder)
				assert.Equal(t, tc.expected.globalIncluded, result.globalIncluded)
				assert.ElementsMatch(t, tc.expected.countryGroupsIncluded, result.countryGroupsIncluded)
				assert.ElementsMatch(t, tc.expected.countryGroupsExcluded, result.countryGroupsExcluded)
				assert.ElementsMatch(t, tc.expected.singleCountriesIncluded, result.singleCountriesIncluded)
				assert.ElementsMatch(t, tc.expected.singleCountriesExcluded, result.singleCountriesExcluded)
			}
		})
	}
}

func TestMarkAllowListWithGroupsExclusions(t *testing.T) {
	// Define test groups
	testCountryGroups := CountryGroups{
		"EEA":           []string{"FRA", "DEU", "ITA"},
		"BALTIC":        []string{"EST", "LVA", "LTU"},
		"NORDIC":        []string{"FIN", "NOR", "SWE"},
		"NORTH_AMERICA": []string{"USA", "CAN", "MEX"},
	}

	testCases := []struct {
		name              string
		config            parsedConfig
		countryExclusions CountryExclusions
		expected          CountryExclusions
	}{
		{
			name: "country-in-group-not-excluded",
			config: parsedConfig{
				bidder:                  "bidder1",
				countryGroupsIncluded:   []string{"NORTH_AMERICA"},
				singleCountriesIncluded: []string{},
				singleCountriesExcluded: []string{},
			},
			countryExclusions: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"GBR": []string{},
				"JPN": []string{},
				"*":   []string{},
			},
			expected: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"GBR": []string{"bidder1"},
				"JPN": []string{"bidder1"},
				"*":   []string{"bidder1"},
			},
		},
		{
			name: "country-in-group-specifically-excluded",
			config: parsedConfig{
				bidder:                  "bidder1",
				countryGroupsIncluded:   []string{"NORTH_AMERICA"},
				singleCountriesIncluded: []string{},
				singleCountriesExcluded: []string{"USA"},
			},
			countryExclusions: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"GBR": []string{},
				"JPN": []string{},
				"*":   []string{},
			},
			expected: CountryExclusions{
				"USA": []string{"bidder1"},
				"CAN": []string{},
				"GBR": []string{"bidder1"},
				"JPN": []string{"bidder1"},
				"*":   []string{"bidder1"},
			},
		},
		{
			name: "country-not-in-group-specifically-included",
			config: parsedConfig{
				bidder:                  "bidder1",
				countryGroupsIncluded:   []string{"NORTH_AMERICA"},
				singleCountriesIncluded: []string{"GBR"},
				singleCountriesExcluded: []string{},
			},
			countryExclusions: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"GBR": []string{},
				"JPN": []string{},
				"*":   []string{},
			},
			expected: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"GBR": []string{},
				"JPN": []string{"bidder1"},
				"*":   []string{"bidder1"},
			},
		},
		{
			name: "multiple-included-groups",
			config: parsedConfig{
				bidder:                  "bidder1",
				countryGroupsIncluded:   []string{"NORTH_AMERICA", "EEA"},
				singleCountriesIncluded: []string{},
				singleCountriesExcluded: []string{},
			},
			countryExclusions: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"FRA": []string{},
				"JPN": []string{},
				"*":   []string{},
			},
			expected: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"FRA": []string{},
				"JPN": []string{"bidder1"},
				"*":   []string{"bidder1"},
			},
		},
		{
			name: "multiple-bidders-in-exclusions",
			config: parsedConfig{
				bidder:                  "bidder1",
				countryGroupsIncluded:   []string{"NORTH_AMERICA"},
				singleCountriesIncluded: []string{},
				singleCountriesExcluded: []string{},
			},
			countryExclusions: CountryExclusions{
				"USA": []string{"existing-bidder"},
				"GBR": []string{"existing-bidder"},
			},
			expected: CountryExclusions{
				"USA": []string{"existing-bidder"},
				"GBR": []string{"existing-bidder", "bidder1"},
			},
		},
		{
			name: "empty-country-exclusions",
			config: parsedConfig{
				bidder:                  "bidder1",
				countryGroupsIncluded:   []string{"NORTH_AMERICA"},
				singleCountriesIncluded: []string{},
				singleCountriesExcluded: []string{},
			},
			countryExclusions: CountryExclusions{},
			expected:          CountryExclusions{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := &bidderConfigRuleSetBuilder[RequestWrapper, ProcessedAuctionHookResult]{
				countryGroups: testCountryGroups,
			}

			builder.markAllowListWithGroupsExclusions(tc.config, tc.countryExclusions)

			// Check that all expected keys exist with the right values
			for country, expectedBidders := range tc.expected {
				assert.ElementsMatch(t, expectedBidders, tc.countryExclusions[country],
					"Country %s should have the expected excluded bidders", country)
			}

			// Check that there are no unexpected keys
			assert.Equal(t, len(tc.expected), len(tc.countryExclusions),
				"Expected and actual CountryExclusions should have the same number of entries")
		})
	}
}

func TestMarkBlockListWithGroupsExclusions(t *testing.T) {
	// Define test groups
	testCountryGroups := CountryGroups{
		"EEA":           []string{"FRA", "DEU", "ITA"},
		"BALTIC":        []string{"EST", "LVA", "LTU"},
		"NORDIC":        []string{"FIN", "NOR", "SWE"},
		"NORTH_AMERICA": []string{"USA", "CAN", "MEX"},
	}

	testCases := []struct {
		name              string
		config            parsedConfig
		countryExclusions CountryExclusions
		expected          CountryExclusions
	}{
		{
			name: "country-in-excluded-group-excluded",
			config: parsedConfig{
				bidder:                  "bidder1",
				countryGroupsExcluded:   []string{"NORTH_AMERICA"},
				singleCountriesIncluded: []string{},
				singleCountriesExcluded: []string{},
			},
			countryExclusions: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"GBR": []string{},
				"JPN": []string{},
				"*":   []string{},
			},
			expected: CountryExclusions{
				"USA": []string{"bidder1"},
				"CAN": []string{"bidder1"},
				"GBR": []string{},
				"JPN": []string{},
				"*":   []string{},
			},
		},
		{
			name: "country-in-excluded-group-specifically-included",
			config: parsedConfig{
				bidder:                  "bidder1",
				countryGroupsExcluded:   []string{"NORTH_AMERICA"},
				singleCountriesIncluded: []string{"USA"},
				singleCountriesExcluded: []string{},
			},
			countryExclusions: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"GBR": []string{},
				"JPN": []string{},
				"*":   []string{},
			},
			expected: CountryExclusions{
				"USA": []string{},
				"CAN": []string{"bidder1"},
				"GBR": []string{},
				"JPN": []string{},
				"*":   []string{},
			},
		},
		{
			name: "country-not-in-excluded-group-specifically-excluded",
			config: parsedConfig{
				bidder:                  "bidder1",
				countryGroupsExcluded:   []string{"NORTH_AMERICA"},
				singleCountriesIncluded: []string{},
				singleCountriesExcluded: []string{"GBR"},
			},
			countryExclusions: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"GBR": []string{},
				"JPN": []string{},
				"*":   []string{},
			},
			expected: CountryExclusions{
				"USA": []string{"bidder1"},
				"CAN": []string{"bidder1"},
				"GBR": []string{"bidder1"},
				"JPN": []string{},
				"*":   []string{},
			},
		},
		{
			name: "multiple-excluded-groups",
			config: parsedConfig{
				bidder:                  "bidder1",
				countryGroupsExcluded:   []string{"NORTH_AMERICA", "EEA"},
				singleCountriesIncluded: []string{},
				singleCountriesExcluded: []string{},
			},
			countryExclusions: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"FRA": []string{},
				"JPN": []string{},
				"*":   []string{},
			},
			expected: CountryExclusions{
				"USA": []string{"bidder1"},
				"CAN": []string{"bidder1"},
				"FRA": []string{"bidder1"},
				"JPN": []string{},
				"*":   []string{},
			},
		},
		{
			name: "multiple-bidders-in-exclusions",
			config: parsedConfig{
				bidder:                  "bidder1",
				countryGroupsExcluded:   []string{"NORTH_AMERICA"},
				singleCountriesIncluded: []string{},
				singleCountriesExcluded: []string{},
			},
			countryExclusions: CountryExclusions{
				"USA": []string{"existing-bidder"},
				"GBR": []string{"existing-bidder"},
			},
			expected: CountryExclusions{
				"USA": []string{"existing-bidder", "bidder1"},
				"GBR": []string{"existing-bidder"},
			},
		},
		{
			name: "country-in-excluded-group-with-specific-exclusion",
			config: parsedConfig{
				bidder:                  "bidder1",
				countryGroupsExcluded:   []string{"NORTH_AMERICA"},
				singleCountriesIncluded: []string{},
				singleCountriesExcluded: []string{"USA"},
			},
			countryExclusions: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"*":   []string{},
			},
			expected: CountryExclusions{
				"USA": []string{"bidder1"},
				"CAN": []string{"bidder1"},
				"*":   []string{},
			},
		},
		{
			name: "empty-country-exclusions",
			config: parsedConfig{
				bidder:                  "bidder1",
				countryGroupsExcluded:   []string{"NORTH_AMERICA"},
				singleCountriesIncluded: []string{},
				singleCountriesExcluded: []string{},
			},
			countryExclusions: CountryExclusions{},
			expected:          CountryExclusions{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := &bidderConfigRuleSetBuilder[RequestWrapper, ProcessedAuctionHookResult]{
				countryGroups: testCountryGroups,
			}

			builder.markBlockListWithGroupsExclusions(tc.config, tc.countryExclusions)

			// Check that all expected keys exist with the right values
			for country, expectedBidders := range tc.expected {
				assert.ElementsMatch(t, expectedBidders, tc.countryExclusions[country],
					"Country %s should have the expected excluded bidders", country)
			}

			// Check that there are no unexpected keys
			assert.Equal(t, len(tc.expected), len(tc.countryExclusions),
				"Expected and actual CountryExclusions should have the same number of entries")
		})
	}
}

func TestMarkAllowListExclusions(t *testing.T) {
	testCases := []struct {
		name              string
		config            parsedConfig
		countryExclusions CountryExclusions
		expected          CountryExclusions
	}{
		{
			name: "single-country-included",
			config: parsedConfig{
				bidder:                  "bidder1",
				singleCountriesIncluded: []string{"USA"},
			},
			countryExclusions: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"GBR": []string{},
				"*":   []string{},
			},
			expected: CountryExclusions{
				"USA": []string{},
				"CAN": []string{"bidder1"},
				"GBR": []string{"bidder1"},
				"*":   []string{"bidder1"},
			},
		},
		{
			name: "multiple-countries-included",
			config: parsedConfig{
				bidder:                  "bidder1",
				singleCountriesIncluded: []string{"USA", "CAN"},
			},
			countryExclusions: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"GBR": []string{},
				"*":   []string{},
			},
			expected: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"GBR": []string{"bidder1"},
				"*":   []string{"bidder1"},
			},
		},
		{
			name: "no-countries-included",
			config: parsedConfig{
				bidder:                  "bidder1",
				singleCountriesIncluded: []string{},
			},
			countryExclusions: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"GBR": []string{},
				"*":   []string{},
			},
			expected: CountryExclusions{
				"USA": []string{"bidder1"},
				"CAN": []string{"bidder1"},
				"GBR": []string{"bidder1"},
				"*":   []string{"bidder1"},
			},
		},
		{
			name: "existing-excluded-bidders-preserved",
			config: parsedConfig{
				bidder:                  "bidder1",
				singleCountriesIncluded: []string{"USA"},
			},
			countryExclusions: CountryExclusions{
				"USA": []string{"existing-bidder"},
				"CAN": []string{"existing-bidder"},
				"GBR": []string{},
				"*":   []string{},
			},
			expected: CountryExclusions{
				"USA": []string{"existing-bidder"},
				"CAN": []string{"existing-bidder", "bidder1"},
				"GBR": []string{"bidder1"},
				"*":   []string{"bidder1"},
			},
		},
		{
			name: "empty-country-exclusions",
			config: parsedConfig{
				bidder:                  "bidder1",
				singleCountriesIncluded: []string{"USA"},
			},
			countryExclusions: CountryExclusions{},
			expected:          CountryExclusions{},
		},
		{
			name: "specifically-included-wildcard",
			config: parsedConfig{
				bidder:                  "bidder1",
				singleCountriesIncluded: []string{"*"},
			},
			countryExclusions: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"GBR": []string{},
				"*":   []string{},
			},
			expected: CountryExclusions{
				"USA": []string{"bidder1"},
				"CAN": []string{"bidder1"},
				"GBR": []string{"bidder1"},
				"*":   []string{},
			},
		},
		{
			name: "country-not-in-exclusions-map",
			config: parsedConfig{
				bidder:                  "bidder1",
				singleCountriesIncluded: []string{"BRA"},
			},
			countryExclusions: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"GBR": []string{},
				"*":   []string{},
			},
			expected: CountryExclusions{
				"USA": []string{"bidder1"},
				"CAN": []string{"bidder1"},
				"GBR": []string{"bidder1"},
				"*":   []string{"bidder1"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := &bidderConfigRuleSetBuilder[RequestWrapper, ProcessedAuctionHookResult]{}

			builder.markAllowListExclusions(tc.config, tc.countryExclusions)

			// Check that all expected keys exist with the right values
			for country, expectedBidders := range tc.expected {
				assert.ElementsMatch(t, expectedBidders, tc.countryExclusions[country],
					"Country %s should have the expected excluded bidders", country)
			}

			// Check that there are no unexpected keys
			assert.Equal(t, len(tc.expected), len(tc.countryExclusions),
				"Expected and actual CountryExclusions should have the same number of entries")
		})
	}
}

func TestMarkBlockListExclusions(t *testing.T) {
	testCases := []struct {
		name              string
		config            parsedConfig
		countryExclusions CountryExclusions
		expected          CountryExclusions
	}{
		{
			name: "single-country-excluded",
			config: parsedConfig{
				bidder:                  "bidder1",
				singleCountriesExcluded: []string{"USA"},
			},
			countryExclusions: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"GBR": []string{},
				"*":   []string{},
			},
			expected: CountryExclusions{
				"USA": []string{"bidder1"},
				"CAN": []string{},
				"GBR": []string{},
				"*":   []string{},
			},
		},
		{
			name: "multiple-countries-excluded",
			config: parsedConfig{
				bidder:                  "bidder1",
				singleCountriesExcluded: []string{"USA", "CAN"},
			},
			countryExclusions: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"GBR": []string{},
				"*":   []string{},
			},
			expected: CountryExclusions{
				"USA": []string{"bidder1"},
				"CAN": []string{"bidder1"},
				"GBR": []string{},
				"*":   []string{},
			},
		},
		{
			name: "no-countries-excluded",
			config: parsedConfig{
				bidder:                  "bidder1",
				singleCountriesExcluded: []string{},
			},
			countryExclusions: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"GBR": []string{},
				"*":   []string{},
			},
			expected: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"GBR": []string{},
				"*":   []string{},
			},
		},
		{
			name: "existing-excluded-bidders-preserved",
			config: parsedConfig{
				bidder:                  "bidder1",
				singleCountriesExcluded: []string{"USA"},
			},
			countryExclusions: CountryExclusions{
				"USA": []string{"existing-bidder"},
				"CAN": []string{"existing-bidder"},
				"GBR": []string{},
			},
			expected: CountryExclusions{
				"USA": []string{"existing-bidder", "bidder1"},
				"CAN": []string{"existing-bidder"},
				"GBR": []string{},
			},
		},
		{
			name: "empty-country-exclusions",
			config: parsedConfig{
				bidder:                  "bidder1",
				singleCountriesExcluded: []string{"USA"},
			},
			countryExclusions: CountryExclusions{},
			expected:          CountryExclusions{},
		},
		{
			name: "specifically-excluded-wildcard",
			config: parsedConfig{
				bidder:                  "bidder1",
				singleCountriesExcluded: []string{"*"},
			},
			countryExclusions: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"*":   []string{},
			},
			expected: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
				"*":   []string{"bidder1"},
			},
		},
		{
			name: "country-not-in-exclusions-map",
			config: parsedConfig{
				bidder:                  "bidder1",
				singleCountriesExcluded: []string{"BRA"},
			},
			countryExclusions: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
			},
			expected: CountryExclusions{
				"USA": []string{},
				"CAN": []string{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := &bidderConfigRuleSetBuilder[RequestWrapper, ProcessedAuctionHookResult]{}

			builder.markBlockListExclusions(tc.config, tc.countryExclusions)

			// Check that all expected keys exist with the right values
			for country, expectedBidders := range tc.expected {
				assert.ElementsMatch(t, expectedBidders, tc.countryExclusions[country],
					"Country %s should have the expected excluded bidders", country)
			}

			// Check that there are no unexpected keys
			assert.Equal(t, len(tc.expected), len(tc.countryExclusions),
				"Expected and actual CountryExclusions should have the same number of entries")
		})
	}
}

func TestAddTreeNodes(t *testing.T) {
	testCases := []struct {
		name              string
		countryExclusions CountryExclusions
		expectedChildren  int
	}{
		{
			name:              "nil-exclusions",
			countryExclusions: CountryExclusions{},
			expectedChildren:  0,
		},
		{
			name:              "empty-exclusions",
			countryExclusions: CountryExclusions{},
			expectedChildren:  0,
		},
		{
			name: "single-country-exclusion",
			countryExclusions: CountryExclusions{
				"USA": []string{"bidder1"},
			},
			expectedChildren: 1,
		},
		{
			name: "multiple-country-exclusions",
			countryExclusions: CountryExclusions{
				"USA": []string{"bidder1"},
				"CAN": []string{"bidder2"},
				"MEX": []string{"bidder3"},
			},
			expectedChildren: 3,
		},
		{
			name: "country-with-multiple-bidders",
			countryExclusions: CountryExclusions{
				"USA": []string{"bidder1", "bidder2", "bidder3"},
			},
			expectedChildren: 1,
		},
		{
			name: "mixed-exclusions",
			countryExclusions: CountryExclusions{
				"USA": []string{"bidder1", "bidder2"},
				"CAN": []string{"bidder3"},
				"*":   []string{"bidder4"},
			},
			expectedChildren: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := &rules.Node[RequestWrapper, ProcessedAuctionHookResult]{
				Children: make(map[string]*rules.Node[RequestWrapper, ProcessedAuctionHookResult]),
			}
			addTreeNodes(root, tc.countryExclusions)

			assert.Len(t, root.Children, tc.expectedChildren)

			// Check that each country has a node with the correct bidders
			for country, excludedBidders := range tc.countryExclusions {
				childNode, exists := root.Children[country]
				assert.True(t, exists, "Expected child node for country %s", country)

				if exists {
					assert.Len(t, childNode.ResultFunctions, 1)

					if len(childNode.ResultFunctions) > 0 {
						assert.IsType(t, &ExcludeBidders{}, childNode.ResultFunctions[0])

						// Check that the result function is of type ExcludeBidders
						excludeBidders, ok := childNode.ResultFunctions[0].(*ExcludeBidders)
						assert.True(t, ok, "Expected result function to be of type ExcludeBidders")

						if ok {
							// Check that the bidders in the config match what we expect
							assert.ElementsMatch(t, excludedBidders, excludeBidders.Args.Bidders,
								"Country %s: Expected bidders %v, got %v", country, excludedBidders, excludeBidders.Args.Bidders)
						}
					}
				}
			}
		})
	}
}
