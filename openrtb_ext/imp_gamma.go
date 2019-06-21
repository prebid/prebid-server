package openrtb_ext

// ExtImpGamma defines the contract for bidrequest.imp[i].ext.gamma
type ExtImpGamma struct {
	PartnerID string `json:"id"`
	ZoneID    string `json:"zid"`
	WebID     string `json:"wid"`
	AppID     string `json:"app_id,omitempty"`
	AppName   string `json:"app_name,omitempty"`
	AppBundle string `json:"app_bundle,omitempty"`
}

// id=1397808490&
// wid=1513150517&
// zid=1513151405&
// app_id=123456789&
// app_name=Game_Danh_Bai&
// app_bundle=com.danhbai.app&
// device_ua=&
// device_ip=]&
// device_ifa=%%ADVERTISING_IDENTIFIER_PLAIN%%&
// device_country=&
// device_model=&
// device_os=&
// device_type=&
// cb=%%CACHEBUSTER%%&
// ret=js
