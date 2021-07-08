package sharethrough

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestGetAdMarkup(t *testing.T) {
	tests := map[string]struct {
		inputResponse   []byte
		inputParams     *StrAdSeverParams
		expectedSuccess []string
		expectedError   error
	}{
		"Sets template variables": {
			inputResponse: []byte(`{"bidId": "bid", "adserverRequestId": "arid"}`),
			inputParams:   &StrAdSeverParams{Pkey: "pkey"},
			expectedSuccess: []string{
				`<img src="//b.sharethrough.com/butler?type=s2s-win&arid=arid&adReceivedAt=2019-09-12T11%3a29%3a00.000123456Z" />`,
				`<div data-str-native-key="pkey" data-stx-response-name="str_response_bid"></div>`,
				fmt.Sprintf(`<script>var str_response_bid = "%s"</script>`, base64.StdEncoding.EncodeToString([]byte(`{"bidId": "bid", "adserverRequestId": "arid"}`))),
			},
			expectedError: nil,
		},
		"Includes sfp.js without iFrame busting logic if iFrame param is true": {
			inputResponse: []byte(`{"bidId": "bid", "adserverRequestId": "arid"}`),
			inputParams:   &StrAdSeverParams{Pkey: "pkey", Iframe: true},
			expectedSuccess: []string{
				`<script src="//native.sharethrough.com/assets/sfp.js"></script>`,
			},
			expectedError: nil,
		},
		"Includes sfp.js with iFrame busting logic if iFrame param is false": {
			inputResponse: []byte(`{"bidId": "bid", "adserverRequestId": "arid"}`),
			inputParams:   &StrAdSeverParams{Pkey: "pkey", Iframe: false},
			expectedSuccess: []string{
				`<script src="//native.sharethrough.com/assets/sfp-set-targeting.js"></script>`,
			},
			expectedError: nil,
		},
		"Includes sfp.js with iFrame busting logic if iFrame param is not provided": {
			inputResponse: []byte(`{"bidId": "bid", "adserverRequestId": "arid"}`),
			inputParams:   &StrAdSeverParams{Pkey: "pkey"},
			expectedSuccess: []string{
				`<script src="//native.sharethrough.com/assets/sfp-set-targeting.js"></script>`,
			},
			expectedError: nil,
		},
	}

	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)
		assert := assert.New(t)

		var strResp openrtb_ext.ExtImpSharethroughResponse
		_ = json.Unmarshal(test.inputResponse, &strResp)

		outputSuccess, outputError := Util{Clock: MockClock{}}.getAdMarkup(test.inputResponse, strResp, test.inputParams)
		for _, markup := range test.expectedSuccess {
			assert.Contains(outputSuccess, markup)
		}
		assert.Equal(test.expectedError, outputError)
	}
}

func TestGetPlacementSize(t *testing.T) {
	tests := map[string]struct {
		imp            openrtb2.Imp
		strImpParams   openrtb_ext.ExtImpSharethrough
		expectedHeight int64
		expectedWidth  int64
	}{
		"Returns size from STR params if provided": {
			imp:            openrtb2.Imp{},
			strImpParams:   openrtb_ext.ExtImpSharethrough{IframeSize: []int{100, 200}},
			expectedHeight: 100,
			expectedWidth:  200,
		},
		"Skips size from STR params if malformed": {
			imp:            openrtb2.Imp{},
			strImpParams:   openrtb_ext.ExtImpSharethrough{IframeSize: []int{100}},
			expectedHeight: 1,
			expectedWidth:  1,
		},
		"Returns size from banner format if provided": {
			imp:            openrtb2.Imp{Banner: &openrtb2.Banner{Format: []openrtb2.Format{{H: 100, W: 200}}}},
			strImpParams:   openrtb_ext.ExtImpSharethrough{},
			expectedHeight: 100,
			expectedWidth:  200,
		},
		"Defaults to 1x1": {
			imp:            openrtb2.Imp{},
			strImpParams:   openrtb_ext.ExtImpSharethrough{},
			expectedHeight: 1,
			expectedWidth:  1,
		},
	}

	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)
		assert := assert.New(t)

		outputHeight, outputWidth := Util{}.getPlacementSize(test.imp, test.strImpParams)
		assert.Equal(test.expectedHeight, outputHeight)
		assert.Equal(test.expectedWidth, outputWidth)
	}
}

func TestGetBestFormat(t *testing.T) {
	tests := map[string]struct {
		input          []openrtb2.Format
		expectedHeight int64
		expectedWidth  int64
	}{
		"Returns default size if empty input": {
			input:          []openrtb2.Format{},
			expectedHeight: 1,
			expectedWidth:  1,
		},
		"Returns size if only one is passed": {
			input:          []openrtb2.Format{{H: 100, W: 100}},
			expectedHeight: 100,
			expectedWidth:  100,
		},
		"Returns biggest size if multiple are passed": {
			input:          []openrtb2.Format{{H: 100, W: 100}, {H: 200, W: 200}, {H: 50, W: 50}},
			expectedHeight: 200,
			expectedWidth:  200,
		},
	}

	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)
		assert := assert.New(t)

		outputHeight, outputWidth := Util{}.getBestFormat(test.input)
		assert.Equal(test.expectedHeight, outputHeight)
		assert.Equal(test.expectedWidth, outputWidth)
	}
}

type userAgentTest struct {
	input    string
	expected bool
}

type userAgentFailureTest struct {
	input string
}

func runUserAgentTests(tests map[string]userAgentTest, fn func(string) bool, t *testing.T) {
	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)
		assert := assert.New(t)

		output := fn(test.input)
		assert.Equal(test.expected, output)
	}
}

func TestCanAutoPlayVideo(t *testing.T) {
	uaParsers := UserAgentParsers{
		ChromeVersion:    regexp.MustCompile(`Chrome\/(?P<ChromeVersion>\d+)`),
		ChromeiOSVersion: regexp.MustCompile(`CriOS\/(?P<chromeiOSVersion>\d+)`),
		SafariVersion:    regexp.MustCompile(`Version\/(?P<safariVersion>\d+)`),
	}

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

	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)
		assert := assert.New(t)

		output := Util{}.canAutoPlayVideo(test.input, uaParsers)
		assert.Equal(test.expected, output)
	}
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
	regex := regexp.MustCompile(`Chrome\/(?P<ChromeVersion>\d+)`)
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

	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)
		assert := assert.New(t)

		output := Util{}.isAtMinChromeVersion(test.input, regex)
		assert.Equal(test.expected, output)
	}
}

func TestIsAtMinChromeIosVersion(t *testing.T) {
	regex := regexp.MustCompile(`CriOS\/(?P<chromeiOSVersion>\d+)`)
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

	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)
		assert := assert.New(t)

		output := Util{}.isAtMinChromeVersion(test.input, regex)
		assert.Equal(test.expected, output)
	}
}

func TestIsAtMinSafariVersion(t *testing.T) {
	regex := regexp.MustCompile(`Version\/(?P<safariVersion>\d+)`)
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

	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)
		assert := assert.New(t)

		output := Util{}.isAtMinSafariVersion(test.input, regex)
		assert.Equal(test.expected, output)
	}
}

func TestGdprApplies(t *testing.T) {
	bidRequestGdpr := openrtb2.BidRequest{
		Regs: &openrtb2.Regs{
			Ext: []byte(`{"gdpr": 1}`),
		},
	}
	bidRequestNonGdpr := openrtb2.BidRequest{
		Regs: &openrtb2.Regs{
			Ext: []byte(`{"gdpr": 0}`),
		},
	}
	bidRequestEmptyGdpr := openrtb2.BidRequest{
		Regs: &openrtb2.Regs{
			Ext: []byte(``),
		},
	}
	bidRequestEmptyRegs := openrtb2.BidRequest{
		Regs: &openrtb2.Regs{},
	}

	tests := map[string]struct {
		input    *openrtb2.BidRequest
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

	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)
		assert := assert.New(t)

		output := Util{}.gdprApplies(test.input)
		assert.Equal(test.expected, output)
	}
}

func TestParseUserInfo(t *testing.T) {
	tests := map[string]struct {
		input    *openrtb2.User
		expected userInfo
	}{
		"Return empty strings if no User": {
			input:    nil,
			expected: userInfo{Consent: "", TtdUid: "", StxUid: ""},
		},
		"Return empty strings if no uids": {
			input:    &openrtb2.User{Ext: []byte(`{ "eids": [{"source": "adserver.org", "uids": []}] }`)},
			expected: userInfo{Consent: "", TtdUid: "", StxUid: ""},
		},
		"Return empty strings if ID is not defined or empty string": {
			input:    &openrtb2.User{Ext: []byte(`{ "eids": [{"source": "adserver.org", "uids": [{"id": null}]}, {"source": "adserver.org", "uids": [{"id": ""}]}] }`)},
			expected: userInfo{Consent: "", TtdUid: "", StxUid: ""},
		},
		"Return consent correctly": {
			input:    &openrtb2.User{Ext: []byte(`{ "consent": "abc" }`)},
			expected: userInfo{Consent: "abc", TtdUid: "", StxUid: ""},
		},
		"Return ttd uid correctly": {
			input:    &openrtb2.User{Ext: []byte(`{ "eids": [{"source": "adserver.org", "uids": [{"id": "abc123"}]}] }`)},
			expected: userInfo{Consent: "", TtdUid: "abc123", StxUid: ""},
		},
		"Ignore non-trade-desk uid": {
			input:    &openrtb2.User{Ext: []byte(`{ "eids": [{"source": "something", "uids": [{"id": "xyz"}]}] }`)},
			expected: userInfo{Consent: "", TtdUid: "", StxUid: ""},
		},
		"Returns STX user id from buyer id": {
			input:    &openrtb2.User{BuyerUID: "myid"},
			expected: userInfo{Consent: "", TtdUid: "", StxUid: "myid"},
		},
		"Full test": {
			input:    &openrtb2.User{BuyerUID: "myid", Ext: []byte(`{ "consent": "abc", "eids": [{"source": "something", "uids": [{"id": "xyz"}]}, {"source": "adserver.org", "uids": [{"id": "abc123"}]}] }`)},
			expected: userInfo{Consent: "abc", TtdUid: "abc123", StxUid: "myid"},
		},
	}

	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)
		assert := assert.New(t)

		output := Util{}.parseUserInfo(test.input)
		assert.Equal(test.expected.Consent, output.Consent)
		assert.Equal(test.expected.TtdUid, output.TtdUid)
		assert.Equal(test.expected.StxUid, output.StxUid)
	}
}

func TestParseDomain(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected string
	}{
		"Parses domain without port": {
			input:    "http://a.domain.com/page?param=value",
			expected: "http://a.domain.com",
		},
		"Parses domain with port": {
			input:    "https://a.domain.com:8000/page?param=value",
			expected: "https://a.domain.com",
		},
		"Returns empty string if cannot parse the domain": {
			input:    "abc",
			expected: "",
		},
	}

	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)
		assert := assert.New(t)

		output := Util{}.parseDomain(test.input)
		assert.Equal(test.expected, output)
	}
}

func TestGetClock(t *testing.T) {
	tests := map[string]struct {
		input    Util
		expected Clock
	}{
		"returns Clock from Utility": {
			input:    Util{Clock: Clock{}},
			expected: Clock{},
		},
	}

	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)
		assert := assert.New(t)

		output := test.input.getClock()
		assert.Equal(test.expected, output)
	}
}
