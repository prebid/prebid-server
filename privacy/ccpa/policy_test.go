package ccpa

import (
	"encoding/json"
	"errors"
	"testing"

	gpplib "github.com/prebid/go-gpp"
	gppConstants "github.com/prebid/go-gpp/constants"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestReadFromRequestWrapper(t *testing.T) {
	testCases := []struct {
		description    string
		request        *openrtb2.BidRequest
		giveGPP        gpplib.GppContainer
		expectedPolicy Policy
		expectedError  bool
	}{
		{
			description: "Success",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{USPrivacy: "ABC"},
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
			description: "Nil Ext",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{USPrivacy: "ABC"},
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
				Regs: &openrtb2.Regs{USPrivacy: "ABC"},
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
				Regs: &openrtb2.Regs{USPrivacy: "ABC"},
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
			description: "GPP Success",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{GPP: "DBACNYA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA~present",
					GPPSID: []int8{6}},
				Ext: json.RawMessage(`{"prebid":{"nosale":["a", "b"]}}`),
			},
			giveGPP: gpplib.GppContainer{Version: 1, SectionTypes: []gppConstants.SectionID{6}, Sections: []gpplib.Section{&upsv1Section}},
			expectedPolicy: Policy{
				Consent:       "gppContainerConsent",
				NoSaleBidders: []string{"a", "b"},
			},
		},
		{
			description: "GPP Success, has Regs.ext",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{GPP: "DBACNYA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA~present",
					GPPSID: []int8{6},
					Ext:    json.RawMessage(`{"us_privacy":"ABC"}`)},
				Ext: json.RawMessage(`{"prebid":{"nosale":["a", "b"]}}`),
			},
			giveGPP: gpplib.GppContainer{Version: 1, SectionTypes: []gppConstants.SectionID{6}, Sections: []gpplib.Section{&upsv1Section}},
			expectedPolicy: Policy{
				Consent:       "gppContainerConsent",
				NoSaleBidders: []string{"a", "b"},
			},
		},
		{
			description: "GPP Success, has regs.us_privacy",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{GPP: "DBACNYA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA~present",
					GPPSID:    []int8{6},
					USPrivacy: "conflicting"},
				Ext: json.RawMessage(`{"prebid":{"nosale":["a", "b"]}}`),
			},
			giveGPP: gpplib.GppContainer{Version: 1, SectionTypes: []gppConstants.SectionID{6}, Sections: []gpplib.Section{&upsv1Section}},
			expectedPolicy: Policy{
				Consent:       "gppContainerConsent",
				NoSaleBidders: []string{"a", "b"},
			},
			expectedError: true,
		},
		{
			description: "Has regs.us_privacy",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{USPrivacy: "present"},
				Ext:  json.RawMessage(`{"prebid":{"nosale":["a", "b"]}}`),
			},
			expectedPolicy: Policy{
				Consent:       "present",
				NoSaleBidders: []string{"a", "b"},
			},
			expectedError: false,
		},
		{
			description: "GPP Success, no USPV1",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{GPP: "DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
					GPPSID: []int8{6}},
			},
			giveGPP: gpplib.GppContainer{Version: 1, SectionTypes: []gppConstants.SectionID{2}, Sections: []gpplib.Section{&tcf1Section}},
			expectedPolicy: Policy{
				Consent: "",
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			reqWrapper := &openrtb_ext.RequestWrapper{BidRequest: test.request}
			result, err := ReadFromRequestWrapper(reqWrapper, test.giveGPP)
			assertError(t, test.expectedError, err, test.description)
			assert.Equal(t, test.expectedPolicy, result)
		})
	}
}

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
				Regs: &openrtb2.Regs{USPrivacy: "ABC"},
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
			description: "GPP Success",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{GPP: "DBACNYA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA~1YNN",
					GPPSID: []int8{6}},
				Ext: json.RawMessage(`{"prebid":{"nosale":["a", "b"]}}`),
			},
			expectedPolicy: Policy{
				Consent:       "1YNN",
				NoSaleBidders: []string{"a", "b"},
			},
		},
		{
			description: "GPP Success, has Regs.ext",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{GPP: "DBACNYA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA~1YNN",
					GPPSID: []int8{6},
					Ext:    json.RawMessage(`{"us_privacy":"ABC"}`)},
				Ext: json.RawMessage(`{"prebid":{"nosale":["a", "b"]}}`),
			},
			expectedPolicy: Policy{
				Consent:       "1YNN",
				NoSaleBidders: []string{"a", "b"},
			},
		},
		{
			description: "GPP Success, no USPV1",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{GPP: "DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
					GPPSID: []int8{6}}},
			expectedPolicy: Policy{
				Consent: "",
			},
		},
		{
			description: "GPP Success, no signal",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{GPP: "DBACNYA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA~1YNN",
					GPPSID: []int8{}}},
			expectedPolicy: Policy{
				Consent: "",
			},
		},
		{
			description: "GPP Success, wrong signal",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{GPP: "DBACNYA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA~1YNN",
					GPPSID: []int8{2}}},
			expectedPolicy: Policy{
				Consent: "",
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			result, err := ReadFromRequest(test.request)
			assertError(t, test.expectedError, err, test.description)
			assert.Equal(t, test.expectedPolicy, result)
		})
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
				Regs: &openrtb2.Regs{USPrivacy: "anyConsent"},
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
			expected:    nil,
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

func TestSelectCCPAConsent(t *testing.T) {
	type testInput struct {
		requestUSPrivacy string
		gpp              gpplib.GppContainer
		gppSIDs          []int8
	}
	testCases := []struct {
		desc         string
		in           testInput
		expectedCCPA string
		expectedErr  error
	}{
		{
			desc: "SectionUSPV1 in both GPP_SID and GPP container. Consent equal to request US_Privacy. Expect valid string and nil error",
			in: testInput{
				requestUSPrivacy: "gppContainerConsent",
				gpp:              gpplib.GppContainer{Version: 1, SectionTypes: []gppConstants.SectionID{gppConstants.SectionUSPV1}, Sections: []gpplib.Section{upsv1Section}},
				gppSIDs:          []int8{int8(6)},
			},
			expectedCCPA: "gppContainerConsent",
			expectedErr:  nil,
		},
		{
			desc: "No SectionUSPV1 in GPP_SID array expect request US_Privacy",
			in: testInput{
				requestUSPrivacy: "requestConsent",
				gpp:              gpplib.GppContainer{Version: 1, SectionTypes: []gppConstants.SectionID{gppConstants.SectionUSPV1}, Sections: []gpplib.Section{upsv1Section}},
				gppSIDs:          []int8{int8(2), int8(4)},
			},
			expectedCCPA: "requestConsent",
			expectedErr:  nil,
		},
		{
			desc: "No SectionUSPV1 in gpp.SectionTypes array expect request US_Privacy",
			in: testInput{
				requestUSPrivacy: "requestConsent",
				gpp:              gpplib.GppContainer{Version: 1, SectionTypes: []gppConstants.SectionID{}, Sections: []gpplib.Section{upsv1Section}},
				gppSIDs:          []int8{int8(6)},
			},
			expectedCCPA: "requestConsent",
			expectedErr:  nil,
		},
		{
			desc: "No SectionUSPV1 in GPP_SID array, blank request US_Privacy, expect blank consent",
			in: testInput{
				requestUSPrivacy: "",
				gpp:              gpplib.GppContainer{Version: 1, SectionTypes: []gppConstants.SectionID{gppConstants.SectionUSPV1}, Sections: []gpplib.Section{upsv1Section}},
				gppSIDs:          []int8{int8(2), int8(4)},
			},
			expectedCCPA: "",
			expectedErr:  nil,
		},
		{
			desc: "No SectionUSPV1 in gpp.SectionTypes array, blank request US_Privacy, expect blank consent",
			in: testInput{
				requestUSPrivacy: "",
				gpp:              gpplib.GppContainer{Version: 1, SectionTypes: []gppConstants.SectionID{}, Sections: []gpplib.Section{upsv1Section}},
				gppSIDs:          []int8{int8(6)},
			},
			expectedCCPA: "",
			expectedErr:  nil,
		},
		{
			desc: "SectionUSPV1 in both GPP_SID and GPP container. Consent equal to request US_Privacy. Expect valid string and nil error",
			in: testInput{
				requestUSPrivacy: "requestConsent",
				gpp:              gpplib.GppContainer{Version: 1, SectionTypes: []gppConstants.SectionID{gppConstants.SectionUSPV1}, Sections: []gpplib.Section{upsv1Section}},
				gppSIDs:          []int8{int8(6)},
			},
			expectedCCPA: "gppContainerConsent",
			expectedErr:  errors.New("request.us_privacy consent does not match uspv1"),
		},
	}
	for _, tc := range testCases {
		out, outErr := SelectCCPAConsent(tc.in.requestUSPrivacy, tc.in.gpp, tc.in.gppSIDs)

		assert.Equal(t, tc.expectedCCPA, out, tc.desc)
		assert.Equal(t, tc.expectedErr, outErr, tc.desc)
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

var upsv1Section mockGPPSection = mockGPPSection{sectionID: 6, value: "gppContainerConsent"}
var tcf1Section mockGPPSection = mockGPPSection{sectionID: 2, value: "BOS2bx5OS2bx5ABABBAAABoAAAAAFA"}

type mockGPPSection struct {
	sectionID gppConstants.SectionID
	value     string
}

func (ms mockGPPSection) GetID() gppConstants.SectionID {
	return ms.sectionID
}

func (ms mockGPPSection) GetValue() string {
	return ms.value
}

func (ms mockGPPSection) Encode(bool) []byte {
	return nil
}
