package openrtb_ext

import (
	"strconv"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
)

// ExtUser defines the contract for bidrequest.user.ext
type ExtUser struct {
	// Consent is a GDPR consent string. See "Advised Extensions" of
	// https://iabtechlab.com/wp-content/uploads/2018/02/OpenRTB_Advisory_GDPR_2018-02.pdf
	Consent string `json:"consent,omitempty"`

	ConsentedProvidersSettings *ConsentedProvidersSettingsIn `json:"ConsentedProvidersSettings,omitempty"`

	ConsentedProvidersSettingsParsed *ConsentedProvidersSettingsOut `json:"consented_providers_settings,omitempty"`

	Prebid *ExtUserPrebid `json:"prebid,omitempty"`

	Eids []openrtb2.EID `json:"eids,omitempty"`
}

// ExtUserPrebid defines the contract for bidrequest.user.ext.prebid
type ExtUserPrebid struct {
	BuyerUIDs map[string]string `json:"buyeruids,omitempty"`
}

type ConsentedProvidersSettingsIn struct {
	ConsentedProvidersString string `json:"consented_providers,omitempty"`
}

type ConsentedProvidersSettingsOut struct {
	ConsentedProvidersList []int `json:"consented_providers,omitempty"`
}

// ParseConsentedProvidersString takes a string formatted as Google's Additional Consent format and returns a list with its
// elements. For instance, the following string "1~1.35.41.101" would result in []int{1, 35, 41, 101}
func ParseConsentedProvidersString(cps string) []int {
	// Additional Consent format version is separated from elements by the '~' character
	parts := strings.Split(cps, "~")
	if len(parts) != 2 {
		return nil
	}

	// Split the individual elements
	providerStringList := strings.Split(parts[1], ".")
	if len(providerStringList) == 0 {
		return nil
	}

	// Convert to ints and add to int array
	var consentedProviders []int
	for _, providerStr := range providerStringList {
		if providerInt, err := strconv.Atoi(providerStr); err == nil {
			consentedProviders = append(consentedProviders, providerInt)
		}
	}

	return consentedProviders
}
