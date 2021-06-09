package ccpa

import (
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// ConsentWriterLegacy implements the old PolicyWriter interface for CCPA.
// This is used where we have not converted to RequestWrapper yet
type ConsentWriterLegacy struct {
	Consent string
}

// Write mutates an OpenRTB bid request with the CCPA consent string.
func (c ConsentWriterLegacy) Write(req *openrtb2.BidRequest) error {
	if req == nil {
		return nil
	}
	reqWrap := &openrtb_ext.RequestWrapper{BidRequest: req}
	if regsExt, err := reqWrap.GetRegExt(); err == nil {
		regsExt.SetUSPrivacy(c.Consent)
	} else {
		return err
	}
	return reqWrap.RebuildRequest()
}
