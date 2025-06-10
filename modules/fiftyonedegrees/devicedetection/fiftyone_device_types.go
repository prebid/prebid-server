package devicedetection

import (
	"github.com/prebid/openrtb/v20/adcom1"
)

type deviceTypeMap = map[deviceType]adcom1.DeviceType

var mobileOrTabletDeviceTypes = []deviceType{
	deviceTypeMobile,
}

var personalComputerDeviceTypes = []deviceType{
	deviceTypeDesktop,
	deviceTypeEReader,
	deviceTypeVehicleDisplay,
}

var tvDeviceTypes = []deviceType{
	deviceTypeTv,
}

var phoneDeviceTypes = []deviceType{
	deviceTypePhone,
	deviceTypeSmartPhone,
}

var tabletDeviceTypes = []deviceType{
	deviceTypeTablet,
}

var connectedDeviceTypes = []deviceType{
	deviceTypeIoT,
	deviceTypeRouter,
	deviceTypeSmallScreen,
	deviceTypeSmartSpeaker,
	deviceTypeSmartWatch,
}

var setTopBoxDeviceTypes = []deviceType{
	deviceTypeMediaHub,
	deviceTypeConsole,
}

var oohDeviceTypes = []deviceType{
	deviceTypeKiosk,
}

func applyCollection(items []deviceType, value adcom1.DeviceType, mappedCollection deviceTypeMap) {
	for _, item := range items {
		mappedCollection[item] = value
	}
}

var deviceTypeMapCollection = deviceTypeMap{}

func init() {
	applyCollection(mobileOrTabletDeviceTypes, adcom1.DeviceMobile, deviceTypeMapCollection)
	applyCollection(personalComputerDeviceTypes, adcom1.DevicePC, deviceTypeMapCollection)
	applyCollection(tvDeviceTypes, adcom1.DeviceTV, deviceTypeMapCollection)
	applyCollection(phoneDeviceTypes, adcom1.DevicePhone, deviceTypeMapCollection)
	applyCollection(tabletDeviceTypes, adcom1.DeviceTablet, deviceTypeMapCollection)
	applyCollection(connectedDeviceTypes, adcom1.DeviceConnected, deviceTypeMapCollection)
	applyCollection(setTopBoxDeviceTypes, adcom1.DeviceSetTopBox, deviceTypeMapCollection)
	applyCollection(oohDeviceTypes, adcom1.DeviceOOH, deviceTypeMapCollection)
}

// fiftyOneDtToRTB converts a 51Degrees device type to an OpenRTB device type.
// If the device type is not recognized, it defaults to PC.
func fiftyOneDtToRTB(val string) adcom1.DeviceType {
	id, ok := deviceTypeMapCollection[deviceType(val)]
	if ok {
		return id
	}

	return adcom1.DevicePC
}
