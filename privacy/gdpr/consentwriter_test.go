package gdpr

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/stretchr/testify/assert"
)

func TestConsentWriter(t *testing.T) {
	testCases := []struct {
		description   string
		consent       string
		request       *openrtb.BidRequest
		expected      *openrtb.BidRequest
		expectedError bool
	}{
		{
			description: "Empty",
			consent:     "",
			request:     &openrtb.BidRequest{},
			expected:    &openrtb.BidRequest{},
		},
		{
			description: "Enabled With Nil Request User Object",
			consent:     "anyConsent",
			request:     &openrtb.BidRequest{},
			expected: &openrtb.BidRequest{User: &openrtb.User{
				Ext: json.RawMessage(`{"consent":"anyConsent"}`)}},
		},
		{
			description: "Enabled With Nil Request User Ext Object",
			consent:     "anyConsent",
			request:     &openrtb.BidRequest{User: &openrtb.User{}},
			expected: &openrtb.BidRequest{User: &openrtb.User{
				Ext: json.RawMessage(`{"consent":"anyConsent"}`)}},
		},
		{
			description: "Enabled With Existing Request User Ext Object - Doesn't Overwrite",
			consent:     "anyConsent",
			request: &openrtb.BidRequest{User: &openrtb.User{
				Ext: json.RawMessage(`{"existing":"any"}`)}},
			expected: &openrtb.BidRequest{User: &openrtb.User{
				Ext: json.RawMessage(`{"consent":"anyConsent","existing":"any"}`)}},
		},
		{
			description: "Enabled With Existing Request User Ext Object - Overwrites",
			consent:     "anyConsent",
			request: &openrtb.BidRequest{User: &openrtb.User{
				Ext: json.RawMessage(`{"existing":"any","consent":"toBeOverwritten"}`)}},
			expected: &openrtb.BidRequest{User: &openrtb.User{
				Ext: json.RawMessage(`{"consent":"anyConsent","existing":"any"}`)}},
		},
		{
			description: "Enabled With Existing Malformed Request User Ext Object",
			consent:     "anyConsent",
			request: &openrtb.BidRequest{User: &openrtb.User{
				Ext: json.RawMessage(`malformed`)}},
			expectedError: true,
		},
		{
			description: "Injection Attack With Nil Request User Object",
			consent:     "BONV8oqONXwgmADACHENAO7pqzAAppY\"},\"oops\":\"malicious\",\"p\":{\"p\":\"",
			request:     &openrtb.BidRequest{},
			expected: &openrtb.BidRequest{User: &openrtb.User{
				Ext: json.RawMessage(`{"consent":"BONV8oqONXwgmADACHENAO7pqzAAppY\"},\"oops\":\"malicious\",\"p\":{\"p\":\""}`),
			}},
		},
		{
			description: "Injection Attack With Nil Request User Ext Object",
			consent:     "BONV8oqONXwgmADACHENAO7pqzAAppY\"},\"oops\":\"malicious\",\"p\":{\"p\":\"",
			request:     &openrtb.BidRequest{User: &openrtb.User{}},
			expected: &openrtb.BidRequest{User: &openrtb.User{
				Ext: json.RawMessage(`{"consent":"BONV8oqONXwgmADACHENAO7pqzAAppY\"},\"oops\":\"malicious\",\"p\":{\"p\":\""}`),
			}},
		},
		{
			description: "Injection Attack With Existing Request User Ext Object",
			consent:     "BONV8oqONXwgmADACHENAO7pqzAAppY\"},\"oops\":\"malicious\",\"p\":{\"p\":\"",
			request: &openrtb.BidRequest{User: &openrtb.User{
				Ext: json.RawMessage(`{"existing":"any"}`),
			}},
			expected: &openrtb.BidRequest{User: &openrtb.User{
				Ext: json.RawMessage(`{"consent":"BONV8oqONXwgmADACHENAO7pqzAAppY\"},\"oops\":\"malicious\",\"p\":{\"p\":\"","existing":"any"}`),
			}},
		},
	}

	for _, test := range testCases {
		writer := ConsentWriter{test.consent}
		err := writer.Write(test.request)

		if test.expectedError {
			assert.Error(t, err, test.description)
		} else {
			assert.NoError(t, err, test.description)
			assert.Equal(t, test.expected, test.request, test.description)
		}
	}
}
