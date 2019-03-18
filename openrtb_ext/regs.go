package openrtb_ext

// ExtRegs defines the contract for bidrequest.regs.ext
type ExtRegs struct {

	// GDPR should be "1" if the caller believes the user is subject to GDPR laws, "0" if not, and undefined
	// if it's unknown. For more info on this parameter, see: https://iabtechlab.com/wp-content/uploads/2018/02/OpenRTB_Advisory_GDPR_2018-02.pdf
	GDPR *int8 `json:"gdpr,omitempty"`
}
