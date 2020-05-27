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
			description: "Empty - No Request",
			request:     nil,
			expectedPolicy: Policy{
				Value:         "",
				NoSaleBidders: nil,
			},
		},
		{
			description: "Empty - No Regs",
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
			description: "Empty - No Regs.Ext",
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
			description: "Empty - No Regs.Ext Value",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{
					Ext: json.RawMessage(`{"anythingElse":"42"}`),
				},
			},
			expectedPolicy: Policy{
				Value: "",
			},
		},
		{
			description: "Empty - No Ext",
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
			description: "Empty - No Ext Value",
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
			description: "Enabled With Nil Request Regs Object",
			policy:      Policy{Value: "anyValue"},
			request:     &openrtb.BidRequest{},
			expected: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"us_privacy":"anyValue"}`)}},
		},
		{
			description: "Enabled With Nil Request Regs Ext Object",
			policy:      Policy{Value: "anyValue"},
			request:     &openrtb.BidRequest{Regs: &openrtb.Regs{}},
			expected: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"us_privacy":"anyValue"}`)}},
		},
		{
			description: "Enabled With Existing Request Regs Ext Object - Doesn't Overwrite",
			policy:      Policy{Value: "anyValue"},
			request: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"existing":"any"}`)}},
			expected: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"existing":"any","us_privacy":"anyValue"}`)}},
		},
		{
			description: "Enabled With Existing Request Regs Ext Object - Overwrites",
			policy:      Policy{Value: "anyValue"},
			request: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"existing":"any","us_privacy":"toBeOverwritten"}`)}},
			expected: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"existing":"any","us_privacy":"anyValue"}`)}},
		},
		{
			description: "Enabled With Existing Malformed Request Regs Ext Object",
			policy:      Policy{Value: "anyValue"},
			request: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`malformed`)}},
			expectedError: true,
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
			description: "Nil",
			policy:      Policy{NoSaleBidders: nil},
			request:     &openrtb.BidRequest{},
			expected:    &openrtb.BidRequest{},
		},
		{
			description: "Empty",
			policy:      Policy{NoSaleBidders: []string{}},
			request:     &openrtb.BidRequest{},
			expected:    &openrtb.BidRequest{},
		},
		{
			description: "Values - Nil Ext",
			policy:      Policy{NoSaleBidders: []string{"a", "b"}},
			request: &openrtb.BidRequest{
				Ext: nil,
			},
			expected: &openrtb.BidRequest{
				Ext: json.RawMessage(`{"prebid":{"nosale":["a","b"]}}`),
			},
		},
		{
			description: "Values - Empty Prebid",
			policy:      Policy{NoSaleBidders: []string{"a", "b"}},
			request: &openrtb.BidRequest{
				Ext: json.RawMessage(`{"prebid":{}}`),
			},
			expected: &openrtb.BidRequest{
				Ext: json.RawMessage(`{"prebid":{"nosale":["a","b"]}}`),
			},
		},
		{
			description: "Values - Existing - Persists Other Values",
			policy:      Policy{NoSaleBidders: []string{"a", "b"}},
			request: &openrtb.BidRequest{
				Ext: json.RawMessage(`{"prebid":{"supportdeals":true}}`),
			},
			expected: &openrtb.BidRequest{
				Ext: json.RawMessage(`{"prebid":{"supportdeals":true,"nosale":["a","b"]}}`),
			},
		},
		{
			description: "Values - Existing - Overwrites Same Value",
			policy:      Policy{NoSaleBidders: []string{"a", "b"}},
			request: &openrtb.BidRequest{
				Ext: json.RawMessage(`{"prebid":{"nosale":["1","2"]}}`),
			},
			expected: &openrtb.BidRequest{
				Ext: json.RawMessage(`{"prebid":{"nosale":["a","b"]}}`),
			},
		},
		{
			description: "Values - Malformed",
			policy:      Policy{NoSaleBidders: []string{"a", "b"}},
			request: &openrtb.BidRequest{
				Ext: json.RawMessage(`malformed`),
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
