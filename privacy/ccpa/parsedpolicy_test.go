package ccpa

import (
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestValidateConsent(t *testing.T) {
	testCases := []struct {
		description string
		consent     string
		expected    bool
	}{
		{
			description: "Empty String",
			consent:     "",
			expected:    true,
		},
		{
			description: "Valid Consent With Opt Out",
			consent:     "1NYN",
			expected:    true,
		},
		{
			description: "Valid Consent Without Opt Out",
			consent:     "1NNN",
			expected:    true,
		},
		{
			description: "Invalid",
			consent:     "malformed",
			expected:    false,
		},
	}

	for _, test := range testCases {
		result := ValidateConsent(test.consent)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestParse(t *testing.T) {
	validBidders := map[string]struct{}{"a": {}}

	testCases := []struct {
		description    string
		consent        string
		noSaleBidders  []string
		expectedPolicy ParsedPolicy
		expectedError  string
	}{
		{
			description:    "Consent Error",
			consent:        "malformed",
			noSaleBidders:  []string{},
			expectedPolicy: ParsedPolicy{},
			expectedError:  "request.regs.ext.us_privacy must contain 4 characters",
		},
		{
			description:    "No Sale Error",
			consent:        "1NYN",
			noSaleBidders:  []string{"b"},
			expectedPolicy: ParsedPolicy{},
			expectedError:  "request.ext.prebid.nosale is invalid: unrecognized bidder 'b'",
		},
		{
			description:   "Success",
			consent:       "1NYN",
			noSaleBidders: []string{"a"},
			expectedPolicy: ParsedPolicy{
				consentSpecified:      true,
				consentOptOutSale:     true,
				noSaleForAllBidders:   false,
				noSaleSpecificBidders: map[string]struct{}{"a": {}},
			},
		},
	}

	for _, test := range testCases {
		policy := Policy{test.consent, test.noSaleBidders}

		result, err := policy.Parse(validBidders)

		if test.expectedError == "" {
			assert.NoError(t, err, test.description)
		} else {
			assert.EqualError(t, err, test.expectedError, test.description)
		}

		assert.Equal(t, test.expectedPolicy, result, test.description)
	}
}

func TestParseConsent(t *testing.T) {
	testCases := []struct {
		description    string
		consent        string
		expectedResult bool
		expectedError  string
	}{
		{
			description:    "Valid",
			consent:        "1NYN",
			expectedResult: true,
		},
		{
			description:    "Valid - Not Sale",
			consent:        "1NNN",
			expectedResult: false,
		},
		{
			description:    "Valid - Not Applicable",
			consent:        "1---",
			expectedResult: false,
		},
		{
			description:    "Valid - Empty",
			consent:        "",
			expectedResult: false,
		},
		{
			description:    "Wrong Length",
			consent:        "1NY",
			expectedResult: false,
			expectedError:  "must contain 4 characters",
		},
		{
			description:    "Wrong Version",
			consent:        "2---",
			expectedResult: false,
			expectedError:  "must specify version 1",
		},
		{
			description:    "Explicit Notice Char",
			consent:        "1X--",
			expectedResult: false,
			expectedError:  "must specify 'N', 'Y', or '-' for the explicit notice",
		},
		{
			description:    "Invalid Explicit Notice Case",
			consent:        "1y--",
			expectedResult: false,
			expectedError:  "must specify 'N', 'Y', or '-' for the explicit notice",
		},
		{
			description:    "Invalid Opt-Out Sale Char",
			consent:        "1-X-",
			expectedResult: false,
			expectedError:  "must specify 'N', 'Y', or '-' for the opt-out sale",
		},
		{
			description:    "Invalid Opt-Out Sale Case",
			consent:        "1-y-",
			expectedResult: false,
			expectedError:  "must specify 'N', 'Y', or '-' for the opt-out sale",
		},
		{
			description:    "Invalid LSPA Char",
			consent:        "1--X",
			expectedResult: false,
			expectedError:  "must specify 'N', 'Y', or '-' for the limited service provider agreement",
		},
		{
			description:    "Invalid LSPA Case",
			consent:        "1--y",
			expectedResult: false,
			expectedError:  "must specify 'N', 'Y', or '-' for the limited service provider agreement",
		},
	}

	for _, test := range testCases {
		result, err := parseConsent(test.consent)

		if test.expectedError == "" {
			assert.NoError(t, err, test.description)
		} else {
			assert.EqualError(t, err, test.expectedError, test.description)
		}

		assert.Equal(t, test.expectedResult, result, test.description)
	}
}

func TestParseNoSaleBidders(t *testing.T) {
	testCases := []struct {
		description                   string
		noSaleBidders                 []string
		validBidders                  []string
		expectedNoSaleForAllBidders   bool
		expectedNoSaleSpecificBidders map[string]struct{}
		expectedError                 string
	}{
		{
			description:                   "Valid - No Bidders",
			noSaleBidders:                 []string{},
			validBidders:                  []string{"a"},
			expectedNoSaleForAllBidders:   false,
			expectedNoSaleSpecificBidders: map[string]struct{}{},
		},
		{
			description:                   "Valid - 1 Bidder",
			noSaleBidders:                 []string{"a"},
			validBidders:                  []string{"a"},
			expectedNoSaleForAllBidders:   false,
			expectedNoSaleSpecificBidders: map[string]struct{}{"a": {}},
		},
		{
			description:                   "Valid - 1+ Bidders",
			noSaleBidders:                 []string{"a", "b"},
			validBidders:                  []string{"a", "b"},
			expectedNoSaleForAllBidders:   false,
			expectedNoSaleSpecificBidders: map[string]struct{}{"a": {}, "b": {}},
		},
		{
			description:                   "Valid - All Bidders",
			noSaleBidders:                 []string{"*"},
			validBidders:                  []string{"a"},
			expectedNoSaleForAllBidders:   true,
			expectedNoSaleSpecificBidders: map[string]struct{}{},
		},
		{
			description:                   "Bidder Not Valid",
			noSaleBidders:                 []string{"b"},
			validBidders:                  []string{"a"},
			expectedError:                 "unrecognized bidder 'b'",
			expectedNoSaleForAllBidders:   false,
			expectedNoSaleSpecificBidders: map[string]struct{}{},
		},
		{
			description:                   "All Bidder Mixed With Other Bidders Is Invalid",
			noSaleBidders:                 []string{"*", "a"},
			validBidders:                  []string{"a"},
			expectedError:                 "can only specify all bidders if no other bidders are provided",
			expectedNoSaleForAllBidders:   false,
			expectedNoSaleSpecificBidders: map[string]struct{}{},
		},
		{
			description:                   "Valid Bidders Case Sensitive",
			noSaleBidders:                 []string{"a"},
			validBidders:                  []string{"A"},
			expectedError:                 "unrecognized bidder 'a'",
			expectedNoSaleForAllBidders:   false,
			expectedNoSaleSpecificBidders: map[string]struct{}{},
		},
	}

	for _, test := range testCases {
		validBiddersMap := make(map[string]struct{})
		for _, v := range test.validBidders {
			validBiddersMap[v] = struct{}{}
		}

		resultNoSaleForAllBidders, resultNoSaleSpecificBidders, err := parseNoSaleBidders(test.noSaleBidders, validBiddersMap)

		if test.expectedError == "" {
			assert.NoError(t, err, test.description+":err")
		} else {
			assert.EqualError(t, err, test.expectedError, test.description+":err")
		}

		assert.Equal(t, test.expectedNoSaleForAllBidders, resultNoSaleForAllBidders, test.description+":allBidders")
		assert.Equal(t, test.expectedNoSaleSpecificBidders, resultNoSaleSpecificBidders, test.description+":specificBidders")
	}
}

func TestCanEnforce(t *testing.T) {
	testCases := []struct {
		description string
		policy      ParsedPolicy
		expected    bool
	}{
		{
			description: "Specified",
			policy: ParsedPolicy{
				consentSpecified:      true,
				consentOptOutSale:     false,
				noSaleForAllBidders:   false,
				noSaleSpecificBidders: map[string]struct{}{},
			},
			expected: true,
		},
		{
			description: "Not Specified",
			policy: ParsedPolicy{
				consentSpecified:      false,
				consentOptOutSale:     false,
				noSaleForAllBidders:   false,
				noSaleSpecificBidders: map[string]struct{}{},
			},
			expected: false,
		},
	}

	for _, test := range testCases {
		result := test.policy.CanEnforce()
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestShouldEnforce(t *testing.T) {
	testCases := []struct {
		description string
		policy      ParsedPolicy
		bidder      string
		expected    bool
	}{
		{
			description: "Not Enforced - All Bidders No Sale",
			policy: ParsedPolicy{
				consentSpecified:      true,
				consentOptOutSale:     true,
				noSaleForAllBidders:   true,
				noSaleSpecificBidders: map[string]struct{}{},
			},
			bidder:   "a",
			expected: false,
		},
		{
			description: "Not Enforced - Specific Bidders No Sale",
			policy: ParsedPolicy{
				consentSpecified:      true,
				consentOptOutSale:     true,
				noSaleForAllBidders:   false,
				noSaleSpecificBidders: map[string]struct{}{"a": {}},
			},
			bidder:   "a",
			expected: false,
		},
		{
			description: "Not Enforced - No Bidder No Sale",
			policy: ParsedPolicy{
				consentSpecified:      true,
				consentOptOutSale:     false,
				noSaleForAllBidders:   false,
				noSaleSpecificBidders: map[string]struct{}{},
			},
			bidder:   "a",
			expected: false,
		},
		{
			description: "Not Enforced - No Sale Case Sensitive",
			policy: ParsedPolicy{
				consentSpecified:      true,
				consentOptOutSale:     false,
				noSaleForAllBidders:   false,
				noSaleSpecificBidders: map[string]struct{}{"A": {}},
			},
			bidder:   "a",
			expected: false,
		},
		{
			description: "Enforced - No Bidder No Sale",
			policy: ParsedPolicy{
				consentSpecified:      true,
				consentOptOutSale:     true,
				noSaleForAllBidders:   false,
				noSaleSpecificBidders: map[string]struct{}{},
			},
			bidder:   "a",
			expected: true,
		},
		{
			description: "Enforced - No Sale Case Sensitive",
			policy: ParsedPolicy{
				consentSpecified:      true,
				consentOptOutSale:     true,
				noSaleForAllBidders:   false,
				noSaleSpecificBidders: map[string]struct{}{"A": {}},
			},
			bidder:   "a",
			expected: true,
		},
	}

	for _, test := range testCases {
		result := test.policy.ShouldEnforce(test.bidder)
		assert.Equal(t, test.expected, result, test.description)
	}
}

type mockPolicWriter struct {
	mock.Mock
}

func (m *mockPolicWriter) Write(req *openrtb2.BidRequest) error {
	args := m.Called(req)
	return args.Error(0)
}
