package openrtb_ext

// ExtUser defines the contract for bidrequest.user.ext
type ExtUser struct {
	DigiTrust *ExtUserDigiTrust `json:"digitrust,omitempty"`
}

// ExtUserDigiTrust defines the contract for bidrequest.user.ext.digitrust
type ExtUserDigiTrust struct {
	ID   string `json:"id"`
	KeyV int    `json:"keyv"`
	Pref int    `json:"pref"`
}
