package openrtb_ext

type ExtImpHuaweiAds struct {
	SlotId              string `json:"slotid"`
	Adtype              string `json:"adtype"`
	PublisherId         string `json:"publisherid"`
	SignKey             string `json:"signkey"`
	KeyId               string `json:"keyid"`
	IsTestAuthorization string `json:"isTestAuthorization,omitempty"`
}

type ExtUserDataHuaweiAds struct {
	Data ExtUserDataDeviceIdHuaweiAds `json:"data,omitempty"`
}

type ExtUserDataDeviceIdHuaweiAds struct {
	Imei       []string `json:"imei,omitempty"`
	Oaid       []string `json:"oaid,omitempty"`
	Gaid       []string `json:"gaid,omitempty"`
	ClientTime []string `json:"clientTime,omitempty"`
}
