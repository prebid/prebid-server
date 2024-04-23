package device_detection

import (
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"log"
	"strconv"
)

// DeviceInfoExtractor is a struct that contains the methods to extract device information
// from the results of the device detection
type DeviceInfoExtractor struct{}

func NewDeviceInfoExtractor() *DeviceInfoExtractor {
	return &DeviceInfoExtractor{}
}

type Results interface {
	ValuesString(string, string) (string, error)
	HasValues(string) (bool, error)
	DeviceId() (string, error)
}

type DeviceInfoProperty string

const (
	DeviceInfoHardwareVendor        DeviceInfoProperty = "HardwareVendor"
	DeviceInfoHardwareName          DeviceInfoProperty = "HardwareName"
	DeviceInfoDeviceType            DeviceInfoProperty = "DeviceType"
	DeviceInfoPlatformVendor        DeviceInfoProperty = "PlatformVendor"
	DeviceInfoPlatformName          DeviceInfoProperty = "PlatformName"
	DeviceInfoPlatformVersion       DeviceInfoProperty = "PlatformVersion"
	DeviceInfoBrowserVendor         DeviceInfoProperty = "BrowserVendor"
	DeviceInfoBrowserName           DeviceInfoProperty = "BrowserName"
	DeviceInfoBrowserVersion        DeviceInfoProperty = "BrowserVersion"
	DeviceInfoScreenPixelsWidth     DeviceInfoProperty = "ScreenPixelsWidth"
	DeviceInfoScreenPixelsHeight    DeviceInfoProperty = "ScreenPixelsHeight"
	DeviceInfoPixelRatio            DeviceInfoProperty = "PixelRatio"
	DeviceInfoJavascript            DeviceInfoProperty = "Javascript"
	DeviceInfoGeoLocation           DeviceInfoProperty = "GeoLocation"
	DeviceInfoHardwareModel         DeviceInfoProperty = "HardwareModel"
	DeviceInfoHardwareFamily        DeviceInfoProperty = "HardwareFamily"
	DeviceInfoHardwareModelVariants DeviceInfoProperty = "HardwareModelVariants"
	DeviceInfoScreenInchesHeight    DeviceInfoProperty = "ScreenInchesHeight"
)

func (x DeviceInfoExtractor) Extract(results Results, ua string) (*DeviceInfo, error) {
	hardwareVendor := x.getValue(results, DeviceInfoHardwareVendor)
	hardwareName := x.getValue(results, DeviceInfoHardwareName)
	deviceType := x.getValue(results, DeviceInfoDeviceType)
	platformVendor := x.getValue(results, DeviceInfoPlatformVendor)
	platformName := x.getValue(results, DeviceInfoPlatformName)
	platformVersion := x.getValue(results, DeviceInfoPlatformVersion)
	browserVendor := x.getValue(results, DeviceInfoBrowserVendor)
	browserName := x.getValue(results, DeviceInfoBrowserName)
	browserVersion := x.getValue(results, DeviceInfoBrowserVersion)
	screenPixelsWidth, _ := strconv.ParseInt(x.getValue(results, DeviceInfoScreenPixelsWidth), 10, 64)
	screenPixelsHeight, _ := strconv.ParseInt(x.getValue(results, DeviceInfoScreenPixelsHeight), 10, 64)
	pixelRatio, _ := strconv.ParseFloat(x.getValue(results, DeviceInfoPixelRatio), 10)
	javascript, _ := strconv.ParseBool(x.getValue(results, DeviceInfoJavascript))
	geoLocation, _ := strconv.ParseBool(x.getValue(results, DeviceInfoGeoLocation))
	deviceId, err := results.DeviceId()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get device id.")
	}
	hardwareModel := x.getValue(results, DeviceInfoHardwareModel)
	hardwareFamily := x.getValue(results, DeviceInfoHardwareFamily)
	hardwareModelVariants := x.getValue(results, DeviceInfoHardwareModelVariants)
	screenInchedHeight, _ := strconv.ParseFloat(x.getValue(results, DeviceInfoScreenInchesHeight), 10)

	p := &DeviceInfo{
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
func (x DeviceInfoExtractor) getValue(
	results Results,
	propertyName DeviceInfoProperty) string {
	// Get the values in string
	value, err := results.ValuesString(
		string(propertyName),
		",",
	)
	if err != nil {
		log.Printf("ERROR: Failed to get results values string.")
	}

	hasValues, err := results.HasValues(string(propertyName))
	if err != nil {
		glog.Errorf("Failed to check if a matched value exists for property %s.\n", propertyName)
	}

	if !hasValues {
		glog.Warningf("Property %s does not have a matched value.\n", propertyName)
		return "Unknown"
	}

	if err != nil {
		return ""
	}

	return value
}
