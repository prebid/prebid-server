package devicedetection

import (
	"os"
	"testing"

	"github.com/51Degrees/device-detection-go/v4/dd"
	"github.com/stretchr/testify/assert"
)

func TestParseConfig(t *testing.T) {
	cfgRaw := []byte(`{ 
		"enabled": true,
		"data_file": {
            "path": "path/to/51Degrees-LiteV4.1.hash",
            "update": {
				"auto": true,
				"url": "https://my.datafile.com/datafile.gz",
				"polling_interval": 3600,
				"license_key": "your_license_key",
				"product": "V4Enterprise",
				"on_startup": true
            }
		},
		"account_filter": {"allow_list": ["123"]},
		"performance": {
			"profile": "default",
			"concurrency": 1,
			"difference": 1,
			"allow_unmatched": true,
			"drift": 1	
		}
	}`)

	cfg, err := parseConfig(cfgRaw)

	assert.NoError(t, err)

	assert.Equal(t, cfg.DataFile.Path, "path/to/51Degrees-LiteV4.1.hash")
	assert.True(t, cfg.DataFile.Update.Auto)
	assert.Equal(t, cfg.DataFile.Update.Url, "https://my.datafile.com/datafile.gz")
	assert.Equal(t, cfg.DataFile.Update.PollingInterval, 3600)
	assert.Equal(t, cfg.DataFile.Update.License, "your_license_key")
	assert.Equal(t, cfg.DataFile.Update.Product, "V4Enterprise")
	assert.True(t, cfg.DataFile.Update.OnStartup)
	assert.Equal(t, cfg.AccountFilter.AllowList, []string{"123"})
	assert.Equal(t, cfg.Performance.Profile, "default")
	assert.Equal(t, *cfg.Performance.Concurrency, 1)
	assert.Equal(t, *cfg.Performance.Difference, 1)
	assert.True(t, *cfg.Performance.AllowUnmatched)
	assert.Equal(t, *cfg.Performance.Drift, 1)
	assert.Equal(t, cfg.getPerformanceProfile(), dd.Default)
}

func TestValidateConfig(t *testing.T) {
	file, err := os.Create("test-validate-config.hash")
	if err != nil {
		t.Errorf("Failed to create file: %v", err)
	}
	defer file.Close()
	defer os.Remove("test-validate-config.hash")

	cfgRaw := []byte(`{ 
		"enabled": true,
		"data_file": {
			"path": "test-validate-config.hash",
			"update": {
				"auto": true,
				"url": "https://my.datafile.com/datafile.gz",
				"polling_interval": 3600,
				"licence_key": "your_licence_key",
				"product": "V4Enterprise"
			}
		},
		"account_filter": {"allow_list": ["123"]},
		"performance": {
			"profile": "default",
			"concurrency": 1,
			"difference": 1,
			"allow_unmatched": true,
			"drift": 1	
		}
	}`)

	cfg, err := parseConfig(cfgRaw)
	assert.NoError(t, err)

	err = validateConfig(cfg)
	assert.NoError(t, err)

}

func TestInvalidPerformanceProfile(t *testing.T) {
	cfgRaw := []byte(`{ 
		"enabled": true,
		"data_file": {
			"path": "test-validate-config.hash",
			"update": {
				"auto": true,
				"url": "https://my.datafile.com/datafile.gz",
				"polling_interval": 3600,
				"licence_key": "your_licence_key",
				"product": "V4Enterprise"
			}
		},
		"account_filter": {"allow_list": ["123"]},
		"performance": {
			"profile": "123",
			"concurrency": 1,
			"difference": 1,
			"allow_unmatched": true,
			"drift": 1	
		}
	}`)
	cfg, err := parseConfig(cfgRaw)
	assert.NoError(t, err)

	assert.Equal(t, cfg.getPerformanceProfile(), dd.Default)
}
