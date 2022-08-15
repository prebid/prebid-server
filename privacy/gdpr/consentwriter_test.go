package gdpr

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
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
				Ext: json.RawMessage(`{"consent":"anyConsent"}`)}},
		},
		{
			description: "Enabled With Nil Request User Ext Object",
			consent:     "anyConsent",
			request:     &openrtb2.BidRequest{User: &openrtb2.User{}},
			expected: &openrtb2.BidRequest{User: &openrtb2.User{
				Ext: json.RawMessage(`{"consent":"anyConsent"}`)}},
		},
		{
			description: "Enabled With Existing Request User Ext Object - Doesn't Overwrite",
			consent:     "anyConsent",
			request: &openrtb2.BidRequest{User: &openrtb2.User{
				Ext: json.RawMessage(`{"existing":"any"}`)}},
			expected: &openrtb2.BidRequest{User: &openrtb2.User{
				Ext: json.RawMessage(`{"consent":"anyConsent","existing":"any"}`)}},
		},
		{
			description: "Enabled With Existing Request User Ext Object - Overwrites",
			consent:     "anyConsent",
			request: &openrtb2.BidRequest{User: &openrtb2.User{
				Ext: json.RawMessage(`{"existing":"any","consent":"toBeOverwritten"}`)}},
			expected: &openrtb2.BidRequest{User: &openrtb2.User{
				Ext: json.RawMessage(`{"consent":"anyConsent","existing":"any"}`)}},
		},
		{
			description: "Enabled With Existing Malformed Request User Ext Object",
			consent:     "anyConsent",
			request: &openrtb2.BidRequest{User: &openrtb2.User{
				Ext: json.RawMessage(`malformed`)}},
			expectedError: true,
		},
		{
			description: "Injection Attack With Nil Request User Object",
			consent:     "BONV8oqONXwgmADACHENAO7pqzAAppY\"},\"oops\":\"malicious\",\"p\":{\"p\":\"",
			request:     &openrtb2.BidRequest{},
			expected: &openrtb2.BidRequest{User: &openrtb2.User{
				Ext: json.RawMessage(`{"consent":"BONV8oqONXwgmADACHENAO7pqzAAppY\"},\"oops\":\"malicious\",\"p\":{\"p\":\""}`),
			}},
		},
		{
			description: "Injection Attack With Nil Request User Ext Object",
			consent:     "BONV8oqONXwgmADACHENAO7pqzAAppY\"},\"oops\":\"malicious\",\"p\":{\"p\":\"",
			request:     &openrtb2.BidRequest{User: &openrtb2.User{}},
			expected: &openrtb2.BidRequest{User: &openrtb2.User{
				Ext: json.RawMessage(`{"consent":"BONV8oqONXwgmADACHENAO7pqzAAppY\"},\"oops\":\"malicious\",\"p\":{\"p\":\""}`),
			}},
		},
		{
			description: "Injection Attack With Existing Request User Ext Object",
			consent:     "BONV8oqONXwgmADACHENAO7pqzAAppY\"},\"oops\":\"malicious\",\"p\":{\"p\":\"",
			request: &openrtb2.BidRequest{User: &openrtb2.User{
				Ext: json.RawMessage(`{"existing":"any"}`),
			}},
			expected: &openrtb2.BidRequest{User: &openrtb2.User{
				Ext: json.RawMessage(`{"consent":"BONV8oqONXwgmADACHENAO7pqzAAppY\"},\"oops\":\"malicious\",\"p\":{\"p\":\"","existing":"any"}`),
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

func TestSetRegExtGDPR(t *testing.T) {
	gdprAppliesFalse := false
	gdprAppliesTrue := true
	type testInput struct {
		gdprApplies *bool
		req         *openrtb2.BidRequest
	}
	testCases := []struct {
		desc            string
		in              testInput
		expectedErrMsg  string
		expectedRegsExt json.RawMessage
	}{
		{
			desc: "gdprApplies was not set and defaulted to nil. This isn't cause for an error",
			in: testInput{
				gdprApplies: nil,
			},
		},
		{
			desc: "gdprApplies isn't nil but the bidRequest is, expect RequestWrapper error",
			in: testInput{
				gdprApplies: &gdprAppliesFalse,
				req:         nil,
			},
			expectedErrMsg: "Requestwrapper Sync called on a nil BidRequest",
		},
		{
			desc: "gdprApplies was set but current req.regs.ext is malformed, expect error",
			in: testInput{
				gdprApplies: &gdprAppliesFalse,
				req:         &openrtb2.BidRequest{Regs: &openrtb2.Regs{Ext: json.RawMessage(`malformed`)}},
			},
			expectedErrMsg: "invalid character 'm' looking for beginning of value",
		},
		{
			desc: "gdprApplies equals false, expect req.Regs.Ext.GDPR to be 0",
			in: testInput{
				gdprApplies: &gdprAppliesFalse,
				req:         &openrtb2.BidRequest{},
			},
			expectedRegsExt: json.RawMessage(`{"gdpr":0}`),
		},
		{
			desc: "gdprApplies equals true, expect req.Regs.Ext.GDPR to be 1",
			in: testInput{
				gdprApplies: &gdprAppliesTrue,
				req:         &openrtb2.BidRequest{},
			},
			expectedRegsExt: json.RawMessage(`{"gdpr":1}`),
		},
	}
	for _, tc := range testCases {
		err := setRegExtGDPR(tc.in.gdprApplies, tc.in.req)

		if len(tc.expectedErrMsg) > 0 {
			assert.Error(t, err, tc.desc)
			assert.Equal(t, tc.expectedErrMsg, err.Error(), tc.desc)
		} else {
			assert.NoError(t, err, tc.desc)
		}

		if len(tc.expectedRegsExt) > 0 {
			assert.JSONEq(t, string(tc.expectedRegsExt), string(tc.in.req.Regs.Ext), tc.desc)
		}
	}
}
