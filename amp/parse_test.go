package amp

import (
	"net/http"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
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
			query: "tag_id=anyTagID&account=anyAccount&curl=anyCurl&consent_string=anyConsent&debug=1&__amp_source_origin=anyOrigin" +
				"&slot=anySlot&timeout=42&h=1&w=2&oh=3&ow=4&ms=10x11,12x13",
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
