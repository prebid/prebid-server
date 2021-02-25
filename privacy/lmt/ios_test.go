package lmt

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/util/iosutil"
	"github.com/stretchr/testify/assert"
)

// TestModifyForIOS is a simple spot check end-to-end test for the integration of all functional components.
func TestModifyForIOS(t *testing.T) {
	testCases := []struct {
		description  string
		givenRequest *openrtb.BidRequest
		expectedLMT  *int8
	}{
		{
			description: "13.0",
			givenRequest: &openrtb.BidRequest{
				App:    &openrtb.App{},
				Device: &openrtb.Device{OS: "iOS", OSV: "13.0", IFA: "", Lmt: nil},
			},
			expectedLMT: nil,
		},
		{
			description: "14.0",
			givenRequest: &openrtb.BidRequest{
				App:    &openrtb.App{},
				Device: &openrtb.Device{OS: "iOS", OSV: "14.0", IFA: "", Lmt: nil},
			},
			expectedLMT: openrtb.Int8Ptr(1),
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
		givenRequest              *openrtb.BidRequest
		expectedModifier140Called bool
		expectedModifier142Called bool
	}{
		{
			description: "Valid 14.0",
			givenRequest: &openrtb.BidRequest{
				App:    &openrtb.App{},
				Device: &openrtb.Device{OS: "iOS", OSV: "14.0"},
			},
			expectedModifier140Called: true,
			expectedModifier142Called: false,
		},
		{
			description: "Valid 14.2 Or Greater",
			givenRequest: &openrtb.BidRequest{
				App:    &openrtb.App{},
				Device: &openrtb.Device{OS: "iOS", OSV: "14.2"},
			},
			expectedModifier140Called: false,
			expectedModifier142Called: true,
		},
		{
			description: "Invalid Version",
			givenRequest: &openrtb.BidRequest{
				App:    &openrtb.App{},
				Device: &openrtb.Device{OS: "iOS", OSV: "invalid"},
			},
			expectedModifier140Called: false,
			expectedModifier142Called: false,
		},
		{
			description: "Invalid OS",
			givenRequest: &openrtb.BidRequest{
				App:    &openrtb.App{},
				Device: &openrtb.Device{OS: "invalid", OSV: "14.0"},
			},
			expectedModifier140Called: false,
			expectedModifier142Called: false,
		},
		{
			description: "Invalid Nil Device",
			givenRequest: &openrtb.BidRequest{
				App:    &openrtb.App{},
				Device: nil,
			},
			expectedModifier140Called: false,
			expectedModifier142Called: false,
		},
	}

	for _, test := range testCases {
		modifierIOS140Called := false
		modifierIOS140 := func(req *openrtb.BidRequest) { modifierIOS140Called = true }

		modifierIOS142Called := false
		modifierIOS142 := func(req *openrtb.BidRequest) { modifierIOS142Called = true }

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
		givenRequest *openrtb.BidRequest
		expected     bool
	}{
		{
			description: "Valid",
			givenRequest: &openrtb.BidRequest{
				App:    &openrtb.App{},
				Device: &openrtb.Device{OS: "iOS"},
			},
			expected: true,
		},
		{
			description: "Valid - OS Case Insensitive",
			givenRequest: &openrtb.BidRequest{
				App:    &openrtb.App{},
				Device: &openrtb.Device{OS: "IOS"},
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
			givenRequest: &openrtb.BidRequest{
				App:    nil,
				Device: &openrtb.Device{OS: "iOS"},
			},
			expected: false,
		},
		{
			description: "Invalid - Nil Device",
			givenRequest: &openrtb.BidRequest{
				App:    &openrtb.App{},
				Device: nil,
			},
			expected: false,
		},
		{
			description: "Invalid - Empty OS",
			givenRequest: &openrtb.BidRequest{
				App:    &openrtb.App{},
				Device: &openrtb.Device{OS: ""},
			},
			expected: false,
		},
		{
			description: "Invalid - Wrong OS",
			givenRequest: &openrtb.BidRequest{
				App:    &openrtb.App{},
				Device: &openrtb.Device{OS: "Android"},
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
		givenDevice openrtb.Device
		expectedLMT *int8
	}{
		{
			description: "IFA Empty",
			givenDevice: openrtb.Device{IFA: "", Lmt: nil},
			expectedLMT: openrtb.Int8Ptr(1),
		},
		{
			description: "IFA Zero UUID",
			givenDevice: openrtb.Device{IFA: "00000000-0000-0000-0000-000000000000", Lmt: nil},
			expectedLMT: openrtb.Int8Ptr(1),
		},
		{
			description: "IFA Populated",
			givenDevice: openrtb.Device{IFA: "any-real-value", Lmt: nil},
			expectedLMT: openrtb.Int8Ptr(0),
		},
		{
			description: "Overwrites Existing",
			givenDevice: openrtb.Device{IFA: "", Lmt: openrtb.Int8Ptr(0)},
			expectedLMT: openrtb.Int8Ptr(1),
		},
	}

	for _, test := range testCases {
		request := &openrtb.BidRequest{Device: &test.givenDevice}
		modifyForIOS14X(request)
		assert.Equal(t, test.expectedLMT, request.Device.Lmt, test.description)
	}
}

func TestModifyForIOS142OrGreater(t *testing.T) {
	testCases := []struct {
		description string
		givenDevice openrtb.Device
		expectedLMT *int8
	}{
		{
			description: "Not Determined",
			givenDevice: openrtb.Device{Ext: json.RawMessage(`{"atts":0}`), Lmt: nil},
			expectedLMT: openrtb.Int8Ptr(0),
		},
		{
			description: "Restricted",
			givenDevice: openrtb.Device{Ext: json.RawMessage(`{"atts":1}`), Lmt: nil},
			expectedLMT: openrtb.Int8Ptr(1),
		},
		{
			description: "Denied",
			givenDevice: openrtb.Device{Ext: json.RawMessage(`{"atts":2}`), Lmt: nil},
			expectedLMT: openrtb.Int8Ptr(1),
		},
		{
			description: "Authorized",
			givenDevice: openrtb.Device{Ext: json.RawMessage(`{"atts":3}`), Lmt: nil},
			expectedLMT: openrtb.Int8Ptr(0),
		},
		{
			description: "Overwrites Existing",
			givenDevice: openrtb.Device{Ext: json.RawMessage(`{"atts":3}`), Lmt: openrtb.Int8Ptr(1)},
			expectedLMT: openrtb.Int8Ptr(0),
		},
		{
			description: "Invalid Value - Unknown",
			givenDevice: openrtb.Device{Ext: json.RawMessage(`{"atts":4}`), Lmt: nil},
			expectedLMT: nil,
		},
		{
			description: "Invalid Value - Unknown - Does Not Overwrite Existing",
			givenDevice: openrtb.Device{Ext: json.RawMessage(`{"atts":4}`), Lmt: openrtb.Int8Ptr(1)},
			expectedLMT: openrtb.Int8Ptr(1),
		},
		{
			description: "Invalid Value - Missing",
			givenDevice: openrtb.Device{Ext: json.RawMessage(`{}`), Lmt: nil},
			expectedLMT: nil,
		},
		{
			description: "Invalid Value - Wrong Type",
			givenDevice: openrtb.Device{Ext: json.RawMessage(`{"atts":"wrong type"}`), Lmt: nil},
			expectedLMT: nil,
		},
	}

	for _, test := range testCases {
		request := &openrtb.BidRequest{Device: &test.givenDevice}
		modifyForIOS142OrGreater(request)
		assert.Equal(t, test.expectedLMT, request.Device.Lmt, test.description)
	}
}
