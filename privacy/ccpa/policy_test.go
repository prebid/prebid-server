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
		description string
		policy      Policy
		expected    string
	}{
		{
			description: "Valid",
			policy:      Policy{Value: "1NYN"},
			expected:    "",
		},
		{
			description: "Valid - Not Applicable",
			policy:      Policy{Value: "1---"},
			expected:    "",
		},
		{
			description: "Valid - Empty",
			policy:      Policy{Value: ""},
			expected:    "",
		},
		{
			description: "Invalid Length",
			policy:      Policy{Value: "1NY"},
			expected:    "request.regs.ext.us_privacy must contain 4 characters",
		},
		{
			description: "Invalid Version",
			policy:      Policy{Value: "2---"},
			expected:    "request.regs.ext.us_privacy must specify version 1",
		},
		{
			description: "Invalid Explicit Notice Char",
			policy:      Policy{Value: "1X--"},
			expected:    "request.regs.ext.us_privacy must specify 'N', 'Y', or '-' for the explicit notice",
		},
		{
			description: "Invalid Explicit Notice Case",
			policy:      Policy{Value: "1y--"},
			expected:    "request.regs.ext.us_privacy must specify 'N', 'Y', or '-' for the explicit notice",
		},
		{
			description: "Invalid Opt-Out Sale Char",
			policy:      Policy{Value: "1-X-"},
			expected:    "request.regs.ext.us_privacy must specify 'N', 'Y', or '-' for the opt-out sale",
		},
		{
			description: "Invalid Opt-Out Sale Case",
			policy:      Policy{Value: "1-y-"},
			expected:    "request.regs.ext.us_privacy must specify 'N', 'Y', or '-' for the opt-out sale",
		},
		{
			description: "Invalid LSPA Char",
			policy:      Policy{Value: "1--X"},
			expected:    "request.regs.ext.us_privacy must specify 'N', 'Y', or '-' for the limited service provider agreement",
		},
		{
			description: "Invalid LSPA Case",
			policy:      Policy{Value: "1--y"},
			expected:    "request.regs.ext.us_privacy must specify 'N', 'Y', or '-' for the limited service provider agreement",
		},
	}

	for _, test := range testCases {
		result := test.policy.Validate()

		if test.expected == "" {
			assert.NoError(t, result, test.description)
		} else {
			assert.EqualError(t, result, test.expected, test.description)
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
