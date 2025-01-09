package gdpr

import (
	"github.com/prebid/openrtb/v20/openrtb2"
)

// ConsentWriter implements the PolicyWriter interface for GDPR TCF.
type ConsentWriter struct {
	Consent string
	GDPR    *int8
}

// Write mutates an OpenRTB bid request with the GDPR TCF consent.
func (c ConsentWriter) Write(req *openrtb2.BidRequest) error {
	if req == nil {
		return nil
	}

	if c.GDPR != nil {
		if req.Regs == nil {
			req.Regs = &openrtb2.Regs{}
		}
		req.Regs.GDPR = c.GDPR
	}

	if c.Consent != "" {
		if req.User == nil {
			req.User = &openrtb2.User{}
		}
		req.User.Consent = c.Consent
	}

	return nil
}
