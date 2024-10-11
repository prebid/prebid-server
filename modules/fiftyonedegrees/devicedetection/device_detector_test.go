package devicedetection

import (
	"fmt"
	"testing"

	"github.com/51Degrees/device-detection-go/v4/dd"
	"github.com/51Degrees/device-detection-go/v4/onpremise"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
						"product": "V4Enterprise",
						"watch_file_system": true,
						"on_startup": true
					},
					"make_temp_copy": true
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
			length: 11,
			// data_file.path, data_file.update.auto:true, url, polling_interval, license_key, product, confighash, properties
			// data_file.update.on_startup:true, data_file.update.watch_file_system:true, data_file.make_temp_copy:true
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
			length: 5, // data_file.update.auto:false, data_file.path, confighash, properties, data_file.update.on_startup:false
		},
	}

	for _, c := range cases {
		cfg, err := parseConfig(c.cfgRaw)
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

func (e *extractorMock) extract(results Results, ua string) (*deviceInfo, error) {
	args := e.Called(results, ua)
	return args.Get(0).(*deviceInfo), args.Error(1)
}

func TestGetDeviceInfo(t *testing.T) {
	tests := []struct {
		name           string
		engineResponse *dd.ResultsHash
		engineError    error
		expectedResult *deviceInfo
		expectedError  string
	}{
		{
			name:           "Success_path",
			engineResponse: &dd.ResultsHash{},
			engineError:    nil,
			expectedResult: &deviceInfo{
				DeviceId: "123",
			},
			expectedError: "",
		},
		{
			name:           "Error_path",
			engineResponse: nil,
			engineError:    fmt.Errorf("error"),
			expectedResult: nil,
			expectedError:  "Failed to process evidence: error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractorM := &extractorMock{}
			extractorM.On("extract", mock.Anything, mock.Anything).Return(
				&deviceInfo{
					DeviceId: "123",
				}, nil,
			)

			engineM := &engineMock{}
			engineM.On("Process", mock.Anything).Return(
				tt.engineResponse, tt.engineError,
			)

			deviceDetector := defaultDeviceDetector{
				cfg:                 nil,
				deviceInfoExtractor: extractorM,
				engine:              engineM,
			}

			result, err := deviceDetector.getDeviceInfo(
				[]onpremise.Evidence{{
					Prefix: dd.HttpEvidenceQuery,
					Key:    "key",
					Value:  "val",
				}}, "ua",
			)

			if tt.expectedError == "" {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedResult.DeviceId, result.DeviceId)
			} else {
				assert.Errorf(t, err, tt.expectedError)
				assert.Nil(t, result)
			}
		})
	}
}

func TestGetSupportedHeaders(t *testing.T) {
	engineM := &engineMock{}

	engineM.On("GetHttpHeaderKeys").Return(
		[]dd.EvidenceKey{{
			Key:    "key",
			Prefix: dd.HttpEvidenceQuery,
		}},
	)

	deviceDetector := defaultDeviceDetector{
		cfg:                 nil,
		deviceInfoExtractor: nil,
		engine:              engineM,
	}

	result := deviceDetector.getSupportedHeaders()
	assert.NotNil(t, result)
	assert.Equal(t, len(result), 1)
	assert.Equal(t, result[0].Key, "key")

}
