package gdpr

import (
	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
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
		if regsExt, err := reqWrap.GetRegExt(); err == nil {
			regsExt.SetGDPR(c.RegExtGDPR)
		} else {
			return err
		}
	}

	if c.Consent != "" {
		if userExt, err := reqWrap.GetUserExt(); err == nil {
			userExt.SetConsent(&c.Consent)
		} else {
			return err
		}
	}

	if err := reqWrap.RebuildRequest(); err != nil {
		return err
	}

	return nil
}
