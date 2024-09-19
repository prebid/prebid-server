package gdpr

import (
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

// ConsentWriter implements the PolicyWriter interface for GDPR TCF.
type ConsentWriter struct {
	Consent    string
	RegExtGDPR *int8
}

// Write mutates an OpenRTB bid request with the GDPR TCF consent.
func (c ConsentWriter) Write(req *openrtb2.BidRequest) error {
	if req == nil {
		return nil
	}
	reqWrap := &openrtb_ext.RequestWrapper{BidRequest: req}

	if c.RegExtGDPR != nil {
		if reqWrap.Regs != nil {
			reqWrap.Regs.GDPR = c.RegExtGDPR
		}
	}

	if c.Consent != "" {
		if reqWrap.User != nil {
			reqWrap.User.Consent = c.Consent
		}
	}

	// do we need to rebuild req here?
	if err := reqWrap.RebuildRequest(); err != nil {
		return err
	}

	return nil
}
