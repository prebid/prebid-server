package wurfl_devicedetection

import (
	"github.com/golang/glog"
	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/tidwall/sjson"
)

const (
	advertisedDeviceOSCapKey        = "advertised_device_os"
	advertisedDeviceOSVersionCapKey = "advertised_device_os_version"
	ajaxSupportJavascriptCapKey     = "ajax_support_javascript"
	brandNameCapKey                 = "brand_name"
	completeDeviceNameCapKey        = "complete_device_name"
	densityClassCapKey              = "density_class"
	formFactorCapKey                = "form_factor"
	isConnectedTVCapKey             = "is_connected_tv"
	isFullDesktopCapKey             = "is_full_desktop"
	isMobileCapKey                  = "is_mobile"
	isOTTCapKey                     = "is_ott"
	isPhoneCapKey                   = "is_phone"
	isTabletCapKey                  = "is_tablet"
	modelNameCapKey                 = "model_name"
	physicalFormFactorCapKey        = "physical_form_factor"
	pixelDensityCapKey              = "pixel_density"
	resolutionHeightCapKey          = "resolution_height"
	resolutionWidthCapKey           = "resolution_width"
)

const (
	ortb2WurflExtKey = "wurfl"
)

const (
	outOfHomeDevice = "out_of_home_device"
	trueString      = "true"
)

var vcaps = []string{
	advertisedDeviceOSCapKey,
	advertisedDeviceOSVersionCapKey,
	completeDeviceNameCapKey,
	isFullDesktopCapKey,
	isMobileCapKey,
	isPhoneCapKey,
	formFactorCapKey,
	pixelDensityCapKey,
}

// wurflDeviceDetection wraps the methods for the WURFL device detection
type wurflDeviceDetection interface {
	DeviceDetection(headers map[string]string) (wurflData, error)
}

// wurflEnricher represents the WURFL Enricher for Prebid
type wurflEnricher struct {
	// WurflData holds the WURFL data
	WurflData wurflData
\t// extCaps if true will enrich the device.ext field with all WURFL caps
	// Default to enrich only with the wurfl_id
	ExtCaps bool
}

// EnrichDevice enriches OpenRTB 2.x device with WURFL data
func (we wurflEnricher) EnrichDevice(device *openrtb2.Device) {
	wd := we.WurflData
	if device.Make == "" {
		if v, err := wd.String(brandNameCapKey); err == nil {
			device.Make = v
		}
	}
	if device.Model == "" {
		if v, err := wd.String(modelNameCapKey); err == nil {
			device.Model = v
		}
	}
	if device.DeviceType == 0 {
		device.DeviceType = we.makeDeviceType()
	}
	if device.OS == "" {
		if v, err := wd.String(advertisedDeviceOSCapKey); err == nil {
			device.OS = v
		}
	}
	if device.OSV == "" {
		if v, err := wd.String(advertisedDeviceOSVersionCapKey); err == nil {
			device.OSV = v
		}
	}
		if v, err := wd.String(advertisedDeviceOSCapKey); err == nil {
			device.OSV = v
		}
	}
	if device.HWV == "" {
		if v, err := wd.String(modelNameCapKey); err == nil {
			device.HWV = v
		}
	}
	if device.H == 0 {
		if v, err := wd.Int64(resolutionHeightCapKey); err == nil {
			device.H = v
		}
	}
	if device.W == 0 {
		if v, err := wd.Int64(resolutionWidthCapKey); err == nil {
			device.W = v
		}
	}
	if device.PPI == 0 {
		if v, err := wd.Int64(pixelDensityCapKey); err == nil {
			device.PPI = v
		}
	}
	if device.PxRatio == 0 {
		if v, err := wd.Float64(densityClassCapKey); err == nil {
			device.PxRatio = v
		}
	}
	if device.JS == nil {
		if v, err := wd.Bool(ajaxSupportJavascriptCapKey); err == nil {
			var js int8
			if v {
				js = 1
			}
			device.JS = &js
		}
	}

	wurflExtData, err := we.wurflExtData()
	if err != nil {
		return
	}
	// merges the WURFL data in device.ext under the wurfl "namespace"
	ext, err := sjson.SetRawBytes(device.Ext, ortb2WurflExtKey, wurflExtData)
	if err != nil {
		return
	}
	device.Ext = ext
}

// wurflExtData returns the WURFL data in JSON format for the device.ext field
func (we wurflEnricher) wurflExtData() ([]byte, error) {
	if we.ExtCaps {
		// return all WURFL data
		return we.WurflData.MarshalJSON()
	}
	// return only the WURFL ID
	return we.WurflData.WurflIDToJSON()
}

// makeDeviceType returns an OpenRTB2 DeviceType from WURFL data
// see https://www.scientiamobile.com/how-to-populate-iab-openrtb-device-object/
func (we wurflEnricher) makeDeviceType() adcom1.DeviceType {
	wd := we.WurflData
	unknownDeviceType := adcom1.DeviceType(0)

	isMobile, err := wd.Bool(isMobileCapKey)
	if err != nil {
		glog.Warning(err)
	}

	isPhone, err := wd.Bool(isPhoneCapKey)
	if err != nil {
		glog.Warning(err)
	}

	isTablet, err := wd.Bool(isTabletCapKey)
	if err != nil {
		glog.Warning(err)
	}

	if isMobile {
		if isPhone || isTablet {
			return adcom1.DeviceMobile
		}
		return adcom1.DeviceConnected
	}

	isFullDesktop, err := wd.Bool(isFullDesktopCapKey)
	if err != nil {
		glog.Warning(err)
	}
	if isFullDesktop {
		return adcom1.DevicePC
	}

	isConnectedTV, err := wd.Bool(isConnectedTVCapKey)
	if err != nil {
		glog.Warning(err)
	}
	if isConnectedTV {
		return adcom1.DeviceTV
	}

	if isPhone {
		return adcom1.DevicePhone
	}

	if isTablet {
		return adcom1.DeviceTablet
	}

	isOTT, err := wd.Bool(isOTTCapKey)
	if err != nil {
		glog.Warning(err)
	}
	if isOTT {
		return adcom1.DeviceSetTopBox
	}

	isOOH, err := wd.String(physicalFormFactorCapKey)
	if err != nil {
		glog.Warning(err)
	}
	if isOOH == outOfHomeDevice {
		return adcom1.DeviceOOH
	}
	return unknownDeviceType
}
