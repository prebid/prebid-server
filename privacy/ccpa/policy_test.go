package ccpa

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/stretchr/testify/assert"
)

func TestReadFromRequest(t *testing.T) {
	testCases := []struct {
		description    string
		request        *openrtb.BidRequest
		expectedPolicy Policy
		expectedError  bool
	}{
		{
			description: "Success",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{Ext: json.RawMessage(`{"us_privacy":"ABC"}`)},
				Ext:  json.RawMessage(`{"prebid":{"nosale":["a", "b"]}}`),
			},
			expectedPolicy: Policy{
				Consent:       "ABC",
				NoSaleBidders: []string{"a", "b"},
			},
		},
		{
			description: "Nil Request",
			request:     nil,
			expectedPolicy: Policy{
				Consent:       "",
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
				Consent:       "",
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
				Consent:       "",
				NoSaleBidders: []string{"a", "b"},
			},
		},
		{
			description: "Empty Regs.Ext",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{Ext: json.RawMessage(`{}`)},
				Ext:  json.RawMessage(`{"prebid":{"nosale":["a", "b"]}}`),
			},
			expectedPolicy: Policy{
				Consent:       "",
				NoSaleBidders: []string{"a", "b"},
			},
		},
		{
			description: "Missing Regs.Ext USPrivacy Value",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{Ext: json.RawMessage(`{"anythingElse":"42"}`)},
				Ext:  json.RawMessage(`{"prebid":{"nosale":["a", "b"]}}`),
			},
			expectedPolicy: Policy{
				Consent:       "",
				NoSaleBidders: []string{"a", "b"},
			},
		},
		{
			description: "Malformed Regs.Ext",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{Ext: json.RawMessage(`malformed`)},
				Ext:  json.RawMessage(`{"prebid":{"nosale":["a", "b"]}}`),
			},
			expectedError: true,
		},
		{
			description: "Invalid Regs.Ext Type",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{Ext: json.RawMessage(`{"us_privacy":123`)},
				Ext:  json.RawMessage(`{"prebid":{"nosale":["a", "b"]}}`),
			},
			expectedError: true,
		},
		{
			description: "Nil Ext",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{Ext: json.RawMessage(`{"us_privacy":"ABC"}`)},
				Ext:  nil,
			},
			expectedPolicy: Policy{
				Consent:       "ABC",
				NoSaleBidders: nil,
			},
		},
		{
			description: "Empty Ext",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{Ext: json.RawMessage(`{"us_privacy":"ABC"}`)},
				Ext:  json.RawMessage(`{}`),
			},
			expectedPolicy: Policy{
				Consent:       "ABC",
				NoSaleBidders: nil,
			},
		},
		{
			description: "Missing Ext.Prebid No Sale Value",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{Ext: json.RawMessage(`{"us_privacy":"ABC"}`)},
				Ext:  json.RawMessage(`{"anythingElse":"42"}`),
			},
			expectedPolicy: Policy{
				Consent:       "ABC",
				NoSaleBidders: nil,
			},
		},
		{
			description: "Malformed Ext",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{Ext: json.RawMessage(`{"us_privacy":"ABC"}`)},
				Ext:  json.RawMessage(`malformed`),
			},
			expectedError: true,
		},
		{
			description: "Invalid Ext.Prebid.NoSale Type",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{Ext: json.RawMessage(`{"us_privacy":"ABC"}`)},
				Ext:  json.RawMessage(`{"prebid":{"nosale":"wrongtype"}}`),
			},
			expectedError: true,
		},
		{
			description: "Injection Attack",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{Ext: json.RawMessage(`{"us_privacy":"1YYY\"},\"oops\":\"malicious\",\"p\":{\"p\":\""}`)},
			},
			expectedPolicy: Policy{
				Consent: "1YYY\"},\"oops\":\"malicious\",\"p\":{\"p\":\"",
			},
		},
	}

	for _, test := range testCases {
		result, err := ReadFromRequest(test.request)
		assertError(t, test.expectedError, err, test.description)
		assert.Equal(t, test.expectedPolicy, result, test.description)
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
			description: "Nil Request",
			policy:      Policy{Consent: "anyConsent", NoSaleBidders: []string{"a", "b"}},
			request:     nil,
			expected:    nil,
		},
		{
			description: "Success",
			policy:      Policy{Consent: "anyConsent", NoSaleBidders: []string{"a", "b"}},
			request:     &openrtb.BidRequest{},
			expected: &openrtb.BidRequest{
				Regs: &openrtb.Regs{Ext: json.RawMessage(`{"us_privacy":"anyConsent"}`)},
				Ext:  json.RawMessage(`{"prebid":{"nosale":["a","b"]}}`),
			},
		},
		{
			description: "Error Regs.Ext - No Partial Update To Request",
			policy:      Policy{Consent: "anyConsent", NoSaleBidders: []string{"a", "b"}},
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{Ext: json.RawMessage(`malformed}`)},
			},
			expectedError: true,
			expected: &openrtb.BidRequest{
				Regs: &openrtb.Regs{Ext: json.RawMessage(`malformed}`)},
			},
		},
		{
			description: "Error Ext - No Partial Update To Request",
			policy:      Policy{Consent: "anyConsent", NoSaleBidders: []string{"a", "b"}},
			request: &openrtb.BidRequest{
				Ext: json.RawMessage(`malformed}`),
			},
			expectedError: true,
			expected: &openrtb.BidRequest{
				Ext: json.RawMessage(`malformed}`),
			},
		},
	}

	for _, test := range testCases {
		err := test.policy.Write(test.request)
		assertError(t, test.expectedError, err, test.description)
		assert.Equal(t, test.expected, test.request, test.description)
	}
}

func TestBuildRegs(t *testing.T) {
	testCases := []struct {
		description   string
		consent       string
		regs          *openrtb.Regs
		expected      *openrtb.Regs
		expectedError bool
	}{
		{
			description: "Clear",
			consent:     "",
			regs: &openrtb.Regs{
				Ext: json.RawMessage(`{"us_privacy":"ABC"}`),
			},
			expected: &openrtb.Regs{},
		},
		{
			description: "Clear - Error",
			consent:     "",
			regs: &openrtb.Regs{
				Ext: json.RawMessage(`malformed`),
			},
			expectedError: true,
		},
		{
			description: "Write",
			consent:     "anyConsent",
			regs:        nil,
			expected: &openrtb.Regs{
				Ext: json.RawMessage(`{"us_privacy":"anyConsent"}`),
			},
		},
		{
			description: "Write - Error",
			consent:     "anyConsent",
			regs: &openrtb.Regs{
				Ext: json.RawMessage(`malformed`),
			},
			expectedError: true,
		},
	}

	for _, test := range testCases {
		result, err := buildRegs(test.consent, test.regs)
		assertError(t, test.expectedError, err, test.description)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestBuildRegsClear(t *testing.T) {
	testCases := []struct {
		description   string
		regs          *openrtb.Regs
		expected      *openrtb.Regs
		expectedError bool
	}{
		{
			description: "Nil Regs",
			regs:        nil,
			expected:    nil,
		},
		{
			description: "Nil Regs.Ext",
			regs:        &openrtb.Regs{Ext: nil},
			expected:    &openrtb.Regs{Ext: nil},
		},
		{
			description: "Empty Regs.Ext",
			regs:        &openrtb.Regs{Ext: json.RawMessage(`{}`)},
			expected:    &openrtb.Regs{},
		},
		{
			description: "Removes Regs.Ext Entirely",
			regs:        &openrtb.Regs{Ext: json.RawMessage(`{"us_privacy":"ABC"}`)},
			expected:    &openrtb.Regs{},
		},
		{
			description: "Leaves Other Regs.Ext Values",
			regs:        &openrtb.Regs{Ext: json.RawMessage(`{"us_privacy":"ABC", "other":"any"}`)},
			expected:    &openrtb.Regs{Ext: json.RawMessage(`{"other":"any"}`)},
		},
		{
			description: "Invalid Regs.Ext Type - Still Cleared",
			regs:        &openrtb.Regs{Ext: json.RawMessage(`{"us_privacy":123}`)},
			expected:    &openrtb.Regs{},
		},
		{
			description:   "Malformed Regs.Ext",
			regs:          &openrtb.Regs{Ext: json.RawMessage(`malformed`)},
			expectedError: true,
		},
	}

	for _, test := range testCases {
		result, err := buildRegsClear(test.regs)
		assertError(t, test.expectedError, err, test.description)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestBuildRegsWrite(t *testing.T) {
	testCases := []struct {
		description   string
		consent       string
		regs          *openrtb.Regs
		expected      *openrtb.Regs
		expectedError bool
	}{
		{
			description: "Nil Regs",
			consent:     "anyConsent",
			regs:        nil,
			expected:    &openrtb.Regs{Ext: json.RawMessage(`{"us_privacy":"anyConsent"}`)},
		},
		{
			description: "Nil Regs.Ext",
			consent:     "anyConsent",
			regs:        &openrtb.Regs{Ext: nil},
			expected:    &openrtb.Regs{Ext: json.RawMessage(`{"us_privacy":"anyConsent"}`)},
		},
		{
			description: "Empty Regs.Ext",
			consent:     "anyConsent",
			regs:        &openrtb.Regs{Ext: json.RawMessage(`{}`)},
			expected:    &openrtb.Regs{Ext: json.RawMessage(`{"us_privacy":"anyConsent"}`)},
		},
		{
			description: "Overwrites Existing",
			consent:     "anyConsent",
			regs:        &openrtb.Regs{Ext: json.RawMessage(`{"us_privacy":"ABC"}`)},
			expected:    &openrtb.Regs{Ext: json.RawMessage(`{"us_privacy":"anyConsent"}`)},
		},
		{
			description: "Leaves Other Ext Values",
			consent:     "anyConsent",
			regs:        &openrtb.Regs{Ext: json.RawMessage(`{"other":"any"}`)},
			expected:    &openrtb.Regs{Ext: json.RawMessage(`{"other":"any","us_privacy":"anyConsent"}`)},
		},
		{
			description: "Invalid Regs.Ext Type - Still Overwrites",
			consent:     "anyConsent",
			regs:        &openrtb.Regs{Ext: json.RawMessage(`{"us_privacy":123}`)},
			expected:    &openrtb.Regs{Ext: json.RawMessage(`{"us_privacy":"anyConsent"}`)},
		},
		{
			description:   "Malformed Regs.Ext",
			consent:       "anyConsent",
			regs:          &openrtb.Regs{Ext: json.RawMessage(`malformed`)},
			expectedError: true,
		},
	}

	for _, test := range testCases {
		result, err := buildRegsWrite(test.consent, test.regs)
		assertError(t, test.expectedError, err, test.description)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestBuildExt(t *testing.T) {
	testCases := []struct {
		description   string
		noSaleBidders []string
		ext           json.RawMessage
		expected      json.RawMessage
		expectedError bool
	}{
		{
			description:   "Clear - Nil",
			noSaleBidders: nil,
			ext:           json.RawMessage(`{"prebid":{"nosale":["a", "b"]}}`),
			expected:      nil,
		},
		{
			description:   "Clear - Empty",
			noSaleBidders: []string{},
			ext:           json.RawMessage(`{"prebid":{"nosale":["a", "b"]}}`),
			expected:      nil,
		},
		{
			description:   "Clear - Error",
			noSaleBidders: []string{},
			ext:           json.RawMessage(`malformed`),
			expectedError: true,
		},
		{
			description:   "Write",
			noSaleBidders: []string{"a", "b"},
			ext:           nil,
			expected:      json.RawMessage(`{"prebid":{"nosale":["a","b"]}}`),
		},
		{
			description:   "Write - Error",
			noSaleBidders: []string{"a", "b"},
			ext:           json.RawMessage(`malformed`),
			expectedError: true,
		},
	}

	for _, test := range testCases {
		result, err := buildExt(test.noSaleBidders, test.ext)
		assertError(t, test.expectedError, err, test.description)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestBuildExtClear(t *testing.T) {
	testCases := []struct {
		description   string
		ext           json.RawMessage
		expected      json.RawMessage
		expectedError bool
	}{
		{
			description: "Nil Ext",
			ext:         nil,
			expected:    nil,
		},
		{
			description: "Empty Ext",
			ext:         json.RawMessage(``),
			expected:    json.RawMessage(``),
		},
		{
			description: "Empty Ext Object",
			ext:         json.RawMessage(`{}`),
			expected:    json.RawMessage(`{}`),
		},
		{
			description: "Empty Ext.Prebid",
			ext:         json.RawMessage(`{"prebid":{}}`),
			expected:    nil,
		},
		{
			description: "Removes Ext Entirely",
			ext:         json.RawMessage(`{"prebid":{"nosale":["a","b"]}}`),
			expected:    nil,
		},
		{
			description: "Leaves Other Ext Values",
			ext:         json.RawMessage(`{"other":"any","prebid":{"nosale":["a","b"]}}`),
			expected:    json.RawMessage(`{"other":"any"}`),
		},
		{
			description: "Leaves Other Ext.Prebid Values",
			ext:         json.RawMessage(`{"prebid":{"nosale":["a","b"],"other":"any"}}`),
			expected:    json.RawMessage(`{"prebid":{"other":"any"}}`),
		},
		{
			description: "Leaves All Other Values",
			ext:         json.RawMessage(`{"other":"ABC","prebid":{"nosale":["a","b"],"other":"123"}}`),
			expected:    json.RawMessage(`{"other":"ABC","prebid":{"other":"123"}}`),
		},
		{
			description:   "Malformed Ext",
			ext:           json.RawMessage(`malformed`),
			expectedError: true,
		},
		{
			description:   "Malformed Ext.Prebid",
			ext:           json.RawMessage(`{"prebid":malformed}`),
			expectedError: true,
		},
		{
			description:   "Invalid Ext.Prebid Type",
			ext:           json.RawMessage(`{"prebid":123}`),
			expectedError: true,
		},
	}

	for _, test := range testCases {
		result, err := buildExtClear(test.ext)
		assertError(t, test.expectedError, err, test.description)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestBuildExtWrite(t *testing.T) {
	testCases := []struct {
		description   string
		noSaleBidders []string
		ext           json.RawMessage
		expected      json.RawMessage
		expectedError bool
	}{
		{
			description:   "Nil Ext",
			noSaleBidders: []string{"a", "b"},
			ext:           nil,
			expected:      json.RawMessage(`{"prebid":{"nosale":["a","b"]}}`),
		},
		{
			description:   "Empty Ext",
			noSaleBidders: []string{"a", "b"},
			ext:           json.RawMessage(``),
			expected:      json.RawMessage(`{"prebid":{"nosale":["a","b"]}}`),
		},
		{
			description:   "Empty Ext Object",
			noSaleBidders: []string{"a", "b"},
			ext:           json.RawMessage(`{}`),
			expected:      json.RawMessage(`{"prebid":{"nosale":["a","b"]}}`),
		},
		{
			description:   "Empty Ext.Prebid",
			noSaleBidders: []string{"a", "b"},
			ext:           json.RawMessage(`{"prebid":{}}`),
			expected:      json.RawMessage(`{"prebid":{"nosale":["a","b"]}}`),
		},
		{
			description:   "Overwrites Existing",
			noSaleBidders: []string{"a", "b"},
			ext:           json.RawMessage(`{"prebid":{"nosale":["x","y"]}}`),
			expected:      json.RawMessage(`{"prebid":{"nosale":["a","b"]}}`),
		},
		{
			description:   "Leaves Other Ext Values",
			noSaleBidders: []string{"a", "b"},
			ext:           json.RawMessage(`{"other":"any"}`),
			expected:      json.RawMessage(`{"other":"any","prebid":{"nosale":["a","b"]}}`),
		},
		{
			description:   "Leaves Other Ext.Prebid Values",
			noSaleBidders: []string{"a", "b"},
			ext:           json.RawMessage(`{"prebid":{"other":"any"}}`),
			expected:      json.RawMessage(`{"prebid":{"nosale":["a","b"],"other":"any"}}`),
		},
		{
			description:   "Leaves All Other Values",
			noSaleBidders: []string{"a", "b"},
			ext:           json.RawMessage(`{"other":"ABC","prebid":{"other":"123"}}`),
			expected:      json.RawMessage(`{"other":"ABC","prebid":{"nosale":["a","b"],"other":"123"}}`),
		},
		{
			description:   "Invalid Ext.Prebid No Sale Type - Still Overrides",
			noSaleBidders: []string{"a", "b"},
			ext:           json.RawMessage(`{"prebid":{"nosale":123}}`),
			expected:      json.RawMessage(`{"prebid":{"nosale":["a","b"]}}`),
		},
		{
			description:   "Invalid Ext.Prebid Type ",
			noSaleBidders: []string{"a", "b"},
			ext:           json.RawMessage(`{"prebid":"wrongtype"}`),
			expectedError: true,
		},
		{
			description:   "Malformed Ext",
			noSaleBidders: []string{"a", "b"},
			ext:           json.RawMessage(`{malformed`),
			expectedError: true,
		},
		{
			description:   "Malformed Ext.Prebid",
			noSaleBidders: []string{"a", "b"},
			ext:           json.RawMessage(`{"prebid":malformed}`),
			expectedError: true,
		},
	}

	for _, test := range testCases {
		result, err := buildExtWrite(test.noSaleBidders, test.ext)
		assertError(t, test.expectedError, err, test.description)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func assertError(t *testing.T, expectError bool, err error, description string) {
	t.Helper()
	if expectError {
		assert.Error(t, err, description)
	} else {
		assert.NoError(t, err, description)
	}
}
