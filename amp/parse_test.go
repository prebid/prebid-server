package amp

import (
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/privacy"
	"github.com/prebid/prebid-server/v3/privacy/ccpa"
	"github.com/prebid/prebid-server/v3/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestParseParams(t *testing.T) {
	var expectedTimeout uint64 = 42

	testCases := []struct {
		description    string
		query          string
		expectedParams Params
		expectedError  string
	}{
		{
			description:   "Empty",
			query:         "",
			expectedError: "AMP requests require an AMP tag_id",
		},
		{
			description: "All Fields",
			// targeting data is encoded string that looks like this: {"gam-key1":"val1","gam-key2":"val2"}
			query: "tag_id=anyTagID&account=anyAccount&curl=anyCurl&consent_string=anyConsent&debug=1&__amp_source_origin=anyOrigin" +
				"&slot=anySlot&timeout=42&h=1&w=2&oh=3&ow=4&ms=10x11,12x13&targeting=%7B%22gam-key1%22%3A%22val1%22%2C%22gam-key2%22%3A%22val2%22%7D&trace=basic",
			expectedParams: Params{
				Account:         "anyAccount",
				CanonicalURL:    "anyCurl",
				Consent:         "anyConsent",
				Debug:           true,
				Origin:          "anyOrigin",
				Slot:            "anySlot",
				StoredRequestID: "anyTagID",
				Timeout:         &expectedTimeout,
				Size: Size{
					Height:         1,
					OverrideHeight: 3,
					OverrideWidth:  4,
					Width:          2,
					Multisize: []openrtb2.Format{
						{W: 10, H: 11}, {W: 12, H: 13},
					},
				},
				Targeting: `{"gam-key1":"val1","gam-key2":"val2"}`,
				Trace:     "basic",
			},
		},
		{
			description:    "Integer Values Ignored If Invalid",
			query:          "tag_id=anyTagID&h=invalid&w=invalid&oh=invalid&ow=invalid&ms=invalid",
			expectedParams: Params{StoredRequestID: "anyTagID"},
		},
		{
			description:    "consent_string Preferred Over gdpr_consent",
			query:          "tag_id=anyTagID&consent_string=consent1&gdpr_consent=consent2",
			expectedParams: Params{StoredRequestID: "anyTagID", Consent: "consent1"},
		},
		{
			description:    "consent_string Preferred Over gdpr_consent - Order Doesn't Matter",
			query:          "tag_id=anyTagID&gdpr_consent=consent2&consent_string=consent1",
			expectedParams: Params{StoredRequestID: "anyTagID", Consent: "consent1"},
		},
		{
			description:    "Just gdpr_consent",
			query:          "tag_id=anyTagID&gdpr_consent=consent2",
			expectedParams: Params{StoredRequestID: "anyTagID", Consent: "consent2"},
		},
		{
			description:    "Debug 0",
			query:          "tag_id=anyTagID&debug=0",
			expectedParams: Params{StoredRequestID: "anyTagID", Debug: false},
		},
		{
			description:    "Debug Ignored If Invalid",
			query:          "tag_id=anyTagID&debug=invalid",
			expectedParams: Params{StoredRequestID: "anyTagID", Debug: false},
		},
	}

	for _, test := range testCases {
		httpRequest, err := http.NewRequest("GET", "http://any.url/anypage?"+test.query, nil)
		assert.NoError(t, err, test.description+":request")

		params, err := ParseParams(httpRequest)
		assert.Equal(t, test.expectedParams, params, test.description+":params")
		if test.expectedError == "" {
			assert.NoError(t, err, test.description+":err")
		} else {
			assert.EqualError(t, err, test.expectedError)
		}
	}
}

func TestParseIntPtr(t *testing.T) {
	var boolZero uint64 = 0
	var boolOne uint64 = 1

	type testResults struct {
		intPtr *uint64
		err    bool
	}

	testCases := []struct {
		desc     string
		input    string
		expected testResults
	}{
		{
			desc:  "Input is an empty string: expect nil pointer and error",
			input: "",
			expected: testResults{
				intPtr: nil,
				err:    true,
			},
		},
		{
			desc:  "Input is negative number: expect a nil pointer and error",
			input: "-1",
			expected: testResults{
				intPtr: nil,
				err:    true,
			},
		},
		{
			desc:  "Input is a string depicting a zero value: expect a reference pointing to zero value, no error",
			input: "0",
			expected: testResults{
				intPtr: &boolZero,
				err:    false,
			},
		},
		{
			desc:  "Input is a string depicting a value of 1: expect a reference pointing to the value of 1 and no error",
			input: "1",
			expected: testResults{
				intPtr: &boolOne,
				err:    false,
			},
		},
	}
	for _, tc := range testCases {
		resultingIntPtr, resultingErr := parseIntPtr(tc.input)

		assert.Equal(t, tc.expected.intPtr, resultingIntPtr, tc.desc)
		if tc.expected.err {
			assert.Error(t, resultingErr, tc.desc)
		} else {
			assert.NoError(t, resultingErr, tc.desc)
		}
	}
}

func TestParseBoolPtr(t *testing.T) {
	boolTrue := true
	boolFalse := false

	type testResults struct {
		boolPtr *bool
		err     bool
	}

	testCases := []struct {
		desc     string
		input    string
		expected testResults
	}{
		{
			desc:  "Input is an empty string: expect nil pointer and error",
			input: "",
			expected: testResults{
				boolPtr: nil,
				err:     true,
			},
		},
		{
			desc:  "Input is neither true nor false: expect a nil pointer and error",
			input: "other",
			expected: testResults{
				boolPtr: nil,
				err:     true,
			},
		},
		{
			desc:  "Input is the word 'false', expect a reference pointing to false value",
			input: "false",
			expected: testResults{
				boolPtr: &boolFalse,
				err:     false,
			},
		},
		{
			desc:  "Input is the word 'true', expect a reference pointing to true value",
			input: "true",
			expected: testResults{
				boolPtr: &boolTrue,
				err:     false,
			},
		},
	}
	for _, tc := range testCases {
		resultingBoolPtr, resultingErr := parseBoolPtr(tc.input)

		assert.Equal(t, tc.expected.boolPtr, resultingBoolPtr, tc.desc)
		if tc.expected.err {
			assert.Error(t, resultingErr, tc.desc)
		} else {
			assert.NoError(t, resultingErr, tc.desc)
		}
	}
}

// TestPrivacyReader asserts the ReadPolicy scenarios
func TestPrivacyReader(t *testing.T) {
	int8Zero := int8(0)
	int8One := int8(1)
	boolTrue := true
	boolFalse := false

	type testInput struct {
		ampParams Params
	}
	type expectedResults struct {
		policyWriter privacy.PolicyWriter
		warning      error
	}
	type testCase struct {
		desc     string
		in       testInput
		expected expectedResults
	}

	testGroups := []struct {
		groupDesc string
		tests     []testCase
	}{
		{
			groupDesc: "No consent string",
			tests: []testCase{
				{
					desc:     "Params comes with an empty consent string, expect nil policy writer. No warning returned",
					expected: expectedResults{policyWriter: privacy.NilPolicyWriter{}, warning: nil},
				},
			},
		},
		{
			groupDesc: "TCF1",
			tests: []testCase{
				{
					desc: "Consent type TCF1: expect nil policy writer. Warning is returned",
					in: testInput{
						ampParams: Params{Consent: "VALID_TCF1_CONSENT", ConsentType: ConsentTCF1},
					},
					expected: expectedResults{
						policyWriter: privacy.NilPolicyWriter{},
						warning:      &errortypes.Warning{Message: "TCF1 consent is deprecated and no longer supported.", WarningCode: errortypes.InvalidPrivacyConsentWarningCode},
					},
				},
			},
		},
		{
			groupDesc: "ConsentNone. In order to be backwards compatible, we'll guess what consent string this is",
			tests: []testCase{
				{
					desc: "No consent type was specified and invalid consent string provided: expect nil policy writer and a warning",
					in: testInput{
						ampParams: Params{Consent: "NOT_CCPA_NOR_GDPR_TCF2"},
					},
					expected: expectedResults{
						policyWriter: privacy.NilPolicyWriter{},
						warning:      &errortypes.Warning{Message: "Consent string 'NOT_CCPA_NOR_GDPR_TCF2' is not recognized as one of the supported formats CCPA or TCF2.", WarningCode: errortypes.InvalidPrivacyConsentWarningCode},
					},
				},
				{
					desc: "No consent type specified but query params come with a valid CCPA consent string: expect a CCPA consent writer and no error nor warning",
					in: testInput{
						ampParams: Params{Consent: "1YYY"},
					},
					expected: expectedResults{
						policyWriter: ccpa.ConsentWriter{Consent: "1YYY"},
						warning:      nil,
					},
				},
				{
					desc: "No consent type, valid CCPA consent string and gdpr_applies set to true: expect a CCPA consent writer and a warning",
					in: testInput{
						ampParams: Params{
							Consent:     "1YYY",
							GdprApplies: &boolTrue,
						},
					},
					expected: expectedResults{
						policyWriter: ccpa.ConsentWriter{Consent: "1YYY"},
						warning: &errortypes.Warning{
							Message:     "AMP request gdpr_applies value was ignored because provided consent string is a CCPA consent string",
							WarningCode: errortypes.InvalidPrivacyConsentWarningCode,
						},
					},
				},
				{
					desc: "No consent type, valid GDPR consent string and gdpr_applies not set: expect a GDPR consent writer and no error nor warning",
					in: testInput{
						ampParams: Params{Consent: "CPdiPIJPdiPIJACABBENAzCv_____3___wAAAQNd_X9cAAAAAAAA"},
					},
					expected: expectedResults{
						policyWriter: gdpr.ConsentWriter{
							Consent: "CPdiPIJPdiPIJACABBENAzCv_____3___wAAAQNd_X9cAAAAAAAA",
							GDPR:    &int8One,
						},
						warning: nil,
					},
				},
			},
		},
		{
			groupDesc: "Unrecognized consent type. In order to be backwards compatible, we'll guess what consent string type it is",
			tests: []testCase{
				{
					desc: "Unrecognized consent type was specified and invalid consent string provided: expect nil policy writer and a warning",
					in: testInput{
						ampParams: Params{
							ConsentType: 101,
							Consent:     "NOT_CCPA_NOR_GDPR_TCF2",
						},
					},
					expected: expectedResults{
						policyWriter: privacy.NilPolicyWriter{},
						warning: &errortypes.Warning{
							Message:     "Consent string 'NOT_CCPA_NOR_GDPR_TCF2' is not recognized as one of the supported formats CCPA or TCF2.",
							WarningCode: errortypes.InvalidPrivacyConsentWarningCode,
						},
					},
				},
				{
					desc: "Unrecognized consent type specified but query params come with a valid CCPA consent string: expect a CCPA consent writer and no error nor warning",
					in: testInput{
						ampParams: Params{
							ConsentType: 101,
							Consent:     "1YYY",
						},
					},
					expected: expectedResults{
						policyWriter: ccpa.ConsentWriter{Consent: "1YYY"},
						warning:      nil,
					},
				},
				{
					desc: "Unrecognized consent type, valid CCPA consent string and gdpr_applies set to true: expect a CCPA consent writer and a warning",
					in: testInput{
						ampParams: Params{
							ConsentType: 101,
							Consent:     "1YYY",
							GdprApplies: &boolTrue,
						},
					},
					expected: expectedResults{
						policyWriter: ccpa.ConsentWriter{Consent: "1YYY"},
						warning: &errortypes.Warning{
							Message:     "AMP request gdpr_applies value was ignored because provided consent string is a CCPA consent string",
							WarningCode: errortypes.InvalidPrivacyConsentWarningCode,
						},
					},
				},
				{
					desc: "Unrecognized consent type, valid TCF2 consent string and gdpr_applies not set: expect GDPR consent writer and no error nor warning",
					in: testInput{
						ampParams: Params{
							ConsentType: 101,
							Consent:     "CPdiPIJPdiPIJACABBENAzCv_____3___wAAAQNd_X9cAAAAAAAA",
						},
					},
					expected: expectedResults{
						policyWriter: gdpr.ConsentWriter{
							Consent: "CPdiPIJPdiPIJACABBENAzCv_____3___wAAAQNd_X9cAAAAAAAA",
							GDPR:    &int8One,
						},
						warning: nil,
					},
				},
			},
		},
		{
			groupDesc: "consent type TCF2. Return a valid GDPR consent writer in all scenarios.",
			tests: []testCase{
				{
					desc: "GDPR consent string is invalid, but consent type is TCF2: return a valid GDPR writer and warn about the GDPR string being invalid",
					in: testInput{
						ampParams: Params{
							Consent:     "INVALID_GDPR",
							ConsentType: ConsentTCF2,
							GdprApplies: nil,
						},
					},
					expected: expectedResults{
						policyWriter: gdpr.ConsentWriter{
							Consent: "INVALID_GDPR",
							GDPR:    &int8One,
						},
						warning: &errortypes.Warning{
							Message:     "Consent string 'INVALID_GDPR' is not a valid TCF2 consent string.",
							WarningCode: errortypes.InvalidPrivacyConsentWarningCode,
						},
					},
				},
				{
					desc: "GDPR consent string is invalid, consent type is TCF2, gdpr_applies is set to true: return a valid GDPR writer and warn about the GDPR string being invalid",
					in: testInput{
						ampParams: Params{
							Consent:     "INVALID_GDPR",
							ConsentType: ConsentTCF2,
							GdprApplies: &boolFalse,
						},
					},
					expected: expectedResults{
						policyWriter: gdpr.ConsentWriter{
							Consent: "INVALID_GDPR",
							GDPR:    &int8Zero,
						},
						warning: &errortypes.Warning{
							Message:     "Consent string 'INVALID_GDPR' is not a valid TCF2 consent string.",
							WarningCode: errortypes.InvalidPrivacyConsentWarningCode,
						},
					},
				},
				{
					desc: "Valid GDPR consent string, gdpr_applies is set to false, return a valid GDPR writer, no warning",
					in: testInput{
						ampParams: Params{
							Consent:     "CPdiPIJPdiPIJACABBENAzCv_____3___wAAAQNd_X9cAAAAAAAA",
							ConsentType: ConsentTCF2,
							GdprApplies: &boolFalse,
						},
					},
					expected: expectedResults{
						policyWriter: gdpr.ConsentWriter{
							Consent: "CPdiPIJPdiPIJACABBENAzCv_____3___wAAAQNd_X9cAAAAAAAA",
							GDPR:    &int8Zero,
						},
						warning: nil,
					},
				},
				{
					desc: "Valid GDPR consent string, gdpr_applies is set to true, return a valid GDPR writer and no warning",
					in: testInput{
						ampParams: Params{
							Consent:     "CPdiPIJPdiPIJACABBENAzCv_____3___wAAAQNd_X9cAAAAAAAA",
							ConsentType: ConsentTCF2,
							GdprApplies: &boolTrue,
						},
					},
					expected: expectedResults{
						policyWriter: gdpr.ConsentWriter{
							Consent: "CPdiPIJPdiPIJACABBENAzCv_____3___wAAAQNd_X9cAAAAAAAA",
							GDPR:    &int8One,
						},
						warning: nil,
					},
				},
				{
					desc: "Valid GDPR consent string, return a valid GDPR writer and no warning",
					in: testInput{
						ampParams: Params{
							Consent:     "CPdiPIJPdiPIJACABBENAzCv_____3___wAAAQNd_X9cAAAAAAAA",
							ConsentType: ConsentTCF2,
						},
					},
					expected: expectedResults{
						policyWriter: gdpr.ConsentWriter{
							Consent: "CPdiPIJPdiPIJACABBENAzCv_____3___wAAAQNd_X9cAAAAAAAA",
							GDPR:    &int8One,
						},
						warning: nil,
					},
				},
			},
		},
		{
			groupDesc: "consent type CCPA. Return a valid CCPA consent writer in all scenarios.",
			tests: []testCase{
				{
					desc: "CCPA consent string is invalid: return a valid writer a warning about the string being invalid",
					in: testInput{
						ampParams: Params{
							Consent:     "XXXX",
							ConsentType: ConsentUSPrivacy,
						},
					},
					expected: expectedResults{
						policyWriter: ccpa.ConsentWriter{Consent: "XXXX"},
						warning: &errortypes.Warning{
							Message:     "Consent string 'XXXX' is not a valid CCPA consent string.",
							WarningCode: errortypes.InvalidPrivacyConsentWarningCode,
						},
					},
				},
				{
					desc: "Valid CCPA consent string, gdpr_applies is set to true: return a valid GDPR writer and warn about the gdpr_applies value.",
					in: testInput{
						ampParams: Params{
							Consent:     "1YYY",
							ConsentType: ConsentUSPrivacy,
							GdprApplies: &boolTrue,
						},
					},
					expected: expectedResults{
						policyWriter: ccpa.ConsentWriter{Consent: "1YYY"},
						warning: &errortypes.Warning{
							Message:     "AMP request gdpr_applies value was ignored because provided consent string is a CCPA consent string",
							WarningCode: errortypes.InvalidPrivacyConsentWarningCode,
						},
					},
				},
				{
					desc: "Valid CCPA consent string, return a valid GDPR writer and no warning",
					in: testInput{
						ampParams: Params{
							Consent:     "1YYY",
							ConsentType: ConsentUSPrivacy,
						},
					},
					expected: expectedResults{
						policyWriter: ccpa.ConsentWriter{Consent: "1YYY"},
						warning:      nil,
					},
				},
			},
		},
	}
	for _, group := range testGroups {
		for _, tc := range group.tests {
			actualPolicyWriter, actualErr := ReadPolicy(tc.in.ampParams, true)

			assert.Equal(t, tc.expected.policyWriter, actualPolicyWriter, tc.desc)
			assert.Equal(t, tc.expected.warning, actualErr, tc.desc)
		}
	}
}

func TestBuildGdprTCF2ConsentWriter(t *testing.T) {
	int8Zero := int8(0)
	int8One := int8(1)
	boolTrue := true
	boolFalse := false
	consentString := "CONSENT"

	testCases := []struct {
		desc           string
		inParams       Params
		expectedWriter gdpr.ConsentWriter
	}{
		{
			desc:     "gdpr_applies not set",
			inParams: Params{Consent: consentString},
			expectedWriter: gdpr.ConsentWriter{
				Consent: consentString,
				GDPR:    &int8One,
			},
		},
		{
			desc: "gdpr_applies set to false",
			inParams: Params{
				Consent:     consentString,
				GdprApplies: &boolFalse,
			},
			expectedWriter: gdpr.ConsentWriter{
				Consent: consentString,
				GDPR:    &int8Zero,
			},
		},
		{
			desc: "gdpr_applies set to true",
			inParams: Params{
				Consent:     consentString,
				GdprApplies: &boolTrue,
			},
			expectedWriter: gdpr.ConsentWriter{
				Consent: consentString,
				GDPR:    &int8One,
			},
		},
	}
	for _, tc := range testCases {
		actualPolicyWriter := buildGdprTCF2ConsentWriter(tc.inParams)
		assert.Equal(t, tc.expectedWriter, actualPolicyWriter, tc.desc)
	}
}

func TestParseMultisize(t *testing.T) {
	testCases := []struct {
		description     string
		multisize       string
		expectedFormats []openrtb2.Format
	}{
		{
			description:     "Empty",
			multisize:       "",
			expectedFormats: nil,
		},
		{
			description:     "One",
			multisize:       "1x2",
			expectedFormats: []openrtb2.Format{{W: 1, H: 2}},
		},
		{
			description:     "Many",
			multisize:       "1x2,3x4",
			expectedFormats: []openrtb2.Format{{W: 1, H: 2}, {W: 3, H: 4}},
		},
		{
			// Existing Behavior: The " 3" token in the second size is parsed as 0.
			description:     "Many With Space - Quirky Result",
			multisize:       "1x2, 3x4",
			expectedFormats: []openrtb2.Format{{W: 1, H: 2}, {W: 0, H: 4}},
		},
		{
			description:     "One - Zero Size - Ignored",
			multisize:       "0x0",
			expectedFormats: nil,
		},
		{
			description:     "Many - Zero Size - All Ignored",
			multisize:       "0x0,3x4",
			expectedFormats: nil,
		},
		{
			description:     "One - Extra Dimension - Ignored",
			multisize:       "1x2x3",
			expectedFormats: nil,
		},
		{
			description:     "Many - Extra Dimension - All Ignored",
			multisize:       "1x2x3,4x5",
			expectedFormats: nil,
		},
		{
			description:     "One - Invalid Values - Ignored",
			multisize:       "INVALIDxINVALID",
			expectedFormats: nil,
		},
		{
			description:     "Many - Invalid Values - All Ignored",
			multisize:       "1x2,INVALIDxINVALID",
			expectedFormats: nil,
		},
		{
			description:     "One - No Pair - Ignored",
			multisize:       "INVALID",
			expectedFormats: nil,
		},
		{
			description:     "Many - No Pair - All Ignored",
			multisize:       "1x2,INVALID",
			expectedFormats: nil,
		},
	}

	for _, test := range testCases {
		result := parseMultisize(test.multisize)
		assert.ElementsMatch(t, test.expectedFormats, result, test.description)
	}
}

func TestParseGdprApplies(t *testing.T) {
	gdprAppliesFalse := false
	gdprAppliesTrue := true

	testCases := []struct {
		desc              string
		inGdprApplies     *bool
		expectRegsExtGdpr int8
	}{
		{
			desc:              "gdprApplies was not set and defaulted to nil, expect 0",
			inGdprApplies:     nil,
			expectRegsExtGdpr: int8(0),
		},
		{
			desc:              "gdprApplies isn't nil and is set to false, expect a value of 0",
			inGdprApplies:     &gdprAppliesFalse,
			expectRegsExtGdpr: int8(0),
		},
		{
			desc:              "gdprApplies isn't nil and is set to true, expect a value of 1",
			inGdprApplies:     &gdprAppliesTrue,
			expectRegsExtGdpr: int8(1),
		},
	}
	for _, tc := range testCases {
		assert.Equal(t, tc.expectRegsExtGdpr, parseGdprApplies(tc.inGdprApplies), tc.desc)
	}
}
