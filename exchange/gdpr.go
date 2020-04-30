package exchange

import (
	"encoding/json"

	"github.com/mxmCherry/openrtb"
)

// ExtractGDPR will pull the gdpr flag from an openrtb request
func extractGDPR(bidRequest *openrtb.BidRequest, usersyncIfAmbiguous bool) (gdpr int) {
	var re regsExt
	var err error
	if bidRequest.Regs != nil {
		err = json.Unmarshal(bidRequest.Regs.Ext, &re)
	}
	if re.GDPR == nil || err != nil {
		if usersyncIfAmbiguous {
			gdpr = 0
		} else {
			gdpr = 1
		}
	} else {
		gdpr = *re.GDPR
	}
	return
}

// ExtractConsent will pull the consent string from an openrtb request
func extractConsent(bidRequest *openrtb.BidRequest) (consent string) {
	var ue userExt
	var err error
	if bidRequest.User != nil {
		err = json.Unmarshal(bidRequest.User.Ext, &ue)
	}
	if err != nil {
		return
	}
	consent = ue.Consent
	return
}

type userExt struct {
	Consent string `json:"consent,omitempty"`
}

type regsExt struct {
	GDPR *int `json:"gdpr,omitempty"`
}
