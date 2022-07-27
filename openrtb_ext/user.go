package openrtb_ext

import (
	"github.com/mxmCherry/openrtb/v16/openrtb2"
)

// ExtUser defines the contract for bidrequest.user.ext
type ExtUser struct {
	// Consent is a GDPR consent string. See "Advised Extensions" of
	// https://iabtechlab.com/wp-content/uploads/2018/02/OpenRTB_Advisory_GDPR_2018-02.pdf
	Consent string `json:"consent,omitempty"`

	Prebid *ExtUserPrebid `json:"prebid,omitempty"`

	Eids []openrtb2.EID `json:"eids,omitempty"`
}

// ExtUserPrebid defines the contract for bidrequest.user.ext.prebid
type ExtUserPrebid struct {
	BuyerUIDs map[string]string `json:"buyeruids,omitempty"`
}
