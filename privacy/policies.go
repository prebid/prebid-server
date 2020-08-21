package privacy

import (
	"github.com/mxmCherry/openrtb"

	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
)

type PolicyWriter interface {
	Write(req *openrtb.BidRequest) error
}

// ReadPolicyFromConsent inspects the consent string and returns a validated policy writer.
func ReadPolicyFromConsent(consent string) (PolicyWriter, bool) {
	if len(consent) == 0 {
		return nil, false
	}

	if err := gdpr.ValidateConsent(consent); err == nil {
		return gdpr.Policy{Consent: consent}, true
	}

	if p, err := ccpa.Parse(ccpa.Policy{Consent: consent}, nil); err == nil {
		return p, true
	}

	return nil, false
}
