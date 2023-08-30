package gdpr

import (
	"errors"
	"testing"
	"time"

	"github.com/prebid/go-gdpr/consentconstants"

	"github.com/stretchr/testify/assert"
)

func TestParseConsent(t *testing.T) {
	validTCF1Consent := "BONV8oqONXwgmADACHENAO7pqzAAppY"
	validTCF2Consent := "CPuKGCPPuKGCPNEAAAENCZCAAAAAAAAAAAAAAAAAAAAA"

	tests := []struct {
		name                    string
		consent                 string
		expectedEncodingVersion uint8
		expectedListVersion     uint16
		expectedSpecVersion     uint16
		expectedError           error
	}{

		{
			name:                    "valid_consent_with_encoding_version_2",
			consent:                 validTCF2Consent,
			expectedEncodingVersion: 2,
			expectedListVersion:     153,
			expectedSpecVersion:     2,
		},
		{
			name:    "invalid_consent_parsing_error",
			consent: "",
			expectedError: &ErrorMalformedConsent{
				Consent: "",
				Cause:   consentconstants.ErrEmptyDecodedConsent,
			},
		},
		{
			name:    "invalid_consent_version_validation_error",
			consent: validTCF1Consent,
			expectedError: &ErrorMalformedConsent{
				Consent: validTCF1Consent,
				Cause:   errors.New("invalid encoding format version: 1"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedConsent, err := parseConsent(tt.consent)

			if tt.expectedError != nil {
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, parsedConsent)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, parsedConsent)
				assert.Equal(t, uint8(2), parsedConsent.encodingVersion)
				assert.Equal(t, uint16(153), parsedConsent.listVersion)
				assert.Equal(t, uint16(2), parsedConsent.specVersion)
				assert.Equal(t, tt.expectedEncodingVersion, parsedConsent.consentMeta.Version())
				assert.Equal(t, tt.expectedListVersion, parsedConsent.consentMeta.VendorListVersion())
			}
		})
	}
}

func TestValidateVersions(t *testing.T) {
	tests := []struct {
		name          string
		version       uint8
		expectedError error
	}{
		{
			name:    "valid_consent_version=2",
			version: 2,
		},
		{
			name:          "invalid_consent_version<2",
			version:       1,
			expectedError: errors.New("invalid encoding format version: 1"),
		},
		{
			name:          "invalid_consent_version>2",
			version:       3,
			expectedError: errors.New("invalid encoding format version: 3"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcs := mockConsentString{
				version: tt.version,
			}
			err := validateVersions(&mcs)
			if tt.expectedError != nil {
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestGetSpecVersion(t *testing.T) {
	tests := []struct {
		name                string
		policyVersion       uint8
		expectedSpecVersion uint16
	}{
		{
			name:                "policy_version_0_gives_spec_version_2",
			policyVersion:       0,
			expectedSpecVersion: 2,
		},
		{
			name:                "policy_version_3_gives_spec_version_2",
			policyVersion:       3,
			expectedSpecVersion: 2,
		},
		{
			name:                "policy_version_4_gives_spec_version_3",
			policyVersion:       4,
			expectedSpecVersion: 3,
		},
		{
			name:                "policy_version_5_gives_spec_version_3",
			policyVersion:       5,
			expectedSpecVersion: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			specVersion := getSpecVersion(tt.policyVersion)
			assert.Equal(t, tt.expectedSpecVersion, specVersion)
		})
	}
}

type mockConsentString struct {
	version       uint8
	policyVersion uint8
}

func (mcs *mockConsentString) Version() uint8                                  { return mcs.version }
func (mcs *mockConsentString) Created() time.Time                              { return time.Time{} }
func (mcs *mockConsentString) LastUpdated() time.Time                          { return time.Time{} }
func (mcs *mockConsentString) CmpID() uint16                                   { return 0 }
func (mcs *mockConsentString) CmpVersion() uint16                              { return 0 }
func (mcs *mockConsentString) ConsentScreen() uint8                            { return 0 }
func (mcs *mockConsentString) ConsentLanguage() string                         { return "" }
func (mcs *mockConsentString) VendorListVersion() uint16                       { return 0 }
func (mcs *mockConsentString) TCFPolicyVersion() uint8                         { return mcs.policyVersion }
func (mcs *mockConsentString) MaxVendorID() uint16                             { return 0 }
func (mcs *mockConsentString) PurposeAllowed(id consentconstants.Purpose) bool { return false }
func (mcs *mockConsentString) VendorConsent(id uint16) bool                    { return false }
