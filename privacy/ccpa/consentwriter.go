package ccpa

import (
	"github.com/prebid/prebid-server/openrtb_ext"
)

// ConsentWriter implements the PolicyWriter interface for CCPA.
type ConsentWriter struct {
	Consent string
}

// Write mutates an OpenRTB bid request with the CCPA consent string.
func (c ConsentWriter) Write(req *openrtb_ext.RequestWrapper) {
	if req == nil {
		return
	}
	buildRegs(c.Consent, req.RegExt)
}
