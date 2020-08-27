package gdpr

import (
	"encoding/json"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/privacy"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/go-gdpr/vendorconsent"
)

// Policy represents the GDPR regulation for an OpenRTB bid request.
type Policy struct {
	Signal  string
	Consent string
}

type consentWriter struct {
	consent string
}

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

func NewConsentWriter(consent string) privacy.PolicyWriter {
	return consentWriter{consent}
}

// ValidateConsent returns true if the consent string is empty or valid per the IAB TCF spec.
func ValidateConsent(consent string) error {
	_, err := vendorconsent.ParseString(consent)
	return err == nil
}
