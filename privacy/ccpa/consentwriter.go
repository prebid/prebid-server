package ccpa

import "github.com/mxmCherry/openrtb/v15/openrtb2"

// ConsentWriter implements the PolicyWriter interface for CCPA.
type ConsentWriter struct {
	Consent string
}

// Write mutates an OpenRTB bid request with the CCPA consent string.
func (c ConsentWriter) Write(req *openrtb2.BidRequest) error {
	if req == nil {
		return nil
	}

	regs, err := buildRegs(c.Consent, req.Regs)
	if err != nil {
		return err
	}
	req.Regs = regs

	return nil
}
