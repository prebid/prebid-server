package openrtb_ext

type ExtImpHuaweiAds struct {
	SlotId             string `json:"slotid"`
	Adtype             string `json:"adtype"`
	PublisherId        string `json:"publisherid"`
	SignKey            string `json:"signkey"`
	KeyId              string `json:"keyid"`
	ClientTime         string `json:"clientTime"`
	IsAddAuthorization string `json:"isAddAuthorization,omitempty"`
}

type ExtUserDataHuaweiAds struct {
	Data ExtUserDataDeviceIdHuaweiAds `json:"data,omitempty"`
}

type ExtUserDataDeviceIdHuaweiAds struct {
	Imei []string `json:"imei,omitempty"`
	Oaid []string `json:"oaid,omitempty"`
	Gaid []string `json:"gaid,omitempty"`
}
