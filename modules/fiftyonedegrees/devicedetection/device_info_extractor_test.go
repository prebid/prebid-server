package devicedetection

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type ResultsHashMock struct {
	mock.Mock
}

func (m *ResultsHashMock) DeviceId() (string, error) {
	return "", nil
}

func (m *ResultsHashMock) ValuesString(prop1 string, prop2 string) (string, error) {
	args := m.Called(prop1, prop2)
	return args.String(0), args.Error(1)
}

func (m *ResultsHashMock) HasValues(prop1 string) (bool, error) {
	args := m.Called(prop1)
	return args.Bool(0), args.Error(1)
}

func TestDeviceInfoExtraction(t *testing.T) {
	results := &ResultsHashMock{}

	extractor := newDeviceInfoExtractor()
	mockValue(results, "HardwareName", "Macbook")
	mockValues(results)

	deviceInfo, _ := extractor.extract(results, "ua")
	assert.NotNil(t, deviceInfo)

	assert.Equal(t, deviceInfo.HardwareName, "Macbook")
	assertDeviceInfo(t, deviceInfo)
}

func TestDeviceInfoExtractionNoProperty(t *testing.T) {
	results := &ResultsHashMock{}

	extractor := newDeviceInfoExtractor()
	results.Mock.On("ValuesString", "HardwareName", ",").Return("", errors.New("Error"))
	results.Mock.On("HasValues", "HardwareName").Return(true, nil)
	mockValues(results)

	deviceInfo, _ := extractor.extract(results, "ua")
	assert.NotNil(t, deviceInfo)

	assertDeviceInfo(t, deviceInfo)
	assert.Equal(t, deviceInfo.HardwareName, "")
}

func TestDeviceInfoExtractionNoValue(t *testing.T) {
	results := &ResultsHashMock{}

	extractor := newDeviceInfoExtractor()
	mockValues(results)
	mockValue(results, "HardwareVendor", "Apple")

	results.Mock.On("ValuesString", "HardwareName", ",").Return("Macbook", nil)
	results.Mock.On("HasValues", "HardwareName").Return(false, nil)

	deviceInfo, _ := extractor.extract(results, "ua")
	assert.NotNil(t, deviceInfo)
	assertDeviceInfo(t, deviceInfo)
	assert.Equal(t, deviceInfo.HardwareName, "Unknown")
}

func TestDeviceInfoExtractionHasValueError(t *testing.T) {
	results := &ResultsHashMock{}

	extractor := newDeviceInfoExtractor()
	mockValue(results, "HardwareVendor", "Apple")

	results.Mock.On("ValuesString", "HardwareName", ",").Return("Macbook", nil)
	results.Mock.On("HasValues", "HardwareName").Return(true, errors.New("error"))

	mockValues(results)

	deviceInfo, _ := extractor.extract(results, "ua")
	assert.NotNil(t, deviceInfo)
	assertDeviceInfo(t, deviceInfo)
	assert.Equal(t, deviceInfo.HardwareName, "")
}

func mockValues(results *ResultsHashMock) {
	mockValue(results, "HardwareVendor", "Apple")
	mockValue(results, "DeviceType", "Desctop")
	mockValue(results, "PlatformVendor", "Apple")
	mockValue(results, "PlatformName", "MacOs")
	mockValue(results, "PlatformVersion", "14")
	mockValue(results, "BrowserVendor", "Google")
	mockValue(results, "BrowserName", "Crome")
	mockValue(results, "BrowserVersion", "12")
	mockValue(results, "ScreenPixelsWidth", "1024")
	mockValue(results, "ScreenPixelsHeight", "1080")
	mockValue(results, "PixelRatio", "223")
	mockValue(results, "Javascript", "true")
	mockValue(results, "GeoLocation", "true")
	mockValue(results, "HardwareModel", "Macbook")
	mockValue(results, "HardwareFamily", "Macbook")
	mockValue(results, "HardwareModelVariants", "Macbook")
	mockValue(results, "ScreenInchesHeight", "12")
}

func assertDeviceInfo(t *testing.T, deviceInfo *deviceInfo) {
	assert.Equal(t, deviceInfo.HardwareVendor, "Apple")
	assert.Equal(t, deviceInfo.DeviceType, "Desctop")
	assert.Equal(t, deviceInfo.PlatformVendor, "Apple")
	assert.Equal(t, deviceInfo.PlatformName, "MacOs")
	assert.Equal(t, deviceInfo.PlatformVersion, "14")
	assert.Equal(t, deviceInfo.BrowserVendor, "Google")
	assert.Equal(t, deviceInfo.BrowserName, "Crome")
	assert.Equal(t, deviceInfo.BrowserVersion, "12")
	assert.Equal(t, deviceInfo.ScreenPixelsWidth, int64(1024))
	assert.Equal(t, deviceInfo.ScreenPixelsHeight, int64(1080))
	assert.Equal(t, deviceInfo.PixelRatio, float64(223))
	assert.Equal(t, deviceInfo.Javascript, true)
	assert.Equal(t, deviceInfo.GeoLocation, true)
}

func mockValue(results *ResultsHashMock, name string, value string) {
	results.Mock.On("ValuesString", name, ",").Return(value, nil)
	results.Mock.On("HasValues", name).Return(true, nil)
}
