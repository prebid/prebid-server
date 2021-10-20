package ccpa

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestReadFromRequest(t *testing.T) {
	testCases := []struct {
		description    string
		request        *openrtb2.BidRequest
		expectedPolicy Policy
		expectedError  bool
	}{
		{
			description: "Success",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"us_privacy":"ABC"}`)},
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
			request: &openrtb2.BidRequest{
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
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{},
				Ext:  json.RawMessage(`{"prebid":{"nosale":["a", "b"]}}`),
			},
			expectedPolicy: Policy{
				Consent:       "",
				NoSaleBidders: []string{"a", "b"},
			},
		},
		{
			description: "Empty Regs.Ext",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{Ext: json.RawMessage(`{}`)},
				Ext:  json.RawMessage(`{"prebid":{"nosale":["a", "b"]}}`),
			},
			expectedPolicy: Policy{
				Consent:       "",
				NoSaleBidders: []string{"a", "b"},
			},
		},
		{
			description: "Missing Regs.Ext USPrivacy Value",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"anythingElse":"42"}`)},
				Ext:  json.RawMessage(`{"prebid":{"nosale":["a", "b"]}}`),
			},
			expectedPolicy: Policy{
				Consent:       "",
				NoSaleBidders: []string{"a", "b"},
			},
		},
		{
			description: "Malformed Regs.Ext",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{Ext: json.RawMessage(`malformed`)},
				Ext:  json.RawMessage(`{"prebid":{"nosale":["a", "b"]}}`),
			},
			expectedError: true,
		},
		{
			description: "Invalid Regs.Ext Type",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"us_privacy":123`)},
				Ext:  json.RawMessage(`{"prebid":{"nosale":["a", "b"]}}`),
			},
			expectedError: true,
		},
		{
			description: "Nil Ext",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"us_privacy":"ABC"}`)},
				Ext:  nil,
			},
			expectedPolicy: Policy{
				Consent:       "ABC",
				NoSaleBidders: nil,
			},
		},
		{
			description: "Empty Ext",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"us_privacy":"ABC"}`)},
				Ext:  json.RawMessage(`{}`),
			},
			expectedPolicy: Policy{
				Consent:       "ABC",
				NoSaleBidders: nil,
			},
		},
		{
			description: "Missing Ext.Prebid No Sale Value",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"us_privacy":"ABC"}`)},
				Ext:  json.RawMessage(`{"anythingElse":"42"}`),
			},
			expectedPolicy: Policy{
				Consent:       "ABC",
				NoSaleBidders: nil,
			},
		},
		{
			description: "Malformed Ext",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"us_privacy":"ABC"}`)},
				Ext:  json.RawMessage(`malformed`),
			},
			expectedError: true,
		},
		{
			description: "Invalid Ext.Prebid.NoSale Type",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"us_privacy":"ABC"}`)},
				Ext:  json.RawMessage(`{"prebid":{"nosale":"wrongtype"}}`),
			},
			expectedError: true,
		},
		{
			description: "Injection Attack",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"us_privacy":"1YYY\"},\"oops\":\"malicious\",\"p\":{\"p\":\""}`)},
			},
			expectedPolicy: Policy{
				Consent: "1YYY\"},\"oops\":\"malicious\",\"p\":{\"p\":\"",
			},
		},
	}

	for _, test := range testCases {
		reqWrapper := &openrtb_ext.RequestWrapper{BidRequest: test.request}
		result, err := ReadFromRequestWrapper(reqWrapper)
		assertError(t, test.expectedError, err, test.description)
		assert.Equal(t, test.expectedPolicy, result, test.description)
	}
}

func TestWrite(t *testing.T) {
	testCases := []struct {
		description   string
		policy        Policy
		request       *openrtb2.BidRequest
		expected      *openrtb2.BidRequest
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
			request:     &openrtb2.BidRequest{},
			expected: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"us_privacy":"anyConsent"}`)},
				Ext:  json.RawMessage(`{"prebid":{"nosale":["a","b"]}}`),
			},
		},
		{
			description: "Error Regs.Ext - No Partial Update To Request",
			policy:      Policy{Consent: "anyConsent", NoSaleBidders: []string{"a", "b"}},
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{Ext: json.RawMessage(`malformed}`)},
			},
			expectedError: true,
			expected: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{Ext: json.RawMessage(`malformed}`)},
			},
		},
		{
			description: "Error Ext - No Partial Update To Request",
			policy:      Policy{Consent: "anyConsent", NoSaleBidders: []string{"a", "b"}},
			request: &openrtb2.BidRequest{
				Ext: json.RawMessage(`malformed}`),
			},
			expectedError: true,
			expected: &openrtb2.BidRequest{
				Ext: json.RawMessage(`malformed}`),
			},
		},
	}

	for _, test := range testCases {
		reqWrapper := &openrtb_ext.RequestWrapper{BidRequest: test.request}
		var err error
		_, err = reqWrapper.GetRegExt()
		if err == nil {
			_, err = reqWrapper.GetRequestExt()
			if err == nil {
				err = test.policy.Write(reqWrapper)
				if err == nil && reqWrapper.BidRequest != nil {
					err = reqWrapper.RebuildRequest()
				}
			}
		}
		assertError(t, test.expectedError, err, test.description)
		assert.Equal(t, test.expected, reqWrapper.BidRequest, test.description)
	}
}

func TestBuildRegs(t *testing.T) {
	testCases := []struct {
		description   string
		consent       string
		regs          *openrtb2.Regs
		expected      *openrtb2.Regs
		expectedError bool
	}{
		{
			description: "Clear",
			consent:     "",
			regs: &openrtb2.Regs{
				Ext: json.RawMessage(`{"us_privacy":"ABC"}`),
			},
			expected: &openrtb2.Regs{},
		},
		{
			description: "Clear - Error",
			consent:     "",
			regs: &openrtb2.Regs{
				Ext: json.RawMessage(`malformed`),
			},
			expected: &openrtb2.Regs{
				Ext: json.RawMessage(`malformed`),
			},
			expectedError: true,
		},
		{
			description: "Write",
			consent:     "anyConsent",
			regs:        nil,
			expected: &openrtb2.Regs{
				Ext: json.RawMessage(`{"us_privacy":"anyConsent"}`),
			},
		},
		{
			description: "Write - Error",
			consent:     "anyConsent",
			regs: &openrtb2.Regs{
				Ext: json.RawMessage(`malformed`),
			},
			expected: &openrtb2.Regs{
				Ext: json.RawMessage(`malformed`),
			},
			expectedError: true,
		},
	}

	for _, test := range testCases {
		request := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{Regs: test.regs}}
		regsExt, err := request.GetRegExt()
		if err == nil {
			regsExt.SetUSPrivacy(test.consent)
			request.RebuildRequest()
		}
		assertError(t, test.expectedError, err, test.description)
		assert.Equal(t, test.expected, request.Regs, test.description)
	}
}

func TestBuildRegsClear(t *testing.T) {
	testCases := []struct {
		description   string
		regs          *openrtb2.Regs
		expected      *openrtb2.Regs
		expectedError bool
	}{
		{
			description: "Nil Regs",
			regs:        nil,
			expected:    &openrtb2.Regs{Ext: nil},
		},
		{
			description: "Nil Regs.Ext",
			regs:        &openrtb2.Regs{Ext: nil},
			expected:    &openrtb2.Regs{Ext: nil},
		},
		{
			description: "Empty Regs.Ext",
			regs:        &openrtb2.Regs{Ext: json.RawMessage(`{}`)},
			expected:    &openrtb2.Regs{},
		},
		{
			description: "Removes Regs.Ext Entirely",
			regs:        &openrtb2.Regs{Ext: json.RawMessage(`{"us_privacy":"ABC"}`)},
			expected:    &openrtb2.Regs{},
		},
		{
			description: "Leaves Other Regs.Ext Values",
			regs:        &openrtb2.Regs{Ext: json.RawMessage(`{"us_privacy":"ABC", "other":"any"}`)},
			expected:    &openrtb2.Regs{Ext: json.RawMessage(`{"other":"any"}`)},
		},
		{
			description:   "Invalid Regs.Ext Type - Returns Error, doesn't clear",
			regs:          &openrtb2.Regs{Ext: json.RawMessage(`{"us_privacy":123}`)},
			expected:      &openrtb2.Regs{Ext: json.RawMessage(`{"us_privacy":123}`)},
			expectedError: true,
		},
		{
			description:   "Malformed Regs.Ext",
			regs:          &openrtb2.Regs{Ext: json.RawMessage(`malformed`)},
			expected:      &openrtb2.Regs{Ext: json.RawMessage(`malformed`)},
			expectedError: true,
		},
	}

	for _, test := range testCases {
		request := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{Regs: test.regs}}
		regsExt, err := request.GetRegExt()
		if err == nil {
			regsExt.SetUSPrivacy("")
			request.RebuildRequest()
		}
		assertError(t, test.expectedError, err, test.description)
		assert.Equal(t, test.expected, request.Regs, test.description)
	}
}

func TestBuildRegsWrite(t *testing.T) {
	testCases := []struct {
		description   string
		consent       string
		regs          *openrtb2.Regs
		expected      *openrtb2.Regs
		expectedError bool
	}{
		{
			description: "Nil Regs",
			consent:     "anyConsent",
			regs:        nil,
			expected:    &openrtb2.Regs{Ext: json.RawMessage(`{"us_privacy":"anyConsent"}`)},
		},
		{
			description: "Nil Regs.Ext",
			consent:     "anyConsent",
			regs:        &openrtb2.Regs{Ext: nil},
			expected:    &openrtb2.Regs{Ext: json.RawMessage(`{"us_privacy":"anyConsent"}`)},
		},
		{
			description: "Empty Regs.Ext",
			consent:     "anyConsent",
			regs:        &openrtb2.Regs{Ext: json.RawMessage(`{}`)},
			expected:    &openrtb2.Regs{Ext: json.RawMessage(`{"us_privacy":"anyConsent"}`)},
		},
		{
			description: "Overwrites Existing",
			consent:     "anyConsent",
			regs:        &openrtb2.Regs{Ext: json.RawMessage(`{"us_privacy":"ABC"}`)},
			expected:    &openrtb2.Regs{Ext: json.RawMessage(`{"us_privacy":"anyConsent"}`)},
		},
		{
			description: "Leaves Other Ext Values",
			consent:     "anyConsent",
			regs:        &openrtb2.Regs{Ext: json.RawMessage(`{"other":"any"}`)},
			expected:    &openrtb2.Regs{Ext: json.RawMessage(`{"other":"any","us_privacy":"anyConsent"}`)},
		},
		{
			description:   "Invalid Regs.Ext Type - Doesn't Overwrite",
			consent:       "anyConsent",
			regs:          &openrtb2.Regs{Ext: json.RawMessage(`{"us_privacy":123}`)},
			expected:      &openrtb2.Regs{Ext: json.RawMessage(`{"us_privacy":123}`)},
			expectedError: true,
		},
		{
			description:   "Malformed Regs.Ext",
			consent:       "anyConsent",
			regs:          &openrtb2.Regs{Ext: json.RawMessage(`malformed`)},
			expected:      &openrtb2.Regs{Ext: json.RawMessage(`malformed`)},
			expectedError: true,
		},
	}

	for _, test := range testCases {
		request := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{Regs: test.regs}}
		regsExt, err := request.GetRegExt()
		if err == nil {
			regsExt.SetUSPrivacy(test.consent)
			request.RebuildRequest()
		}
		assertError(t, test.expectedError, err, test.description)
		assert.Equal(t, test.expected, request.Regs, test.description)
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
		request := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{Ext: test.ext}}
		reqExt, err := request.GetRequestExt()
		var result json.RawMessage
		if err == nil {
			setPrebidNoSale(test.noSaleBidders, reqExt)
			err = request.RebuildRequest()
			result = request.Ext
		}
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
			ext:         json.RawMessage(`{"prebid":{"nosale":["a","b"],"aliases":{"a":"b"}}}`),
			expected:    json.RawMessage(`{"prebid":{"aliases":{"a":"b"}}}`),
		},
		{
			description: "Leaves All Other Values",
			ext:         json.RawMessage(`{"other":"ABC","prebid":{"nosale":["a","b"],"supportdeals":true}}`),
			expected:    json.RawMessage(`{"other":"ABC","prebid":{"supportdeals":true}}`),
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
		request := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{Ext: test.ext}}
		reqExt, err := request.GetRequestExt()
		var result json.RawMessage
		if err == nil {
			setPrebidNoSaleClear(reqExt)
			err = request.RebuildRequest()
			result = request.Ext
		}
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
			ext:           json.RawMessage(`{"prebid":{"supportdeals":true}}`),
			expected:      json.RawMessage(`{"prebid":{"supportdeals":true,"nosale":["a","b"]}}`),
		},
		{
			description:   "Leaves All Other Values",
			noSaleBidders: []string{"a", "b"},
			ext:           json.RawMessage(`{"other":"ABC","prebid":{"aliases":{"a":"b"}}}`),
			expected:      json.RawMessage(`{"other":"ABC","prebid":{"aliases":{"a":"b"},"nosale":["a","b"]}}`),
		},
		{
			description:   "Invalid Ext.Prebid No Sale Type - Still Overrides",
			noSaleBidders: []string{"a", "b"},
			ext:           json.RawMessage(`{"prebid":{"nosale":123}}`),
			expected:      json.RawMessage(`{"prebid":{"nosale":123}}`),
			expectedError: true,
		},
		{
			description:   "Invalid Ext.Prebid Type ",
			noSaleBidders: []string{"a", "b"},
			ext:           json.RawMessage(`{"prebid":"wrongtype"}`),
			expected:      json.RawMessage(`{"prebid":"wrongtype"}`),
			expectedError: true,
		},
		{
			description:   "Malformed Ext",
			noSaleBidders: []string{"a", "b"},
			ext:           json.RawMessage(`{malformed`),
			expected:      json.RawMessage(`{malformed`),
			expectedError: true,
		},
		{
			description:   "Malformed Ext.Prebid",
			noSaleBidders: []string{"a", "b"},
			ext:           json.RawMessage(`{"prebid":malformed}`),
			expected:      json.RawMessage(`{"prebid":malformed}`),
			expectedError: true,
		},
	}

	for _, test := range testCases {
		request := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{Ext: test.ext}}
		reqExt, err := request.GetRequestExt()
		var result json.RawMessage
		if err == nil {
			setPrebidNoSaleWrite(test.noSaleBidders, reqExt)
			err = request.RebuildRequest()
			result = request.Ext
		} else {
			result = test.ext
		}
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
