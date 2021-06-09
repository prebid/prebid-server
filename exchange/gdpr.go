package exchange

import (
	"encoding/json"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/gdpr"
)

// ExtractGDPR will pull the gdpr flag from an openrtb request
func extractGDPR(bidRequest *openrtb2.BidRequest) (gdpr.Signal, error) {
	var re regsExt
	var err error
	if bidRequest.Regs != nil && bidRequest.Regs.Ext != nil {
		err = json.Unmarshal(bidRequest.Regs.Ext, &re)
	}
	if re.GDPR == nil || err != nil {
		return gdpr.SignalAmbiguous, err
	}
	return gdpr.Signal(*re.GDPR), nil
}

// ExtractConsent will pull the consent string from an openrtb request
func extractConsent(bidRequest *openrtb2.BidRequest) (consent string, err error) {
	var ue userExt
	if bidRequest.User != nil && bidRequest.User.Ext != nil {
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
