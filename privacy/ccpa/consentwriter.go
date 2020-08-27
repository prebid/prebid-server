package ccpa

import (
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/privacy"
)

type consentWriter struct {
	consent string
}

// Write mutates an OpenRTB bid request with the CCPA consent.
func (c consentWriter) Write(req *openrtb.BidRequest) error {
	if req == nil {
		return nil
	}

	regs, err := buildRegs(c.consent, req.Regs)
	if err != nil {
		return err
	}
	req.Regs = regs

	return nil
}

// NewConsentWriter constructs a privacy.PolicyWriter to write the CCPA consent to an OpenRTB bid request.
func NewConsentWriter(consent string) privacy.PolicyWriter {
	return consentWriter{consent}
}
