package device_detection

import (
	"github.com/prebid/prebid-server/v2/hooks/hookstage"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"math"
)

// DeviceInformationMapper is a struct that contains the device information
// and is used to hydrate the device information in the request payload
type DeviceInformationMapper struct {
	deviceInfo *DeviceInfo
}

func NewDeviceInformationMapper(deviceInfo *DeviceInfo) *DeviceInformationMapper {
	return &DeviceInformationMapper{
		deviceInfo,
	}
}

func (s DeviceInformationMapper) HydrateDeviceType(payload hookstage.RawAuctionRequestPayload) ([]byte, error) {
	deviceTypeResult := gjson.GetBytes(payload, "device.devicetype")
	if !deviceTypeResult.Exists() {
		newPayload, err := sjson.SetBytes(payload, "device.devicetype", fiftyOneDtToRTB(s.deviceInfo.DeviceType))
		if err != nil {
			return payload, err
		}

		return newPayload, nil
	}

	return payload, nil
}

func (s DeviceInformationMapper) HydrateUserAgent(payload hookstage.RawAuctionRequestPayload) ([]byte, error) {
	uaResult := gjson.GetBytes(payload, "device.ua")
	if !uaResult.Exists() && s.deviceInfo.UserAgent != DdUnknown {
		newPayload, err := sjson.SetBytes(payload, "device.ua", s.deviceInfo.UserAgent)
		if err != nil {
			return payload, err
		}

		return newPayload, nil
	}

	return payload, nil
}

func (s DeviceInformationMapper) HydrateMake(device hookstage.RawAuctionRequestPayload) ([]byte, error) {
	makeResult := gjson.GetBytes(device, "device.make")
	if !makeResult.Exists() && s.deviceInfo.HardwareVendor != DdUnknown {
		newPayload, err := sjson.SetBytes(device, "device.make", s.deviceInfo.HardwareVendor)
		if err != nil {
			return device, err
		}
		return newPayload, nil
	}

	return device, nil
}

func (s DeviceInformationMapper) HydrateModel(payload hookstage.RawAuctionRequestPayload, extMap map[string]any) ([]byte, error) {
	newVal := s.deviceInfo.HardwareModel
	if newVal == DdUnknown {
		newVal = s.deviceInfo.HardwareName
	}

	if newVal != DdUnknown {
		newPayload, err := sjson.SetBytes(payload, "device.model", newVal)
		if err != nil {
			return newPayload, err
		}
		return newPayload, nil
	}

	return payload, nil
}

func (s DeviceInformationMapper) HydrateOS(payload hookstage.RawAuctionRequestPayload) ([]byte, error) {
	osResult := gjson.GetBytes(payload, "device.os")
	if !osResult.Exists() && s.deviceInfo.PlatformName != DdUnknown {
		newPayload, err := sjson.SetBytes(payload, "device.os", s.deviceInfo.PlatformName)
		if err != nil {
			return payload, err
		}

		return newPayload, nil
	}

	return payload, nil
}

func (s DeviceInformationMapper) HydrateOSVersion(payload hookstage.RawAuctionRequestPayload) ([]byte, error) {
	osvResult := gjson.GetBytes(payload, "device.osv")
	if !osvResult.Exists() && s.deviceInfo.PlatformVersion != DdUnknown {
		newPayload, err := sjson.SetBytes(payload, "device.osv", s.deviceInfo.PlatformVersion)
		if err != nil {
			return payload, err
		}

		return newPayload, nil
	}

	return payload, nil
}

func (s DeviceInformationMapper) HydrateScreenHeight(payload hookstage.RawAuctionRequestPayload) ([]byte, error) {
	heightResult := gjson.GetBytes(payload, "device.h")
	if !heightResult.Exists() {
		newPayload, err := sjson.SetBytes(payload, "device.h", s.deviceInfo.ScreenPixelsHeight)
		if err != nil {
			return payload, err
		}
		return newPayload, nil
	}

	return payload, nil
}

func (s DeviceInformationMapper) HydrateScreenWidth(payload hookstage.RawAuctionRequestPayload) ([]byte, error) {
	widthResult := gjson.GetBytes(payload, "device.w")
	if !widthResult.Exists() {
		newPayload, err := sjson.SetBytes(payload, "device.w", s.deviceInfo.ScreenPixelsWidth)
		if err != nil {
			return payload, err
		}

		return newPayload, nil
	}

	return payload, nil
}

func (s DeviceInformationMapper) HydratePixelRatio(payload hookstage.RawAuctionRequestPayload) ([]byte, error) {
	pxRationResult := gjson.GetBytes(payload, "device.pxratio")
	if !pxRationResult.Exists() {
		newPayload, err := sjson.SetBytes(payload, "device.pxratio", s.deviceInfo.PixelRatio)
		if err != nil {
			return payload, err
		}

		return newPayload, nil
	}

	return payload, nil
}

func (s DeviceInformationMapper) HydrateJavascript(payload hookstage.RawAuctionRequestPayload) ([]byte, error) {
	jsResult := gjson.GetBytes(payload, "device.js")
	if !jsResult.Exists() {
		val := 0
		if s.deviceInfo.Javascript {
			val = 1
		}
		newPayload, err := sjson.SetBytes(payload, "device.js", val)
		if err != nil {
			return payload, err
		}

		return newPayload, nil
	}

	return payload, nil
}

func (s DeviceInformationMapper) HydrateGeoLocation(payload hookstage.RawAuctionRequestPayload) ([]byte, error) {
	geoFetchResult := gjson.GetBytes(payload, "device.geoFetch")
	if !geoFetchResult.Exists() {
		val := 0
		if s.deviceInfo.GeoLocation {
			val = 1
		}

		newPayload, err := sjson.SetBytes(payload, "device.geoFetch", val)
		if err != nil {
			return payload, err
		}

		return newPayload, nil
	}

	return payload, nil
}

func (s DeviceInformationMapper) HydratePPI(payload hookstage.RawAuctionRequestPayload) ([]byte, error) {
	if s.deviceInfo.ScreenPixelsHeight > 0 && s.deviceInfo.ScreenInchesHeight > 0 {
		ppi := float64(s.deviceInfo.ScreenPixelsHeight) / s.deviceInfo.ScreenInchesHeight
		ppi = math.Round(ppi)
		newPayload, err := sjson.SetBytes(payload, "device.ppi", int(ppi))
		if err != nil {
			return payload, err
		}

		return newPayload, nil
	}

	return payload, nil
}
