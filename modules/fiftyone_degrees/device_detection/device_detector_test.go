package device_detection

import (
	"fmt"
	"github.com/51Degrees/device-detection-go/v4/dd"
	"github.com/51Degrees/device-detection-go/v4/onpremise"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestBuildEngineOptions(t *testing.T) {
	cases := []struct {
		cfgRaw []byte
		length int
	}{
		{
			cfgRaw: []byte(`{ 
"enabled": true,
          "data_file": {
            "path": "path/to/51Degrees-LiteV4.1.hash",
            "update": {
              "auto": true,
              "url": "https://my.datafile.com/datafile.gz",
              "polling_interval": 3600,
              "license_key": "your_license_key",
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
}`),
			length: 8,
			// data_file.path, data_file.update.auto:true, url, polling_interval, license_key, product, confighash, properties
		},
		{
			cfgRaw: []byte(`{ 
"enabled": true,
          "data_file": {
            "path": "path/to/51Degrees-LiteV4.1.hash"
          },
          "account_filter": {"allow_list": ["123"]},
				  "performance": {
					"profile": "default",
					"concurrency": 1,
					"difference": 1,
					"allow_unmatched": true,
					"drift": 1	
				  }
}`),
			length: 4, // data_file.update.auto:false, data_file.path, confighash, properties
		},
	}

	for _, c := range cases {
		cfg, err := ParseConfig(c.cfgRaw)
		assert.NoError(t, err)
		configHash := configHashFromConfig(&cfg)
		options := buildEngineOptions(&cfg, configHash)
		assert.Equal(t, c.length, len(options))
	}
}

type engineMock struct {
	mock.Mock
}

func (e *engineMock) Process(evidences []onpremise.Evidence) (*dd.ResultsHash, error) {
	args := e.Called(evidences)
	res := args.Get(0)
	if res == nil {
		return nil, args.Error(1)
	}

	return res.(*dd.ResultsHash), args.Error(1)
}

func (e *engineMock) GetHttpHeaderKeys() []dd.EvidenceKey {
	args := e.Called()
	return args.Get(0).([]dd.EvidenceKey)
}

type extractorMock struct {
	mock.Mock
}

func (e *extractorMock) Extract(results Results, ua string) (*DeviceInfo, error) {
	args := e.Called(results, ua)
	return args.Get(0).(*DeviceInfo), args.Error(1)
}

func TestGetDeviceInfo(t *testing.T) {
	var extractorM = &extractorMock{}

	extractorM.On("Extract", mock.Anything, mock.Anything).Return(
		&DeviceInfo{
			DeviceId: "123",
		}, nil,
	)

	var engineM = &engineMock{}

	engineM.On("Process", mock.Anything).Return(
		&dd.ResultsHash{}, nil,
	)

	deviceDetector := DeviceDetector{
		cfg:                 nil,
		deviceInfoExtractor: extractorM,
		engine:              engineM,
	}

	result, err := deviceDetector.GetDeviceInfo(
		[]onpremise.Evidence{{
			Prefix: dd.HttpEvidenceQuery,
			Key:    "key",
			Value:  "val",
		}}, "ua",
	)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, result.DeviceId, "123")

	errorEngineM := &engineMock{}

	errorEngineM.On("Process", mock.Anything).Return(nil, fmt.Errorf("error"))

	deviceDetector = DeviceDetector{
		cfg:                 nil,
		deviceInfoExtractor: extractorM,
		engine:              errorEngineM,
	}

	result, err = deviceDetector.GetDeviceInfo(
		[]onpremise.Evidence{{
			Prefix: dd.HttpEvidenceQuery,
			Key:    "key",
			Value:  "val",
		}},
		"ua",
	)
	assert.Errorf(t, err, "Failed to process evidence: error")
	assert.Nil(t, result)
}

func TestGetSupportedHeaders(t *testing.T) {
	engineM := &engineMock{}

	engineM.On("GetHttpHeaderKeys").Return(
		[]dd.EvidenceKey{{
			Key:    "key",
			Prefix: dd.HttpEvidenceQuery,
		}},
	)

	deviceDetector := DeviceDetector{
		cfg:                 nil,
		deviceInfoExtractor: nil,
		engine:              engineM,
	}

	result := deviceDetector.GetSupportedHeaders()
	assert.NotNil(t, result)
	assert.Equal(t, len(result), 1)
	assert.Equal(t, result[0].Key, "key")

}
