package device_detection

import (
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v2/hooks/hookexecution"
	"github.com/prebid/prebid-server/v2/hooks/hookstage"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type deviceMapper interface {
	HydrateDeviceType(payload hookstage.RawAuctionRequestPayload) ([]byte, error)
	HydrateUserAgent(payload hookstage.RawAuctionRequestPayload) ([]byte, error)
	HydrateMake(device hookstage.RawAuctionRequestPayload) ([]byte, error)
	HydrateModel(payload hookstage.RawAuctionRequestPayload, extMap map[string]any) ([]byte, error)
	HydrateOS(payload hookstage.RawAuctionRequestPayload) ([]byte, error)
	HydrateOSVersion(payload hookstage.RawAuctionRequestPayload) ([]byte, error)
	HydrateScreenHeight(payload hookstage.RawAuctionRequestPayload) ([]byte, error)
	HydrateScreenWidth(payload hookstage.RawAuctionRequestPayload) ([]byte, error)
	HydratePixelRatio(payload hookstage.RawAuctionRequestPayload) ([]byte, error)
	HydrateJavascript(payload hookstage.RawAuctionRequestPayload) ([]byte, error)
	HydrateGeoLocation(payload hookstage.RawAuctionRequestPayload) ([]byte, error)
	HydratePPI(payload hookstage.RawAuctionRequestPayload) ([]byte, error)
}

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
			deviceMapper := NewDeviceInformationMapper(deviceInfo)

			result, err := hydrateFields(deviceInfo, deviceMapper, rawPayload)
			if err != nil {
				glog.Errorf("error hydrating fields %s", err)
			}

			return result, nil
		}, hookstage.MutationUpdate,
	)

	return result, nil
}

// hydrateFields hydrates the fields in the raw auction request payload with the device information
func hydrateFields(fiftyOneDd *DeviceInfo, deviceMapper deviceMapper, payload hookstage.RawAuctionRequestPayload) (hookstage.RawAuctionRequestPayload, error) {
	extMap := map[string]any{}

	var (
		errs       []error
		err        error
		newPayload []byte = payload
	)

	newPayload, err = deviceMapper.HydrateDeviceType(payload)
	if err != nil {
		errs = append(errs, fmt.Errorf("error hydrating device type %s", err))
	}

	newPayload, err = deviceMapper.HydrateUserAgent(newPayload)
	if err != nil {
		errs = append(errs, fmt.Errorf("error hydrating user agent %s", err))
	}

	newPayload, err = deviceMapper.HydrateMake(newPayload)
	if err != nil {
		errs = append(errs, fmt.Errorf("error hydrating make %s", err))
	}

	newPayload, err = deviceMapper.HydrateModel(newPayload, extMap)
	if err != nil {
		errs = append(errs, fmt.Errorf("error hydrating model %s", err))
	}

	newPayload, err = deviceMapper.HydrateOS(newPayload)
	if err != nil {
		errs = append(errs, fmt.Errorf("error hydrating OS %s", err))
	}

	newPayload, err = deviceMapper.HydrateOSVersion(newPayload)
	if err != nil {
		errs = append(errs, fmt.Errorf("error hydrating OS version %s", err))
	}

	newPayload, err = deviceMapper.HydrateScreenHeight(newPayload)
	if err != nil {
		errs = append(errs, fmt.Errorf("error hydrating screen height %s", err))
	}

	newPayload, err = deviceMapper.HydrateScreenWidth(newPayload)
	if err != nil {
		errs = append(errs, fmt.Errorf("error hydrating screen width %s", err))
	}

	newPayload, err = deviceMapper.HydratePixelRatio(newPayload)
	if err != nil {
		errs = append(errs, fmt.Errorf("error hydrating pixel ratio %s", err))
	}

	newPayload, err = deviceMapper.HydrateJavascript(newPayload)
	if err != nil {
		errs = append(errs, fmt.Errorf("error hydrating javascript %s", err))
	}

	newPayload, err = deviceMapper.HydrateGeoLocation(newPayload)
	if err != nil {
		errs = append(errs, fmt.Errorf("error hydrating geo location %s", err))
	}

	newPayload, err = deviceMapper.HydratePPI(newPayload)
	if err != nil {
		errs = append(errs, fmt.Errorf("error hydrating ppi %s", err))
	}

	newPayload, err = signDeviceData(newPayload, fiftyOneDd, extMap)
	if err != nil {
		errs = append(errs, fmt.Errorf("error signing device data %s", err))
	}

	return newPayload, errors.Join(errs...)
}

// signDeviceData signs the device data with the device information in the ext map of the raw auction request payload
func signDeviceData(payload hookstage.RawAuctionRequestPayload, deviceInfo *DeviceInfo, extra map[string]any) ([]byte, error) {
	var (
		err        error
		newPayload []byte = []byte(payload)
	)
	extResult := gjson.GetBytes(payload, "device.ext")

	if !extResult.Exists() {
		newPayload, err = sjson.SetBytes(newPayload, "device.ext", map[string]any{})
		if err != nil {
			return payload, err
		}
	}

	newPayload, err = sjson.SetBytes(newPayload, "device.ext.fiftyonedegrees_deviceId", deviceInfo.DeviceId)
	if err != nil {
		return payload, err
	}

	for k, v := range extra {
		newPayload, err = sjson.SetBytes(newPayload, "device.ext."+k, v)
		if err != nil {
			return payload, err
		}
	}

	return newPayload, nil
}
