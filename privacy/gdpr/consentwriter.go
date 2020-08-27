package gdpr

import (
	"encoding/json"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/privacy"

	"github.com/mxmCherry/openrtb"
)

type consentWriter struct {
	consent string
}

// Write mutates an OpenRTB bid request with the GDPR TCF consent.
func (c consentWriter) Write(req *openrtb.BidRequest) error {
	if c.consent == "" {
		return nil
	}

	if req.User == nil {
		req.User = &openrtb.User{}
	}

	if req.User.Ext == nil {
		ext, err := json.Marshal(openrtb_ext.ExtUser{Consent: c.consent})
		if err == nil {
			req.User.Ext = ext
		}
		return err
	}

	var extMap map[string]interface{}
	err := json.Unmarshal(req.User.Ext, &extMap)
	if err == nil {
		extMap["consent"] = c.consent
		ext, err := json.Marshal(extMap)
		if err == nil {
			req.User.Ext = ext
		}
	}
	return err
}

// NewConsentWriter constructs a privacy.PolicyWriter to write the GDPR TCF consent to an OpenRTB bid request.
func NewConsentWriter(consent string) privacy.PolicyWriter {
	return consentWriter{consent}
}
