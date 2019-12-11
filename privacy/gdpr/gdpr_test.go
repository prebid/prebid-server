package gdpr

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
			policy:      Policy{Consent: ""},
			request:     &openrtb.BidRequest{},
			expected:    &openrtb.BidRequest{},
		},
		{
			description: "Enabled With Nil Request User Object",
			policy:      Policy{Consent: "anyConsent"},
			request:     &openrtb.BidRequest{},
			expected: &openrtb.BidRequest{User: &openrtb.User{
				Ext: json.RawMessage(`{"consent":"anyConsent"}`)}},
		},
		{
			description: "Enabled With Nil Request User Ext Object",
			policy:      Policy{Consent: "anyConsent"},
			request:     &openrtb.BidRequest{User: &openrtb.User{}},
			expected: &openrtb.BidRequest{User: &openrtb.User{
				Ext: json.RawMessage(`{"consent":"anyConsent"}`)}},
		},
		{
			description: "Enabled With Existing Request User Ext Object - Doesn't Overwrite",
			policy:      Policy{Consent: "anyConsent"},
			request: &openrtb.BidRequest{User: &openrtb.User{
				Ext: json.RawMessage(`{"existing":"any"}`)}},
			expected: &openrtb.BidRequest{User: &openrtb.User{
				Ext: json.RawMessage(`{"existing":"any","consent":"anyConsent"}`)}},
		},
		{
			description: "Enabled With Existing Request User Ext Object - Overwrites",
			policy:      Policy{Consent: "anyConsent"},
			request: &openrtb.BidRequest{User: &openrtb.User{
				Ext: json.RawMessage(`{"existing":"any","consent":"toBeOverwritten"}`)}},
			expected: &openrtb.BidRequest{User: &openrtb.User{
				Ext: json.RawMessage(`{"existing":"any","consent":"anyConsent"}`)}},
		},
		{
			description: "Enabled With Existing Malformed Request User Ext Object",
			policy:      Policy{Consent: "anyConsent"},
			request: &openrtb.BidRequest{User: &openrtb.User{
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
