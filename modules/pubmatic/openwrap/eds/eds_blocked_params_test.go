package eds

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsEmptyJSONObject(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "empty object", input: `{}`, want: true},
		{name: "outer whitespace", input: `  {}  `, want: true},
		{name: "has keys", input: `{"boottime":1}`, want: false},
		{name: "not an object", input: `"value"`, want: false},
		{name: "array", input: `[]`, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isEmptyJSONObject([]byte(tt.input)))
		})
	}
}

func TestStripEDSTier1ParamsForBlockedCountry(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "strips blocked device and app params and keeps allowed fields",
			input:    `{"pubmatic":{"eds":{"device":{"boottime":123,"charging":1,"diskspace":10.5,"screenbright":0.8,"totaldisk":100,"inputlanguage":"en","totalmem":2048},"app":{"install_time":456,"first_launch_time":789,"other":"keep"}}}}`,
			expected: `{"pubmatic":{"eds":{"device":{"charging":1,"screenbright":0.8},"app":{"other":"keep"}}}}`,
		},
		{
			name:     "empty bidderparams object returns nil",
			input:    `{}`,
			expected: ``,
		},
		{
			name:     "no pubmatic key is unchanged",
			input:    `{"rubicon":{"account":"123"}}`,
			expected: `{"rubicon":{"account":"123"}}`,
		},
		{
			name:     "pubmatic without eds is unchanged",
			input:    `{"pubmatic":{"profileid":1}}`,
			expected: `{"pubmatic":{"profileid":1}}`,
		},
		{
			name:     "eds without device or app is unchanged",
			input:    `{"pubmatic":{"eds":{"other":"value"}}}`,
			expected: `{"pubmatic":{"eds":{"other":"value"}}}`,
		},
		{
			name:     "device is not an object is unchanged",
			input:    `{"pubmatic":{"eds":{"device":"not-an-object","app":{"other":"keep"}}}}`,
			expected: `{"pubmatic":{"eds":{"device":"not-an-object","app":{"other":"keep"}}}}`,
		},
		{
			name:     "app is not an object is unchanged",
			input:    `{"pubmatic":{"eds":{"device":{"charging":1},"app":123}}}`,
			expected: `{"pubmatic":{"eds":{"device":{"charging":1},"app":123}}}`,
		},
		{
			name:     "only blocked device params removes empty device object",
			input:    `{"pubmatic":{"eds":{"device":{"boottime":1,"diskspace":2,"totaldisk":3,"inputlanguage":"en","totalmem":4},"app":{"other":"keep"}}}}`,
			expected: `{"pubmatic":{"eds":{"app":{"other":"keep"}}}}`,
		},
		{
			name:     "only blocked app params removes empty app object",
			input:    `{"pubmatic":{"eds":{"device":{"charging":1},"app":{"install_time":1,"first_launch_time":2}}}}`,
			expected: `{"pubmatic":{"eds":{"device":{"charging":1}}}}`,
		},
		{
			name:     "only blocked params in eds removes eds object",
			input:    `{"pubmatic":{"eds":{"device":{"boottime":1},"app":{"install_time":2}},"profileid":1}}`,
			expected: `{"pubmatic":{"profileid":1}}`,
		},
		{
			name:     "no blocked params present is unchanged",
			input:    `{"pubmatic":{"eds":{"device":{"charging":1},"app":{"other":"keep"}}}}`,
			expected: `{"pubmatic":{"eds":{"device":{"charging":1},"app":{"other":"keep"}}}}`,
		},
		{
			name:     "partial blocked params strips only present keys",
			input:    `{"pubmatic":{"eds":{"device":{"boottime":1,"charging":1},"app":{"install_time":2,"other":"keep"}}}}`,
			expected: `{"pubmatic":{"eds":{"device":{"charging":1},"app":{"other":"keep"}}}}`,
		},
		{
			name:     "other bidder keys are preserved",
			input:    `{"pubmatic":{"eds":{"device":{"boottime":1,"charging":1}}},"rubicon":{"account":"abc"}}`,
			expected: `{"pubmatic":{"eds":{"device":{"charging":1}}},"rubicon":{"account":"abc"}}`,
		},
		{
			name:     "strips each blocked device param individually",
			input:    `{"pubmatic":{"eds":{"device":{"boottime":1}}}}`,
			expected: `{"pubmatic":{}}`,
		},
		{
			name:     "strips diskspace from device",
			input:    `{"pubmatic":{"eds":{"device":{"diskspace":10.5,"charging":1}}}}`,
			expected: `{"pubmatic":{"eds":{"device":{"charging":1}}}}`,
		},
		{
			name:     "strips totaldisk from device",
			input:    `{"pubmatic":{"eds":{"device":{"totaldisk":100,"charging":1}}}}`,
			expected: `{"pubmatic":{"eds":{"device":{"charging":1}}}}`,
		},
		{
			name:     "strips inputlanguage from device",
			input:    `{"pubmatic":{"eds":{"device":{"inputlanguage":"en","charging":1}}}}`,
			expected: `{"pubmatic":{"eds":{"device":{"charging":1}}}}`,
		},
		{
			name:     "strips totalmem from device",
			input:    `{"pubmatic":{"eds":{"device":{"totalmem":2048,"charging":1}}}}`,
			expected: `{"pubmatic":{"eds":{"device":{"charging":1}}}}`,
		},
		{
			name:     "strips install_time from app",
			input:    `{"pubmatic":{"eds":{"app":{"install_time":456,"other":"keep"}}}}`,
			expected: `{"pubmatic":{"eds":{"app":{"other":"keep"}}}}`,
		},
		{
			name:     "strips first_launch_time from app",
			input:    `{"pubmatic":{"eds":{"app":{"first_launch_time":789,"other":"keep"}}}}`,
			expected: `{"pubmatic":{"eds":{"app":{"other":"keep"}}}}`,
		},
		{
			name:     "removes already empty device object without iterating blocked keys",
			input:    `{"pubmatic":{"eds":{"device":{},"app":{"other":"keep"}}}}`,
			expected: `{"pubmatic":{"eds":{"app":{"other":"keep"}}}}`,
		},
		{
			name:     "removes already empty app object without iterating blocked keys",
			input:    `{"pubmatic":{"eds":{"device":{"charging":1},"app":{}}}}`,
			expected: `{"pubmatic":{"eds":{"device":{"charging":1}}}}`,
		},
		{
			name:     "removes already empty device and app objects and then eds",
			input:    `{"pubmatic":{"eds":{"device":{},"app":{}},"profileid":1}}`,
			expected: `{"pubmatic":{"profileid":1}}`,
		},
		{
			name:     "removes already empty eds object",
			input:    `{"pubmatic":{"eds":{},"profileid":1}}`,
			expected: `{"pubmatic":{"profileid":1}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripEDSTier1ParamsForBlockedCountry(json.RawMessage(tt.input))

			if tt.expected == "" {
				assert.Nil(t, got)
				return
			}

			assert.JSONEq(t, tt.expected, string(got))
		})
	}
}

func TestStripEDSTier1ParamsForBlockedCountry_emptyInput(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		assert.Nil(t, StripEDSTier1ParamsForBlockedCountry(nil))
	})

	t.Run("empty byte slice", func(t *testing.T) {
		got := StripEDSTier1ParamsForBlockedCountry(json.RawMessage{})
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})
}
