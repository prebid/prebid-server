package gdpr

import (
	"github.com/prebid/go-gdpr/vendorconsent"
)

// Policy represents the GDPR regulation for an OpenRTB bid request.
type Policy struct {
	Signal  string
	Consent string
}

type GDPRConsentValidator struct{}

// ValidateConsent returns true if the consent string is empty or valid per the IAB TCF spec.
func (GDPRConsentValidator) ValidateConsent(consent string) bool {
	_, err := vendorconsent.ParseString(consent)
	return err == nil
}
