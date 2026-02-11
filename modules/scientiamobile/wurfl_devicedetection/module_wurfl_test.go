//go:build wurfl

package wurfl_devicedetection

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/modules/moduledeps"
	"github.com/stretchr/testify/assert"
)

func TestBuilder(t *testing.T) {
	tests := []struct {
		name        string
		configRaw   json.RawMessage
		expectedErr bool
		validate    func(t *testing.T, module interface{})
	}{
		{
			name: "Valid configuration",
			configRaw: json.RawMessage(`{
				"wurfl_snapshot_url": "http://example.com/wurfl-data",
				"wurfl_file_path": "/tmp/wurfl.zip",
				"allowed_publisher_ids": ["pub1", "pub2"],
				"ext_caps": true
			}`),
			expectedErr: false,
			validate: func(t *testing.T, module interface{}) {
				m, ok := module.(Module)
				assert.True(t, ok, "Module type assertion failed")
				assert.Equal(t, map[string]struct{}{"pub1": {}, "pub2": {}}, m.allowedPublisherIDs)
				assert.True(t, m.extCaps)
				assert.NotNil(t, m.we)
			},
		},
		{
			name:        "Invalid configuration - newConfig fails",
			configRaw:   json.RawMessage(`{ "wurfl_snapshot_url": "http://example.com/wurfl-data" }`), // Missing required fields
			expectedErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			module, err := Builder(tc.configRaw, moduledeps.ModuleDeps{})

			if tc.expectedErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tc.validate != nil {
				tc.validate(t, module)
			}
		})
	}
}
