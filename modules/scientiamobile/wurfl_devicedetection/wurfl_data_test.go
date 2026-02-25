package wurfl_devicedetection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWurflData_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		data     wurflData
		expected string
	}{
		{
			name: "Non-empty wurflData",
			data: wurflData{
				"brand_name": "BrandX",
				"model_name": "ModelY",
			},
			expected: `{"brand_name":"BrandX","model_name":"ModelY"}`,
		},
		{
			name:     "Empty wurflData",
			data:     wurflData{},
			expected: `{}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.data.MarshalJSON()
			assert.NoError(t, err)
			assert.JSONEq(t, tc.expected, string(result))
		})
	}
}

func TestWurflData_SON(t *testing.T) {
	tests := []struct {
		name        string
		data        wurflData
		expected    string
		expectedErr bool
	}{
		{
			name: "Non-empty wurflData",
			data: wurflData{
				"brand_name": "BrandX",
				"model_name": "ModelY",
				"wurfl_id":   "test",
			},
			expected: `{"wurfl_id":"test"}`,
		},
		{
			name:        "Missed WURFL ID",
			data:        wurflData{},
			expectedErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.data.WurflIDToJSON()
			if tc.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				return
			}
			assert.NoError(t, err)
			assert.JSONEq(t, tc.expected, string(result))
		})
	}
}

func TestWurflData_Bool(t *testing.T) {
	data := wurflData{
		"ajax_support_javascript": "true",
		"invalid_value":           "not_a_bool",
	}

	v, err := data.Bool("ajax_support_javascript")
	assert.NoError(t, err)
	assert.True(t, v)

	v, err = data.Bool("invalid_value")
	assert.Error(t, err)
	assert.Empty(t, v)

	v, err = data.Bool("non_existent_key")
	assert.Error(t, err)
	assert.Empty(t, v)
}

func TestWurflData_Float64(t *testing.T) {
	data := wurflData{
		"density_class": "2.5",
		"invalid_value": "not_a_number",
	}

	v, err := data.Float64("density_class")
	assert.NoError(t, err)
	assert.Equal(t, 2.5, v)

	v, err = data.Float64("invalid_value")
	assert.Empty(t, v)
	assert.Error(t, err)

	v, err = data.Float64("non_existent_key")
	assert.Empty(t, v)
	assert.Error(t, err)
}

func TestWurflData_Int64(t *testing.T) {
	data := wurflData{
		"resolution_height": "1080",
		"invalid_value":     "not_a_number",
	}

	v, err := data.Int64("resolution_height")
	assert.NoError(t, err)
	assert.Equal(t, int64(1080), v)

	v, err = data.Int64("invalid_value")
	assert.Empty(t, v)
	assert.Error(t, err)

	v, err = data.Int64("non_existent_key")
	assert.Empty(t, v)
	assert.Error(t, err)
}

func TestWurflData_String(t *testing.T) {
	data := wurflData{
		"brand_name": "BrandX",
	}

	v, err := data.String("brand_name")
	assert.NoError(t, err)
	assert.Equal(t, "BrandX", v)

	v, err = data.String("non_existent_key")
	assert.Empty(t, v)
	assert.Error(t, err)
}
