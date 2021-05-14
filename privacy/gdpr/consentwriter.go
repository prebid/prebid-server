package gdpr

import (
	"encoding/json"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// ConsentWriter implements the PolicyWriter interface for GDPR TCF.
type ConsentWriter struct {
	Consent string
}

// Write mutates an OpenRTB bid request with the GDPR TCF consent.
func (c ConsentWriter) Write(req *openrtb2.BidRequest) error {
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
