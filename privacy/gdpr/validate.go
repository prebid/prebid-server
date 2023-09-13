package gdpr

import (
	"github.com/prebid/go-gdpr/vendorconsent"
)

// ValidateConsent returns true if the consent string is empty or valid per the IAB TCF spec.
func ValidateConsent(consent string) bool {
	_, err := vendorconsent.ParseString(consent)
	return err == nil
}
