package ccpa

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/stretchr/testify/assert"
)

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
			policy:      Policy{Signal: ""},
			request:     &openrtb.BidRequest{},
			expected:    &openrtb.BidRequest{},
		},
		{
			description: "Enabled With Nil Request Regs Object",
			policy:      Policy{Signal: "anySignal"},
			request:     &openrtb.BidRequest{},
			expected: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"us_privacy":"anySignal"}`)}},
		},
		{
			description: "Enabled With Nil Request Regs Ext Object",
			policy:      Policy{Signal: "anySignal"},
			request:     &openrtb.BidRequest{Regs: &openrtb.Regs{}},
			expected: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"us_privacy":"anySignal"}`)}},
		},
		{
			description: "Enabled With Existing Request Regs Ext Object - Doesn't Overwrite",
			policy:      Policy{Signal: "anySignal"},
			request: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"existing":"any"}`)}},
			expected: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"existing":"any","us_privacy":"anySignal"}`)}},
		},
		{
			description: "Enabled With Existing Request Regs Ext Object - Overwrites",
			policy:      Policy{Signal: "anySignal"},
			request: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"existing":"any","us_privacy":"toBeOverwritten"}`)}},
			expected: &openrtb.BidRequest{Regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"existing":"any","us_privacy":"anySignal"}`)}},
		},
		{
			description: "Enabled With Existing Malformed Request Regs Ext Object",
			policy:      Policy{Signal: "anySignal"},
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
