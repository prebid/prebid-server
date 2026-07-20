package openwrap

import (
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

func populateDeviceContext(dvc *models.DeviceCtx, device *openrtb2.Device) {
	if device == nil {
		return
	}
	//this is needed in determine ifa_type parameter
	dvc.DeviceIFA = strings.TrimSpace(device.IFA)
	dvc.Model = device.Model
	dvc.ID = getDeviceID(dvc, device)

	if device.Ext == nil {
		return
	}

	//unmarshal device ext
	var deviceExt models.ExtDevice
	if err := deviceExt.UnmarshalJSON(device.Ext); err != nil {
		return
	}
	dvc.Ext = &deviceExt

	if dvc.ID == "" {
		dvc.ID, _ = dvc.Ext.GetSessionID()
	}
	//update device IFA Details
	updateDeviceIFADetails(dvc)
}

// getDeviceID retrieves deviceID for logging purpose
func getDeviceID(dvc *models.DeviceCtx, device *openrtb2.Device) string {
	if dvc.DeviceIFA != "" {
		return dvc.DeviceIFA
	}
	if device.DIDSHA1 != "" {
		return device.DIDSHA1
	}
	if device.DIDMD5 != "" {
		return device.DIDMD5
	}
	if device.DPIDSHA1 != "" {
		return device.DPIDSHA1
	}
	if device.DPIDMD5 != "" {
		return device.DPIDMD5
	}
	if device.MACSHA1 != "" {
		return device.MACSHA1
	}
	if device.MACMD5 != "" {
		return device.MACMD5
	}
	return ""
}

func updateDeviceIFADetails(dvc *models.DeviceCtx) {
	if dvc == nil || dvc.Ext == nil {
		return
	}

	deviceExt := dvc.Ext
	extIFAType, ifaTypeFound := deviceExt.GetIFAType()
	extSessionID, _ := deviceExt.GetSessionID()

	if ifaTypeFound {
		if dvc.DeviceIFA != "" {
			if ifaTypeID, ok := models.DeviceIFATypeID[strings.ToLower(extIFAType)]; !ok {
				deviceExt.DeleteIFAType()
			} else {
				dvc.IFATypeID = &ifaTypeID
				deviceExt.SetIFAType(extIFAType)
			}
		} else if extSessionID != "" {
			dvc.DeviceIFA = extSessionID
			dvc.IFATypeID = ptrutil.ToPtr(models.DeviceIfaTypeIdSessionId)
			deviceExt.SetIFAType(models.DeviceIFATypeSESSIONID)
		} else {
			deviceExt.DeleteIFAType()
		}
	} else if extSessionID != "" && dvc.DeviceIFA == "" {
		dvc.DeviceIFA = extSessionID
		dvc.IFATypeID = ptrutil.ToPtr(models.DeviceIfaTypeIdSessionId)
		deviceExt.SetIFAType(models.DeviceIFATypeSESSIONID)
	}
}

func amendDeviceObject(device *openrtb2.Device, dvc *models.DeviceCtx) {
	if device == nil || dvc == nil {
		return
	}

	//update device IFA
	if len(dvc.DeviceIFA) > 0 {
		device.IFA = dvc.DeviceIFA
	}

	//update device extension
	if dvc.Ext == nil {
		return
	}
	if dvc.Ext.IsEmpty() {
		device.Ext = nil
		return
	}
	device.Ext, _ = dvc.Ext.MarshalJSON()
}
