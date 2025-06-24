package exchange

import (
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"

	gpplib "github.com/prebid/go-gpp"
	gppConstants "github.com/prebid/go-gpp/constants"
	"github.com/prebid/prebid-server/v3/gdpr"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	gppPolicy "github.com/prebid/prebid-server/v3/privacy/gpp"
)

// GetGDPR will pull the gdpr flag from an openrtb request
func GetGDPR(req *openrtb_ext.RequestWrapper) (gdpr.Signal, error) {
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

// GetConsent will pull the consent string from an openrtb request
func GetConsent(req *openrtb_ext.RequestWrapper, gpp gpplib.GppContainer) (consent string, err error) {
	if i := gppPolicy.IndexOfSID(gpp, gppConstants.SectionTCFEU2); i >= 0 {
		consent = gpp.Sections[i].GetValue()
		return
	}
	if req.User != nil {
		return req.User.Consent, nil
	}
	return
}

// EnforceGDPR determines if GDPR should be enforced based on the request signal and whether the channel is enabled
func EnforceGDPR(signal gdpr.Signal, defaultValue gdpr.Signal, channelEnabled bool) bool {
	gdprApplies := signal == gdpr.SignalYes || (signal == gdpr.SignalAmbiguous && defaultValue == gdpr.SignalYes)
	return gdprApplies && channelEnabled
}

// SelectEEACountries selects the EEA countries based on host and account configurations.
// Account-level configuration takes precedence over the host-level configuration.
func SelectEEACountries(hostEEACountries []string, accountEEACountries []string) []string {
	if accountEEACountries != nil {
		return accountEEACountries
	}
	return hostEEACountries
}

// ParseGDPRDefaultValue determines the default GDPR signal based on the request, configuration, and EEA countries.
func ParseGDPRDefaultValue(r *openrtb_ext.RequestWrapper, cfgDefault string, eeaCountries []string) gdpr.Signal {
	gdprDefaultValue := gdpr.SignalYes
	if cfgDefault == "0" {
		gdprDefaultValue = gdpr.SignalNo
	}

	var geo *openrtb2.Geo
	if r.User != nil && r.User.Geo != nil {
		geo = r.User.Geo
	} else if r.Device != nil && r.Device.Geo != nil {
		geo = r.Device.Geo
	}

	if geo != nil {
		// If the country is in the EEA list, GDPR applies.
		// Otherwise, if the country code is properly formatted (3 characters), GDPR does not apply.
		if isEEACountry(geo.Country, eeaCountries) {
			gdprDefaultValue = gdpr.SignalYes
		} else if len(geo.Country) == 3 {
			gdprDefaultValue = gdpr.SignalNo
		}
	}

	return gdprDefaultValue
}

// isEEACountry checks if the given country is part of the EEA countries list.
func isEEACountry(country string, eeaCountries []string) bool {
	if len(eeaCountries) == 0 {
		return false
	}

	country = strings.ToUpper(country)
	for _, c := range eeaCountries {
		if strings.ToUpper(c) == country {
			return true
		}
	}
	return false
}
