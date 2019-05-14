package sharethrough

import (
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"strings"
	"testing"
)

func TestGetAdMarkup(t *testing.T) {
	tests := map[string]struct {
		inputResponse   openrtb_ext.ExtImpSharethroughResponse
		inputParams     *StrAdSeverParams
		expectedSuccess []string
		expectedError   error
	}{
		"Sets template variables": {
			inputResponse: openrtb_ext.ExtImpSharethroughResponse{BidID: "bid", AdServerRequestID: "arid"},
			inputParams:   &StrAdSeverParams{Pkey: "pkey"},
			expectedSuccess: []string{
				`<img src="//b.sharethrough.com/butler?type=s2s-win&arid=arid" />`,
				`<div data-str-native-key="pkey" data-stx-response-name="str_response_bid"></div>`,
				`<script>var str_response_bid = "eyJhZHNlcnZlclJlcXVlc3RJZCI6ImFyaWQiLCJiaWRJZCI6ImJpZCIsImNvb2tpZVN5bmNVcmxzIjpudWxsLCJjcmVhdGl2ZXMiOm51bGwsInBsYWNlbWVudCI6eyJhbGxvd19pbnN0YW50X3BsYXkiOmZhbHNlLCJhcnRpY2xlc19iZWZvcmVfZmlyc3RfYWQiOjAsImFydGljbGVzX2JldHdlZW5fYWRzIjowLCJsYXlvdXQiOiIiLCJtZXRhZGF0YSI6bnVsbCwicGxhY2VtZW50QXR0cmlidXRlcyI6eyJhZF9zZXJ2ZXJfa2V5IjoiIiwiYWRfc2VydmVyX3BhdGgiOiIiLCJhbGxvd19keW5hbWljX2Nyb3BwaW5nIjpmYWxzZSwiYXBwX3RoaXJkX3BhcnR5X3BhcnRuZXJzIjpudWxsLCJjdXN0b21fY2FyZF9jc3MiOiIiLCJkZnBfcGF0aCI6IiIsImRpcmVjdF9zZWxsX3Byb21vdGVkX2J5X3RleHQiOiIiLCJkb21haW4iOiIiLCJlbmFibGVfbGlua19yZWRpcmVjdGlvbiI6ZmFsc2UsImZlYXR1cmVkX2NvbnRlbnQiOm51bGwsIm1heF9oZWFkbGluZV9sZW5ndGgiOjAsIm11bHRpX2FkX3BsYWNlbWVudCI6ZmFsc2UsInByb21vdGVkX2J5X3RleHQiOiIiLCJwdWJsaXNoZXJfa2V5IjoiIiwicmVuZGVyaW5nX3BpeGVsX29mZnNldCI6MCwic2FmZV9mcmFtZV9zaXplIjpudWxsLCJzaXRlX2tleSI6IiIsInN0cl9vcHRfb3V0X3VybCI6IiIsInRlbXBsYXRlIjoiIiwidGhpcmRfcGFydHlfcGFydG5lcnMiOm51bGx9LCJzdGF0dXMiOiIifSwic3R4VXNlcklkIjoiIn0="</script>`,
			},
			expectedError: nil,
		},
		"Includes sfp.js without iFrame busting logic if iFrame param is true": {
			inputResponse: openrtb_ext.ExtImpSharethroughResponse{BidID: "bid", AdServerRequestID: "arid"},
			inputParams:   &StrAdSeverParams{Pkey: "pkey", Iframe: true},
			expectedSuccess: []string{
				`<script src="//native.sharethrough.com/assets/sfp.js"></script>`,
			},
			expectedError: nil,
		},
		"Includes sfp.js with iFrame busting logic if iFrame param is false": {
			inputResponse: openrtb_ext.ExtImpSharethroughResponse{BidID: "bid", AdServerRequestID: "arid"},
			inputParams:   &StrAdSeverParams{Pkey: "pkey", Iframe: false},
			expectedSuccess: []string{
				`<script src="//native.sharethrough.com/assets/sfp-set-targeting.js"></script>`,
			},
			expectedError: nil,
		},
		"Includes sfp.js with iFrame busting logic if iFrame param is not provided": {
			inputResponse: openrtb_ext.ExtImpSharethroughResponse{BidID: "bid", AdServerRequestID: "arid"},
			inputParams:   &StrAdSeverParams{Pkey: "pkey"},
			expectedSuccess: []string{
				`<script src="//native.sharethrough.com/assets/sfp-set-targeting.js"></script>`,
			},
			expectedError: nil,
		},
	}

	util := Util{}
	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)

		outputSuccess, outputError := util.getAdMarkup(test.inputResponse, test.inputParams)
		for _, markup := range test.expectedSuccess {
			if !strings.Contains(outputSuccess, markup) {
				t.Errorf("Expected Ad Markup to contain: %s, got %s\n", markup, outputSuccess)
			}
		}
		if outputError != test.expectedError {
			t.Errorf("Expected Error to be: %s, got %s\n", test.expectedError, outputError)
		}
	}
}

func TestGetPlacementSize(t *testing.T) {
	tests := map[string]struct {
		input          []openrtb.Format
		expectedHeight uint64
		expectedWidth  uint64
	}{
		"Returns default size if empty input": {
			input:          []openrtb.Format{},
			expectedHeight: 1,
			expectedWidth:  1,
		},
		"Returns size if only one is passed": {
			input:          []openrtb.Format{{H: 100, W: 100}},
			expectedHeight: 100,
			expectedWidth:  100,
		},
		"Returns biggest size if multiple are passed": {
			input:          []openrtb.Format{{H: 100, W: 100}, {H: 200, W: 200}, {H: 50, W: 50}},
			expectedHeight: 200,
			expectedWidth:  200,
		},
	}

	util := Util{}
	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)

		outputHeight, outputWidth := util.getPlacementSize(test.input)
		if outputHeight != test.expectedHeight {
			t.Errorf("Expected Height: %d, got %d\n", test.expectedHeight, outputHeight)
		}
		if outputWidth != test.expectedWidth {
			t.Errorf("Expected Width: %d, got %d\n", test.expectedWidth, outputWidth)
		}
	}
}

type userAgentTest struct {
	input    string
	expected bool
}

func runUserAgentTests(tests map[string]userAgentTest, fn func(string) bool, t *testing.T) {
	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)

		output := fn(test.input)
		if output != test.expected {
			t.Errorf("Expected: %t, got %t\n", test.expected, output)
		}
	}
}

func TestCanAutoPlayVideo(t *testing.T) {
	ableAgents := map[string]string{
		"Android at min Chrome version": "Android Chrome/60.0",
		"iOS at min Chrome version":     "iPhone CriOS/60.0",
		"iOS at min Safari version":     "iPad Version/14.0",
		"Neither Android or iOS":        "Some User Agent",
	}
	unableAgents := map[string]string{
		"Android not at min Chrome version": "Android Chrome/12",
		"iOS not at min Chrome version":     "iPod Chrome/12",
		"iOS not at min Safari version":     "iPod Version/8",
	}

	tests := map[string]userAgentTest{}
	for testName, agent := range ableAgents {
		tests[testName] = userAgentTest{
			input:    agent,
			expected: true,
		}
	}
	for testName, agent := range unableAgents {
		tests[testName] = userAgentTest{
			input:    agent,
			expected: false,
		}
	}

	runUserAgentTests(tests, Util{}.canAutoPlayVideo, t)
}

func TestIsAndroid(t *testing.T) {
	goodUserAgent := "Mozilla/5.0 (Linux; Android 6.0.1; Nexus 6P Build/MMB29P)"
	badUserAgent := "fake user agent"

	// This is an alternate way to do testing if you have many test cases that only change the input and output
	tests := map[string]userAgentTest{
		"Match the Android user agent": {
			input:    goodUserAgent,
			expected: true,
		},
		"Does not match Android user agent": {
			input:    badUserAgent,
			expected: false,
		},
	}

	runUserAgentTests(tests, Util{}.isAndroid, t)
}

func TestIsiOS(t *testing.T) {
	iPhoneUserAgent := "Some string containing iPhone"
	iPadUserAgent := "Some string containing iPad"
	iPodUserAgent := "Some string containing iPOD"
	badUserAgent := "Fake User Agent"

	tests := map[string]userAgentTest{
		"Match the iPhone user agent": {
			input:    iPhoneUserAgent,
			expected: true,
		},
		"Match the iPad user agent": {
			input:    iPadUserAgent,
			expected: true,
		},
		"Match the iPod user agent": {
			input:    iPodUserAgent,
			expected: true,
		},
		"Does not match Android user agent": {
			input:    badUserAgent,
			expected: false,
		},
	}

	runUserAgentTests(tests, Util{}.isiOS, t)
}

func TestIsAtMinChromeVersion(t *testing.T) {
	v60ChromeUA := "Mozilla/5.0 Chrome/60.0.3112.113"
	v12ChromeUA := "Mozilla/5.0 Chrome/12.0.3112.113"
	badUA := "Fake User Agent"

	tests := map[string]userAgentTest{
		"Return true if greater than min (53)": {
			input:    v60ChromeUA,
			expected: true,
		},
		"Return false if lower than min (53)": {
			input:    v12ChromeUA,
			expected: false,
		},
		"Return false if no version found": {
			input:    badUA,
			expected: false,
		},
	}

	runUserAgentTests(tests, Util{}.isAtMinChromeVersion, t)
}

func TestIsAtMinChromeIosVersion(t *testing.T) {
	v60ChrIosUA := "Mozilla/5.0 CriOS/60.0.3112.113"
	v12ChrIosUA := "Mozilla/5.0 CriOS/12.0.3112.113"
	badUA := "Fake User Agent"

	tests := map[string]userAgentTest{
		"Return true if greater than min (53)": {
			input:    v60ChrIosUA,
			expected: true,
		},
		"Return false if lower than min (53)": {
			input:    v12ChrIosUA,
			expected: false,
		},
		"Return false if no version found": {
			input:    badUA,
			expected: false,
		},
	}

	runUserAgentTests(tests, Util{}.isAtMinChromeIosVersion, t)
}

func TestIsAtMinSafariVersion(t *testing.T) {
	v12SafariUA := "Mozilla/5.0 Version/12.0.3112.113"
	v07SafariUA := "Mozilla/5.0 Version/07.0.3112.113"
	badUA := "Fake User Agent"

	tests := map[string]userAgentTest{
		"Return true if greater than min (10)": {
			input:    v12SafariUA,
			expected: true,
		},
		"Return false if lower than min (10)": {
			input:    v07SafariUA,
			expected: false,
		},
		"Return false if no version found": {
			input:    badUA,
			expected: false,
		},
	}

	runUserAgentTests(tests, Util{}.isAtMinSafariVersion, t)
}

func TestGdprApplies(t *testing.T) {
	bidRequestGdpr := openrtb.BidRequest{
		Regs: &openrtb.Regs{
			Ext: []byte(`{"gdpr": 1}`),
		},
	}
	bidRequestNonGdpr := openrtb.BidRequest{
		Regs: &openrtb.Regs{
			Ext: []byte(`{"gdpr": 0}`),
		},
	}
	bidRequestEmptyGdpr := openrtb.BidRequest{
		Regs: &openrtb.Regs{
			Ext: []byte(``),
		},
	}
	bidRequestEmptyRegs := openrtb.BidRequest{
		Regs: &openrtb.Regs{},
	}

	tests := map[string]struct {
		input    *openrtb.BidRequest
		expected bool
	}{
		"Return true if gdpr set to 1": {
			input:    &bidRequestGdpr,
			expected: true,
		},
		"Return false if gdpr set to 0": {
			input:    &bidRequestNonGdpr,
			expected: false,
		},
		"Return false if no gdpr set": {
			input:    &bidRequestEmptyGdpr,
			expected: false,
		},
		"Return false if no Regs set": {
			input:    &bidRequestEmptyRegs,
			expected: false,
		},
	}

	util := Util{}
	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)

		output := util.gdprApplies(test.input)
		if output != test.expected {
			t.Errorf("Expected: %t, got %t\n", test.expected, output)
		}
	}
}

func TestGdprConsentString(t *testing.T) {
	bidRequestWithConsent := openrtb.BidRequest{
		User: &openrtb.User{
			Ext: []byte(`{"consent": "abc"}`),
		},
	}
	bidRequestWithEmptyConsent := openrtb.BidRequest{
		User: &openrtb.User{
			Ext: []byte(`{"consent": ""}`),
		},
	}
	bidRequestWithoutConsent := openrtb.BidRequest{
		User: &openrtb.User{
			Ext: []byte(`{"other": "abc"}`),
		},
	}
	bidRequestWithUserExt := openrtb.BidRequest{
		User: &openrtb.User{},
	}

	tests := map[string]struct {
		input    *openrtb.BidRequest
		expected string
	}{
		"Return consent string if provided": {
			input:    &bidRequestWithConsent,
			expected: "abc",
		},
		"Return empty string if consent string empty": {
			input:    &bidRequestWithEmptyConsent,
			expected: "",
		},
		"Return empty string if no consent string provided": {
			input:    &bidRequestWithoutConsent,
			expected: "",
		},
		"Return empty string if User set": {
			input:    &bidRequestWithUserExt,
			expected: "",
		},
	}

	util := Util{}
	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)

		output := util.gdprConsentString(test.input)
		if output != test.expected {
			t.Errorf("Expected: %s, got %s\n", test.expected, output)
		}
	}
}
