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
			},
			expectedPolicy: Policy{
				Value: "ABC",
			},
		},
		{
			description: "Empty - No Request",
			request:     nil,
			expectedPolicy: Policy{
				Value: "",
			},
		},
		{
			description: "Empty - No Regs",
			request: &openrtb.BidRequest{
				Regs: nil,
			},
			expectedPolicy: Policy{
				Value: "",
			},
		},
		{
			description: "Empty - No Ext",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{},
			},
			expectedPolicy: Policy{
				Value: "",
			},
		},
		{
			description: "Empty - No Value",
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
			description: "Serialization Issue",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{
					Ext: json.RawMessage(`malformed`),
				},
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
			description: "Disabled",
			policy:      Policy{Value: ""},
			request:     &openrtb.BidRequest{},
			expected:    &openrtb.BidRequest{},
		},
		{
			description: "Disabled - Nil Request",
			policy:      Policy{Value: ""},
			request:     nil,
			expected:    nil,
		},
		{
			description: "Disabled - Empty Regs.Ext",
			policy:      Policy{Value: ""},
			request:     &openrtb.BidRequest{Regs: &openrtb.Regs{}},
			expected:    &openrtb.BidRequest{Regs: &openrtb.Regs{}},
		},
		{
			description: "Disabled - Remove From Request",
			policy:      Policy{Value: ""},
			request: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"us_privacy":"toBeRemoved"}`)}},
			expected: &openrtb.BidRequest{Regs: &openrtb.Regs{}},
		},
		{
			description: "Disabled - Remove From Request, Leave Other req Values",
			policy:      Policy{Value: ""},
			request: &openrtb.BidRequest{Regs: &openrtb.Regs{
				COPPA: 42,
				Ext:   json.RawMessage(`{"us_privacy":"toBeRemoved"}`)}},
			expected: &openrtb.BidRequest{Regs: &openrtb.Regs{
				COPPA: 42}},
		},
		{
			description: "Disabled - Remove From Request, Leave Other req.ext Values",
			policy:      Policy{Value: ""},
			request: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"existing":"any","us_privacy":"toBeRemoved"}`)}},
			expected: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"existing":"any"}`)}},
		},
		{
			description: "Enabled - Nil Request",
			policy:      Policy{Value: "anyValue"},
			request:     nil,
			expected:    nil,
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
		{
			description: "Injection Attack With Nil Request Regs Object",
			policy:      Policy{Value: "1YYY\"},\"oops\":\"malicious\",\"p\":{\"p\":\""},
			request:     &openrtb.BidRequest{},
			expected: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"us_privacy":"1YYY\"},\"oops\":\"malicious\",\"p\":{\"p\":\""}`),
			}},
		},
		{
			description: "Injection Attack With Nil Request Regs Ext Object",
			policy:      Policy{Value: "1YYY\"},\"oops\":\"malicious\",\"p\":{\"p\":\""},
			request:     &openrtb.BidRequest{Regs: &openrtb.Regs{}},
			expected: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"us_privacy":"1YYY\"},\"oops\":\"malicious\",\"p\":{\"p\":\""}`),
			}},
		},
		{
			description: "Injection Attack With Existing Request Regs Ext Object",
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
		err := test.policy.Write(test.request)

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
		policy      Policy
		expected    bool
	}{
		{
			description: "Enforceable",
			policy:      Policy{Value: "1-Y-"},
			expected:    true,
		},
		{
			description: "Not Enforceable - Not Present",
			policy:      Policy{Value: ""},
			expected:    false,
		},
		{
			description: "Not Enforceable - Opt-Out Unknown",
			policy:      Policy{Value: "1---"},
			expected:    false,
		},
		{
			description: "Not Enforceable - Opt-Out Explicitly No",
			policy:      Policy{Value: "1-N-"},
			expected:    false,
		},
		{
			description: "Invalid",
			policy:      Policy{Value: "2---"},
			expected:    false,
		},
	}

	for _, test := range testCases {
		result := test.policy.ShouldEnforce()
		assert.Equal(t, test.expected, result, test.description)
	}
}
