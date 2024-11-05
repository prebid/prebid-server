package devicedetection

// Prefixes in literal format
const queryPrefix = "query."
const headerPrefix = "header."
const ddUnknown = "Unknown"

// Evidence where all fields are in string format
type stringEvidence struct {
	Prefix string
	Key    string
	Value  string
}

func getEvidenceByKey(e []stringEvidence, key string) (stringEvidence, bool) {
	for _, evidence := range e {
		if evidence.Key == key {
			return evidence, true
		}
	}
	return stringEvidence{}, false
}

type deviceType string

const (
	deviceTypePhone          = "Phone"
	deviceTypeConsole        = "Console"
	deviceTypeDesktop        = "Desktop"
	deviceTypeEReader        = "EReader"
	deviceTypeIoT            = "IoT"
	deviceTypeKiosk          = "Kiosk"
	deviceTypeMediaHub       = "MediaHub"
	deviceTypeMobile         = "Mobile"
	deviceTypeRouter         = "Router"
	deviceTypeSmallScreen    = "SmallScreen"
	deviceTypeSmartPhone     = "SmartPhone"
	deviceTypeSmartSpeaker   = "SmartSpeaker"
	deviceTypeSmartWatch     = "SmartWatch"
	deviceTypeTablet         = "Tablet"
	deviceTypeTv             = "Tv"
	deviceTypeVehicleDisplay = "Vehicle Display"
)

type deviceInfo struct {
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
