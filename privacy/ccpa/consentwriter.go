package ccpa

import (
	"github.com/prebid/prebid-server/openrtb_ext"
)

// ConsentWriter implements the PolicyWriter interface for CCPA.
type ConsentWriter struct {
	Consent string
}

// Write mutates an OpenRTB bid request with the CCPA consent string.
func (c ConsentWriter) Write(req *openrtb_ext.RequestWrapper) error {
	if req == nil {
		return nil
	}
	// START BELOW HERE
	regs, err := buildRegs(c.Consent, req.Regs)
	if err != nil {
		return err
	}
	req.Regs = regs

	return nil
}
