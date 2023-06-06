package exchange

import (
	gpplib "github.com/prebid/go-gpp"
	gppConstants "github.com/prebid/go-gpp/constants"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// GetGDPR will pull the gdpr flag from an openrtb request
func GetGDPR(req *openrtb_ext.RequestWrapper) (gdpr.Signal, error) {
	if req.Regs != nil && len(req.Regs.GPPSID) > 0 {
		for _, id := range req.Regs.GPPSID {
			if id == int8(gppConstants.SectionTCFEU2) {
				return gdpr.SignalYes, nil
			}
		}
		return gdpr.SignalNo, nil
	}
	re, err := req.GetRegExt()
	if re == nil || re.GetGDPR() == nil || err != nil {
		return gdpr.SignalAmbiguous, err
	}
	return gdpr.Signal(*re.GetGDPR()), nil
}

// GetConsent will pull the consent string from an openrtb request
func GetConsent(req *openrtb_ext.RequestWrapper, gpp gpplib.GppContainer) (consent string, err error) {
	for i, id := range gpp.SectionTypes {
		if id == gppConstants.SectionTCFEU2 {
			consent = gpp.Sections[i].GetValue()
			return
		}
	}
	ue, err := req.GetUserExt()
	if ue == nil || ue.GetConsent() == nil || err != nil {
		return
	}
	return *ue.GetConsent(), nil
}

func GDPRApplies(gdprSignal gdpr.Signal, defaultValue string) bool {
	if gdprSignal == gdpr.SignalYes {
		return true
	}
	if gdprSignal == gdpr.SignalAmbiguous && defaultValue == "1" {
		return true
	}
	return false
}
