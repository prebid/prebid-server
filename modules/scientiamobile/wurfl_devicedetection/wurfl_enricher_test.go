package wurfl_devicedetection

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestWurflEnricher_EnrichDevice(t *testing.T) {
	data := wurflData{
		"brand_name":              "BrandX",
		"model_name":              "ModelY",
		"advertised_device_os":    "Android",
		"resolution_height":       "1080",
		"resolution_width":        "1920",
		"pixel_density":           "300",
		"density_class":           "2.5",
		"ajax_support_javascript": "true",
		"is_mobile":               "true",
		"is_phone":                "true",
		"is_tablet":               "false",
	}

	device := &openrtb2.Device{}

	we := &wurflEnricher{
		WurflData: data,
	}
	we.EnrichDevice(device)

	assert.Equal(t, "BrandX", device.Make)
	assert.Equal(t, "ModelY", device.Model)
	assert.Equal(t, "Android", device.OS)
	assert.Equal(t, int64(1080), device.H)
	assert.Equal(t, int64(1920), device.W)
	assert.Equal(t, int64(300), device.PPI)
	assert.Equal(t, 2.5, device.PxRatio)
	assert.NotNil(t, device.JS)
	assert.Equal(t, int8(1), *device.JS)
	assert.Nil(t, device.Ext)
}

func TestWurflEnricher_EnrichDeviceExt(t *testing.T) {
	tests := []struct {
		name          string
		wurflData     wurflData
		initialExt    json.RawMessage
		expectedExt   string
		expectNoError bool
	}{
		{
			name: "Add wurfl data to empty device ext",
			wurflData: wurflData{
				"brand_name": "BrandX",
				"model_name": "ModelY",
			},
			initialExt:    nil,
			expectedExt:   `{"wurfl":{"brand_name":"BrandX","model_name":"ModelY"}}`,
			expectNoError: true,
		},
		{
			name: "Merge wurfl data into existing device ext",
			wurflData: wurflData{
				"brand_name": "BrandZ",
			},
			initialExt:    json.RawMessage(`{"existing_key":"existing_value"}`),
			expectedExt:   `{"existing_key":"existing_value","wurfl":{"brand_name":"BrandZ"}}`,
			expectNoError: true,
		},
		{
			name: "Invalid initial ext JSON",
			wurflData: wurflData{
				"brand_name": "BrandX",
			},
			initialExt:    json.RawMessage(`{"invalid_json":`), // Malformed JSON
			expectedExt:   `{"invalid_json":`,                  // Should remain as is
			expectNoError: false,
		},
		{
			name:          "Empty wurfl data",
			wurflData:     wurflData{},
			initialExt:    nil,
			expectedExt:   `{"wurfl":{}}`,
			expectNoError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			device := &openrtb2.Device{Ext: tc.initialExt}

			we := &wurflEnricher{
				WurflData: tc.wurflData,
				ExtCaps:   true,
			}
			// Call the method being tested
			we.EnrichDevice(device)

			// Assert the results
			if tc.expectNoError {
				assert.JSONEq(t, tc.expectedExt, string(device.Ext))
			} else {
				assert.NotEqual(t, tc.expectedExt, string(device.Ext))
			}
		})
	}
}

func TestWurflEnricher_MakeDeviceType(t *testing.T) {
	tests := []struct {
		name     string
		data     wurflData
		expected adcom1.DeviceType
	}{
		{
			name: "Mobile device - form_factor Other Mobile",
			data: wurflData{
				"form_factor": "Other Mobile",
			},
			expected: adcom1.DeviceMobile,
		},
		{
			name: "Smartphone device - form_factor Smartphone",
			data: wurflData{
				"form_factor": "Smartphone",
			},
			expected: adcom1.DevicePhone,
		},
		{
			name: "Feature Phone device - form_factor Feature Phone",
			data: wurflData{
				"form_factor": "Feature Phone",
			},
			expected: adcom1.DevicePhone,
		},
		{
			name: "Connected TV - form_factor Smart-TV",
			data: wurflData{
				"form_factor": "Smart-TV",
			},
			expected: adcom1.DeviceTV,
		},
		{
			name: "Full desktop - form_factor Desktop",
			data: wurflData{
				"form_factor": "Desktop",
			},
			expected: adcom1.DevicePC,
		},
		{
			name: "Tablet device - form_factor Tablet",
			data: wurflData{
				"form_factor": "Tablet",
			},
			expected: adcom1.DeviceTablet,
		},
		{
			name: "Connected device - form_factor Other Non-Mobile",
			data: wurflData{
				"form_factor": "Other Non-Mobile",
			},
			expected: adcom1.DeviceConnected,
		},
		{
			name: "Set-top box (OTT) - is_ott has priority",
			data: wurflData{
				"is_ott":      "true",
				"form_factor": "Desktop",
			},
			expected: adcom1.DeviceSetTopBox,
		},
		{
			name: "Console device - is_console has priority",
			data: wurflData{
				"is_console":  "true",
				"form_factor": "Desktop",
			},
			expected: adcom1.DeviceConnected,
		},
		{
			name: "Out-of-home device - physical_form_factor has priority",
			data: wurflData{
				"physical_form_factor": "out_of_home_device",
				"form_factor":          "Desktop",
			},
			expected: adcom1.DeviceOOH,
		},
		{
			name:     "Unknown device type - no form_factor",
			data:     wurflData{},
			expected: adcom1.DeviceType(0),
		},
		{
			name: "Unknown device type - invalid form_factor",
			data: wurflData{
				"form_factor": "Unknown",
			},
			expected: adcom1.DeviceType(0),
		},
	}

	for _, tc := range tests {
		we := &wurflEnricher{
			WurflData: tc.data,
		}
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, we.makeDeviceType())
		})
	}
}
