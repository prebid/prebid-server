package privacy

import (
	"github.com/mxmCherry/openrtb"

	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
)

// Policies represents the privacy regulations for an OpenRTB bid request.
type Policies struct {
	GDPR gdpr.Policy
	CCPA ccpa.Policy
}

type policyWriter interface {
	Write(req *openrtb.BidRequest) error
}

// Write mutates an OpenRTB bid request with the policies applied.
func (p Policies) Write(req *openrtb.BidRequest) error {
	return writePolicies(req, []policyWriter{
		p.GDPR, p.CCPA,
	})
}

func writePolicies(req *openrtb.BidRequest, writers []policyWriter) error {
	for _, writer := range writers {
		if err := writer.Write(req); err != nil {
			return err
		}
	}

	return nil
}

// ReadPoliciesFromConsent inspects the consent string kind and sets the corresponding values in a new Policies object.
func ReadPoliciesFromConsent(consent string) (Policies, bool) {
	if len(consent) == 0 {
		return Policies{}, false
	}

	if err := gdpr.ValidateConsent(consent); err == nil {
		return Policies{
			GDPR: gdpr.Policy{
				Consent: consent,
			},
		}, true
	}

	if err := ccpa.ValidateConsent(consent); err == nil {
		return Policies{
			CCPA: ccpa.Policy{
				Value: consent,
			},
		}, true
	}

	return Policies{}, false
}
