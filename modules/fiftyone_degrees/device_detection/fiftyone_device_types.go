package device_detection

import (
	"github.com/prebid/openrtb/v20/adcom1"
)

type deviceTypeMap = map[DeviceType]adcom1.DeviceType

var mobileOrTabletDeviceTypes = []DeviceType{
	DeviceTypeMobile,
	DeviceTypeSmartPhone,
}

var personalComputerDeviceTypes = []DeviceType{
	DeviceTypeDesktop,
	DeviceTypeEReader,
	DeviceTypeVehicleDisplay,
}

var tvDeviceTypes = []DeviceType{
	DeviceTypeTv,
}

var phoneDeviceTypes = []DeviceType{
	DeviceTypePhone,
}

var tabletDeviceTypes = []DeviceType{
	DeviceTypeTablet,
}

var connectedDeviceTypes = []DeviceType{
	DeviceTypeConsole,
	DeviceTypeIoT,
	DeviceTypeRouter,
	DeviceTypeSmallScreen,
	DeviceTypeSmartSpeaker,
	DeviceTypeSmartWatch,
}

var setTopBoxDeviceTypes = []DeviceType{
	DeviceTypeMediaHub,
	DeviceTypeConsole,
}

var oohDeviceTypes = []DeviceType{
	DeviceTypeKiosk,
}

func applyCollection(items []DeviceType, value adcom1.DeviceType, mappedCollection deviceTypeMap) {
	for _, item := range items {
		mappedCollection[item] = value
	}
}

var deviceTypeMapCollection = deviceTypeMap{}

// fiftyOneDtToRTB converts a 51Degrees device type to an OpenRTB device type.
// If the device type is not recognized, it defaults to PC.
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

func fiftyOneDtToRTB(val string) adcom1.DeviceType {
	id, ok := deviceTypeMapCollection[DeviceType(val)]
	if ok {
		return id
	}

	return adcom1.DevicePC
}
