package gdpr

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestConsentWriter(t *testing.T) {
	testCases := []struct {
		description   string
		consent       string
		request       *openrtb2.BidRequest
		expected      *openrtb2.BidRequest
		expectedError bool
	}{
		{
			description: "Empty",
			consent:     "",
			request:     &openrtb2.BidRequest{},
			expected:    &openrtb2.BidRequest{},
		},
		{
			description: "Enabled With Nil Request User Object",
			consent:     "anyConsent",
			request:     &openrtb2.BidRequest{},
			expected: &openrtb2.BidRequest{User: &openrtb2.User{
				Consent: "anyConsent"}},
		},
		{
			description: "Enabled With Nil Request User Ext Object",
			consent:     "anyConsent",
			request:     &openrtb2.BidRequest{User: &openrtb2.User{}},
			expected: &openrtb2.BidRequest{User: &openrtb2.User{
				Consent: "anyConsent"}},
		},
		{
			description: "Enabled With Existing Request User Ext Object - Doesn't Overwrite",
			consent:     "anyConsent",
			request: &openrtb2.BidRequest{User: &openrtb2.User{
				Ext: json.RawMessage(`{"existing":"any"}`)}},
			expected: &openrtb2.BidRequest{User: &openrtb2.User{
				Consent: "anyConsent",
				Ext:     json.RawMessage(`{"existing":"any"}`)}},
		},
		{
			description: "Enabled With Existing Request User Ext Object - Overwrites",
			consent:     "anyConsent",
			request: &openrtb2.BidRequest{User: &openrtb2.User{
				Consent: "toBeOverwritten",
				Ext:     json.RawMessage(`{"existing":"any"}`)}},
			expected: &openrtb2.BidRequest{User: &openrtb2.User{
				Consent: "anyConsent",
				Ext:     json.RawMessage(`{"existing":"any"}`)}},
		},
		{
			description: "Injection Attack With Nil Request User Object",
			consent:     "BONV8oqONXwgmADACHENAO7pqzAAppY\"},\"oops\":\"malicious\",\"p\":{\"p\":\"",
			request:     &openrtb2.BidRequest{},
			expected: &openrtb2.BidRequest{User: &openrtb2.User{
				Consent: "BONV8oqONXwgmADACHENAO7pqzAAppY\"},\"oops\":\"malicious\",\"p\":{\"p\":\"",
			}},
		},
		{
			description: "Injection Attack With Nil Request User Ext Object",
			consent:     "BONV8oqONXwgmADACHENAO7pqzAAppY\"},\"oops\":\"malicious\",\"p\":{\"p\":\"",
			request:     &openrtb2.BidRequest{User: &openrtb2.User{}},
			expected: &openrtb2.BidRequest{User: &openrtb2.User{
				Consent: "BONV8oqONXwgmADACHENAO7pqzAAppY\"},\"oops\":\"malicious\",\"p\":{\"p\":\"",
				Ext:     nil,
			}},
		},
		{
			description: "Injection Attack With Existing Request User Ext Object",
			consent:     "BONV8oqONXwgmADACHENAO7pqzAAppY\"},\"oops\":\"malicious\",\"p\":{\"p\":\"",
			request: &openrtb2.BidRequest{User: &openrtb2.User{
				Ext: json.RawMessage(`{"existing":"any"}`),
			}},
			expected: &openrtb2.BidRequest{User: &openrtb2.User{
				Consent: "BONV8oqONXwgmADACHENAO7pqzAAppY\"},\"oops\":\"malicious\",\"p\":{\"p\":\"",
				Ext:     json.RawMessage(`{"existing":"any"}`),
			}},
		},
	}

	for _, test := range testCases {
		writer := ConsentWriter{test.consent, nil}
		err := writer.Write(test.request)

		if test.expectedError {
			assert.Error(t, err, test.description)
		} else {
			assert.NoError(t, err, test.description)
			assert.Equal(t, test.expected, test.request, test.description)
		}
	}
}
