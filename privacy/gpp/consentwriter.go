package gpp

import (
	"github.com/prebid/openrtb/v20/openrtb2"
)

// ConsentWriter implements the PolicyWriter interface for GPP.
// GppSid is pre-parsed by the caller (see amp.ReadPolicy) so the string is parsed once and a
// malformed gpp_sid can be surfaced as a warning without dropping the GPP consent string.
type ConsentWriter struct {
	Consent string
	GppSid  []int8
}

// Write mutates an OpenRTB bid request with the GPP consent.
func (c ConsentWriter) Write(req *openrtb2.BidRequest) error {
	if req == nil {
		return nil
	}

	if req.Regs == nil {
		req.Regs = &openrtb2.Regs{}
	}

	if c.Consent != "" {
		req.Regs.GPP = c.Consent
	}

	if len(c.GppSid) > 0 {
		req.Regs.GPPSID = c.GppSid
	}

	return nil
}
