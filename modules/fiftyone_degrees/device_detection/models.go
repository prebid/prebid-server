package device_detection

// Prefixes in literal format
const QueryPrefix = "query."
const HeaderPrefix = "header."
const DdUnknown = "Unknown"

// Evidence where all fields are in string format
type StringEvidence struct {
	Prefix string
	Key    string
	Value  string
}

func GetEvidenceByKey(e []StringEvidence, key string) (StringEvidence, bool) {
	for _, evidence := range e {
		if evidence.Key == key {
			return evidence, true
		}
	}
	return StringEvidence{}, false
}

type DeviceType string

const (
	DeviceTypePhone          = "Phone"
	DeviceTypeConsole        = "Console"
	DeviceTypeDesktop        = "Desktop"
	DeviceTypeEReader        = "EReader"
	DeviceTypeIoT            = "IoT"
	DeviceTypeKiosk          = "Kiosk"
	DeviceTypeMediaHub       = "MediaHub"
	DeviceTypeMobile         = "Mobile"
	DeviceTypeRouter         = "Router"
	DeviceTypeSmallScreen    = "SmallScreen"
	DeviceTypeSmartPhone     = "SmartPhone"
	DeviceTypeSmartSpeaker   = "SmartSpeaker"
	DeviceTypeSmartWatch     = "SmartWatch"
	DeviceTypeTablet         = "Tablet"
	DeviceTypeTv             = "Tv"
	DeviceTypeVehicleDisplay = "Vehicle Display"
)

type DeviceInfo struct {
	HardwareVendor        string
	HardwareName          string
	DeviceType            string
	PlatformVendor        string
	PlatformName          string
	PlatformVersion       string
	BrowserVendor         string
	BrowserName           string
	BrowserVersion        string
	ScreenPixelsWidth     int64
	ScreenPixelsHeight    int64
	PixelRatio            float64
	Javascript            bool
	GeoLocation           bool
	HardwareFamily        string
	HardwareModel         string
	HardwareModelVariants string
	UserAgent             string
	DeviceId              string
	ScreenInchesHeight    float64
}
