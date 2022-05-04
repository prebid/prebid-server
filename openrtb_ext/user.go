package openrtb_ext

import "encoding/json"

// ExtUser defines the contract for bidrequest.user.ext
type ExtUser struct {
	// Consent is a GDPR consent string. See "Advised Extensions" of
	// https://iabtechlab.com/wp-content/uploads/2018/02/OpenRTB_Advisory_GDPR_2018-02.pdf
	Consent string `json:"consent,omitempty"`

	Prebid *ExtUserPrebid `json:"prebid,omitempty"`

	Eids []ExtUserEid `json:"eids,omitempty"`
}

// ExtUserPrebid defines the contract for bidrequest.user.ext.prebid
type ExtUserPrebid struct {
	BuyerUIDs map[string]string `json:"buyeruids,omitempty"`
}

// ExtUserEid defines the contract for bidrequest.user.ext.eids
// Responsible for the Universal User ID support: establishing pseudonymous IDs for users.
// See https://github.com/prebid/Prebid.js/issues/3900 for details.
type ExtUserEid struct {
	Source string          `json:"source"`
	ID     string          `json:"id,omitempty"`
	Uids   []ExtUserEidUid `json:"uids,omitempty"`
	Ext    json.RawMessage `json:"ext,omitempty"`
}

// ExtUserEidUid defines the contract for bidrequest.user.ext.eids[i].uids[j]
type ExtUserEidUid struct {
	ID    string          `json:"id"`
	Atype int             `json:"atype,omitempty"`
	Ext   json.RawMessage `json:"ext,omitempty"`
}
