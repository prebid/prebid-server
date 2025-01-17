package wurfl_devicedetection

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name        string
		input       json.RawMessage
		expectedErr bool
		validate    func(t *testing.T, cfg config)
	}{
		{
			name: "Valid config with default cache size",
			input: json.RawMessage(`{
				"wurfl_snapshot_url": "http://example.com/wurfl-data",
				"wurfl_file_dir_path": "/tmp/wurfl",
				"wurfl_run_updater": false 
			}`),
			expectedErr: false,
			validate: func(t *testing.T, cfg config) {
				assert.Equal(t, "http://example.com/wurfl-data", cfg.WURFLSnapshotURL)
				assert.Equal(t, "/tmp/wurfl", cfg.WURFLFileDirPath)
				assert.False(t, *cfg.WURFLRunUpdater)
				assert.Equal(t, defaultCacheSize, cfg.WURFLEngineCacheSize())
			},
		},
		{
			name: "Valid config with custom cache size",
			input: json.RawMessage(`{
				"wurfl_snapshot_url": "http://example.com/wurfl-data",
				"wurfl_file_dir_path": "/tmp/wurfl",
				"wurfl_cache_size": 5000,
				"wurfl_run_updater": true
			}`),
			expectedErr: false,
			validate: func(t *testing.T, cfg config) {
				assert.Equal(t, "5000", cfg.WURFLEngineCacheSize())
				assert.True(t, *cfg.WURFLRunUpdater)
			},
		},
		{
			name: "Invalid config - missing wurfl_snapshot_url",
			input: json.RawMessage(`{
				"wurfl_file_dir_path": "/tmp/wurfl",
			}`),
			expectedErr: true,
		},
		{
			name: "Invalid config - missing wurfl_file_dir_path",
			input: json.RawMessage(`{
				"wurfl_snapshot_url": "http://example.com/wurfl-data",
			}`),
			expectedErr: true,
		},
		{
			name: "Default wurfl_run_updater",
			input: json.RawMessage(`{
				"wurfl_snapshot_url": "http://example.com/wurfl-data",
				"wurfl_file_dir_path": "/tmp/wurfl"
			}`),
			expectedErr: false,
			validate: func(t *testing.T, cfg config) {
				assert.Nil(t, cfg.WURFLRunUpdater)
			},
		},
		{
			name:        "Invalid config - malformed JSON",
			input:       json.RawMessage(`{ "wurfl_snapshot_url": "http://example.com/wurfl-data", "wurfl_file_dir_path": "/tmp/wurfl",`), // Malformed JSON
			expectedErr: true,
		},
		{
			name:        "Empty config",
			input:       json.RawMessage(`{}`),
			expectedErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := newConfig(tc.input)

			if tc.expectedErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tc.validate != nil {
				tc.validate(t, cfg)
			}
		})
	}
}

func TestWURFLFilePath(t *testing.T) {
	cfg := config{
		WURFLFileDirPath: "/tmp/wurfl",
		WURFLSnapshotURL: "http://example.com/wurfl-data/wurfl.zip",
	}
	expectedPath := "/tmp/wurfl/wurfl.zip"
	assert.Equal(t, expectedPath, cfg.WURFLFilePath())
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		cfg         config
		expectedErr bool
	}{
		{
			name: "Valid config",
			cfg: config{
				WURFLSnapshotURL: "http://example.com/wurfl-data",
				WURFLFileDirPath: "/tmp/wurfl",
			},
			expectedErr: false,
		},
		{
			name: "Invalid config - missing wurfl_snapshot_url",
			cfg: config{
				WURFLFileDirPath: "/tmp/wurfl",
			},
			expectedErr: true,
		},
		{
			name: "Invalid config - missing wurfl_file_dir_path",
			cfg: config{
				WURFLSnapshotURL: "http://example.com/wurfl-data",
			},
			expectedErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.validate()

			if tc.expectedErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}
