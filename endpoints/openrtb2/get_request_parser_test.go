package openrtb2

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	jsonpatch "gopkg.in/evanphx/json-patch.v5"
)

// parseGETResult is a helper that parses the raw JSON returned by parseGETRequest
// into a generic map for easy field access.
func parseGETResult(t *testing.T, rawQuery string) map[string]interface{} {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/openrtb2/auction?"+rawQuery, nil)
	data, err := parseGETRequest(req, 0)
	require.NoError(t, err)
	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &out))
	return out
}

// getExtPrebid extracts ext.prebid as a map from a parsed bid-request map.
func getExtPrebid(t *testing.T, m map[string]interface{}) map[string]interface{} {
	t.Helper()
	extRaw, ok := m["ext"]
	require.True(t, ok, "ext missing")
	extMap, ok := extRaw.(map[string]interface{})
	require.True(t, ok, "ext not a map")
	prebidRaw, ok := extMap["prebid"]
	require.True(t, ok, "ext.prebid missing")
	prebidMap, ok := prebidRaw.(map[string]interface{})
	require.True(t, ok, "ext.prebid not a map")
	return prebidMap
}

// getImpOverrideMap extracts ext.prebid.getImpOverride as a map.
// Returns nil when no imp-specific params were set.
func getImpOverrideMap(t *testing.T, m map[string]interface{}) map[string]interface{} {
	t.Helper()
	prebid := getExtPrebid(t, m)
	raw, ok := prebid["getImpOverride"]
	if !ok {
		return nil
	}
	override, ok := raw.(map[string]interface{})
	require.True(t, ok, "getImpOverride is not a map")
	return override
}

// --- TestParseGETRequest_RequiresSrid ---

func TestParseGETRequest_RequiresSrid(t *testing.T) {
	t.Run("missing srid returns error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/openrtb2/auction", nil)
		_, err := parseGETRequest(req, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "srid")
	})

	t.Run("srid present returns no error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/openrtb2/auction?srid=abc", nil)
		_, err := parseGETRequest(req, 0)
		assert.NoError(t, err)
	})
}

// --- TestParseGETRequest_SridInStoredRequest ---

func TestParseGETRequest_SridInStoredRequest(t *testing.T) {
	m := parseGETResult(t, "srid=abc123")
	prebid := getExtPrebid(t, m)
	sr, ok := prebid["storedrequest"].(map[string]interface{})
	require.True(t, ok, "ext.prebid.storedrequest missing or wrong type")
	assert.Equal(t, "abc123", sr["id"])
}

// --- TestParseGETRequest_Tmax ---

func TestParseGETRequest_Tmax(t *testing.T) {
	t.Run("valid tmax=300 is set", func(t *testing.T) {
		m := parseGETResult(t, "srid=x&tmax=300")
		assert.EqualValues(t, float64(300), m["tmax"])
	})

	t.Run("tmax below minimum (50) is ignored", func(t *testing.T) {
		m := parseGETResult(t, "srid=x&tmax=50")
		val, exists := m["tmax"]
		if exists {
			assert.EqualValues(t, float64(0), val)
		}
	})

	t.Run("invalid tmax (abc) is ignored, no error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/openrtb2/auction?srid=x&tmax=abc", nil)
		_, err := parseGETRequest(req, 0)
		assert.NoError(t, err)
	})
}

// --- TestParseGETRequest_Debug ---

func TestParseGETRequest_Debug(t *testing.T) {
	t.Run("debug=1 sets debug true", func(t *testing.T) {
		m := parseGETResult(t, "srid=x&debug=1")
		prebid := getExtPrebid(t, m)
		assert.Equal(t, true, prebid["debug"])
	})

	t.Run("debug=0 does not set debug true", func(t *testing.T) {
		m := parseGETResult(t, "srid=x&debug=0")
		prebid := getExtPrebid(t, m)
		val, exists := prebid["debug"]
		if exists {
			assert.NotEqual(t, true, val)
		}
	})
}

// --- TestParseGETRequest_OutputFormat ---

func TestParseGETRequest_OutputFormat(t *testing.T) {
	t.Run("of sets OutputFormat", func(t *testing.T) {
		m := parseGETResult(t, "srid=x&of=vast4")
		prebid := getExtPrebid(t, m)
		assert.Equal(t, "vast4", prebid["of"])
	})

	t.Run("om sets OutputModule", func(t *testing.T) {
		m := parseGETResult(t, "srid=x&om=prebid.ctv_vast_enrichment")
		prebid := getExtPrebid(t, m)
		assert.Equal(t, "prebid.ctv_vast_enrichment", prebid["om"])
	})
}

// --- TestParseGETRequest_SlotMapsToTagID ---

// Imp-specific params now live in ext.prebid.getImpOverride (applied post-merge).
func TestParseGETRequest_SlotMapsToTagID(t *testing.T) {
	m := parseGETResult(t, "srid=x&slot=my-slot")
	override := getImpOverrideMap(t, m)
	require.NotNil(t, override, "getImpOverride missing")
	assert.Equal(t, "my-slot", override["tagid"])
}

// --- TestParseGETRequest_VideoParams ---

func TestParseGETRequest_VideoParams(t *testing.T) {
	t.Run("mindur/maxdur/w/h set on video imp", func(t *testing.T) {
		m := parseGETResult(t, "srid=x&mtype=2&mindur=5&maxdur=30&w=640&h=360")
		override := getImpOverrideMap(t, m)
		require.NotNil(t, override, "getImpOverride missing")
		video, ok := override["video"].(map[string]interface{})
		require.True(t, ok, "getImpOverride.video missing")
		assert.EqualValues(t, float64(5), video["minduration"])
		assert.EqualValues(t, float64(30), video["maxduration"])
		assert.EqualValues(t, float64(640), video["w"])
		assert.EqualValues(t, float64(360), video["h"])
	})

	t.Run("skip/skipmin/skipafter set on video imp", func(t *testing.T) {
		m := parseGETResult(t, "srid=x&mtype=vid&skip=1&skipmin=5&skipafter=3")
		override := getImpOverrideMap(t, m)
		require.NotNil(t, override, "getImpOverride missing")
		video, ok := override["video"].(map[string]interface{})
		require.True(t, ok, "getImpOverride.video missing")
		assert.EqualValues(t, float64(1), video["skip"])
		assert.EqualValues(t, float64(5), video["skipmin"])
		assert.EqualValues(t, float64(3), video["skipafter"])
	})
}

// --- TestParseGETRequest_AudioParams ---

func TestParseGETRequest_AudioParams(t *testing.T) {
	m := parseGETResult(t, "srid=x&mtype=3&mindur=10&maxdur=60")
	override := getImpOverrideMap(t, m)
	require.NotNil(t, override, "getImpOverride missing")
	audio, ok := override["audio"].(map[string]interface{})
	require.True(t, ok, "getImpOverride.audio missing")
	assert.EqualValues(t, float64(10), audio["minduration"])
	assert.EqualValues(t, float64(60), audio["maxduration"])
}

// --- TestParseGETRequest_BannerDefault ---

func TestParseGETRequest_BannerDefault(t *testing.T) {
	t.Run("no imp-specific params produce no getImpOverride", func(t *testing.T) {
		m := parseGETResult(t, "srid=x")
		override := getImpOverrideMap(t, m)
		assert.Nil(t, override, "no imp-specific params should produce no getImpOverride")
	})

	t.Run("mtype=1 with w/h sets banner dimensions in override", func(t *testing.T) {
		m := parseGETResult(t, "srid=x&mtype=1&w=300&h=250")
		override := getImpOverrideMap(t, m)
		require.NotNil(t, override, "getImpOverride missing")
		banner, ok := override["banner"].(map[string]interface{})
		require.True(t, ok, "getImpOverride.banner missing")
		assert.EqualValues(t, float64(300), banner["w"])
	})
}

// --- TestParseGETRequest_PubID ---

func TestParseGETRequest_PubID(t *testing.T) {
	m := parseGETResult(t, "srid=x&pubid=pub-123")
	site, ok := m["site"].(map[string]interface{})
	require.True(t, ok, "site missing")
	publisher, ok := site["publisher"].(map[string]interface{})
	require.True(t, ok, "site.publisher missing")
	assert.Equal(t, "pub-123", publisher["id"])
}

// --- TestParseGETRequest_Privacy ---

func TestParseGETRequest_Privacy(t *testing.T) {
	t.Run("gdpr and gdpr_consent", func(t *testing.T) {
		m := parseGETResult(t, "srid=x&gdpr=1&gdpr_consent=BOXAaa")
		regs, ok := m["regs"].(map[string]interface{})
		require.True(t, ok, "regs missing")
		assert.EqualValues(t, float64(1), regs["gdpr"])
		user, ok := m["user"].(map[string]interface{})
		require.True(t, ok, "user missing")
		assert.Equal(t, "BOXAaa", user["consent"])
	})

	t.Run("gppc sets gpp", func(t *testing.T) {
		m := parseGETResult(t, "srid=x&gppc=DBACNYA")
		regs, ok := m["regs"].(map[string]interface{})
		require.True(t, ok, "regs missing")
		assert.Equal(t, "DBACNYA", regs["gpp"])
	})

	t.Run("coppa=1 sets coppa", func(t *testing.T) {
		m := parseGETResult(t, "srid=x&coppa=1")
		regs, ok := m["regs"].(map[string]interface{})
		require.True(t, ok, "regs missing")
		assert.EqualValues(t, float64(1), regs["coppa"])
	})
}

// TestParseGETRequest_CoppaZeroIsExplicit verifies that coppa=0 appears in the
// generated JSON rather than being silently omitted by omitempty. Without this,
// a stored request with coppa=1 would not be overridden by the GET param.
func TestParseGETRequest_CoppaZeroIsExplicit(t *testing.T) {
	m := parseGETResult(t, "srid=x&coppa=0")
	regs, ok := m["regs"].(map[string]interface{})
	require.True(t, ok, "regs missing")
	val, exists := regs["coppa"]
	assert.True(t, exists, "coppa=0 must be present in JSON, not omitted by omitempty")
	assert.EqualValues(t, float64(0), val)
}

// --- TestParseGETRequest_ContentParams ---

func TestParseGETRequest_ContentParams(t *testing.T) {
	m := parseGETResult(t, "srid=x&cgenre=comedy&clang=pl&ctitle=Test")
	site, ok := m["site"].(map[string]interface{})
	require.True(t, ok, "site missing")
	content, ok := site["content"].(map[string]interface{})
	require.True(t, ok, "site.content missing")
	assert.Equal(t, "comedy", content["genre"])
	assert.Equal(t, "pl", content["language"])
	assert.Equal(t, "Test", content["title"])
}

// --- TestParseGETRequest_CSVParams ---

func TestParseGETRequest_CSVParams(t *testing.T) {
	t.Run("proto CSV sets video protocols", func(t *testing.T) {
		m := parseGETResult(t, "srid=x&mtype=2&proto=2,3,5")
		override := getImpOverrideMap(t, m)
		require.NotNil(t, override, "getImpOverride missing")
		video, ok := override["video"].(map[string]interface{})
		require.True(t, ok, "getImpOverride.video missing")
		protocols, ok := video["protocols"].([]interface{})
		require.True(t, ok, "video.protocols missing")
		assert.Equal(t, []interface{}{float64(2), float64(3), float64(5)}, protocols)
	})

	t.Run("api CSV sets video api", func(t *testing.T) {
		m := parseGETResult(t, "srid=x&mtype=2&api=1,2")
		override := getImpOverrideMap(t, m)
		require.NotNil(t, override, "getImpOverride missing")
		video, ok := override["video"].(map[string]interface{})
		require.True(t, ok, "getImpOverride.video missing")
		api, ok := video["api"].([]interface{})
		require.True(t, ok, "video.api missing")
		assert.Equal(t, []interface{}{float64(1), float64(2)}, api)
	})
}

// TestParseGETRequest_SaridSetsStoredAuctionResponse verifies sarid lands in
// getImpOverride.ext.prebid.storedauctionresponse without clobbering sibling keys.
func TestParseGETRequest_SaridSetsStoredAuctionResponse(t *testing.T) {
	m := parseGETResult(t, "srid=test-req&sarid=stored-resp-1")
	override := getImpOverrideMap(t, m)
	require.NotNil(t, override, "getImpOverride missing")

	extRaw, ok := override["ext"].(map[string]interface{})
	require.True(t, ok, "getImpOverride.ext missing")
	prebid, ok := extRaw["prebid"].(map[string]interface{})
	require.True(t, ok, "getImpOverride.ext.prebid missing")
	sar, ok := prebid["storedauctionresponse"].(map[string]interface{})
	require.True(t, ok, "storedauctionresponse missing or not an object")
	assert.Equal(t, "stored-resp-1", sar["id"])
}

// TestSetGETImpExtField_InvalidExistingExt verifies that malformed imp.ext is
// reported instead of being silently discarded.
func TestSetGETImpExtField_InvalidExistingExt(t *testing.T) {
	_, err := setGETImpExtField(json.RawMessage(`{not json`), "prebid", "storedauctionresponse", map[string]string{"id": "a"})
	assert.Error(t, err)
}

// TestSetGETImpExtField_NonObjectOuterKey verifies we do not clobber a conflicting
// non-object value at imp.ext.prebid.
func TestSetGETImpExtField_NonObjectOuterKey(t *testing.T) {
	_, err := setGETImpExtField(json.RawMessage(`{"prebid":"scalar"}`), "prebid", "storedauctionresponse", map[string]string{"id": "a"})
	assert.Error(t, err)
}

// TestSetGETImpExtField_PreservesUnrelatedKeys verifies sibling keys survive a merge.
func TestSetGETImpExtField_PreservesUnrelatedKeys(t *testing.T) {
	out, err := setGETImpExtField(json.RawMessage(`{"bidder":{"x":1},"prebid":{"keep":"me"}}`), "prebid", "storedauctionresponse", map[string]string{"id": "a"})
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(out, &m))
	assert.Contains(t, m, "bidder")

	prebid, ok := m["prebid"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "me", prebid["keep"])
	assert.Equal(t, map[string]interface{}{"id": "a"}, prebid["storedauctionresponse"])
}

// parseGETResultWithHeaders parses a GET request with both query params and headers.
func parseGETResultWithHeaders(t *testing.T, rawQuery string, headers map[string]string) map[string]interface{} {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/openrtb2/auction?"+rawQuery, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	data, err := parseGETRequest(req, 0)
	require.NoError(t, err)
	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &out))
	return out
}

// getDevice extracts the device object from a parsed bid-request map.
func getDevice(t *testing.T, m map[string]interface{}) map[string]interface{} {
	t.Helper()
	raw, ok := m["device"]
	require.True(t, ok, "device missing")
	dev, ok := raw.(map[string]interface{})
	require.True(t, ok, "device not a map")
	return dev
}

// TestParseGETRequest_RequiredDeviceHeaders covers the Audio Req12-14 mandatory
// headers: X-Device-IP -> device.ip and X-Device-User-Agent -> device.ua.
func TestParseGETRequest_RequiredDeviceHeaders(t *testing.T) {
	m := parseGETResultWithHeaders(t, "srid=test-req", map[string]string{
		"X-Device-IP":         "203.0.113.10",
		"X-Device-User-Agent": "AudioPlayer/2.1 (Roku)",
	})
	dev := getDevice(t, m)
	assert.Equal(t, "203.0.113.10", dev["ip"])
	assert.Equal(t, "AudioPlayer/2.1 (Roku)", dev["ua"])
}

// TestParseGETRequest_OptionalDeviceHeaders covers make/model/os plus the player
// header which maps to getImpOverride.displaymanager (not the device object).
func TestParseGETRequest_OptionalDeviceHeaders(t *testing.T) {
	m := parseGETResultWithHeaders(t, "srid=test-req", map[string]string{
		"X-Device-Make":   "Roku",
		"X-Device-Model":  "Ultra",
		"X-Device-Os":     "RokuOS",
		"X-Device-Player": "SuperPlayer 4.2",
	})
	dev := getDevice(t, m)
	assert.Equal(t, "Roku", dev["make"])
	assert.Equal(t, "Ultra", dev["model"])
	assert.Equal(t, "RokuOS", dev["os"])

	override := getImpOverrideMap(t, m)
	require.NotNil(t, override, "getImpOverride missing — X-Device-Player should be there")
	assert.Equal(t, "SuperPlayer 4.2", override["displaymanager"])
}

// TestParseGETRequest_HeadersOverrideQueryParams asserts the Tech Response 3.1
// rule 4 precedence: X-Device-* headers win over conflicting query string values.
func TestParseGETRequest_HeadersOverrideQueryParams(t *testing.T) {
	m := parseGETResultWithHeaders(t, "srid=test-req&ua=QueryAgent/1.0", map[string]string{
		"X-Device-User-Agent": "HeaderAgent/2.0",
	})
	dev := getDevice(t, m)
	assert.Equal(t, "HeaderAgent/2.0", dev["ua"], "X-Device-User-Agent must override query param")
}

// TestParseGETRequest_QueryUsedWhenHeaderAbsent verifies the query value survives
// when no corresponding header is supplied.
func TestParseGETRequest_QueryUsedWhenHeaderAbsent(t *testing.T) {
	m := parseGETResultWithHeaders(t, "srid=test-req&ua=QueryAgent/1.0", nil)
	dev := getDevice(t, m)
	assert.Equal(t, "QueryAgent/1.0", dev["ua"])
}

// TestParseGETRequest_QueryUASurvivesTransportUserAgent verifies the three-tier
// precedence for device.ua: a query param ?ua= must not be overwritten by the
// transport-layer User-Agent header.  In SSAI/CTV the stitcher's UA appears on
// the wire, not the viewer's, so the explicitly supplied query param is the more
// trustworthy source.
func TestParseGETRequest_QueryUASurvivesTransportUserAgent(t *testing.T) {
	m := parseGETResultWithHeaders(t, "srid=test-req&ua=ViewerPlayer/3.0", map[string]string{
		"User-Agent": "SSAIStitcher/1.0",
	})
	dev := getDevice(t, m)
	assert.Equal(t, "ViewerPlayer/3.0", dev["ua"],
		"transport User-Agent must not overwrite query param ua when X-Device-User-Agent is absent")
}

// TestParseGETRequest_XForwardedForFallbackUsedWhenNoQueryIP verifies that
// X-Forwarded-For is still used as a fallback when neither X-Device-IP nor a
// query ip is present.
func TestParseGETRequest_XForwardedForFallbackUsedWhenNoQueryIP(t *testing.T) {
	m := parseGETResultWithHeaders(t, "srid=test-req", map[string]string{
		"X-Forwarded-For": "203.0.113.55",
	})
	dev := getDevice(t, m)
	assert.Equal(t, "203.0.113.55", dev["ip"],
		"X-Forwarded-For should still be used as a fallback when no X-Device-IP or query ip is present")
}

// TestParseGETRequest_XDeviceIPBeatsProxyHeaders verifies the explicit device
// header is preferred over proxy-populated forwarding headers.
func TestParseGETRequest_XDeviceIPBeatsProxyHeaders(t *testing.T) {
	m := parseGETResultWithHeaders(t, "srid=test-req", map[string]string{
		"X-Device-IP":     "203.0.113.10",
		"X-Forwarded-For": "198.51.100.7",
		"X-Real-IP":       "198.51.100.8",
	})
	dev := getDevice(t, m)
	assert.Equal(t, "203.0.113.10", dev["ip"])
}

// TestParseGETRequest_ForwardedForChain verifies only the originating client IP
// is taken from a comma-separated X-Forwarded-For chain.
func TestParseGETRequest_ForwardedForChain(t *testing.T) {
	m := parseGETResultWithHeaders(t, "srid=test-req", map[string]string{
		"X-Forwarded-For": "198.51.100.7, 10.0.0.1, 10.0.0.2",
	})
	dev := getDevice(t, m)
	assert.Equal(t, "198.51.100.7", dev["ip"])
}

// TestParseGETRequest_IPv6Header verifies an IPv6 device header lands on
// device.ipv6 rather than device.ip.
func TestParseGETRequest_IPv6Header(t *testing.T) {
	m := parseGETResultWithHeaders(t, "srid=test-req", map[string]string{
		"X-Device-IP": "2001:db8::1",
	})
	dev := getDevice(t, m)
	assert.Equal(t, "2001:db8::1", dev["ipv6"])
	assert.NotContains(t, dev, "ip")
}

// TestParseGETRequest_MalformedIPHeaderIgnored verifies an unparsable IP is
// dropped rather than written through to the bid request.
func TestParseGETRequest_MalformedIPHeaderIgnored(t *testing.T) {
	m := parseGETResultWithHeaders(t, "srid=test-req", map[string]string{
		"X-Device-IP": "not-an-ip",
	})
	if raw, ok := m["device"]; ok {
		dev := raw.(map[string]interface{})
		assert.NotContains(t, dev, "ip")
		assert.NotContains(t, dev, "ipv6")
	}
}

// TestParseGETRequest_PlayerHeaderWithoutImpIsSafe verifies that X-Device-Player
// goes into the imp override (not a direct imp mutation), so there is no panic
// when the header arrives on a request with no other imp params.
func TestParseGETRequest_PlayerHeaderWithoutImpIsSafe(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/openrtb2/auction?srid=test-req", nil)
	req.Header.Set("X-Device-Player", "SuperPlayer 4.2")
	assert.NotPanics(t, func() {
		data, err := parseGETRequest(req, 0)
		require.NoError(t, err)
		var m map[string]interface{}
		require.NoError(t, json.Unmarshal(data, &m))
		override := getImpOverrideMap(t, m)
		require.NotNil(t, override)
		assert.Equal(t, "SuperPlayer 4.2", override["displaymanager"])
	})
}

// TestParseGETRequest_AppliesOverridesToStoredRequest is the integration test from
// the PR reviewer that exercises both bugs together:
//  1. stored imp fields (id, mimes, ext/bidder params) must survive the merge
//  2. coppa=0 must override the stored coppa=1 (not be silently dropped by omitempty)
func TestParseGETRequest_AppliesOverridesToStoredRequest(t *testing.T) {
	stored := json.RawMessage(`{"regs":{"coppa":1},"imp":[{"id":"stored-imp",` +
		`"video":{"mimes":["video/mp4"]},` +
		`"ext":{"prebid":{"bidder":{"appnexus":{"placementId":123}}}}}]}`)

	requestJSON, err := parseGETRequest(httptest.NewRequest(http.MethodGet,
		"/openrtb2/auction?srid=x&mtype=2&w=640&coppa=0", nil), 0)
	require.NoError(t, err)

	// Simulate processStoredRequests: stored is base, GET request is patch.
	merged, err := jsonpatch.MergePatch(stored, requestJSON)
	require.NoError(t, err)

	// Simulate auction.go post-merge step.
	merged, err = applyGETImpOverrideJSON(merged)
	require.NoError(t, err)

	var request openrtb2.BidRequest
	require.NoError(t, json.Unmarshal(merged, &request))
	require.Len(t, request.Imp, 1)
	require.NotNil(t, request.Regs)

	assert.Equal(t, "stored-imp", request.Imp[0].ID, "stored imp id must survive")
	require.NotNil(t, request.Imp[0].Video)
	assert.Equal(t, []string{"video/mp4"}, request.Imp[0].Video.MIMEs, "stored mimes must survive")
	assert.Contains(t, string(request.Imp[0].Ext), "placementId", "stored bidder params must survive")
	assert.EqualValues(t, 0, request.Regs.COPPA, "coppa=0 must override stored coppa=1")

	// GET-specific field must be applied
	require.NotNil(t, request.Imp[0].Video.W)
	assert.EqualValues(t, 640, *request.Imp[0].Video.W)
}

// --- Query string length limit (Tech Response 3.1, consideration 1) ---

// TestParseGETRequest_MaxInitialLineLength verifies the request line cap that guards
// against malicious resource exhaustion attacks via oversized query strings.
func TestParseGETRequest_MaxInitialLineLength(t *testing.T) {
	testCases := []struct {
		name                 string
		rawQuery             string
		maxInitialLineLength int
		expectedErr          string
	}{
		{
			name:                 "limit disabled accepts long query string",
			rawQuery:             "srid=test-req&kv=" + strings.Repeat("a", 20000),
			maxInitialLineLength: 0,
		},
		{
			name:                 "query string within limit",
			rawQuery:             "srid=test-req",
			maxInitialLineLength: 8192,
		},
		{
			name:                 "query string over limit is rejected",
			rawQuery:             "srid=test-req&kv=" + strings.Repeat("a", 9000),
			maxInitialLineLength: 8192,
			expectedErr:          "request line exceeded max size of 8192 bytes",
		},
		{
			name:                 "negative limit behaves as disabled",
			rawQuery:             "srid=test-req&kv=" + strings.Repeat("a", 9000),
			maxInitialLineLength: -1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/openrtb2/auction?"+test.rawQuery, nil)

			result, err := parseGETRequest(req, test.maxInitialLineLength)

			if test.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectedErr)
				assert.Nil(t, result)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, result)
		})
	}
}

// --- Single impression assumption (Tech Response 3.1) ---

// TestEnforceSingleImp verifies that impressions after the first are discarded, since
// the GET interface assumes exactly one impression per request.
func TestEnforceSingleImp(t *testing.T) {
	testCases := []struct {
		name         string
		imps         []openrtb2.Imp
		expectedImps []string
	}{
		{
			name:         "nil imps untouched",
			imps:         nil,
			expectedImps: nil,
		},
		{
			name:         "single imp untouched",
			imps:         []openrtb2.Imp{{ID: "imp-1"}},
			expectedImps: []string{"imp-1"},
		},
		{
			name:         "extra imps discarded",
			imps:         []openrtb2.Imp{{ID: "imp-1"}, {ID: "imp-2"}, {ID: "imp-3"}},
			expectedImps: []string{"imp-1"},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			httpReq := httptest.NewRequest(http.MethodGet, "/openrtb2/auction?srid=test-req", nil)
			httpReq.Header.Set("Referer", "https://publisher.example.com/show")
			bidReq := &openrtb2.BidRequest{ID: "req-id", Imp: test.imps}

			enforceSingleImp(httpReq, bidReq, "test-account")

			actualImps := make([]string, 0, len(bidReq.Imp))
			for _, imp := range bidReq.Imp {
				actualImps = append(actualImps, imp.ID)
			}
			if test.expectedImps == nil {
				assert.Empty(t, actualImps)
				return
			}
			assert.Equal(t, test.expectedImps, actualImps)
		})
	}
}

// TestEnforceSingleImpNilRequest guards against a panic when the bid request is nil.
func TestEnforceSingleImpNilRequest(t *testing.T) {
	httpReq := httptest.NewRequest(http.MethodGet, "/openrtb2/auction?srid=test-req", nil)
	assert.NotPanics(t, func() {
		enforceSingleImp(httpReq, nil, "test-account")
	})
}
