package ccpa

import (
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
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
	reqWrap := &openrtb_ext.RequestWrapper{BidRequest: req}

	// Set consent string in USPrivacy
	if c.Consent != "" {
		if reqWrap.Regs != nil {
			reqWrap.Regs.USPrivacy = c.Consent
		}
	}

	// do we need to rebuild req here?
	return reqWrap.RebuildRequest()
}
