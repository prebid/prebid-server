package ccpa

import (
	"github.com/mxmCherry/openrtb/v14/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// ConsentWriter implements the PolicyWriter interface for CCPA.
type ConsentWriter struct {
	Consent string
}

// Write mutates an OpenRTB bid request with the CCPA consent string.
func (c ConsentWriter) Write(req *openrtb_ext.RequestWrapper) {
	if req == nil || req.Request == nil {
		return
	}
	buildRegs(c.Consent, req.RegExt)
}

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
	reqWrap := &openrtb_ext.RequestWrapper{Request: req}
	reqWrap.ExtractRegExt()
	buildRegs(c.Consent, reqWrap.RegExt)
	return reqWrap.Sync()
}
