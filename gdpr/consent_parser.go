package gdpr

import (
	"errors"
	"fmt"

	"github.com/prebid/go-gdpr/api"
	"github.com/prebid/go-gdpr/vendorconsent"
	tcf2 "github.com/prebid/go-gdpr/vendorconsent/tcf2"
)

// parsedConsent represents a parsed consent string containing notable version information and a convenient
// metadata object that allows easy examination of encoded purpose and vendor information
type parsedConsent struct {
	encodingVersion uint8
	listVersion     uint16
	specVersion     uint16
	consentMeta     tcf2.ConsentMetadata
}

// parseConsent parses and validates the specified consent string returning an instance of parsedConsent
func parseConsent(consent string) (*parsedConsent, error) {
	pc, err := vendorconsent.ParseString(consent)
	if err != nil {
		return nil, &ErrorMalformedConsent{
			Consent: consent,
			Cause:   err,
		}
	}
	if err = validateVersions(pc); err != nil {
		return nil, &ErrorMalformedConsent{
			Consent: consent,
			Cause:   err,
		}
	}
	cm, ok := pc.(tcf2.ConsentMetadata)
	if !ok {
		err = errors.New("Unable to access TCF2 parsed consent")
		return nil, err
	}
	return &parsedConsent{
		encodingVersion: pc.Version(),
		listVersion:     pc.VendorListVersion(),
		specVersion:     getSpecVersion(pc.TCFPolicyVersion()),
		consentMeta:     cm,
	}, nil
}

// validateVersions ensures that certain version fields in the consent string contain valid values.
// An error is returned if at least one of them is invalid
func validateVersions(pc api.VendorConsents) (err error) {
	version := pc.Version()
	if version != 2 {
		return fmt.Errorf("invalid encoding format version: %d", version)
	}
	return
}

// getSpecVersion looks at the TCF policy version and determines the corresponding GVL specification
// version that should be used to calculate legal basis. A zero value is returned if the policy version
// is invalid
func getSpecVersion(policyVersion uint8) uint16 {
	if policyVersion >= 4 {
		return 3
	}
	return 2
}
