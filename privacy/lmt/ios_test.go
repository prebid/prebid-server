package lmt

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/util/iosutil"
	"github.com/stretchr/testify/assert"
)

// TestModifyForIOS is a simple spot check end-to-end test for the integration of all functional components.
func TestModifyForIOS(t *testing.T) {
	testCases := []struct {
		description  string
		givenRequest *openrtb2.BidRequest
		expectedLMT  *int8
	}{
		{
			description: "13.0",
			givenRequest: &openrtb2.BidRequest{
				App:    &openrtb2.App{},
				Device: &openrtb2.Device{OS: "iOS", OSV: "13.0", IFA: "", Lmt: nil},
			},
			expectedLMT: nil,
		},
		{
			description: "14.0",
			givenRequest: &openrtb2.BidRequest{
				App:    &openrtb2.App{},
				Device: &openrtb2.Device{OS: "iOS", OSV: "14.0", IFA: "", Lmt: nil},
			},
			expectedLMT: openrtb2.Int8Ptr(1),
		},
	}

	for _, test := range testCases {
		ModifyForIOS(test.givenRequest)
		assert.Equal(t, test.expectedLMT, test.givenRequest.Device.Lmt, test.description)
	}
}

func TestModifyForIOSHelper(t *testing.T) {
	testCases := []struct {
		description               string
		givenRequest              *openrtb2.BidRequest
		expectedModifier140Called bool
		expectedModifier142Called bool
	}{
		{
			description: "Valid 14.0",
			givenRequest: &openrtb2.BidRequest{
				App:    &openrtb2.App{},
				Device: &openrtb2.Device{OS: "iOS", OSV: "14.0"},
			},
			expectedModifier140Called: true,
			expectedModifier142Called: false,
		},
		{
			description: "Valid 14.2 Or Greater",
			givenRequest: &openrtb2.BidRequest{
				App:    &openrtb2.App{},
				Device: &openrtb2.Device{OS: "iOS", OSV: "14.2"},
			},
			expectedModifier140Called: false,
			expectedModifier142Called: true,
		},
		{
			description: "Invalid Version",
			givenRequest: &openrtb2.BidRequest{
				App:    &openrtb2.App{},
				Device: &openrtb2.Device{OS: "iOS", OSV: "invalid"},
			},
			expectedModifier140Called: false,
			expectedModifier142Called: false,
		},
		{
			description: "Invalid OS",
			givenRequest: &openrtb2.BidRequest{
				App:    &openrtb2.App{},
				Device: &openrtb2.Device{OS: "invalid", OSV: "14.0"},
			},
			expectedModifier140Called: false,
			expectedModifier142Called: false,
		},
		{
			description: "Invalid Nil Device",
			givenRequest: &openrtb2.BidRequest{
				App:    &openrtb2.App{},
				Device: nil,
			},
			expectedModifier140Called: false,
			expectedModifier142Called: false,
		},
	}

	for _, test := range testCases {
		modifierIOS140Called := false
		modifierIOS140 := func(req *openrtb2.BidRequest) { modifierIOS140Called = true }

		modifierIOS142Called := false
		modifierIOS142 := func(req *openrtb2.BidRequest) { modifierIOS142Called = true }

		modifiers := map[iosutil.VersionClassification]modifier{
			iosutil.Version140:          modifierIOS140,
			iosutil.Version142OrGreater: modifierIOS142,
		}

		modifyForIOS(test.givenRequest, modifiers)

		assert.Equal(t, test.expectedModifier140Called, modifierIOS140Called, test.description)
		assert.Equal(t, test.expectedModifier142Called, modifierIOS142Called, test.description)
	}
}

func TestIsRequestForIOS(t *testing.T) {
	testCases := []struct {
		description  string
		givenRequest *openrtb2.BidRequest
		expected     bool
	}{
		{
			description: "Valid",
			givenRequest: &openrtb2.BidRequest{
				App:    &openrtb2.App{},
				Device: &openrtb2.Device{OS: "iOS"},
			},
			expected: true,
		},
		{
			description: "Valid - OS Case Insensitive",
			givenRequest: &openrtb2.BidRequest{
				App:    &openrtb2.App{},
				Device: &openrtb2.Device{OS: "IOS"},
			},
			expected: true,
		},
		{
			description:  "Invalid - Nil Request",
			givenRequest: nil,
			expected:     false,
		},
		{
			description: "Invalid - Nil App",
			givenRequest: &openrtb2.BidRequest{
				App:    nil,
				Device: &openrtb2.Device{OS: "iOS"},
			},
			expected: false,
		},
		{
			description: "Invalid - Nil Device",
			givenRequest: &openrtb2.BidRequest{
				App:    &openrtb2.App{},
				Device: nil,
			},
			expected: false,
		},
		{
			description: "Invalid - Empty OS",
			givenRequest: &openrtb2.BidRequest{
				App:    &openrtb2.App{},
				Device: &openrtb2.Device{OS: ""},
			},
			expected: false,
		},
		{
			description: "Invalid - Wrong OS",
			givenRequest: &openrtb2.BidRequest{
				App:    &openrtb2.App{},
				Device: &openrtb2.Device{OS: "Android"},
			},
			expected: false,
		},
	}

	for _, test := range testCases {
		result := isRequestForIOS(test.givenRequest)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestModifyForIOS14X(t *testing.T) {
	testCases := []struct {
		description string
		givenDevice openrtb2.Device
		expectedLMT *int8
	}{
		{
			description: "IFA Empty",
			givenDevice: openrtb2.Device{IFA: "", Lmt: nil},
			expectedLMT: openrtb2.Int8Ptr(1),
		},
		{
			description: "IFA Zero UUID",
			givenDevice: openrtb2.Device{IFA: "00000000-0000-0000-0000-000000000000", Lmt: nil},
			expectedLMT: openrtb2.Int8Ptr(1),
		},
		{
			description: "IFA Populated",
			givenDevice: openrtb2.Device{IFA: "any-real-value", Lmt: nil},
			expectedLMT: openrtb2.Int8Ptr(0),
		},
		{
			description: "Overwrites Existing",
			givenDevice: openrtb2.Device{IFA: "", Lmt: openrtb2.Int8Ptr(0)},
			expectedLMT: openrtb2.Int8Ptr(1),
		},
	}

	for _, test := range testCases {
		request := &openrtb2.BidRequest{Device: &test.givenDevice}
		modifyForIOS14X(request)
		assert.Equal(t, test.expectedLMT, request.Device.Lmt, test.description)
	}
}

func TestModifyForIOS142OrGreater(t *testing.T) {
	testCases := []struct {
		description string
		givenDevice openrtb2.Device
		expectedLMT *int8
	}{
		{
			description: "Not Determined",
			givenDevice: openrtb2.Device{Ext: json.RawMessage(`{"atts":0}`), Lmt: nil},
			expectedLMT: openrtb2.Int8Ptr(0),
		},
		{
			description: "Restricted",
			givenDevice: openrtb2.Device{Ext: json.RawMessage(`{"atts":1}`), Lmt: nil},
			expectedLMT: openrtb2.Int8Ptr(1),
		},
		{
			description: "Denied",
			givenDevice: openrtb2.Device{Ext: json.RawMessage(`{"atts":2}`), Lmt: nil},
			expectedLMT: openrtb2.Int8Ptr(1),
		},
		{
			description: "Authorized",
			givenDevice: openrtb2.Device{Ext: json.RawMessage(`{"atts":3}`), Lmt: nil},
			expectedLMT: openrtb2.Int8Ptr(0),
		},
		{
			description: "Overwrites Existing",
			givenDevice: openrtb2.Device{Ext: json.RawMessage(`{"atts":3}`), Lmt: openrtb2.Int8Ptr(1)},
			expectedLMT: openrtb2.Int8Ptr(0),
		},
		{
			description: "Invalid Value - Unknown",
			givenDevice: openrtb2.Device{Ext: json.RawMessage(`{"atts":4}`), Lmt: nil},
			expectedLMT: nil,
		},
		{
			description: "Invalid Value - Unknown - Does Not Overwrite Existing",
			givenDevice: openrtb2.Device{Ext: json.RawMessage(`{"atts":4}`), Lmt: openrtb2.Int8Ptr(1)},
			expectedLMT: openrtb2.Int8Ptr(1),
		},
		{
			description: "Invalid Value - Missing",
			givenDevice: openrtb2.Device{Ext: json.RawMessage(`{}`), Lmt: nil},
			expectedLMT: nil,
		},
		{
			description: "Invalid Value - Wrong Type",
			givenDevice: openrtb2.Device{Ext: json.RawMessage(`{"atts":"wrong type"}`), Lmt: nil},
			expectedLMT: nil,
		},
	}

	for _, test := range testCases {
		request := &openrtb2.BidRequest{Device: &test.givenDevice}
		modifyForIOS142OrGreater(request)
		assert.Equal(t, test.expectedLMT, request.Device.Lmt, test.description)
	}
}
