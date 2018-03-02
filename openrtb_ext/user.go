package openrtb_ext

// ExtUser defines the contract for bidrequest.user.ext
type ExtUser struct {

	// Consent is a GDPR consent string. See "Advised Extensions" of
	// https://iabtechlab.com/wp-content/uploads/2018/02/OpenRTB_Advisory_GDPR_2018-02.pdf
	Consent string `json:"consent,omitempty"`

	Prebid *ExtUserPrebid `json:"prebid,omitempty"`

	// DigiTrust breaks the typical Prebid Server convention of namespacing "global" options inside "ext.prebid.*"
	// to match the recommendation from the broader digitrust community.
	// For more info, see: https://github.com/digi-trust/dt-cdn/wiki/OpenRTB-extension#openrtb-2x
	DigiTrust *ExtUserDigiTrust `json:"digitrust,omitempty"`
}

// ExtUserPrebid defines the contract for bidrequest.user.ext.prebid
type ExtUserPrebid struct {
	BuyerUIDs map[string]string `json:"buyeruids,omitempty"`
}

// ExtUserDigiTrust defines the contract for bidrequest.user.ext.digitrust
// More info on DigiTrust can be found here: https://github.com/digi-trust/dt-cdn/wiki/Integration-Guide
type ExtUserDigiTrust struct {
	ID   string `json:"id"`   // Unique device identifier
	KeyV int    `json:"keyv"` // Key version used to encrypt ID
	Pref int    `json:"pref"` // User optout preference
}
