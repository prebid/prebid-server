package config

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccountModulesHookFunc(t *testing.T) {
	// Create the hook function to test
	hookFunc := AccountModulesHookFunc()

	// Define AccountModules type for reference
	var accountModulesType = reflect.TypeOf(AccountModules{})

	// Define some test types
	type OtherType struct{}
	var otherType = reflect.TypeOf(OtherType{})
	var mapStringInterface = reflect.TypeOf(map[string]interface{}{})
	var sliceType = reflect.TypeOf([]string{})

	tests := []struct {
		name          string
		fromType      reflect.Type
		toType        reflect.Type
		inputData     interface{}
		expectedData  interface{}
		expectSamePtr bool
		expectError   bool
		errorContains string
	}{
		{
			name:          "non-map-source-returns-same-data",
			fromType:      sliceType,
			toType:        accountModulesType,
			inputData:     []string{"test"},
			expectedData:  []string{"test"},
			expectSamePtr: true,
			expectError:   false,
		},
		{
			name:          "different-target-type-returns-same-data",
			fromType:      mapStringInterface,
			toType:        otherType,
			inputData:     map[string]interface{}{"key": "value"},
			expectedData:  map[string]interface{}{"key": "value"},
			expectSamePtr: true,
			expectError:   false,
		},
		{
			name:         "empty-map-converts-to-empty-account-modules",
			fromType:     mapStringInterface,
			toType:       accountModulesType,
			inputData:    map[string]interface{}{},
			expectedData: map[string]map[string]json.RawMessage{},
			expectError:  false,
		},
		{
			name:     "valid-input-successfully-converts",
			fromType: mapStringInterface,
			toType:   accountModulesType,
			inputData: map[string]interface{}{
				"vendor1": map[string]interface{}{
					"module1": true,
					"module2": map[string]interface{}{
						"key": map[string]interface{}{
							"nestedKey": map[string]interface{}{
								"subKey": "subValue",
							},
						},
					},
				},
			},
			expectedData: map[string]map[string]json.RawMessage{
				"vendor1": {
					"module1": json.RawMessage(`true`),
					"module2": json.RawMessage(`{"key":{"nestedKey":{"subKey":"subValue"}}}`),
				},
			},
			expectError: false,
		},
		{
			name:     "nil-vendor-config-handled-correctly",
			fromType: mapStringInterface,
			toType:   accountModulesType,
			inputData: map[string]interface{}{
				"vendor1": nil,
			},
			expectedData: map[string]map[string]json.RawMessage{
				"vendor1": nil,
			},
			expectError: false,
		},
		{
			name:     "invalid-inner-map-returns-error",
			fromType: mapStringInterface,
			toType:   accountModulesType,
			inputData: map[string]interface{}{
				"vendor1": "not-a-map",
			},
			expectError:   true,
			errorContains: "failed to convert inner map for vendor1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := hookFunc(tt.fromType, tt.toType, tt.inputData)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)

				if tt.expectSamePtr {
					// When the input should be returned unchanged
					assert.Equal(t, tt.inputData, result)
				} else {
					// For converted data, compare structures
					expectedVendors, ok := tt.expectedData.(map[string]map[string]json.RawMessage)
					assert.True(t, ok, "Expected data should be map[string]map[string]json.RawMessage")

					resultVendors, ok := result.(map[string]map[string]json.RawMessage)
					assert.True(t, ok, "Result should be map[string]map[string]json.RawMessage")

					// Verify maps have same key count
					assert.Len(t, resultVendors, len(expectedVendors))

					// Compare each module's config
					for vendorName, expectedModules := range expectedVendors {
						actualModules, exists := resultVendors[vendorName]
						assert.True(t, exists, "Expected vendor %s in result", vendorName)

						if expectedModules == nil {
							assert.Nil(t, actualModules, "Expected nil modules for vendor %s", vendorName)
							continue
						}

						// Compare each config key
						assert.Len(t, actualModules, len(expectedModules),
							"Config for vendor %s should have %d modules", vendorName, len(expectedModules))

						for moduleName, expectedConfig := range expectedModules {
							actualConfig, exists := actualModules[moduleName]
							assert.True(t, exists, "Expected module %s for vendor %s", moduleName, vendorName)

							// Compare JSON raw messages by converting to string
							assert.JSONEq(t, string(expectedConfig), string(actualConfig),
								"Value mismatch for vendor %s, module %s", vendorName, moduleName)
						}
					}
				}
			}
		})
	}
}

func TestConvertToRawMessageMap(t *testing.T) {
	tests := []struct {
		name           string
		input          map[string]interface{}
		expectedOutput map[string]map[string]json.RawMessage
		expectError    bool
		errorContains  string
	}{
		{
			name:           "nil-input",
			input:          nil,
			expectedOutput: map[string]map[string]json.RawMessage{},
			expectError:    false,
		},
		{
			name:           "empty-input",
			input:          map[string]interface{}{},
			expectedOutput: map[string]map[string]json.RawMessage{},
			expectError:    false,
		},
		{
			name: "nil-vendor-config",
			input: map[string]interface{}{
				"vendor1": nil,
			},
			expectedOutput: map[string]map[string]json.RawMessage{
				"vendor1": nil,
			},
			expectError: false,
		},
		{
			name: "nil-module-config",
			input: map[string]interface{}{
				"vendor1": map[string]interface{}{
					"module1": nil,
				},
			},
			expectedOutput: map[string]map[string]json.RawMessage{
				"vendor1": {
					"module1": json.RawMessage(`null`),
				},
			},
			expectError: false,
		},
		{
			name: "single-vendor-simple-module-values",
			input: map[string]interface{}{
				"vendor1": map[string]interface{}{
					"module1": true,
					"module2": 42,
					"module3": "test-module",
				},
			},
			expectedOutput: map[string]map[string]json.RawMessage{
				"vendor1": {
					"module1": json.RawMessage(`true`),
					"module2": json.RawMessage(`42`),
					"module3": json.RawMessage(`"test-module"`),
				},
			},
			expectError: false,
		},
		{
			name: "multiple-vendors-complex-module-values",
			input: map[string]interface{}{
				"vendor1": map[string]interface{}{
					"module1": map[string]interface{}{
						"key": "value",
					},
					"module2": map[string]interface{}{
						"key": map[string]interface{}{
							"nestedKey": map[string]interface{}{
								"subKey": "subValue",
							},
						},
					},
				},
				"vendor2": map[string]interface{}{
					"module3": false,
					"module4": []interface{}{1, 2, 3},
				},
			},
			expectedOutput: map[string]map[string]json.RawMessage{
				"vendor1": {
					"module1": json.RawMessage(`{"key":"value"}`),
					"module2": json.RawMessage(`{"key":{"nestedKey":{"subKey":"subValue"}}}`),
				},
				"vendor2": {
					"module3": json.RawMessage(`false`),
					"module4": json.RawMessage(`[1,2,3]`),
				},
			},
			expectError: false,
		},
		{
			name: "invalid-inner-map-type",
			input: map[string]interface{}{
				"vendor1": "not-a-map",
			},
			expectError:   true,
			errorContains: "failed to convert inner map for vendor1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertToRawMessageMap(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)

				assert.Equal(t, len(tt.expectedOutput), len(result), "Output maps should have the same number of keys")

				// Compare each vendor's config
				for vendor, expectedModules := range tt.expectedOutput {
					modules, exists := result[vendor]
					assert.True(t, exists, "Expected vendor %s in result", vendor)

					if expectedModules == nil {
						assert.Nil(t, modules, "Expected nil config for vendor %s", vendor)
						continue
					}

					// Compare each module key
					assert.Len(t, modules, len(expectedModules),
						"Config for vendor %s should have %d modules", vendor, len(expectedModules))

					for module, expectedConfig := range expectedModules {
						actualConfig, exists := modules[module]
						assert.True(t, exists, "Expected config key %s for module %s", module, vendor)

						assert.JSONEq(t, string(expectedConfig), string(actualConfig),
							"Config mismatch for vendor %s, module %s", vendor, module)
					}
				}
			}
		})
	}
}
