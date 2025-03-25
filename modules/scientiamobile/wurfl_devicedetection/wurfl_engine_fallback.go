//go:build !wurfl

package wurfl_devicedetection

import "github.com/golang/glog"

// declare conformity with  wurflDeviceDetection interface
var _ wurflDeviceDetection = (*wurflEngine)(nil)

// newWurflEngine creates a new Enricher
func newWurflEngine(_ config) (wurflDeviceDetection, error) {
	glog.Error("WURFL module is enabled but not installed correctly. Running fallback implementation with fake data.")
	return &wurflEngine{}, nil
}

// wurflEngine is the ortb2 enricher powered by WURFL
type wurflEngine struct{}

// deviceDetection performs device detection using the WURFL engine.
func (e *wurflEngine) DeviceDetection(headers map[string]string) (wurflData, error) {
	// wd represents the data as per sample request
	wd := map[string]string{
		"physical_screen_width":        "71",
		"model_name":                   "Pixel 9 Pro XL",
		"pixel_density":                "481",
		"device_os_version":            "15.0",
		"pointing_method":              "touchscreen",
		"is_wireless_device":           "true",
		"is_smarttv":                   "false",
		"is_phone":                     "true",
		"device_os":                    "Android",
		"density_class":                "2.55",
		"resolution_width":             "1344",
		"resolution_height":            "2992",
		"ux_full_desktop":              "false",
		"is_full_desktop":              "false",
		"marketing_name":               "",
		"mobile_browser":               "Chrome Mobile",
		"preferred_markup":             "html_web_4_0",
		"is_connected_tv":              "false",
		"physical_screen_height":       "158",
		"advertised_device_os_version": "15",
		"form_factor":                  "Smartphone",
		"mobile_browser_version":       "",
		"ajax_support_javascript":      "true",
		"can_assign_phone_number":      "true",
		"is_ott":                       "false",
		"advertised_device_os":         "Android",
		"wurfl_id":                     "google_pixel_9_pro_xl_ver1_suban150",
		"complete_device_name":         "Google Pixel 9 Pro XL",
		"is_mobile":                    "true",
		"is_tablet":                    "false",
		"physical_form_factor":         "phone_phablet",
		"xhtml_support_level":          "4",
		"brand_name":                   "Google",
	}
	return wd, nil
}
