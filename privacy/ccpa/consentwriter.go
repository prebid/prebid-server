package ccpa

import (
	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
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
		if regsExt, err := reqWrap.GetRegExt(); err == nil {
			regsExt.SetUSPrivacy(c.Consent)
		} else {
			return err
		}
	}

	return reqWrap.RebuildRequest()
}
