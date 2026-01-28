package gpp

import (
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/util/stringutil"
)

// ConsentWriter implements the PolicyWriter interface for GPP.
type ConsentWriter struct {
	Consent string
	GppSid  string
}

// Write mutates an OpenRTB bid request with the GPP consent.
func (c ConsentWriter) Write(req *openrtb2.BidRequest) error {
	if req == nil {
		return nil
	}

	if req.Regs == nil {
		req.Regs = &openrtb2.Regs{}
	}

	// Set GPP consent string
	if c.Consent != "" {
		req.Regs.GPP = c.Consent
	}

	// Parse and set GPP SID
	if c.GppSid != "" {
		gppSID, err := stringutil.StrToInt8Slice(c.GppSid)
		if err == nil {
			req.Regs.GPPSID = gppSID
		}
		// If parsing fails, GPPSID remains nil (as per spec)
	}

	return nil
}
