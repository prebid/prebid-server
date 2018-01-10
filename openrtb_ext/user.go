package openrtb_ext

// ExtUser defines the contract for bidrequest.user.ext
type ExtUser struct {
	DigiTrust *ExtUserDigiTrust `json:"digitrust,omitempty"`
}

// ExtUserDigiTrust defines the contract for bidrequest.user.ext.digitrust
// More info on DigiTrust can be found here: https://github.com/digi-trust/dt-cdn/wiki/Integration-Guide
type ExtUserDigiTrust struct {
	ID   string `json:"id"`   // Unique device identifier
	KeyV int    `json:"keyv"` // Key version used to encrypt ID
	Pref int    `json:"pref"` // User optout preference
}
