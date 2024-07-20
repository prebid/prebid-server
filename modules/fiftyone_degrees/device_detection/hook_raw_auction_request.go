package device_detection

import (
	"math"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v2/hooks/hookexecution"
	"github.com/prebid/prebid-server/v2/hooks/hookstage"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func handleAuctionRequestHook(ctx hookstage.ModuleInvocationContext, deviceDetector deviceDetector, evidenceExtractor evidenceExtractor) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
	var result hookstage.HookResult[hookstage.RawAuctionRequestPayload]

	// If the hook is not enabled, return the result without any changes
	if ctx.ModuleContext[DDEnabledCtxKey] == nil || ctx.ModuleContext[DDEnabledCtxKey] == false {
		return result, nil
	}

	if ctx.ModuleContext == nil || deviceDetector == nil {
		return result, hookexecution.NewFailure("error getting device detector")
	}

	result.ChangeSet.AddMutation(
		func(rawPayload hookstage.RawAuctionRequestPayload) (hookstage.RawAuctionRequestPayload, error) {
			evidence, ua, err := evidenceExtractor.Extract(ctx.ModuleContext)
			if err != nil {
				return rawPayload, hookexecution.NewFailure("error extracting evidence %s", err)
			}
			if evidence == nil {
				return rawPayload, hookexecution.NewFailure("error extracting evidence")
			}

			deviceInfo, err := deviceDetector.GetDeviceInfo(evidence, ua)
			if err != nil {
				return rawPayload, hookexecution.NewFailure("error getting device info %s", err)
			}

			result, err := hydrateFields(deviceInfo, rawPayload)
			if err != nil {
				glog.Errorf("error hydrating fields %s", err)
			}

			return result, nil
		}, hookstage.MutationUpdate,
	)

	return result, nil
}

// hydrateFields hydrates the fields in the raw auction request payload with the device information
func hydrateFields(fiftyOneDd *DeviceInfo, payload hookstage.RawAuctionRequestPayload) (hookstage.RawAuctionRequestPayload, error) {
	devicePayload := gjson.GetBytes(payload, "device")
	dPV := devicePayload.Value()
	if dPV == nil {
		return payload, nil
	}

	deviceObject := dPV.(map[string]any)
	deviceObject = setMissingFields(deviceObject, fiftyOneDd)
	deviceObject = signDeviceData(deviceObject, fiftyOneDd)

	return mergeDeviceIntoPayload(payload, deviceObject)
}

// setMissingFields sets fields such as ["devicetype", "ua", "make", "os", "osv", "h", "w", "pxratio", "js", "geoFetch", "model", "ppi"]
// if they are not already present in the device object
func setMissingFields(deviceObj map[string]any, fiftyOneDd *DeviceInfo) map[string]any {
	optionalFields := map[string]func() any{
		"devicetype": func() any {
			return fiftyOneDtToRTB(fiftyOneDd.DeviceType)
		},
		"ua": func() any {
			if fiftyOneDd.UserAgent != DdUnknown {
				return fiftyOneDd.UserAgent
			}
			return nil
		},
		"make": func() any {
			if fiftyOneDd.HardwareVendor != DdUnknown {
				return fiftyOneDd.HardwareVendor
			}
			return nil
		},
		"os": func() any {
			if fiftyOneDd.PlatformName != DdUnknown {
				return fiftyOneDd.PlatformName
			}
			return nil
		},
		"osv": func() any {
			if fiftyOneDd.PlatformVersion != DdUnknown {
				return fiftyOneDd.PlatformVersion
			}
			return nil
		},
		"h": func() any {
			return fiftyOneDd.ScreenPixelsHeight
		},
		"w": func() any {
			return fiftyOneDd.ScreenPixelsWidth
		},
		"pxratio": func() any {
			return fiftyOneDd.PixelRatio
		},
		"js": func() any {
			val := 0
			if fiftyOneDd.Javascript {
				val = 1
			}
			return val
		},
		"geoFetch": func() any {
			val := 0
			if fiftyOneDd.GeoLocation {
				val = 1
			}
			return val
		},
		"model": func() any {
			newVal := fiftyOneDd.HardwareModel
			if newVal == DdUnknown {
				newVal = fiftyOneDd.HardwareName
			}
			if newVal != DdUnknown {
				return newVal
			}
			return nil
		},
		"ppi": func() any {
			if fiftyOneDd.ScreenPixelsHeight > 0 && fiftyOneDd.ScreenInchesHeight > 0 {
				ppi := float64(fiftyOneDd.ScreenPixelsHeight) / fiftyOneDd.ScreenInchesHeight
				return int(math.Round(ppi))
			}
			return nil
		},
	}

	for field, valFunc := range optionalFields {
		_, ok := deviceObj[field]
		if !ok {
			val := valFunc()
			if val != nil {
				deviceObj[field] = val
			}
		}
	}

	return deviceObj
}

// signDeviceData signs the device data with the device information in the ext map of the device object
func signDeviceData(deviceObj map[string]any, fiftyOneDd *DeviceInfo) map[string]any {
	extObj, ok := deviceObj["ext"]
	var ext map[string]any
	if ok {
		ext = extObj.(map[string]any)
	} else {
		ext = make(map[string]any)
	}

	ext["fiftyonedegrees_deviceId"] = fiftyOneDd.DeviceId
	deviceObj["ext"] = ext

	return deviceObj
}

// mergeDeviceIntoPayload merges the modified device object back into the RawAuctionRequestPayload
func mergeDeviceIntoPayload(payload hookstage.RawAuctionRequestPayload, deviceObject map[string]any) (hookstage.RawAuctionRequestPayload, error) {
	newPayload, err := sjson.SetBytes(payload, "device", deviceObject)
	if err != nil {
		return payload, err
	}

	return newPayload, nil
}
