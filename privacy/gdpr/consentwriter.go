package gdpr

import (
	"encoding/json"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// ConsentWriter implements the PolicyWriter interface for GDPR TCF.
type ConsentWriter struct {
	Consent     string
	GDPRApplies *bool
}

// Write mutates an OpenRTB bid request with the GDPR TCF consent.
func (c ConsentWriter) Write(req *openrtb2.BidRequest) error {
	if err := setRegExtGDPR(c.GDPRApplies, req); err != nil {
		return err
	}

	if c.Consent == "" {
		return nil
	}

	if req.User == nil {
		req.User = &openrtb2.User{}
	}

	if req.User.Ext == nil {
		ext, err := json.Marshal(openrtb_ext.ExtUser{Consent: c.Consent})
		if err == nil {
			req.User.Ext = ext
		}
		return err
	}

	var extMap map[string]interface{}
	err := json.Unmarshal(req.User.Ext, &extMap)
	if err == nil {
		extMap["consent"] = c.Consent
		ext, err := json.Marshal(extMap)
		if err == nil {
			req.User.Ext = ext
		}
	}

	return err
}

// setRegExtGDPR sets regs.ext.gdpr to either 0 or 1 only if it was set in the request query
func setRegExtGDPR(gdprApplies *bool, req *openrtb2.BidRequest) error {
	if gdprApplies == nil {
		return nil
	}

	gdpr := int8(0)
	if *gdprApplies {
		gdpr++
	}

	reqWrap := &openrtb_ext.RequestWrapper{BidRequest: req}

	regsExt, err := reqWrap.GetRegExt()
	if err != nil {
		return err
	}

	regsExt.SetGDPR(&gdpr)

	return reqWrap.RebuildRequest()
}
