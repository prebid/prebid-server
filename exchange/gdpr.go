package exchange

import (
	gpplib "github.com/prebid/go-gpp"
	gppConstants "github.com/prebid/go-gpp/constants"
	"github.com/prebid/prebid-server/v3/gdpr"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	gppPolicy "github.com/prebid/prebid-server/v3/privacy/gpp"
)

// getGDPR will pull the gdpr flag from an openrtb request
func getGDPR(req *openrtb_ext.RequestWrapper) (gdpr.Signal, error) {
	if req.Regs != nil && len(req.Regs.GPPSID) > 0 {
		if gppPolicy.IsSIDInList(req.Regs.GPPSID, gppConstants.SectionTCFEU2) {
			return gdpr.SignalYes, nil
		}
		return gdpr.SignalNo, nil
	}
	if req.Regs != nil && req.Regs.GDPR != nil {
		return gdpr.IntSignalParse(int(*req.Regs.GDPR))
	}
	return gdpr.SignalAmbiguous, nil

}

// getConsent will pull the consent string from an openrtb request
func getConsent(req *openrtb_ext.RequestWrapper, gpp gpplib.GppContainer) (consent string, err error) {
	if i := gppPolicy.IndexOfSID(gpp, gppConstants.SectionTCFEU2); i >= 0 {
		consent = gpp.Sections[i].GetValue()
		return
	}
	if req.User != nil {
		return req.User.Consent, nil
	}
	return
}

// enforceGDPR determines if GDPR should be enforced based on the request signal and whether the channel is enabled
func enforceGDPR(signal gdpr.Signal, defaultValue gdpr.Signal, channelEnabled bool) bool {
	gdprApplies := signal == gdpr.SignalYes || (signal == gdpr.SignalAmbiguous && defaultValue == gdpr.SignalYes)
	return gdprApplies && channelEnabled
}

// SelectEEACountries selects the EEA countries based on host and account configurations.
// Account-level configuration takes precedence over the host-level configuration.
func selectEEACountries(hostEEACountries []string, accountEEACountries []string) []string {
	if accountEEACountries != nil {
		return accountEEACountries
	}
	return hostEEACountries
}
