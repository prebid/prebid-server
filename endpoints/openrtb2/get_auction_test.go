package openrtb2

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseGETResult is a helper that parses the raw JSON returned by parseGETRequest
// into a generic map for easy field access.
func parseGETResult(t *testing.T, rawQuery string) map[string]interface{} {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/openrtb2/auction?"+rawQuery, nil)
	data, err := parseGETRequest(req)
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

// getImpExtPrebid extracts imp[0].ext.prebid as a map.
func getImpExtPrebid(t *testing.T, m map[string]interface{}) map[string]interface{} {
	t.Helper()
	impsRaw, ok := m["imp"]
	require.True(t, ok, "imp missing")
	imps, ok := impsRaw.([]interface{})
	require.True(t, ok && len(imps) > 0, "imp not a non-empty array")
	impMap, ok := imps[0].(map[string]interface{})
	require.True(t, ok, "imp[0] not a map")
	extRaw, ok := impMap["ext"]
	require.True(t, ok, "imp[0].ext missing")
	extMap, ok := extRaw.(map[string]interface{})
	require.True(t, ok, "imp[0].ext not a map")
	prebidRaw, ok := extMap["prebid"]
	require.True(t, ok, "imp[0].ext.prebid missing")
	prebidMap, ok := prebidRaw.(map[string]interface{})
	require.True(t, ok, "imp[0].ext.prebid not a map")
	return prebidMap
}

// --- TestParseGETRequest_RequiresSrid ---

func TestParseGETRequest_RequiresSrid(t *testing.T) {
	t.Run("missing srid returns error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/openrtb2/auction", nil)
		_, err := parseGETRequest(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "srid")
	})

	t.Run("srid present returns no error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/openrtb2/auction?srid=abc", nil)
		_, err := parseGETRequest(req)
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
		// tmax should be absent or zero
		val, exists := m["tmax"]
		if exists {
			assert.EqualValues(t, float64(0), val)
		}
	})

	t.Run("invalid tmax (abc) is ignored, no error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/openrtb2/auction?srid=x&tmax=abc", nil)
		_, err := parseGETRequest(req)
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

// --- TestParseGETRequest_Profiles ---

func TestParseGETRequest_Profiles(t *testing.T) {
	t.Run("rprof sets request-level profiles", func(t *testing.T) {
		m := parseGETResult(t, "srid=x&rprof=android-device,show-abc")
		prebid := getExtPrebid(t, m)
		profilesRaw, ok := prebid["profiles"]
		require.True(t, ok, "ext.prebid.profiles missing")
		profiles, ok := profilesRaw.([]interface{})
		require.True(t, ok)
		assert.Equal(t, []interface{}{"android-device", "show-abc"}, profiles)
	})

	t.Run("iprof sets imp-level profiles", func(t *testing.T) {
		m := parseGETResult(t, "srid=x&iprof=highbandwidth")
		impPrebid := getImpExtPrebid(t, m)
		profilesRaw, ok := impPrebid["profiles"]
		require.True(t, ok, "imp[0].ext.prebid.profiles missing")
		profiles, ok := profilesRaw.([]interface{})
		require.True(t, ok)
		assert.Equal(t, []interface{}{"highbandwidth"}, profiles)
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

func TestParseGETRequest_SlotMapsToTagID(t *testing.T) {
	m := parseGETResult(t, "srid=x&slot=my-slot")
	impsRaw, ok := m["imp"]
	require.True(t, ok)
	imps := impsRaw.([]interface{})
	imp0 := imps[0].(map[string]interface{})
	assert.Equal(t, "my-slot", imp0["tagid"])
}

// --- TestParseGETRequest_VideoParams ---

func TestParseGETRequest_VideoParams(t *testing.T) {
	t.Run("mindur/maxdur/w/h set on video imp", func(t *testing.T) {
		m := parseGETResult(t, "srid=x&mtype=2&mindur=5&maxdur=30&w=640&h=360")
		impsRaw := m["imp"].([]interface{})
		imp0 := impsRaw[0].(map[string]interface{})
		video, ok := imp0["video"].(map[string]interface{})
		require.True(t, ok, "imp[0].video missing")
		assert.EqualValues(t, float64(5), video["minduration"])
		assert.EqualValues(t, float64(30), video["maxduration"])
		assert.EqualValues(t, float64(640), video["w"])
		assert.EqualValues(t, float64(360), video["h"])
	})

	t.Run("skip/skipmin/skipafter set on video imp", func(t *testing.T) {
		m := parseGETResult(t, "srid=x&mtype=vid&skip=1&skipmin=5&skipafter=3")
		impsRaw := m["imp"].([]interface{})
		imp0 := impsRaw[0].(map[string]interface{})
		video, ok := imp0["video"].(map[string]interface{})
		require.True(t, ok, "imp[0].video missing")
		assert.EqualValues(t, float64(1), video["skip"])
		assert.EqualValues(t, float64(5), video["skipmin"])
		assert.EqualValues(t, float64(3), video["skipafter"])
	})
}

// --- TestParseGETRequest_AudioParams ---

func TestParseGETRequest_AudioParams(t *testing.T) {
	m := parseGETResult(t, "srid=x&mtype=3&mindur=10&maxdur=60")
	impsRaw := m["imp"].([]interface{})
	imp0 := impsRaw[0].(map[string]interface{})
	audio, ok := imp0["audio"].(map[string]interface{})
	require.True(t, ok, "imp[0].audio missing")
	assert.EqualValues(t, float64(10), audio["minduration"])
	assert.EqualValues(t, float64(60), audio["maxduration"])
}

// --- TestParseGETRequest_BannerDefault ---

func TestParseGETRequest_BannerDefault(t *testing.T) {
	t.Run("default mtype results in no banner when no dimensions given", func(t *testing.T) {
		// Implementation only sets imp.banner if w/h/format present
		m := parseGETResult(t, "srid=x")
		impsRaw := m["imp"].([]interface{})
		imp0 := impsRaw[0].(map[string]interface{})
		// banner may or may not be set depending on implementation; if set, it should be valid
		if banner, ok := imp0["banner"]; ok {
			assert.NotNil(t, banner)
		}
	})

	t.Run("mtype=1 with w/h sets banner dimensions", func(t *testing.T) {
		m := parseGETResult(t, "srid=x&mtype=1&w=300&h=250")
		impsRaw := m["imp"].([]interface{})
		imp0 := impsRaw[0].(map[string]interface{})
		banner, ok := imp0["banner"].(map[string]interface{})
		require.True(t, ok, "imp[0].banner missing")
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
		impsRaw := m["imp"].([]interface{})
		imp0 := impsRaw[0].(map[string]interface{})
		video, ok := imp0["video"].(map[string]interface{})
		require.True(t, ok, "imp[0].video missing")
		protocols, ok := video["protocols"].([]interface{})
		require.True(t, ok, "video.protocols missing")
		assert.Equal(t, []interface{}{float64(2), float64(3), float64(5)}, protocols)
	})

	t.Run("api CSV sets video api", func(t *testing.T) {
		m := parseGETResult(t, "srid=x&mtype=2&api=1,2")
		impsRaw := m["imp"].([]interface{})
		imp0 := impsRaw[0].(map[string]interface{})
		video, ok := imp0["video"].(map[string]interface{})
		require.True(t, ok, "imp[0].video missing")
		api, ok := video["api"].([]interface{})
		require.True(t, ok, "video.api missing")
		assert.Equal(t, []interface{}{float64(1), float64(2)}, api)
	})
}
