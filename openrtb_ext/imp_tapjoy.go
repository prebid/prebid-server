package openrtb_ext

import (
	"encoding/json"
)

// ExtImpTapjoy defines the contract for bidrequest.imp[i].ext.tapjoy
type ExtImpTapjoy struct {
	App        TJApp        `json:"app"`
	Device     TJDevice     `json:"device"`
	Request    TJRequest    `json:"request"`
	Extensions TJExtensions `json:"extensions"`

	Region         string `json:"region"`
	Reward         int    `json:"reward"`
	SKADNSupported bool   `json:"skadn_supported"`
	MRAIDSupported bool   `json:"mraid_supported"`
}

type TJApp struct {
	ID string `json:"id"`
}

type TJDevice struct {
	OS            string `json:"os"`
	OSV           string `json:"osv"`
	HWV           string `json:"hwv"`
	Make          string `json:"make"`
	Model         string `json:"model"`
	DeviceType    int8   `json:"device_type"`
	CountryAlpha2 string `json:"country_alpha_2"`
}

type TJRequest struct {
	ID string `json:"id"`
}

type TJExtensions struct {
	AppExt       json.RawMessage `json:"app_ext"`
	ImpExt       json.RawMessage `json:"imp_ext"`
	RegsExt      json.RawMessage `json:"regs_ext"`
	UserExt      json.RawMessage `json:"user_ext"`
	VideoExt     json.RawMessage `json:"video_ext"`
	DeviceExt    json.RawMessage `json:"device_ext"`
	SourceExt    json.RawMessage `json:"source_ext"`
	RequestExt   json.RawMessage `json:"request_ext"`
	PublisherExt json.RawMessage `json:"publisher_ext"`
}
