package config

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// AccountModulesHookFunc returns a mapstructure.DecodeHookFuncType that converts
// a map[string]interface{} to a map[string]map[string]json.RawMessage for the
// AccountModules type. This is used to handle the custom decoding of account modules
// in the configuration.
func AccountModulesHookFunc() mapstructure.DecodeHookFuncType {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if f.Kind() != reflect.Map {
			return data, nil
		}

		if t != reflect.TypeOf(AccountModules{}) {
			return data, nil
		}

		result, err := convertToRawMessageMap(data.(map[string]interface{}))
		if err != nil {
			return nil, fmt.Errorf("failed to convert account modules map: %w", err)
		}
		return result, nil
	}
}

// convertToRawMessageMap converts a map[string]interface{} to a map[string]map[string]json.RawMessage.
// It marshals each inner value to json.RawMessage, allowing for flexible storage of arbitrary JSON structures.
func convertToRawMessageMap(input map[string]interface{}) (map[string]map[string]json.RawMessage, error) {
	result := make(map[string]map[string]json.RawMessage, len(input))

	for outerKey, outerValue := range input {
		if outerValue == nil {
			result[outerKey] = nil
			continue
		}

		innerMap, ok := outerValue.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("failed to convert inner map for %s", outerKey)
		}

		resultInnerMap := make(map[string]json.RawMessage, len(innerMap))
		for innerKey, innerValue := range innerMap {
			rawBytes, err := jsonutil.Marshal(innerValue)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal value for %s.%s: %w", outerKey, innerKey, err)
			}
			resultInnerMap[innerKey] = rawBytes
		}
		result[outerKey] = resultInnerMap
	}
	return result, nil
}
