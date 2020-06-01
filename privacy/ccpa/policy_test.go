package ccpa

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/stretchr/testify/assert"
)

func TestRead(t *testing.T) {
	testCases := []struct {
		description    string
		request        *openrtb.BidRequest
		expectedPolicy Policy
		expectedError  bool
	}{
		{
			description: "Success",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{
					Ext: json.RawMessage(`{"us_privacy":"ABC"}`),
				},
				Ext: json.RawMessage(`{"prebid":{"nosale":["a", "b"]}}`),
			},
			expectedPolicy: Policy{
				Value:         "ABC",
				NoSaleBidders: []string{"a", "b"},
			},
		},
		{
			description: "Nil Request",
			request:     nil,
			expectedPolicy: Policy{
				Value:         "",
				NoSaleBidders: nil,
			},
		},
		{
			description: "Nil Regs",
			request: &openrtb.BidRequest{
				Regs: nil,
				Ext:  json.RawMessage(`{"prebid":{"nosale":["a", "b"]}}`),
			},
			expectedPolicy: Policy{
				Value:         "",
				NoSaleBidders: []string{"a", "b"},
			},
		},
		{
			description: "Nil Regs.Ext",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{},
				Ext:  json.RawMessage(`{"prebid":{"nosale":["a", "b"]}}`),
			},
			expectedPolicy: Policy{
				Value:         "",
				NoSaleBidders: []string{"a", "b"},
			},
		},
		{
			description: "Empty Regs.Ext",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{
					Ext: json.RawMessage(`{}`),
				},
				Ext: json.RawMessage(`{"prebid":{"nosale":["a", "b"]}}`),
			},
			expectedPolicy: Policy{
				Value:         "",
				NoSaleBidders: []string{"a", "b"},
			},
		},
		{
			description: "Nil Regs.Ext Value",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{
					Ext: json.RawMessage(`{"anythingElse":"42"}`),
				},
				Ext: json.RawMessage(`{"prebid":{"nosale":["a", "b"]}}`),
			},
			expectedPolicy: Policy{
				Value:         "",
				NoSaleBidders: []string{"a", "b"},
			},
		},
		{
			description: "Nil Ext",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{
					Ext: json.RawMessage(`{"us_privacy":"ABC"}`),
				},
				Ext: nil,
			},
			expectedPolicy: Policy{
				Value:         "ABC",
				NoSaleBidders: nil,
			},
		},
		{
			description: "Empty Ext",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{
					Ext: json.RawMessage(`{"us_privacy":"ABC"}`),
				},
				Ext: json.RawMessage(`{}`),
			},
			expectedPolicy: Policy{
				Value:         "ABC",
				NoSaleBidders: nil,
			},
		},
		{
			description: "Nil Ext.Prebid.NoSale",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{
					Ext: json.RawMessage(`{"us_privacy":"ABC"}`),
				},
				Ext: json.RawMessage(`{"anythingElse":"42"}`),
			},
			expectedPolicy: Policy{
				Value:         "ABC",
				NoSaleBidders: nil,
			},
		},
		{
			description: "Malformed Regs.Ext",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{
					Ext: json.RawMessage(`malformed`),
				},
				Ext: json.RawMessage(`{"prebid":{"nosale":["a", "b"]}}`),
			},
			expectedError: true,
		},
		{
			description: "Malformed Ext",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{
					Ext: json.RawMessage(`{"us_privacy":"ABC"}`),
				},
				Ext: json.RawMessage(`malformed`),
			},
			expectedError: true,
		},
		{
			description: "Incorrect Ext.Prebid.NoSale Type",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{
					Ext: json.RawMessage(`{"us_privacy":"ABC"}`),
				},
				Ext: json.RawMessage(`{"prebid":{"nosale":"wrongtype"}}`),
			},
			expectedError: true,
		},
		{
			description: "Injection Attack",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{
					Ext: json.RawMessage(`{"us_privacy":"1YYY\"},\"oops\":\"malicious\",\"p\":{\"p\":\""}`),
				},
			},
			expectedPolicy: Policy{
				Value: "1YYY\"},\"oops\":\"malicious\",\"p\":{\"p\":\"",
			},
		},
	}

	for _, test := range testCases {

		p, e := ReadPolicy(test.request)

		if test.expectedError {
			assert.Error(t, e, test.description)
		} else {
			assert.NoError(t, e, test.description)
		}

		assert.Equal(t, test.expectedPolicy, p, test.description)
	}
}

func TestWrite(t *testing.T) {
	testCases := []struct {
		description   string
		policy        Policy
		request       *openrtb.BidRequest
		expected      *openrtb.BidRequest
		expectedError bool
	}{
		{
			description: "Success",
			policy:      Policy{Value: "anyValue", NoSaleBidders: []string{"a", "b"}},
			request:     &openrtb.BidRequest{},
			expected: &openrtb.BidRequest{
				Regs: &openrtb.Regs{
					Ext: json.RawMessage(`{"us_privacy":"anyValue"}`),
				},
				Ext: json.RawMessage(`{"prebid":{"nosale":["a","b"]}}`),
			},
		},
		{
			description: "Error Regs.Ext",
			policy:      Policy{Value: "anyValue", NoSaleBidders: []string{"a", "b"}},
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{
					Ext: json.RawMessage(`malformed}`),
				},
			},
			expectedError: true,
		},
		{
			description: "Error Ext",
			policy:      Policy{Value: "anyValue", NoSaleBidders: []string{"a", "b"}},
			request: &openrtb.BidRequest{
				Ext: json.RawMessage(`malformed}`),
			},
			expectedError: true,
		},
	}

	for _, test := range testCases {
		err := test.policy.Write(test.request)

		if test.expectedError {
			assert.Error(t, err, test.description)
		} else {
			assert.NoError(t, err, test.description)
			assert.Equal(t, test.expected, test.request, test.description)
		}
	}
}

func TestWriteRegsExt(t *testing.T) {
	testCases := []struct {
		description   string
		policy        Policy
		request       *openrtb.BidRequest
		expected      *openrtb.BidRequest
		expectedError bool
	}{
		{
			description: "Disabled",
			policy:      Policy{Value: ""},
			request:     &openrtb.BidRequest{},
			expected:    &openrtb.BidRequest{},
		},
		{
			description: "Enabled - Nil Regs",
			policy:      Policy{Value: "anyValue"},
			request:     &openrtb.BidRequest{},
			expected: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"us_privacy":"anyValue"}`)}},
		},
		{
			description: "Enabled - Nil Regs.Ext",
			policy:      Policy{Value: "anyValue"},
			request:     &openrtb.BidRequest{Regs: &openrtb.Regs{}},
			expected: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"us_privacy":"anyValue"}`)}},
		},
		{
			description: "Enabled - Existing Regs.Ext - Doesn't Overwrite",
			policy:      Policy{Value: "anyValue"},
			request: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"existing":"any"}`)}},
			expected: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"existing":"any","us_privacy":"anyValue"}`)}},
		},
		{
			description: "Enabled - Existing Regs.Ext.US_Privacy - Overwrites",
			policy:      Policy{Value: "anyValue"},
			request: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"existing":"any","us_privacy":"toBeOverwritten"}`)}},
			expected: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"existing":"any","us_privacy":"anyValue"}`)}},
		},
		{
			description: "Enabled -  Malformed Regs.Ext",
			policy:      Policy{Value: "anyValue"},
			request: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`malformed`)}},
			expectedError: true,
		},
		{
			description: "Injection Attack - Nil Regs",
			policy:      Policy{Value: "1YYY\"},\"oops\":\"malicious\",\"p\":{\"p\":\""},
			request:     &openrtb.BidRequest{},
			expected: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"us_privacy":"1YYY\"},\"oops\":\"malicious\",\"p\":{\"p\":\""}`),
			}},
		},
		{
			description: "Injection Attack - Nil Regs.Ext",
			policy:      Policy{Value: "1YYY\"},\"oops\":\"malicious\",\"p\":{\"p\":\""},
			request:     &openrtb.BidRequest{Regs: &openrtb.Regs{}},
			expected: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"us_privacy":"1YYY\"},\"oops\":\"malicious\",\"p\":{\"p\":\""}`),
			}},
		},
		{
			description: "Injection Attack - Existing Regs.Ext",
			policy:      Policy{Value: "1YYY\"},\"oops\":\"malicious\",\"p\":{\"p\":\""},
			request: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"existing":"any"}`),
			}},
			expected: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"existing":"any","us_privacy":"1YYY\"},\"oops\":\"malicious\",\"p\":{\"p\":\""}`),
			}},
		},
	}

	for _, test := range testCases {
		err := test.policy.writeRegsExt(test.request)

		if test.expectedError {
			assert.Error(t, err, test.description)
		} else {
			assert.NoError(t, err, test.description)
			assert.Equal(t, test.expected, test.request, test.description)
		}
	}
}

func TestWriteExt(t *testing.T) {
	testCases := []struct {
		description   string
		policy        Policy
		request       *openrtb.BidRequest
		expected      *openrtb.BidRequest
		expectedError bool
	}{
		{
			description: "Nil NoSaleBidders",
			policy:      Policy{NoSaleBidders: nil},
			request:     &openrtb.BidRequest{},
			expected:    &openrtb.BidRequest{},
		},
		{
			description: "Empty NoSaleBidders",
			policy:      Policy{NoSaleBidders: []string{}},
			request:     &openrtb.BidRequest{},
			expected:    &openrtb.BidRequest{},
		},
		{
			description: "Nil Ext",
			policy:      Policy{NoSaleBidders: []string{"a", "b"}},
			request: &openrtb.BidRequest{
				Ext: nil,
			},
			expected: &openrtb.BidRequest{
				Ext: json.RawMessage(`{"prebid":{"nosale":["a","b"]}}`),
			},
		},
		{
			description: "Empty Ext",
			policy:      Policy{NoSaleBidders: []string{"a", "b"}},
			request: &openrtb.BidRequest{
				Ext: json.RawMessage(`{}`),
			},
			expected: &openrtb.BidRequest{
				Ext: json.RawMessage(`{"prebid":{"nosale":["a","b"]}}`),
			},
		},
		{
			description: "Empty Ext.Prebid",
			policy:      Policy{NoSaleBidders: []string{"a", "b"}},
			request: &openrtb.BidRequest{
				Ext: json.RawMessage(`{"prebid":{}}`),
			},
			expected: &openrtb.BidRequest{
				Ext: json.RawMessage(`{"prebid":{"nosale":["a","b"]}}`),
			},
		},
		{
			description: "Existing Values In Ext",
			policy:      Policy{NoSaleBidders: []string{"a", "b"}},
			request: &openrtb.BidRequest{
				Ext: json.RawMessage(`{"existing":true,"prebid":{}}`),
			},
			expected: &openrtb.BidRequest{
				Ext: json.RawMessage(`{"existing":true,"prebid":{"nosale":["a","b"]}}`),
			},
		},
		{
			description: "Existing Values In Ext.Prebid",
			policy:      Policy{NoSaleBidders: []string{"a", "b"}},
			request: &openrtb.BidRequest{
				Ext: json.RawMessage(`{"prebid":{"existing":true}}`),
			},
			expected: &openrtb.BidRequest{
				Ext: json.RawMessage(`{"prebid":{"existing":true,"nosale":["a","b"]}}`),
			},
		},
		{
			description: "Overwrite Existing In Ext.Prebid",
			policy:      Policy{NoSaleBidders: []string{"a", "b"}},
			request: &openrtb.BidRequest{
				Ext: json.RawMessage(`{"prebid":{"nosale":["1","2"]}}`),
			},
			expected: &openrtb.BidRequest{
				Ext: json.RawMessage(`{"prebid":{"nosale":["a","b"]}}`),
			},
		},
		{
			description: "Malformed Ext",
			policy:      Policy{NoSaleBidders: []string{"a", "b"}},
			request: &openrtb.BidRequest{
				Ext: json.RawMessage(`malformed`),
			},
			expectedError: true,
		},
		{
			description: "Invalid Ext.Prebid Type",
			policy:      Policy{NoSaleBidders: []string{"a", "b"}},
			request: &openrtb.BidRequest{
				Ext: json.RawMessage(`{"prebid":42}`),
			},
			expectedError: true,
		},
	}

	for _, test := range testCases {
		err := test.policy.writeExt(test.request)

		if test.expectedError {
			assert.Error(t, err, test.description)
		} else {
			assert.NoError(t, err, test.description)
			assert.Equal(t, test.expected, test.request, test.description)
		}
	}
}

func TestValidate(t *testing.T) {
	testCases := []struct {
		description   string
		policy        Policy
		expectedError string
	}{
		{
			description:   "Valid",
			policy:        Policy{Value: "1NYN"},
			expectedError: "",
		},
		{
			description:   "Valid - Not Applicable",
			policy:        Policy{Value: "1---"},
			expectedError: "",
		},
		{
			description:   "Valid - Empty",
			policy:        Policy{Value: ""},
			expectedError: "",
		},
		{
			description:   "Invalid Length",
			policy:        Policy{Value: "1NY"},
			expectedError: "request.regs.ext.us_privacy must contain 4 characters",
		},
		{
			description:   "Invalid Version",
			policy:        Policy{Value: "2---"},
			expectedError: "request.regs.ext.us_privacy must specify version 1",
		},
		{
			description:   "Invalid Explicit Notice Char",
			policy:        Policy{Value: "1X--"},
			expectedError: "request.regs.ext.us_privacy must specify 'N', 'Y', or '-' for the explicit notice",
		},
		{
			description:   "Invalid Explicit Notice Case",
			policy:        Policy{Value: "1y--"},
			expectedError: "request.regs.ext.us_privacy must specify 'N', 'Y', or '-' for the explicit notice",
		},
		{
			description:   "Invalid Opt-Out Sale Char",
			policy:        Policy{Value: "1-X-"},
			expectedError: "request.regs.ext.us_privacy must specify 'N', 'Y', or '-' for the opt-out sale",
		},
		{
			description:   "Invalid Opt-Out Sale Case",
			policy:        Policy{Value: "1-y-"},
			expectedError: "request.regs.ext.us_privacy must specify 'N', 'Y', or '-' for the opt-out sale",
		},
		{
			description:   "Invalid LSPA Char",
			policy:        Policy{Value: "1--X"},
			expectedError: "request.regs.ext.us_privacy must specify 'N', 'Y', or '-' for the limited service provider agreement",
		},
		{
			description:   "Invalid LSPA Case",
			policy:        Policy{Value: "1--y"},
			expectedError: "request.regs.ext.us_privacy must specify 'N', 'Y', or '-' for the limited service provider agreement",
		},
	}

	for _, test := range testCases {
		result := test.policy.Validate()

		if test.expectedError == "" {
			assert.NoError(t, result, test.description)
		} else {
			assert.EqualError(t, result, test.expectedError, test.description)
		}
	}
}

func TestValidateConsent(t *testing.T) {
	testCases := []struct {
		description   string
		consent       string
		expectedError string
	}{
		{
			description:   "Valid",
			consent:       "1NYN",
			expectedError: "",
		},
		{
			description:   "Valid - Not Applicable",
			consent:       "1---",
			expectedError: "",
		},
		{
			description:   "Invalid Empty",
			consent:       "",
			expectedError: "",
		},
		{
			description:   "Invalid Length",
			consent:       "1NY",
			expectedError: "must contain 4 characters",
		},
		{
			description:   "Invalid Version",
			consent:       "2---",
			expectedError: "must specify version 1",
		},
		{
			description:   "Invalid Explicit Notice Char",
			consent:       "1X--",
			expectedError: "must specify 'N', 'Y', or '-' for the explicit notice",
		},
		{
			description:   "Invalid Explicit Notice Case",
			consent:       "1y--",
			expectedError: "must specify 'N', 'Y', or '-' for the explicit notice",
		},
		{
			description:   "Invalid Opt-Out Sale Char",
			consent:       "1-X-",
			expectedError: "must specify 'N', 'Y', or '-' for the opt-out sale",
		},
		{
			description:   "Invalid Opt-Out Sale Case",
			consent:       "1-y-",
			expectedError: "must specify 'N', 'Y', or '-' for the opt-out sale",
		},
		{
			description:   "Invalid LSPA Char",
			consent:       "1--X",
			expectedError: "must specify 'N', 'Y', or '-' for the limited service provider agreement",
		},
		{
			description:   "Invalid LSPA Case",
			consent:       "1--y",
			expectedError: "must specify 'N', 'Y', or '-' for the limited service provider agreement",
		},
	}

	for _, test := range testCases {
		result := ValidateConsent(test.consent)

		if test.expectedError == "" {
			assert.NoError(t, result, test.description)
		} else {
			assert.EqualError(t, result, test.expectedError, test.description)
		}
	}
}

func TestShouldEnforce(t *testing.T) {
	testCases := []struct {
		description string
		bidder      string
		policy      Policy
		expected    bool
	}{
		{
			description: "Enforceable",
			bidder:      "a",
			policy:      Policy{Value: "1-Y-"},
			expected:    true,
		},
		{
			description: "Enforceable - No Sale For Different Bidder",
			bidder:      "a",
			policy:      Policy{Value: "1-Y-", NoSaleBidders: []string{"b"}},
			expected:    true,
		},
		{
			description: "Not Enforceable - Not Present",
			bidder:      "a",
			policy:      Policy{Value: ""},
			expected:    false,
		},
		{
			description: "Not Enforceable - Opt-Out Unknown",
			bidder:      "a",
			policy:      Policy{Value: "1---"},
			expected:    false,
		},
		{
			description: "Not Enforceable - Opt-Out Explicitly No",
			bidder:      "a",
			policy:      Policy{Value: "1-N-"},
			expected:    false,
		},
		{
			description: "Not Enforceable - No Sale All Bidders",
			bidder:      "a",
			policy:      Policy{Value: "1-Y-", NoSaleBidders: []string{"*"}},
			expected:    false,
		},
		{
			description: "Not Enforceable - No Sale All Bidders Mixed With Specific Bidders",
			bidder:      "a",
			policy:      Policy{Value: "1-Y-", NoSaleBidders: []string{"b", "*", "c"}},
			expected:    false,
		},
		{
			description: "Not Enforceable - No Sale Specific Bidder",
			bidder:      "a",
			policy:      Policy{Value: "1-Y-", NoSaleBidders: []string{"a"}},
			expected:    false,
		},
		{
			description: "Not Enforceable - No Sale Specific Bidder Case Insensitive",
			bidder:      "a",
			policy:      Policy{Value: "1-Y-", NoSaleBidders: []string{"A"}},
			expected:    false,
		},
		{
			description: "Invalid",
			bidder:      "a",
			policy:      Policy{Value: "2---"},
			expected:    false,
		},
	}

	for _, test := range testCases {
		result := test.policy.ShouldEnforce(test.bidder)
		assert.Equal(t, test.expected, result, test.description)
	}
}
