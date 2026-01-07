package wurfl_devicedetection

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strconv"

	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

const (
	wurflID = "wurfl_id"
)

var ErrWURFLIDNotExist = errors.New("WURFL ID does not exist")

// declare conformity with json.Marshaler interface
var _ json.Marshaler = wurflData{}

// wurflData represents the WURFL data
type wurflData map[string]string

// Bool retrieves a capability value as a bool
func (wd wurflData) Bool(key string) (bool, error) {
	val, exists := wd[key]
	if !exists {
		return false, fmt.Errorf("capability not found: %q", key)
	}
	result, err := strconv.ParseBool(val)
	if err != nil {
\t\treturn false, fmt.Errorf("could not parse %q to bool for capability %q", val, key)
	}
	return result, nil
}

// Int64 retrieves a capability value as an int64
func (wd wurflData) Int64(key string) (int64, error) {
	val, exists := wd[key]
	if !exists {
		return 0, fmt.Errorf("capability not found: %q", key)
	}
	result, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cound not parse %q to int64 for capability %q", val, key)
	}
	return result, nil
}

// Float64 retrieves a capability value as a float64
func (wd wurflData) Float64(key string) (float64, error) {
	val, exists := wd[key]
	if !exists {
		return 0.0, fmt.Errorf("capability not found: %q", key)
	}
	result, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0.0, fmt.Errorf("cound not parse %q to float64 for capability %q", val, key)
	}
	return result, nil
}

// String retrieves a capability value as a string
func (wd wurflData) String(key string) (string, error) {
	val, exists := wd[key]
	if !exists {
		return "", fmt.Errorf("capability not found: %q", key)
	}
	return val, nil
}

// WurflIDToJSON returns a JSON representation of the WURFL ID
func (wd wurflData) WurflIDToJSON() ([]byte, error) {
	m := make(map[string]string)
	v, ok := wd[wurflID]
	if !ok {
		return nil, ErrWURFLIDNotExist
	}
	m[wurflID] = v
	return jsonutil.Marshal(m)
}

// MarshalJSON customizes the JSON marshaling for wurflData
func (wd wurflData) MarshalJSON() ([]byte, error) {
	return jsonutil.Marshal(maps.Clone(map[string]string(wd)))
}
