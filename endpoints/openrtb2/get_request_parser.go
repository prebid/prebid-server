package openrtb2

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/config/util"
	"github.com/prebid/prebid-server/v4/logger"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
	jsonpatch "gopkg.in/evanphx/json-patch.v5"
)

// getMultiImpLogSampleRate is the fraction of discarded-impression events written to the
// application log. The condition is not expected in normal operation, but a misconfigured
// stored request can trigger it on every request, so the log entry is sampled.
const getMultiImpLogSampleRate = 0.01

// enforceSingleImp implements the GET interface assumption that a request carries exactly
// one impression. If merging the stored request yields more than one imp, every
// imp after the first is discarded and a sampled log entry is emitted with the referrer and
// account so the misconfiguration can be traced back to its publisher.
func enforceSingleImp(httpRequest *http.Request, req *openrtb2.BidRequest, accountID string) {
	if req == nil || len(req.Imp) <= 1 {
		return
	}

	discarded := len(req.Imp) - 1
	req.Imp = req.Imp[:1]

	util.LogRandomSample(
		fmt.Sprintf(
			"GET /openrtb2/auction resolved to %d impressions; discarded %d after the first. referrer=%q account=%q",
			discarded+1, discarded, httpRequest.Referer(), accountID,
		),
		logger.Warnf,
		getMultiImpLogSampleRate,
	)
}

// parseGETRequest builds an OpenRTB BidRequest JSON from HTTP GET query parameters.
// The stored request ID (srid) is required — without it we cannot know the auction structure.
// All other parameters are optional and overlay on top of the stored request.
//
// Parameter precedence (lowest → highest):
//  1. Stored request (loaded later in the normal parseRequest / processStoredRequests flow)
//  2. Individual GET query params mapped to OpenRTB fields
//  3. X-Device-* HTTP headers override all conflicting values (Tech Response §3.1 rule 4).
//
// Standard transport headers (User-Agent, X-Forwarded-For, …) sit below query
// params in the precedence order: they are only used when neither an X-Device-*
// header nor a query param set the field.
//
// Imp-level params (video/audio/banner dimensions, slot, sarid, displaymanager) are
// stored in ext.prebid.getImpOverride and applied to each imp AFTER processStoredRequests
// via applyGETImpOverrideJSON. This preserves stored imp fields (id, ext, mimes) that
// would otherwise be lost when the imp array is replaced during JSON Merge Patch.
//
// Request-level params are built as a sparse map so that explicit zero values (e.g.
// coppa=0) survive JSON marshalling and correctly override stored request values.
func parseGETRequest(r *http.Request, maxInitialLineLength int) ([]byte, error) {
	if maxInitialLineLength > 0 {
		if lineLength := len(r.Method) + 1 + len(r.URL.RequestURI()) + 1 + len(r.Proto); lineLength > maxInitialLineLength {
			return nil, fmt.Errorf("request line exceeded max size of %d bytes", maxInitialLineLength)
		}
	}

	q := r.URL.Query()

	srid := qFirst(q, "srid")
	if srid == "" {
		return nil, fmt.Errorf("GET /openrtb2/auction requires 'srid' (stored request ID) query parameter")
	}

	reqMap := map[string]interface{}{}

	// ext.prebid skeleton
	prebidMap := map[string]interface{}{
		"storedrequest": map[string]interface{}{"id": srid},
		"server":        map[string]interface{}{"http_method": "GET"},
	}
	if of := qFirst(q, "of"); of != "" {
		prebidMap["of"] = of
	}
	if om := qFirst(q, "om"); om != "" {
		prebidMap["om"] = om
	}
	if d := qFirst(q, "debug"); d == "1" || d == "true" {
		prebidMap["debug"] = true
	}

	// tmax
	if tmaxStr := qFirst(q, "tmax"); tmaxStr != "" {
		if tmax, err := strconv.ParseInt(tmaxStr, 10, 64); err == nil && tmax >= 100 {
			reqMap["tmax"] = tmax
		}
	}

	// Privacy params (sparse — coppa=0 is preserved)
	applyGETPrivacyParamsToMap(q, reqMap)

	// Publisher ID
	if pubid := qFirst(q, "pubid"); pubid != "" {
		site, _ := reqMap["site"].(map[string]interface{})
		if site == nil {
			site = map[string]interface{}{}
			reqMap["site"] = site
		}
		pub, _ := site["publisher"].(map[string]interface{})
		if pub == nil {
			pub = map[string]interface{}{}
			site["publisher"] = pub
		}
		pub["id"] = pubid
	}

	// Content params
	applyGETContentParamsToMap(q, reqMap)

	// Blocking
	if bcat := qCSV(q, "bcat"); len(bcat) > 0 {
		reqMap["bcat"] = bcat
	}
	if badv := qCSV(q, "badv"); len(badv) > 0 {
		reqMap["badv"] = badv
	}

	// Imp override — not placed directly in imp array to avoid RFC 7396 array replacement.
	// Applied to each imp after processStoredRequests by applyGETImpOverrideJSON.
	impOverride, err := buildImpOverrideFromGET(r.Header, q)
	if err != nil {
		return nil, err
	}
	if len(impOverride) > 0 {
		prebidMap["getImpOverride"] = json.RawMessage(impOverride)
	}

	// Header params (device fields; X-Device-Player goes into imp override above)
	applyGETHeaderParamsToMap(r.Header, reqMap)

	reqMap["ext"] = map[string]interface{}{"prebid": prebidMap}

	return json.Marshal(reqMap)
}

// applyGETPrivacyParamsToMap writes privacy-related query params into reqMap as a
// sparse map. Using maps (rather than marshalling structs) ensures that explicit
// zero values like coppa=0 appear in the JSON output and correctly override stored
// request values during the MergePatch merge.
func applyGETPrivacyParamsToMap(q url.Values, reqMap map[string]interface{}) {
	regsMap := map[string]interface{}{}

	if gdpr := qInt(q, "gdpr", "gdpr_applies"); gdpr >= 0 {
		regsMap["gdpr"] = gdpr
	}
	if gpp := qFirst(q, "gppc"); gpp != "" {
		regsMap["gpp"] = gpp
	}
	if gpps := qCSV(q, "gpps"); len(gpps) > 0 {
		var ids []int
		for _, s := range gpps {
			if i := qParseInt(s); i > 0 {
				ids = append(ids, i)
			}
		}
		if len(ids) > 0 {
			regsMap["gppsid"] = ids
		}
	}
	if coppa := qInt(q, "coppa"); coppa >= 0 {
		regsMap["coppa"] = coppa
	}
	if usp := qFirst(q, "usp"); usp != "" {
		regsMap["us_privacy"] = usp
	}
	if len(regsMap) > 0 {
		reqMap["regs"] = regsMap
	}

	if consent := qFirst(q, "gdpr_consent", "consent_string", "tcfc", "cs"); consent != "" {
		userMap, _ := reqMap["user"].(map[string]interface{})
		if userMap == nil {
			userMap = map[string]interface{}{}
		}
		userMap["consent"] = consent
		reqMap["user"] = userMap
	}

	deviceMap, _ := reqMap["device"].(map[string]interface{})
	if deviceMap == nil {
		deviceMap = map[string]interface{}{}
	}
	if dnt := qInt(q, "dnt"); dnt >= 0 {
		deviceMap["dnt"] = dnt
	}
	if lmt := qInt(q, "lmt"); lmt >= 0 {
		deviceMap["lmt"] = lmt
	}
	if ifa := qFirst(q, "ifa"); ifa != "" {
		deviceMap["ifa"] = ifa
	}
	if ua := qFirst(q, "ua"); ua != "" {
		deviceMap["ua"] = ua
	}
	if dtype := qFirst(q, "dtype"); dtype != "" {
		if dt := qParseInt(dtype); dt > 0 {
			deviceMap["devicetype"] = dt
		}
	}
	if len(deviceMap) > 0 {
		reqMap["device"] = deviceMap
	}
}

// applyGETContentParamsToMap writes content-related query params into the
// site.content or app.content sub-object in reqMap.
func applyGETContentParamsToMap(q url.Values, reqMap map[string]interface{}) {
	contentMap := map[string]interface{}{}

	if genre := qFirst(q, "cgenre"); genre != "" {
		contentMap["genre"] = genre
	}
	if lang := qFirst(q, "clang"); lang != "" {
		contentMap["language"] = lang
	}
	if rating := qFirst(q, "crating"); rating != "" {
		contentMap["contentrating"] = rating
	}
	if title := qFirst(q, "ctitle"); title != "" {
		contentMap["title"] = title
	}
	if series := qFirst(q, "cseries"); series != "" {
		contentMap["series"] = series
	}
	if curl := qFirst(q, "curl", "url_override"); curl != "" {
		contentMap["url"] = curl
	}
	if livestream := qInt(q, "clivestream"); livestream >= 0 {
		contentMap["livestream"] = livestream
	}

	if len(contentMap) == 0 {
		return
	}

	if site, ok := reqMap["site"].(map[string]interface{}); ok {
		site["content"] = contentMap
	} else if app, ok := reqMap["app"].(map[string]interface{}); ok {
		app["content"] = contentMap
	} else {
		reqMap["site"] = map[string]interface{}{"content": contentMap}
	}
}

// applyGETHeaderParamsToMap writes X-Device-* header values into the device sub-map
// of reqMap, applying three-tier precedence for ua and ip:
//  1. X-Device-User-Agent / X-Device-IP — always override, including query params.
//  2. Query params (already in deviceMap) — survive when X-Device-* is absent.
//  3. Standard transport headers (User-Agent, X-Forwarded-For, …) — fallback only.
//
// X-Device-Player is handled in buildImpOverrideFromGET (imp-level field).
func applyGETHeaderParamsToMap(h http.Header, reqMap map[string]interface{}) {
	deviceMap, _ := reqMap["device"].(map[string]interface{})
	if deviceMap == nil {
		deviceMap = map[string]interface{}{}
	}

	// ua — three-tier
	if xdua := strings.TrimSpace(h.Get("X-Device-User-Agent")); xdua != "" {
		deviceMap["ua"] = xdua
	} else if _, hasUA := deviceMap["ua"]; !hasUA {
		if ua := strings.TrimSpace(h.Get("User-Agent")); ua != "" {
			deviceMap["ua"] = ua
		}
	}

	// ip — three-tier
	applyGETIPToMap(h, deviceMap)

	if make := firstHeader(h, "X-Device-Make"); make != "" {
		deviceMap["make"] = make
	}
	if model := firstHeader(h, "X-Device-Model"); model != "" {
		deviceMap["model"] = model
	}
	if os := firstHeader(h, "X-Device-Os"); os != "" {
		deviceMap["os"] = os
	}

	if len(deviceMap) > 0 {
		reqMap["device"] = deviceMap
	}
}

// applyGETIPToMap applies the three-tier IP precedence directly to a device map.
func applyGETIPToMap(h http.Header, deviceMap map[string]interface{}) {
	if xdip := strings.TrimSpace(h.Get("X-Device-IP")); xdip != "" {
		if comma := strings.IndexByte(xdip, ','); comma >= 0 {
			xdip = xdip[:comma]
		}
		xdip = strings.TrimSpace(xdip)
		if parsed := net.ParseIP(xdip); parsed != nil {
			if parsed.To4() != nil {
				deviceMap["ip"] = xdip
			} else {
				deviceMap["ipv6"] = xdip
			}
		}
		return
	}

	// Transport headers are fallback only; don't overwrite a query-param-set ip.
	_, hasIP := deviceMap["ip"]
	_, hasIPv6 := deviceMap["ipv6"]
	if hasIP || hasIPv6 {
		return
	}

	if ip := firstHeader(h, "X-Forwarded-For", "X-Real-IP", "True-Client-IP"); ip != "" {
		if comma := strings.IndexByte(ip, ','); comma >= 0 {
			ip = ip[:comma]
		}
		ip = strings.TrimSpace(ip)
		if parsed := net.ParseIP(ip); parsed != nil {
			if parsed.To4() != nil {
				deviceMap["ip"] = ip
			} else {
				deviceMap["ipv6"] = ip
			}
		}
	}
}

// buildImpOverrideFromGET builds a sparse imp JSON object containing only the
// query params that were explicitly set. It is stored in ext.prebid.getImpOverride
// and applied to each imp after processStoredRequests, so stored imp fields (id,
// ext with bidder params, mimes) are preserved and only the GET-specific fields
// (w, h, etc.) are overlaid via RFC 7396 recursive object merge.
func buildImpOverrideFromGET(h http.Header, q url.Values) (json.RawMessage, error) {
	impMap := map[string]interface{}{}

	if slot := qFirst(q, "slot"); slot != "" {
		impMap["tagid"] = slot
	}

	if sarid := qFirst(q, "sarid"); sarid != "" {
		ext, err := setGETImpExtField(nil, "prebid", "storedauctionresponse", map[string]string{"id": sarid})
		if err != nil {
			return nil, err
		}
		impMap["ext"] = json.RawMessage(ext)
	}

	mtype := qFirst(q, "mtype")
	switch mtype {
	case "2", "vid":
		if vm := buildSparseVideoParams(q); len(vm) > 0 {
			impMap["video"] = vm
		}
	case "3", "aud":
		if am := buildSparseAudioParams(q); len(am) > 0 {
			impMap["audio"] = am
		}
	default:
		if bm := buildSparseBannerParams(q); len(bm) > 0 {
			impMap["banner"] = bm
		}
	}

	if player := strings.TrimSpace(h.Get("X-Device-Player")); player != "" {
		impMap["displaymanager"] = player
	}

	if len(impMap) == 0 {
		return nil, nil
	}
	return json.Marshal(impMap)
}

func buildSparseVideoParams(q url.Values) map[string]interface{} {
	m := map[string]interface{}{}
	if v := qInt(q, "mindur"); v > 0 {
		m["minduration"] = v
	}
	if v := qInt(q, "maxdur"); v > 0 {
		m["maxduration"] = v
	}
	if v := qInt(q, "w"); v > 0 {
		m["w"] = v
	}
	if v := qInt(q, "h"); v > 0 {
		m["h"] = v
	}
	if v := qInt(q, "skip"); v >= 0 {
		m["skip"] = v
	}
	if v := qInt(q, "skipmin"); v > 0 {
		m["skipmin"] = v
	}
	if v := qInt(q, "skipafter"); v > 0 {
		m["skipafter"] = v
	}
	if v := qInt(q, "startdelay"); v != -1 {
		m["startdelay"] = v
	}
	if v := qInt(q, "linearity"); v > 0 {
		m["linearity"] = v
	}
	if v := qInt(q, "placement"); v > 0 {
		m["placement"] = v
	}
	if v := qInt(q, "plcmt"); v > 0 {
		m["plcmt"] = v
	}
	if v := qInt(q, "pos"); v >= 0 {
		m["pos"] = v
	}
	if v := qInt(q, "poddur"); v > 0 {
		m["poddur"] = v
	}
	if v := qFirst(q, "podid"); v != "" {
		m["podid"] = v
	}
	if v := qInt(q, "podseq"); v != -1 {
		m["podseq"] = v
	}
	if v := qInt(q, "seq"); v > 0 {
		m["sequence"] = v
	}
	if v := qInt(q, "slotinpod"); v != -1 {
		m["slotinpod"] = v
	}
	if v := qInt(q, "minbr"); v > 0 {
		m["minbitrate"] = v
	}
	if v := qInt(q, "maxbr"); v > 0 {
		m["maxbitrate"] = v
	}
	if v := qInt(q, "maxex"); v != -1 {
		m["maxextended"] = v
	}
	if v := qInt(q, "playbackend"); v > 0 {
		m["playbackend"] = v
	}
	if v := qInt(q, "boxingallowed"); v >= 0 {
		m["boxingallowed"] = v
	}
	if v := qInt(q, "maxseq"); v > 0 {
		m["maxseq"] = v
	}
	if v := qInt(q, "mincpms"); v > 0 {
		m["mincpmpersec"] = float64(v)
	}
	if v := qCSV(q, "mimes"); len(v) > 0 {
		m["mimes"] = v
	}
	if v := qInts(q, "proto"); len(v) > 0 {
		m["protocols"] = v
	}
	if v := qInts(q, "api"); len(v) > 0 {
		m["api"] = v
	}
	if v := qInts(q, "delivery"); len(v) > 0 {
		m["delivery"] = v
	}
	if v := qInts(q, "battr"); len(v) > 0 {
		m["battr"] = v
	}
	if v := qInts(q, "playbackmethod"); len(v) > 0 {
		m["playbackmethod"] = v
	}
	if v := qInts(q, "rqddurs"); len(v) > 0 {
		m["rqddurs"] = v
	}
	return m
}

func buildSparseAudioParams(q url.Values) map[string]interface{} {
	m := map[string]interface{}{}
	if v := qInt(q, "mindur"); v > 0 {
		m["minduration"] = v
	}
	if v := qInt(q, "maxdur"); v > 0 {
		m["maxduration"] = v
	}
	if v := qInt(q, "minbr"); v > 0 {
		m["minbitrate"] = v
	}
	if v := qInt(q, "maxbr"); v > 0 {
		m["maxbitrate"] = v
	}
	if v := qInt(q, "maxseq"); v > 0 {
		m["maxseq"] = v
	}
	if v := qInt(q, "stitched"); v >= 0 {
		m["stitched"] = v
	}
	if v := qInt(q, "feed"); v > 0 {
		m["feed"] = v
	}
	if v := qInt(q, "nvol"); v > 0 {
		m["nvol"] = v
	}
	if v := qCSV(q, "mimes"); len(v) > 0 {
		m["mimes"] = v
	}
	if v := qInts(q, "api"); len(v) > 0 {
		m["api"] = v
	}
	if v := qInts(q, "delivery"); len(v) > 0 {
		m["delivery"] = v
	}
	if v := qInts(q, "battr"); len(v) > 0 {
		m["battr"] = v
	}
	if v := qInts(q, "proto"); len(v) > 0 {
		m["protocols"] = v
	}
	if v := qInt(q, "startdelay"); v != -1 {
		m["startdelay"] = v
	}
	if v := qInt(q, "poddur"); v > 0 {
		m["poddur"] = v
	}
	if v := qFirst(q, "podid"); v != "" {
		m["podid"] = v
	}
	if v := qInt(q, "podseq"); v != -1 {
		m["podseq"] = v
	}
	if v := qInt(q, "seq"); v > 0 {
		m["sequence"] = v
	}
	if v := qInt(q, "slotinpod"); v != -1 {
		m["slotinpod"] = v
	}
	if v := qInt(q, "mincpms"); v > 0 {
		m["mincpmpersec"] = float64(v)
	}
	if v := qInts(q, "rqddurs"); len(v) > 0 {
		m["rqddurs"] = v
	}
	return m
}

func buildSparseBannerParams(q url.Values) map[string]interface{} {
	m := map[string]interface{}{}
	if v := qInt(q, "w"); v > 0 {
		m["w"] = v
	}
	if v := qInt(q, "h"); v > 0 {
		m["h"] = v
	}
	if v := qInt(q, "pos"); v >= 0 {
		m["pos"] = v
	}
	if v := qInt(q, "topframe"); v >= 0 {
		m["topframe"] = v
	}
	if v := qInts(q, "battr"); len(v) > 0 {
		m["battr"] = v
	}
	if v := qInts(q, "btype"); len(v) > 0 {
		m["btype"] = v
	}
	if v := qInts(q, "expdir"); len(v) > 0 {
		m["expdir"] = v
	}
	if v := qCSV(q, "mimes"); len(v) > 0 {
		m["mimes"] = v
	}
	if v := qInts(q, "api"); len(v) > 0 {
		m["api"] = v
	}
	return m
}

// applyGETImpOverrideJSON extracts ext.prebid.getImpOverride from requestJSON,
// applies it as a MergePatch to every imp in the request, and returns the
// cleaned-up JSON with getImpOverride removed.
//
// Called by auction.go immediately after processStoredRequests for GET requests,
// so stored imp fields survive the top-level merge and GET-specific fields are
// overlaid per RFC 7396 recursive object merging rules.
func applyGETImpOverrideJSON(requestJSON []byte) ([]byte, error) {
	var reqMap map[string]json.RawMessage
	if err := jsonutil.UnmarshalValid(requestJSON, &reqMap); err != nil {
		return nil, err
	}

	extRaw, hasExt := reqMap["ext"]
	if !hasExt {
		return requestJSON, nil
	}
	var extMap map[string]json.RawMessage
	if err := jsonutil.UnmarshalValid(extRaw, &extMap); err != nil {
		return nil, err
	}

	prebidRaw, hasPrebid := extMap["prebid"]
	if !hasPrebid {
		return requestJSON, nil
	}
	var prebidMap map[string]json.RawMessage
	if err := jsonutil.UnmarshalValid(prebidRaw, &prebidMap); err != nil {
		return nil, err
	}

	impOverride, hasOverride := prebidMap["getImpOverride"]
	if !hasOverride {
		return requestJSON, nil
	}

	delete(prebidMap, "getImpOverride")

	impsRaw, hasImps := reqMap["imp"]
	if hasImps && len(impsRaw) > 0 {
		var imps []json.RawMessage
		if err := jsonutil.UnmarshalValid(impsRaw, &imps); err != nil {
			return nil, err
		}
		for i, imp := range imps {
			merged, err := jsonpatch.MergePatch(imp, impOverride)
			if err != nil {
				return nil, fmt.Errorf("applying GET imp override to imp[%d]: %w", i, err)
			}
			imps[i] = merged
		}
		impsBytes, err := json.Marshal(imps)
		if err != nil {
			return nil, err
		}
		reqMap["imp"] = impsBytes
	}

	prebidBytes, err := json.Marshal(prebidMap)
	if err != nil {
		return nil, err
	}
	extMap["prebid"] = prebidBytes

	extBytes, err := json.Marshal(extMap)
	if err != nil {
		return nil, err
	}
	reqMap["ext"] = extBytes

	return json.Marshal(reqMap)
}

// firstHeader returns the first non-empty value among the given header names.
func firstHeader(h http.Header, names ...string) string {
	for _, name := range names {
		if v := strings.TrimSpace(h.Get(name)); v != "" {
			return v
		}
	}
	return ""
}

// setGETImpExtField merges a value into imp.ext at path ext[outerKey][innerKey],
// preserving any keys that are already present. It returns an error rather than
// silently discarding malformed JSON, so the caller can reject the request
// instead of emitting an imp.ext that quietly lost data.
func setGETImpExtField(ext json.RawMessage, outerKey, innerKey string, value interface{}) (json.RawMessage, error) {
	m := map[string]interface{}{}
	if len(ext) > 0 {
		if err := jsonutil.Unmarshal(ext, &m); err != nil {
			return nil, fmt.Errorf("failed to unmarshal imp.ext while setting %s.%s: %w", outerKey, innerKey, err)
		}
	}

	outer, ok := m[outerKey].(map[string]interface{})
	if !ok {
		if existing, present := m[outerKey]; present && existing != nil {
			return nil, fmt.Errorf("imp.ext.%s must be a JSON object to set %s", outerKey, innerKey)
		}
		outer = map[string]interface{}{}
	}

	outer[innerKey] = value
	m[outerKey] = outer

	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal imp.ext while setting %s.%s: %w", outerKey, innerKey, err)
	}
	return b, nil
}

// --- query param helpers ---

// qFirst returns the first non-empty value from the given param names (alias-aware).
func qFirst(q url.Values, names ...string) string {
	for _, name := range names {
		if v := q.Get(name); v != "" {
			return v
		}
	}
	return ""
}

// qCSV returns a parsed comma-separated list from the first matching param name.
func qCSV(q url.Values, names ...string) []string {
	v := qFirst(q, names...)
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// qInt parses the first matching param (alias-aware) as an integer.
//
// A malformed value is deliberately treated the same as an absent one: both
// return -1, which callers interpret as "leave the field unset" so the value
// falls back to the stored request default.
//
// This is intentional and follows the GET interface requirements, which state
// that invalid parameter values are dropped and unknown parameters are silently
// ignored, rather than failing the request. GET query strings can be truncated
// or mangled by intermediate proxies, so a single bad parameter must not reject
// an otherwise serviceable auction request.
//
// Note that -1 is also a legitimate sentinel for several OpenRTB fields (e.g.
// startdelay, podseq, slotinpod), which is why call sites use field-specific
// comparisons such as `> 0` or `>= 0` instead of a blanket check.
func qInt(q url.Values, names ...string) int {
	return qParseInt(qFirst(q, names...))
}

// qParseInt parses a string as an int, returning -1 for both an empty string and
// an unparsable value. See qInt for why invalid input is dropped rather than
// surfaced as an error.
func qParseInt(s string) int {
	if s == "" {
		return -1
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}
	return i
}

// qInts parses a comma-separated string of integers from the first matching param.
// Individual entries that are not positive integers are dropped, consistent with
// the invalid-values-are-dropped rule described on qInt.
func qInts(q url.Values, names ...string) []int {
	s := qFirst(q, names...)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		if i := qParseInt(strings.TrimSpace(p)); i > 0 {
			result = append(result, i)
		}
	}
	return result
}
