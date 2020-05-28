package gdpr

import (
	"encoding/json"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/go-gdpr/vendorconsent"
)

// Policy represents the GDPR regulation for an OpenRTB bid request.
type Policy struct {
	Signal  string
	Consent string
}

// Write mutates an OpenRTB bid request with the context of the GDPR policy.
func (p Policy) Write(req *openrtb.BidRequest) error {
	if p.Consent == "" {
		return nil
	}

	if req.User == nil {
		req.User = &openrtb.User{}
	}

	if req.User.Ext == nil {
		ext, err := json.Marshal(openrtb_ext.ExtUser{Consent: p.Consent})
		if err == nil {
			req.User.Ext = ext
		}
		return err
	}

	var extMap map[string]interface{}
	err := json.Unmarshal(req.User.Ext, &extMap)
	if err == nil {
		extMap["consent"] = p.Consent
		ext, err := json.Marshal(extMap)
		if err == nil {
			req.User.Ext = ext
		}
	}
	return err
}

// ValidateConsent returns an error if the GDPR consent string does not adhere to the IAB TCF spec.
func ValidateConsent(consent string) error {
	_, err := vendorconsent.ParseString(consent)
	return err
}
