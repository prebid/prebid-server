package exchange

import (
	gpplib "github.com/prebid/go-gpp"
	gppConstants "github.com/prebid/go-gpp/constants"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/openrtb_ext"
	gppPolicy "github.com/prebid/prebid-server/privacy/gpp"
)

// getGDPR will pull the gdpr flag from an openrtb request
func getGDPR(req *openrtb_ext.RequestWrapper) (gdpr.Signal, error) {
	if req.Regs != nil && len(req.Regs.GPPSID) > 0 {
		if gppPolicy.IsSIDInList(req.Regs.GPPSID, gppConstants.SectionTCFEU2) {
			return gdpr.SignalYes, nil
		}
		return gdpr.SignalNo, nil
	}
	re, err := req.GetRegExt()
	if re == nil || re.GetGDPR() == nil || err != nil {
		return gdpr.SignalAmbiguous, err
	}
	return gdpr.Signal(*re.GetGDPR()), nil
}

// getConsent will pull the consent string from an openrtb request
func getConsent(req *openrtb_ext.RequestWrapper, gpp gpplib.GppContainer) (consent string, err error) {
	if i := gppPolicy.IndexOfSID(gpp, gppConstants.SectionTCFEU2); i >= 0 {
		consent = gpp.Sections[i].GetValue()
		return
	}
	ue, err := req.GetUserExt()
	if ue == nil || ue.GetConsent() == nil || err != nil {
		return
	}
	return *ue.GetConsent(), nil
}
