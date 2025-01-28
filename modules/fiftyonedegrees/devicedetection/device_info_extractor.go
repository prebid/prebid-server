package devicedetection

import (
	"fmt"
	"strconv"

	"github.com/golang/glog"
)

// deviceInfoExtractor is a struct that contains the methods to extract device information
// from the results of the device detection
type deviceInfoExtractor struct{}

func newDeviceInfoExtractor() deviceInfoExtractor {
	return deviceInfoExtractor{}
}

type Results interface {
	ValuesString(string, string) (string, error)
	HasValues(string) (bool, error)
	DeviceId() (string, error)
}

type deviceInfoProperty string

const (
	deviceInfoHardwareVendor        deviceInfoProperty = "HardwareVendor"
	deviceInfoHardwareName          deviceInfoProperty = "HardwareName"
	deviceInfoDeviceType            deviceInfoProperty = "DeviceType"
	deviceInfoPlatformVendor        deviceInfoProperty = "PlatformVendor"
	deviceInfoPlatformName          deviceInfoProperty = "PlatformName"
	deviceInfoPlatformVersion       deviceInfoProperty = "PlatformVersion"
	deviceInfoBrowserVendor         deviceInfoProperty = "BrowserVendor"
	deviceInfoBrowserName           deviceInfoProperty = "BrowserName"
	deviceInfoBrowserVersion        deviceInfoProperty = "BrowserVersion"
	deviceInfoScreenPixelsWidth     deviceInfoProperty = "ScreenPixelsWidth"
	deviceInfoScreenPixelsHeight    deviceInfoProperty = "ScreenPixelsHeight"
	deviceInfoPixelRatio            deviceInfoProperty = "PixelRatio"
	deviceInfoJavascript            deviceInfoProperty = "Javascript"
	deviceInfoGeoLocation           deviceInfoProperty = "GeoLocation"
	deviceInfoHardwareModel         deviceInfoProperty = "HardwareModel"
	deviceInfoHardwareFamily        deviceInfoProperty = "HardwareFamily"
	deviceInfoHardwareModelVariants deviceInfoProperty = "HardwareModelVariants"
	deviceInfoScreenInchesHeight    deviceInfoProperty = "ScreenInchesHeight"
)

func (x deviceInfoExtractor) extract(results Results, ua string) (*deviceInfo, error) {
	hardwareVendor := x.getValue(results, deviceInfoHardwareVendor)
	hardwareName := x.getValue(results, deviceInfoHardwareName)
	deviceType := x.getValue(results, deviceInfoDeviceType)
	platformVendor := x.getValue(results, deviceInfoPlatformVendor)
	platformName := x.getValue(results, deviceInfoPlatformName)
	platformVersion := x.getValue(results, deviceInfoPlatformVersion)
	browserVendor := x.getValue(results, deviceInfoBrowserVendor)
	browserName := x.getValue(results, deviceInfoBrowserName)
	browserVersion := x.getValue(results, deviceInfoBrowserVersion)
	screenPixelsWidth, _ := strconv.ParseInt(x.getValue(results, deviceInfoScreenPixelsWidth), 10, 64)
	screenPixelsHeight, _ := strconv.ParseInt(x.getValue(results, deviceInfoScreenPixelsHeight), 10, 64)
	pixelRatio, _ := strconv.ParseFloat(x.getValue(results, deviceInfoPixelRatio), 10)
	javascript, _ := strconv.ParseBool(x.getValue(results, deviceInfoJavascript))
	geoLocation, _ := strconv.ParseBool(x.getValue(results, deviceInfoGeoLocation))
	deviceId, err := results.DeviceId()
	if err != nil {
		return nil, fmt.Errorf("failed to get device id: %w", err)
	}
	hardwareModel := x.getValue(results, deviceInfoHardwareModel)
	hardwareFamily := x.getValue(results, deviceInfoHardwareFamily)
	hardwareModelVariants := x.getValue(results, deviceInfoHardwareModelVariants)
	screenInchedHeight, _ := strconv.ParseFloat(x.getValue(results, deviceInfoScreenInchesHeight), 10)

	p := &deviceInfo{
		HardwareVendor:        hardwareVendor,
		HardwareName:          hardwareName,
		DeviceType:            deviceType,
		PlatformVendor:        platformVendor,
		PlatformName:          platformName,
		PlatformVersion:       platformVersion,
		BrowserVendor:         browserVendor,
		BrowserName:           browserName,
		BrowserVersion:        browserVersion,
		ScreenPixelsWidth:     screenPixelsWidth,
		ScreenPixelsHeight:    screenPixelsHeight,
		PixelRatio:            pixelRatio,
		Javascript:            javascript,
		GeoLocation:           geoLocation,
		UserAgent:             ua,
		DeviceId:              deviceId,
		HardwareModel:         hardwareModel,
		HardwareFamily:        hardwareFamily,
		HardwareModelVariants: hardwareModelVariants,
		ScreenInchesHeight:    screenInchedHeight,
	}

	return p, nil
}

// function getValue return a value results for a property
func (x deviceInfoExtractor) getValue(results Results, propertyName deviceInfoProperty) string {
	// Get the values in string
	value, err := results.ValuesString(
		string(propertyName),
		",",
	)
	if err != nil {
		glog.Errorf("Failed to get results values string.")
		return ""
	}

	hasValues, err := results.HasValues(string(propertyName))
	if err != nil {
		glog.Errorf("Failed to check if a matched value exists for property %s.\n", propertyName)
		return ""
	}

	if !hasValues {
		glog.Warningf("Property %s does not have a matched value.\n", propertyName)
		return "Unknown"
	}

	return value
}
