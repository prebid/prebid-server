package ccpa

import (
	"github.com/prebid/openrtb/v20/openrtb2"
)

// ConsentWriter implements the old PolicyWriter interface for CCPA.
// This is used where we have not converted to RequestWrapper yet
type ConsentWriter struct {
	Consent string
}

// Write mutates an OpenRTB bid request with the CCPA consent string.
func (c ConsentWriter) Write(req *openrtb2.BidRequest) error {
	if req == nil {
		return nil
	}

	// Set consent string in USPrivacy
	if c.Consent != "" {
		if req.Regs == nil {
			req.Regs = &openrtb2.Regs{}
		}
		req.Regs.USPrivacy = c.Consent
	}

	return nil
}
