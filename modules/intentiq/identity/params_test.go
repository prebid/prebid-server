package identity

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

// baseURL is the fixed prefix every resolution URL starts with (partner id and source echoed in).
const testEndpoint = "https://dev.example.com/resolve"

func wrap(req *openrtb2.BidRequest) *openrtb_ext.RequestWrapper {
	return &openrtb_ext.RequestWrapper{BidRequest: req}
}

func TestResolveURLConstantParams(t *testing.T) {
	cfg := Config{APIEndpoint: testEndpoint, PartnerID: "partner-42"}
	got := resolveURL(cfg, wrap(&openrtb2.BidRequest{}))

	assert.Equal(t,
		testEndpoint+"?at=39&mi=10&dpi=partner-42&pt=17&dpn=1&srvrReq=true&source=pbsgo",
		got)
}

func TestResolveURLAppendsToExistingQuery(t *testing.T) {
	cfg := Config{APIEndpoint: "https://dev.example.com/resolve?x=1", PartnerID: "p"}
	got := resolveURL(cfg, wrap(&openrtb2.BidRequest{}))

	assert.True(t, strings.HasPrefix(got, "https://dev.example.com/resolve?x=1&at=39&"), got)
}

func TestResolveURLDeviceIPIPv6AndUserAgentEncoding(t *testing.T) {
	cfg := Config{APIEndpoint: testEndpoint, PartnerID: "383342646"}
	req := &openrtb2.BidRequest{Device: &openrtb2.Device{
		IP:   "125.253.50.47",
		IPv6: "2001:db8::1",
		UA:   "Mozilla/5.0 (iPhone)",
	}}

	got := resolveURL(cfg, wrap(req))

	assert.Equal(t,
		testEndpoint+"?at=39&mi=10&dpi=383342646&pt=17&dpn=1&srvrReq=true&source=pbsgo"+
			"&ip=125.253.50.47&ipv6=2001%3Adb8%3A%3A1&uas=Mozilla%2F5.0%20%28iPhone%29",
		got)
}

func TestResolveURLDeviceIfaMobileIdtype4(t *testing.T) {
	cfg := Config{APIEndpoint: testEndpoint, PartnerID: "123"}
	req := &openrtb2.BidRequest{Device: &openrtb2.Device{IFA: "maid-AbC", DeviceType: 1}}

	assert.Contains(t, resolveURL(cfg, wrap(req)), "&pcid=maid-AbC&idtype=4")
}

func TestResolveURLDeviceIfaCtvUppercasedIdtype8(t *testing.T) {
	cfg := Config{APIEndpoint: testEndpoint, PartnerID: "123"}
	req := &openrtb2.BidRequest{Device: &openrtb2.Device{IFA: "rida-abc", DeviceType: 3}}

	assert.Contains(t, resolveURL(cfg, wrap(req)), "&pcid=RIDA-ABC&idtype=8")
}

func TestResolveURLDeviceIfaCtvDeviceType7(t *testing.T) {
	cfg := Config{APIEndpoint: testEndpoint, PartnerID: "123"}
	req := &openrtb2.BidRequest{Device: &openrtb2.Device{IFA: "rida-xyz", DeviceType: 7}}

	assert.Contains(t, resolveURL(cfg, wrap(req)), "&pcid=RIDA-XYZ&idtype=8")
}

func TestResolveURLSkipsDeviceIdWhenLimitAdTracking(t *testing.T) {
	cfg := Config{APIEndpoint: testEndpoint, PartnerID: "123"}
	lmt := int8(1)
	req := &openrtb2.BidRequest{Device: &openrtb2.Device{IFA: "maid-1", Lmt: &lmt}}

	got := resolveURL(cfg, wrap(req))
	assert.NotContains(t, got, "pcid")
	assert.NotContains(t, got, "idtype")
}

func TestResolveURLSkipsDeviceIdWhenIfaBlank(t *testing.T) {
	cfg := Config{APIEndpoint: testEndpoint, PartnerID: "123"}
	req := &openrtb2.BidRequest{Device: &openrtb2.Device{IFA: "   "}}

	assert.NotContains(t, resolveURL(cfg, wrap(req)), "pcid")
}

func TestResolveURLRefFromSiteDomain(t *testing.T) {
	cfg := Config{APIEndpoint: testEndpoint, PartnerID: "123"}
	req := &openrtb2.BidRequest{Site: &openrtb2.Site{Domain: "example.com", Page: "https://example.com/p"}}

	assert.Contains(t, resolveURL(cfg, wrap(req)), "&ref=example.com")
}

func TestResolveURLRefFallsBackToSitePage(t *testing.T) {
	cfg := Config{APIEndpoint: testEndpoint, PartnerID: "123"}
	req := &openrtb2.BidRequest{Site: &openrtb2.Site{Page: "https://example.com/p"}}

	assert.Contains(t, resolveURL(cfg, wrap(req)), "&ref=https%3A%2F%2Fexample.com%2Fp")
}

func TestResolveURLRefFromAppBundleThenName(t *testing.T) {
	cfg := Config{APIEndpoint: testEndpoint, PartnerID: "123"}
	assert.Contains(t, resolveURL(cfg, wrap(&openrtb2.BidRequest{App: &openrtb2.App{Bundle: "com.x.y"}})), "&ref=com.x.y")
	assert.Contains(t, resolveURL(cfg, wrap(&openrtb2.BidRequest{App: &openrtb2.App{Name: "MyApp"}})), "&ref=MyApp")
}

func TestResolveURLIiqUidFromEid(t *testing.T) {
	cfg := Config{APIEndpoint: testEndpoint, PartnerID: "123"}
	req := &openrtb2.BidRequest{User: &openrtb2.User{EIDs: []openrtb2.EID{
		{Source: "other.com", UIDs: []openrtb2.UID{{ID: "x"}}},
		{Source: iiqSource, UIDs: []openrtb2.UID{{ID: "IIQ-UID-1"}}},
	}}}

	assert.Contains(t, resolveURL(cfg, wrap(req)), "&iiquid=IIQ-UID-1")
}

func TestResolveURLConsentParamsFromTopLevelFields(t *testing.T) {
	cfg := Config{APIEndpoint: testEndpoint, PartnerID: "123"}
	gdpr := int8(1)
	req := &openrtb2.BidRequest{
		Regs: &openrtb2.Regs{GDPR: &gdpr, USPrivacy: "1YNN", GPP: "DBABMA~CONSENT", GPPSID: []int8{2, 6}},
		User: &openrtb2.User{Consent: "CO-TCF-STRING"},
	}

	got := resolveURL(cfg, wrap(req))
	assert.Contains(t, got, "&gdpr=1")
	assert.Contains(t, got, "&us_privacy=1YNN")
	// NOTE: url.QueryEscape leaves '~' unescaped (Java's URLEncoder emits %7E); see report.
	assert.Contains(t, got, "&gpp=DBABMA~CONSENT")
	assert.Contains(t, got, "&gpp_sid=2%2C6")
	// Consent is a header, never a query param.
	assert.NotContains(t, got, "CO-TCF-STRING")
}

func TestResolveURLConsentParamsFromRegsExtFallback(t *testing.T) {
	cfg := Config{APIEndpoint: testEndpoint, PartnerID: "123"}
	req := &openrtb2.BidRequest{
		Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"gdpr":1,"us_privacy":"1NYN"}`)},
	}

	got := resolveURL(cfg, wrap(req))
	assert.Contains(t, got, "&gdpr=1")
	assert.Contains(t, got, "&us_privacy=1NYN")
}

func TestResolveURLNoConsentParamsWhenAbsent(t *testing.T) {
	cfg := Config{APIEndpoint: testEndpoint, PartnerID: "123"}
	got := resolveURL(cfg, wrap(&openrtb2.BidRequest{}))

	assert.NotContains(t, got, "gdpr")
	assert.NotContains(t, got, "us_privacy")
	assert.NotContains(t, got, "gpp")
}

func TestResolveConsentFromUserConsent(t *testing.T) {
	req := &openrtb2.BidRequest{User: &openrtb2.User{Consent: "CO-TCF-STRING"}}
	assert.Equal(t, "CO-TCF-STRING", resolveConsent(wrap(req)))
}

func TestResolveConsentFromUserExtFallback(t *testing.T) {
	req := &openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consent":"EXT-TCF-STRING"}`)}}
	assert.Equal(t, "EXT-TCF-STRING", resolveConsent(wrap(req)))
}

func TestResolveConsentEmptyWhenAbsent(t *testing.T) {
	assert.Equal(t, "", resolveConsent(wrap(&openrtb2.BidRequest{})))
}

func TestBuildUaHintsFromDeviceSua(t *testing.T) {
	mobile := int8(0)
	sua := &openrtb2.UserAgent{
		Source: 2,
		Browsers: []openrtb2.BrandVersion{
			{Brand: "Chromium", Version: []string{"108", "0", "5359", "125"}},
			{Brand: "Google Chrome", Version: []string{"108", "0", "5359", "125"}},
			{Brand: "Not?A_Brand", Version: []string{"8", "0", "0", "0"}},
		},
		Platform:     &openrtb2.BrandVersion{Brand: "Windows", Version: []string{"15", "0", "0"}},
		Mobile:       &mobile,
		Architecture: "x86",
		Bitness:      "64",
	}

	var uh map[string]string
	assert.NoError(t, json.Unmarshal([]byte(buildUaHints(sua)), &uh))

	assert.Equal(t, `"Chromium";v="108", "Google Chrome";v="108", "Not?A_Brand";v="8"`, uh["0"])
	assert.Equal(t, `"Chromium";v="108.0.5359.125", "Google Chrome";v="108.0.5359.125", "Not?A_Brand";v="8.0.0.0"`, uh["8"])
	assert.Equal(t, "?0", uh["1"])
	assert.Equal(t, `"Windows"`, uh["2"])
	assert.Equal(t, `"x86"`, uh["3"])
	assert.Equal(t, `"64"`, uh["4"])
	assert.Equal(t, `"15.0.0"`, uh["6"])
	_, has5 := uh["5"]
	_, has7 := uh["7"]
	assert.False(t, has5)
	assert.False(t, has7)
}

func TestBuildUaHintsModelUnderKey5(t *testing.T) {
	sua := &openrtb2.UserAgent{Source: 2, Model: "Pixel 7"}
	var uh map[string]string
	assert.NoError(t, json.Unmarshal([]byte(buildUaHints(sua)), &uh))
	assert.Equal(t, `"Pixel 7"`, uh["5"])
}

func TestBuildUaHintsEmptyWhenNotHighEntropy(t *testing.T) {
	sua := &openrtb2.UserAgent{Source: 1, Browsers: []openrtb2.BrandVersion{{Brand: "Chrome", Version: []string{"120"}}}}
	assert.Equal(t, "", buildUaHints(sua))
}

func TestBuildUaHintsEmptyWhenNilOrNoData(t *testing.T) {
	assert.Equal(t, "", buildUaHints(nil))
	assert.Equal(t, "", buildUaHints(&openrtb2.UserAgent{Source: 2}))
}

func TestResolveURLUhParamPresentForHighEntropySua(t *testing.T) {
	cfg := Config{APIEndpoint: testEndpoint, PartnerID: "123"}
	sua := &openrtb2.UserAgent{Source: 2, Model: "Pixel"}
	req := &openrtb2.BidRequest{Device: &openrtb2.Device{SUA: sua}}
	assert.Contains(t, resolveURL(cfg, wrap(req)), "&uh=")
}

func TestResolveURLNoUhParamForLowEntropySua(t *testing.T) {
	cfg := Config{APIEndpoint: testEndpoint, PartnerID: "123"}
	sua := &openrtb2.UserAgent{Source: 1, Model: "Pixel"}
	req := &openrtb2.BidRequest{Device: &openrtb2.Device{SUA: sua}}
	assert.NotContains(t, resolveURL(cfg, wrap(req)), "uh=")
}
