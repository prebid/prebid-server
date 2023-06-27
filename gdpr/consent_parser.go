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
	specVersion     uint16
	listVersion     uint16
	consentMeta     tcf2.ConsentMetadata
}

// parseConsent parses and validates the specified consent string returning an instance of parsedConsent
func parseConsent(consent string) (parsedConsent, error) {
	pc := parsedConsent{}

	parsedConsent, err := vendorconsent.ParseString(consent)
	if err != nil {
		err = &ErrorMalformedConsent{
			Consent: consent,
			Cause:   err,
		}
		return pc, err
	}

	err = validateVersions(parsedConsent)
	if err != nil {
		err = &ErrorMalformedConsent{
			Consent: consent,
			Cause:   err,
		}
		return pc, err
	}

	pc.encodingVersion = parsedConsent.Version()
	if pc.encodingVersion == 1 {
		return pc, nil
	}

	pc.specVersion = getSpecVersion(parsedConsent.TCFPolicyVersion())
	pc.listVersion = parsedConsent.VendorListVersion()
	cm, ok := parsedConsent.(tcf2.ConsentMetadata)
	if !ok {
		err = errors.New("Unable to access TCF2 parsed consent")
		return pc, err
	}
	pc.consentMeta = cm

	return pc, nil
}

// validateVersions ensures that certain version fields in the consent string contain valid values.
// An error is returned if at least one of them is invalid
func validateVersions(pc api.VendorConsents) (err error) {
	version := pc.Version()
	if version != 1 && version != 2 {
		return fmt.Errorf("invalid encoding format version: %d", version)
	}
	policyVersion := pc.TCFPolicyVersion()
	if policyVersion > 4 {
		return fmt.Errorf("invalid TCF policy version: %d", policyVersion)
	}
	return
}

// getSpecVersion looks at the TCF policy version and determines the corresponding GVL specification
// version that should be used to calculate legal basis. A zero value is returned if the policy version
// is invalid
func getSpecVersion(policyVersion uint8) uint16 {
	if policyVersion == 4 {
		return 3
	}
	if policyVersion < 4 {
		return 2
	}
	return 0
}
