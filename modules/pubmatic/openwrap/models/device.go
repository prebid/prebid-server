package models

import (
	"encoding/json"
	"strings"
)

const (
	//Device.DeviceType values as per OpenRTB-API-Specification-Version-2-5
	DeviceTypeMobile           = 1
	DeviceTypePersonalComputer = 2
	DeviceTypeConnectedTv      = 3
	DeviceTypePhone            = 4
	DeviceTypeTablet           = 5
	DeviceTypeConnectedDevice  = 6
	DeviceTypeSetTopBox        = 7
)

// DevicePlatform defines enums as per int values from KomliAdServer.platform table
type DevicePlatform int

const (
	DevicePlatformUnknown          DevicePlatform = -1
	DevicePlatformDesktop          DevicePlatform = 1 //Desktop Web
	DevicePlatformMobileWeb        DevicePlatform = 2 //Mobile Web
	DevicePlatformNotDefined       DevicePlatform = 3
	DevicePlatformMobileAppIos     DevicePlatform = 4 //In-App iOS
	DevicePlatformMobileAppAndroid DevicePlatform = 5 //In-App Android
	DevicePlatformMobileAppWindows DevicePlatform = 6
	DevicePlatformConnectedTv      DevicePlatform = 7 //Connected TV
)

// DeviceIFAType defines respective logger int id for device type
type DeviceIFAType = int

// DeviceIFATypeID
var DeviceIFATypeID = map[string]DeviceIFAType{
	DeviceIFATypeDPID:      1,
	DeviceIFATypeRIDA:      2,
	DeviceIFATypeAAID:      3,
	DeviceIFATypeIDFA:      4,
	DeviceIFATypeAFAI:      5,
	DeviceIFATypeMSAI:      6,
	DeviceIFATypePPID:      7,
	DeviceIFATypeSSPID:     8,
	DeviceIFATypeSESSIONID: 9,
}

// Device Ifa type constants
const (
	DeviceIFATypeDPID        = "dpid"
	DeviceIFATypeRIDA        = "rida"
	DeviceIFATypeAAID        = "aaid"
	DeviceIFATypeIDFA        = "idfa"
	DeviceIFATypeAFAI        = "afai"
	DeviceIFATypeMSAI        = "msai"
	DeviceIFATypePPID        = "ppid"
	DeviceIFATypeSSPID       = "sspid"
	DeviceIFATypeSESSIONID   = "sessionid"
	DeviceIfaTypeIdSessionId = 9
)

// device.ext related keys
const (
	ExtDeviceIFAType   = "ifa_type"
	ExtDeviceSessionID = "session_id"
	ExtDeviceAtts      = "atts"
)

// ExtDevice will store device.ext parameters
type ExtDevice struct {
	data map[string]any
}

func NewExtDevice() *ExtDevice {
	return &ExtDevice{
		data: make(map[string]any),
	}
}

func (e *ExtDevice) UnmarshalJSON(data []byte) error {
	var intermediate map[string]interface{}
	if err := json.Unmarshal(data, &intermediate); err != nil {
		return err
	}
	e.data = intermediate
	return nil
}

func (e *ExtDevice) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.data)
}

func (e *ExtDevice) getStringValue(key string) (value string, found bool) {
	if e.data == nil {
		return value, found
	}
	val, found := e.data[key]
	if !found {
		return "", found
	}
	var ok bool
	value, ok = val.(string)
	if !ok {
		delete(e.data, key)
	}
	return strings.TrimSpace(value), found
}

func (e *ExtDevice) GetIFAType() (value string, found bool) {
	return e.getStringValue(ExtDeviceIFAType)
}

func (e *ExtDevice) GetSessionID() (value string, found bool) {
	return e.getStringValue(ExtDeviceSessionID)
}

func (e *ExtDevice) getFloatValue(key string) (value float64, found bool) {
	if e.data == nil {
		return value, found
	}
	val, found := e.data[key]
	if !found {
		return 0, found
	}
	value, found = val.(float64)
	return value, found
}

func (e *ExtDevice) GetAtts() (value *float64, found bool) {
	val, ok := e.getFloatValue(ExtDeviceAtts)
	if !ok {
		return nil, ok
	}
	return &val, ok
}

func (e *ExtDevice) setStringValue(key, value string) {
	if e.data == nil {
		e.data = make(map[string]any)
	}
	e.data[key] = value
}

func (e *ExtDevice) SetIFAType(ifaType string) {
	e.setStringValue(ExtDeviceIFAType, ifaType)
}

func (e *ExtDevice) SetSessionID(sessionID string) {
	e.setStringValue(ExtDeviceSessionID, sessionID)
}

func (e *ExtDevice) DeleteIFAType() {
	delete(e.data, ExtDeviceIFAType)
}

func (e *ExtDevice) DeleteSessionID() {
	delete(e.data, ExtDeviceSessionID)
}

// DeleteEds removes PubMatic-only enrichment data from the cached device.ext map.
func (e *ExtDevice) DeleteEds() {
	delete(e.data, "eds")
}

// IsEmpty reports whether device.ext has no keys left to marshal onto the request.
func (e *ExtDevice) IsEmpty() bool {
	return e == nil || len(e.data) == 0
}
