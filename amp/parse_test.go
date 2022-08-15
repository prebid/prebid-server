package amp

import (
	"net/http"
	"testing"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
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
				"&slot=anySlot&timeout=42&h=1&w=2&oh=3&ow=4&ms=10x11,12x13&targeting=%7B%22gam-key1%22%3A%22val1%22%2C%22gam-key2%22%3A%22val2%22%7D",
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

func TestParseBoolPtr(t *testing.T) {
	boolTrue := true
	boolFalse := false

	testCases := []struct {
		desc     string
		input    string
		expected *bool
	}{
		{
			desc:     "Input is an empty string",
			input:    "",
			expected: nil,
		},
		{
			desc:     "Input is neither true nor false, expect a nil pointer",
			input:    "other",
			expected: nil,
		},
		{
			desc:     "Input is the word 'false', expect a reference pointing to false value",
			input:    "false",
			expected: &boolFalse,
		},
		{
			desc:     "Input is the word 'true', expect a reference pointing to true value",
			input:    "true",
			expected: &boolTrue,
		},
	}
	for _, tc := range testCases {
		actual := parseBoolPtr(tc.input)

		assert.Equal(t, tc.expected, actual, tc.desc)
	}
}

// TestPrivacyReader asserts the ReadPolicy scenarios
func TestPrivacyReader(t *testing.T) {

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
			groupDesc: "Consent type denied, unrecognized or TCF1, which is deprecated",
			tests: []testCase{
				{
					desc: "Consent type denied: expect nil policy writer. Warning is returned",
					in: testInput{
						ampParams: Params{Consent: "NOT_CCPA_NOR_GDPR_TCF2", ConsentType: ConsentDenied},
					},
					expected: expectedResults{
						policyWriter: privacy.NilPolicyWriter{},
						warning:      &errortypes.Warning{Message: "Consent denied. Consent string ignored.", WarningCode: errortypes.InvalidPrivacyConsentWarningCode},
					},
				},
				{
					desc: "Consent type TCF1: expect nil policy writer. Warning is returned",
					in: testInput{
						ampParams: Params{Consent: "NOT_CCPA_NOR_GDPR_TCF2", ConsentType: ConsentTCF1},
					},
					expected: expectedResults{
						policyWriter: privacy.NilPolicyWriter{},
						warning:      &errortypes.Warning{Message: "TCF1 consent is deprecated and no longer supported.", WarningCode: errortypes.InvalidPrivacyConsentWarningCode},
					},
				},
				{
					desc: "Consent type unknown: expect nil policy writer. Warning is returned",
					in: testInput{
						ampParams: Params{Consent: "NOT_CCPA_NOR_GDPR_TCF2", ConsentType: 101},
					},
					expected: expectedResults{
						policyWriter: privacy.NilPolicyWriter{},
						warning:      &errortypes.Warning{Message: "Consent '101' is not recognized as either CCPA or GDPR TCF2.", WarningCode: errortypes.InvalidPrivacyConsentWarningCode},
					},
				},
			},
		},
		{
			groupDesc: "consent type TCF2",
			tests: []testCase{
				{
					desc: "GDPR consent string is invalid, but consent type is TCF2: return a valid GDPR writer even and warn about the GDPR string being invalid",
					in: testInput{
						ampParams: Params{Consent: "INVALID_GDPR", ConsentType: ConsentTCF2},
					},
					expected: expectedResults{
						policyWriter: gdpr.ConsentWriter{"INVALID_GDPR"},
						warning:      &errortypes.Warning{Message: "Consent string 'INVALID_GDPR' is not a valid TCF2 consent string.", WarningCode: errortypes.InvalidPrivacyConsentWarningCode},
					},
				},
				{
					desc: "Valid GDPR consent string, return a valid GDPR writer and no warning",
					in: testInput{
						ampParams: Params{Consent: "CPdiPIJPdiPIJACABBENAzCv_____3___wAAAQNd_X9cAAAAAAAA", ConsentType: ConsentTCF2},
					},
					expected: expectedResults{
						policyWriter: gdpr.ConsentWriter{"CPdiPIJPdiPIJACABBENAzCv_____3___wAAAQNd_X9cAAAAAAAA"},
						warning:      nil,
					},
				},
			},
		},
		{
			groupDesc: "consent type CCPA",
			tests: []testCase{
				{
					desc: "CCPA consent string is invalid, but consent type is CCPA: return a nil writer a warning",
					in: testInput{
						ampParams: Params{Consent: "XXXX", ConsentType: ConsentUSPrivacy},
					},
					expected: expectedResults{
						policyWriter: privacy.NilPolicyWriter{},
						warning:      &errortypes.Warning{Message: "Consent string 'XXXX' is not a valid CCPA consent string.", WarningCode: errortypes.InvalidPrivacyConsentWarningCode},
					},
				},
				{
					desc: "Valid CCPA consent string, return a valid GDPR writer and no warning",
					in: testInput{
						ampParams: Params{Consent: "1YYY", ConsentType: ConsentUSPrivacy},
					},
					expected: expectedResults{
						policyWriter: ccpa.ConsentWriter{"1YYY"},
						warning:      nil,
					},
				},
			},
		},
	}
	for _, group := range testGroups {
		for _, tc := range group.tests {
			actualPolicyWriter, actualErr := ReadPolicy(tc.in.ampParams, nil, true)

			assert.Equal(t, tc.expected.policyWriter, actualPolicyWriter, tc.desc)
			assert.Equal(t, tc.expected.warning, actualErr, tc.desc)
		}
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
