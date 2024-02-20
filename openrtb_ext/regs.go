package openrtb_ext

// ExtRegs defines the contract for bidrequest.regs.ext
type ExtRegs struct {
	// DSA is an object containing DSA transparency information, see https://github.com/InteractiveAdvertisingBureau/openrtb/blob/main/extensions/community_extensions/dsa_transparency.md
	DSA *ExtRegsDSA `json:"dsa,omitempty"`

	// GDPR should be "1" if the caller believes the user is subject to GDPR laws, "0" if not, and undefined
	// if it's unknown. For more info on this parameter, see: https://iabtechlab.com/wp-content/uploads/2018/02/OpenRTB_Advisory_GDPR_2018-02.pdf
	GDPR *int8 `json:"gdpr,omitempty"`

	// USPrivacy should be a four character string, see: https://iabtechlab.com/wp-content/uploads/2019/11/OpenRTB-Extension-U.S.-Privacy-IAB-Tech-Lab.pdf
	USPrivacy string `json:"us_privacy,omitempty"`
}

// ExtRegsDSA defines the contract for bidrequest.regs.ext.dsa
type ExtRegsDSA struct {
	// Required should be a between 0 and 3 inclusive, see https://github.com/InteractiveAdvertisingBureau/openrtb/blob/main/extensions/community_extensions/dsa_transparency.md
	Required int8 `json:"dsarequired,omitempty"`
	// PubRender should be between 0 and 2 inclusive, see https://github.com/InteractiveAdvertisingBureau/openrtb/blob/main/extensions/community_extensions/dsa_transparency.md
	PubRender int8 `json:"pubrender,omitempty"`
}
