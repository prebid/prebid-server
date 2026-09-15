package openrtb2

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

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
// Called on raw JSON before applyGETImpPatch so the patch is applied only to imp[0].
func enforceSingleImp(httpRequest *http.Request, requestJson []byte, accountID string) ([]byte, error) {
	if len(requestJson) == 0 {
		return requestJson, nil
	}

	var reqMap map[string]json.RawMessage
	if err := json.Unmarshal(requestJson, &reqMap); err != nil {
		return nil, err
	}

	impsRaw, hasImps := reqMap["imp"]
	if !hasImps {
		return requestJson, nil
	}

	var imps []json.RawMessage
	if err := json.Unmarshal(impsRaw, &imps); err != nil {
		return nil, err
	}

	if len(imps) <= 1 {
		return requestJson, nil
	}

	discarded := len(imps) - 1
	util.LogRandomSample(
		fmt.Sprintf(
			"GET /openrtb2/auction resolved to %d impressions; discarded %d after the first. referrer=%q account=%q",
			discarded+1, discarded, httpRequest.Referer(), accountID,
		),
		logger.Warnf,
		getMultiImpLogSampleRate,
	)

	trimmed, err := json.Marshal(imps[:1])
	if err != nil {
		return nil, err
	}
	reqMap["imp"] = trimmed
	return json.Marshal(reqMap)
}

// reUnfilledMacro matches a query param value that is entirely an unexpanded ad-server macro.
// Such values are dropped before parsing so they never end up as literal strings in the bid
// request.  The four common placeholder styles are matched (after URL-decoding by url.Values):
//
//	[MACRO]      square-bracket, uppercase/digits/underscore only
//	%%MACRO%%    double-percent
//	${MACRO}     dollar-brace
//	{MACRO}      bare curly-brace
//
// Values like "[Live] Player" intentionally do NOT match because they contain mixed case and
// spaces — they are display text that happens to include brackets, not an unfilled placeholder.
var reUnfilledMacro = regexp.MustCompile(
	`^\[([A-Z][A-Z0-9_]*)\]$` +
		`|^%%([A-Z][A-Z0-9_]*)%%$` +
		`|^\$\{([A-Z][A-Z0-9_]*)\}$` +
		`|^\{([A-Z][A-Z0-9_]*)\}$`,
)

// sanitizeGETQuery returns a copy of q with unfilled macro values dropped and
// control characters stripped from all remaining string values.
func sanitizeGETQuery(q url.Values) url.Values {
	out := make(url.Values, len(q))
	for key, vals := range q {
		var cleaned []string
		for _, v := range vals {
			if reUnfilledMacro.MatchString(v) {
				continue
			}
			stripped := strings.Map(func(r rune) rune {
				if r < 0x20 || r == 0x7F {
					return -1
				}
				return r
			}, v)
			cleaned = append(cleaned, stripped)
		}
		if len(cleaned) > 0 {
			out[key] = cleaned
		}
	}
	return out
}

// getParams holds the output of parseGETRequest: a bid request JSON skeleton, an
// imp-level merge patch applied after stored request merge, and inventory fields that
// are deferred until the stored request is known so they can be routed to the correct
// context object (site/app/dooh).
type getParams struct {
	impPatch json.RawMessage
	pubid    string
	page     string
	content  map[string]interface{}
}

func (p *getParams) hasInventory() bool {
	return p.pubid != "" || p.page != "" || len(p.content) > 0
}

// applyInventory writes the deferred inventory fields (pubid, page, content) into
// requestJson. It inspects the stored request to determine the target context object
// (site by default, app or dooh when the stored request declares one), so the fields
// land on the correct top-level object regardless of which context the stored request
// declares.
func (p *getParams) applyInventory(requestJson []byte, storedRequest json.RawMessage) ([]byte, error) {
	if !p.hasInventory() {
		return requestJson, nil
	}

	ctxKey := "site"
	var stored map[string]json.RawMessage
	if json.Unmarshal(storedRequest, &stored) == nil {
		if _, ok := stored["app"]; ok {
			ctxKey = "app"
		} else if _, ok := stored["dooh"]; ok {
			ctxKey = "dooh"
		}
	}

	ctxPatch := map[string]interface{}{}
	if p.pubid != "" {
		ctxPatch["publisher"] = map[string]interface{}{"id": p.pubid}
	}
	if p.page != "" && ctxKey == "site" {
		ctxPatch["page"] = p.page
	}
	if len(p.content) > 0 {
		ctxPatch["content"] = p.content
	}
	if len(ctxPatch) == 0 {
		return requestJson, nil
	}

	patchJSON, err := json.Marshal(map[string]interface{}{ctxKey: ctxPatch})
	if err != nil {
		return nil, err
	}
	return jsonpatch.MergePatch(requestJson, patchJSON)
}

// parseGETRequest builds an OpenRTB BidRequest JSON from HTTP GET query parameters.
// The stored request ID (srid) is required — without it we cannot know the auction structure.
// All other parameters are optional and overlay on top of the stored request.
//
// Returns (requestJSON, *getParams, error). getParams carries the imp-level merge patch and
// the inventory fields (pubid, page, content); both must be applied AFTER processStoredRequests
// via getParams.applyInventory and applyGETImpPatch. Keeping these separate avoids writing
// inventory fields to the wrong context object when the stored request declares app or dooh.
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
// Request-level params are built as a sparse map so that explicit zero values (e.g.
// coppa=0) survive JSON marshalling and correctly override stored request values.
func parseGETRequest(r *http.Request, maxInitialLineLength int) ([]byte, *getParams, error) {
	if maxInitialLineLength > 0 {
		if lineLength := len(r.Method) + 1 + len(r.URL.RequestURI()) + 1 + len(r.Proto); lineLength > maxInitialLineLength {
			return nil, nil, fmt.Errorf("request line exceeded max size of %d bytes", maxInitialLineLength)
		}
	}

	q := sanitizeGETQuery(r.URL.Query())

	srid := qFirst(q, "srid", "tag_id")
	if srid == "" {
		return nil, nil, fmt.Errorf("GET /openrtb2/auction requires 'srid' (stored request ID) query parameter")
	}

	// Fast path: only srid provided — all auction structure comes from the stored request.
	// Device headers (X-Device-*, User-Agent, X-Forwarded-For) are still applied so
	// the request carries accurate device context even when no query params overlay it.
	// X-Device-Player goes into the imp patch even here.
	if len(q) == 1 {
		reqMap := map[string]interface{}{
			"ext": map[string]interface{}{
				"prebid": map[string]interface{}{
					"storedrequest": map[string]interface{}{"id": srid},
					"server":        map[string]interface{}{"http_method": "GET"},
				},
			},
		}
		applyGETHeaderParamsToMap(r.Header, reqMap)
		var fastImpPatch json.RawMessage
		if player := strings.TrimSpace(r.Header.Get("X-Device-Player")); player != "" {
			p, perr := json.Marshal(map[string]interface{}{"displaymanager": player})
			if perr != nil {
				return nil, nil, perr
			}
			fastImpPatch = p
		}
		b, err := json.Marshal(reqMap)
		return b, &getParams{impPatch: fastImpPatch}, err
	}

	reqMap := map[string]interface{}{}

	// ext.prebid skeleton
	prebidMap := map[string]interface{}{
		"storedrequest": map[string]interface{}{"id": srid},
		"server":        map[string]interface{}{"http_method": "GET"},
	}
	pm := newMapWriter(q, prebidMap)
	pm.text("of")
	pm.text("om")
	if d := qFirst(q, "debug"); d == "1" || d == "true" {
		prebidMap["debug"] = true
	}
	if test := qFirst(q, "test"); test == "1" || test == "true" {
		reqMap["test"] = 1
	}

	// tmax
	if tmaxStr := qFirst(q, "tmax"); tmaxStr != "" {
		if tmax, err := strconv.ParseInt(tmaxStr, 10, 64); err == nil && tmax > 0 {
			reqMap["tmax"] = tmax
		}
	}

	// Privacy params (sparse — coppa=0 is preserved)
	applyGETPrivacyParamsToMap(q, reqMap)

	// Inventory params (pubid, page, content) are deferred to applyInventory, called
	// after processStoredRequests, so we can detect whether the stored request declares
	// app or dooh and route the fields to the correct context object.
	gp := &getParams{
		pubid:   qFirst(q, "pubid", "account"),
		page:    qFirst(q, "page"),
		content: buildGETContentMap(q),
	}

	// Blocking
	rw := newMapWriter(q, reqMap)
	rw.csv("bcat")
	rw.csv("badv")

	// Build the imp-level patch; returned separately so ext.prebid stays clean.
	// Applied by the caller via applyGETImpPatch after processStoredRequests.
	impPatch, err := buildImpOverrideFromGET(r.Header, q)
	if err != nil {
		return nil, nil, err
	}
	gp.impPatch = impPatch

	// Header params (device fields; X-Device-Player goes into imp patch above)
	applyGETHeaderParamsToMap(r.Header, reqMap)

	reqMap["ext"] = map[string]interface{}{"prebid": prebidMap}

	b, err := json.Marshal(reqMap)
	return b, gp, err
}

// applyGETPrivacyParamsToMap writes privacy-related query params into reqMap as a
// sparse map. Using maps (rather than marshalling structs) ensures that explicit
// zero values like coppa=0 appear in the JSON output and correctly override stored
// request values during the MergePatch merge.
func applyGETPrivacyParamsToMap(q url.Values, reqMap map[string]interface{}) {
	regsMap := map[string]interface{}{}
	rw := newMapWriter(q, regsMap)
	rw.intN("gdpr", getDomainInt8, "gdpr", "gdpr_applies")
	rw.text("gpp", "gppc")
	// gpp_sid: positive-integer CSV — filtered individually, not a simple domain check.
	if gpps := qCSV(q, "gpps"); len(gpps) > 0 {
		var ids []int
		for _, s := range gpps {
			if i := qParseInt(s); i > 0 {
				ids = append(ids, i)
			}
		}
		if len(ids) > 0 {
			regsMap["gpp_sid"] = ids
		}
	}
	rw.intN("coppa", getDomainInt8)
	rw.text("us_privacy", "usp")
	// gpc lives in regs.ext, not regs directly.
	if gpc, ok := qIntIn(q, getDomainNonNegative, "gpc"); ok {
		regsExt, _ := regsMap["ext"].(map[string]interface{})
		if regsExt == nil {
			regsExt = map[string]interface{}{}
		}
		regsExt["gpc"] = gpc
		regsMap["ext"] = regsExt
	}
	if len(regsMap) > 0 {
		reqMap["regs"] = regsMap
	}

	if consent := qFirst(q, "tcfc", "gdpr_consent", "consent_string", "cs"); consent != "" {
		userMap, _ := reqMap["user"].(map[string]interface{})
		if userMap == nil {
			userMap = map[string]interface{}{}
		}
		userMap["consent"] = consent
		reqMap["user"] = userMap
	}
	if addtlConsent := qFirst(q, "addtl_consent"); addtlConsent != "" {
		userMap, _ := reqMap["user"].(map[string]interface{})
		if userMap == nil {
			userMap = map[string]interface{}{}
		}
		userExt, _ := userMap["ext"].(map[string]interface{})
		if userExt == nil {
			userExt = map[string]interface{}{}
		}
		userExt["ConsentedProvidersSettings"] = map[string]interface{}{
			"consented_providers": addtlConsent,
		}
		userMap["ext"] = userExt
		reqMap["user"] = userMap
	}

	deviceMap, _ := reqMap["device"].(map[string]interface{})
	if deviceMap == nil {
		deviceMap = map[string]interface{}{}
	}
	dw := newMapWriter(q, deviceMap)
	dw.intN("dnt", getDomainInt8)
	dw.intN("lmt", getDomainInt8)
	// Reject the all-zero UUID placeholder produced by unfilled query macros.
	if ifa := qFirst(q, "ifa"); ifa != "" && ifa != "00000000-0000-0000-0000-000000000000" {
		deviceMap["ifa"] = ifa
	}
	dw.text("ua")
	dw.intN("devicetype", getDomainInt8, "dtype")
	// ifa_type lives in device.ext, not device directly.
	if ifaType := qFirst(q, "ifat"); ifaType != "" {
		devExt, _ := deviceMap["ext"].(map[string]interface{})
		if devExt == nil {
			devExt = map[string]interface{}{}
		}
		devExt["ifa_type"] = ifaType
		deviceMap["ext"] = devExt
	}
	// ip/ipv6: custom — requires net.ParseIP normalisation and To4 routing.
	if parsed := net.ParseIP(strings.TrimSpace(qFirst(q, "ip"))); parsed != nil {
		if parsed.To4() != nil {
			deviceMap["ip"] = parsed.String()
		} else {
			deviceMap["ipv6"] = parsed.String()
		}
	}
	if parsed := net.ParseIP(strings.TrimSpace(qFirst(q, "ipv6"))); parsed != nil && parsed.To4() == nil {
		deviceMap["ipv6"] = parsed.String()
	}
	if len(deviceMap) > 0 {
		reqMap["device"] = deviceMap
	}
}

// buildGETContentMap builds the content sub-object from content-related query params.
// Returns nil when no content params are present.
func buildGETContentMap(q url.Values) map[string]interface{} {
	contentMap := map[string]interface{}{}
	cw := newMapWriter(q, contentMap)
	cw.text("id", "cid")
	cw.text("title", "ctitle")
	cw.text("series", "cseries", "rss_feed")
	cw.text("season", "cseason")
	cw.text("artist", "cartist")
	cw.text("album", "calbum")
	cw.text("isrc", "cisrc")
	cw.text("genre", "cgenre")
	cw.text("url", "curl", "url_override")
	cw.text("language", "clang")
	cw.text("langb", "clangb")
	cw.text("contentrating", "crating")
	cw.text("userrating", "cuserrating")
	cw.text("keywords", "ckeywords")
	cw.intN("episode", getDomainAny, "cepisode")
	cw.intN("prodq", getDomainInt8, "cprodq")
	cw.intN("context", getDomainInt8, "ccontext")
	cw.intN("qagmediarating", getDomainInt8, "cqagmediarating")
	cw.intN("sourcerelationship", getDomainInt8, "csourcerelationship")
	cw.intN("len", getDomainAny, "clen")
	cw.intN("embeddable", getDomainInt8, "cembeddable")
	cw.intN("livestream", getDomainInt8, "clivestream")
	cw.csv("cat", "ccat")
	cw.intN("cattax", getDomainNonNegative, "ccattax")
	if name := qFirst(q, "cchannel"); name != "" {
		contentMap["channel"] = map[string]interface{}{"name": name}
	}
	if name := qFirst(q, "cnetwork"); name != "" {
		contentMap["network"] = map[string]interface{}{"name": name}
	}
	return contentMap
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

// buildImpOverrideFromGET builds a sparse imp JSON Merge Patch containing only the
// query params that were explicitly set. The caller (parseGETRequest) returns it as
// the second value and passes it to applyGETImpPatch after processStoredRequests,
// so stored imp fields (id, ext with bidder params, mimes) are preserved and only
// the GET-specific fields (w, h, etc.) are overlaid via RFC 7396 recursive merge.
func buildImpOverrideFromGET(h http.Header, q url.Values) (json.RawMessage, error) {
	impMap := map[string]interface{}{}

	if slot := qFirst(q, "slot"); slot != "" {
		impMap["tagid"] = slot
	}

	impExtMap := map[string]interface{}{}

	if sarid := qFirst(q, "sarid"); sarid != "" {
		impExtMap["prebid"] = map[string]interface{}{
			"storedauctionresponse": map[string]string{"id": sarid},
		}
	}

	if targeting := qFirst(q, "targeting"); targeting != "" {
		dataMap := map[string]interface{}{}
		if json.Unmarshal([]byte(targeting), &dataMap) == nil && len(dataMap) > 0 {
			impExtMap["data"] = dataMap
		}
	}

	if len(impExtMap) > 0 {
		impMap["ext"] = impExtMap
	}

	mtype := qFirst(q, "mtype")
	switch mtype {
	case "2", "vid":
		// Explicit video: overlay video params, delete any banner/audio from stored request.
		if vm := buildSparseVideoParams(q); len(vm) > 0 {
			impMap["video"] = vm
		}
		impMap["banner"] = nil
		impMap["audio"] = nil
	case "3", "aud":
		// Explicit audio: overlay audio params, delete any banner/video from stored request.
		if am := buildSparseAudioParams(q); len(am) > 0 {
			impMap["audio"] = am
		}
		impMap["banner"] = nil
		impMap["video"] = nil
	case "1", "ban":
		// Explicit banner: overlay banner params, delete any video/audio from stored request.
		if bm := buildSparseBannerParams(q); len(bm) > 0 {
			impMap["banner"] = bm
		}
		impMap["video"] = nil
		impMap["audio"] = nil
	case "":
		// No mtype: build all three media param maps and defer selection until the stored
		// imp has been resolved. applyGETImpPatch will detect the existing media type and
		// apply only the matching sub-map via the "_deferred_media" key.
		deferredMedia := map[string]interface{}{}
		if bm := buildSparseBannerParams(q); len(bm) > 0 {
			deferredMedia["banner"] = bm
		}
		if vm := buildSparseVideoParams(q); len(vm) > 0 {
			deferredMedia["video"] = vm
		}
		if am := buildSparseAudioParams(q); len(am) > 0 {
			deferredMedia["audio"] = am
		}
		if len(deferredMedia) > 0 {
			impMap["_deferred_media"] = deferredMedia
		}
	default:
		// Unknown/native mtype: no GET media params apply.
		// Null banner/video/audio so stored values for the wrong type are removed.
		impMap["banner"] = nil
		impMap["video"] = nil
		impMap["audio"] = nil
	}

	if player := strings.TrimSpace(h.Get("X-Device-Player")); player != "" {
		impMap["displaymanager"] = player
	}

	if len(impMap) == 0 {
		return nil, nil
	}
	return json.Marshal(impMap)
}

// applySharedAVParams writes all params shared between video and audio into m,
// including rqddurs/duration exclusivity enforcement.
func applySharedAVParams(q url.Values, m map[string]interface{}) {
	vw := newMapWriter(q, m)
	vw.intN("minduration", getDomainNonNegative, "mindur") // 0 is a valid lower bound
	vw.intN("maxduration", getDomainPositive, "maxdur")
	vw.intN("minbitrate", getDomainPositive, "minbr")
	vw.intN("maxbitrate", getDomainPositive, "maxbr")
	vw.intN("maxseq", getDomainPositive)
	vw.intN("maxextended", getDomainAny, "maxex") // -1 is valid per AdCOM (no end time)
	vw.intN("startdelay", getDomainAny)           // -1/-2 are valid AdCOM values
	vw.intN("poddur", getDomainPositive)
	vw.intN("podseq", getDomainInt8)                 // -1 = any pod
	vw.intN("sequence", getDomainNonNegative, "seq") // 0 is a valid sequence number
	vw.intN("slotinpod", getDomainInt8)              // -1 = any slot
	vw.intsN("protocols", getDomainInt8, "proto")
	vw.intsN("api", getDomainInt8)
	vw.intsN("delivery", getDomainInt8)
	vw.intsN("battr", getDomainInt8)
	vw.intsN("rqddurs", getDomainPositive)
	vw.csv("mimes")
	vw.float("mincpmpersec", "mincpms")
	vw.text("podid")
	enforceRqddursDurationExclusivity(m)
}

// enforceRqddursDurationExclusivity implements the OpenRTB 2.6 rule that rqddurs and
// minduration/maxduration are mutually exclusive. The losing side is set to nil so that
// RFC 7396 merge-patch semantics delete those fields from the stored request.
// When both sides appear in the GET params, rqddurs takes precedence.
func enforceRqddursDurationExclusivity(m map[string]interface{}) {
	_, hasRqddurs := m["rqddurs"]
	_, hasMin := m["minduration"]
	_, hasMax := m["maxduration"]

	if hasRqddurs {
		m["minduration"] = nil
		m["maxduration"] = nil
	} else if hasMin || hasMax {
		m["rqddurs"] = nil
	}
}

func buildSparseVideoParams(q url.Values) map[string]interface{} {
	m := map[string]interface{}{}
	applySharedAVParams(q, m)
	vw := newMapWriter(q, m)
	vw.intN("w", getDomainMinDim)
	vw.intN("h", getDomainMinDim)
	vw.intN("skip", getDomainInt8)
	vw.intN("skipmin", getDomainInt8)
	vw.intN("skipafter", getDomainInt8)
	vw.intN("linearity", getDomainInt8)
	vw.intN("placement", getDomainInt8)
	vw.intN("plcmt", getDomainInt8)
	vw.intN("pos", getDomainInt8)
	vw.intN("playbackend", getDomainInt8)
	vw.intN("boxingallowed", getDomainInt8)
	vw.intsN("playbackmethod", getDomainInt8)
	return m
}

func buildSparseAudioParams(q url.Values) map[string]interface{} {
	m := map[string]interface{}{}
	applySharedAVParams(q, m)
	vw := newMapWriter(q, m)
	vw.intN("stitched", getDomainInt8)
	vw.intN("feed", getDomainInt8)
	vw.intN("nvol", getDomainInt8) // 0 is a valid volume normalisation value
	return m
}

// minAdDim is the minimum pixel value accepted for any ad width or height.
// Dimensions below this threshold are not usable for ad serving.
const minAdDim = 10

// appendSizeFormats appends valid WxH entries from the sizes CSV to formats and returns the result.
func appendSizeFormats(formats []map[string]interface{}, sizes []string) []map[string]interface{} {
	for _, s := range sizes {
		parts := strings.SplitN(s, "x", 2)
		if len(parts) != 2 {
			continue
		}
		sw, sh := qParseInt(parts[0]), qParseInt(parts[1])
		if sw >= minAdDim && sh >= minAdDim {
			formats = append(formats, map[string]interface{}{"w": sw, "h": sh})
		}
	}
	return formats
}

func buildSparseBannerParams(q url.Values) map[string]interface{} {
	m := map[string]interface{}{}

	// ms=WxH fills the format list. Standalone w and h override top-level
	// banner dimensions independently; they do not affect the ms-derived format.
	if ms := qFirst(q, "ms"); ms != "" {
		if parts := strings.SplitN(ms, "x", 2); len(parts) == 2 {
			mw, mh := qParseInt(parts[0]), qParseInt(parts[1])
			if mw >= minAdDim && mh >= minAdDim {
				m["format"] = appendSizeFormats([]map[string]interface{}{{"w": mw, "h": mh}}, qCSV(q, "sizes"))
			}
		}
	}
	w, h := 0, 0
	if v := qInt(q, "w"); v >= minAdDim {
		w = v
		m["w"] = v
	}
	if v := qInt(q, "h"); v >= minAdDim {
		h = v
		m["h"] = v
	}
	// When both dimensions are provided without ms, they also define the primary format entry.
	if _, hasFormat := m["format"]; !hasFormat && w >= minAdDim && h >= minAdDim {
		m["format"] = appendSizeFormats([]map[string]interface{}{{"w": w, "h": h}}, qCSV(q, "sizes"))
	}

	vw := newMapWriter(q, m)
	vw.intN("pos", getDomainInt8)
	vw.intN("topframe", getDomainInt8)
	vw.intsN("battr", getDomainInt8)
	vw.intsN("btype", getDomainInt8)
	vw.intsN("expdir", getDomainInt8)
	vw.intsN("api", getDomainInt8)
	vw.csv("mimes")
	return m
}

// applyGETImpPatch applies impPatch as a JSON Merge Patch to every imp in requestJSON.
// Called after processStoredRequests so stored imp fields (id, ext, mimes) are
// preserved and only the GET-specific fields are overlaid via RFC 7396 rules.
//
// When impPatch contains "_deferred_media" (no explicit mtype in the GET request),
// this function detects which media type each stored imp already has and applies only
// the matching sub-map from the deferred set, leaving the imp's other media objects
// untouched.
//
// impPatch is the second return value of parseGETRequest; a nil or empty patch is a no-op.
func applyGETImpPatch(requestJSON []byte, impPatch json.RawMessage) ([]byte, error) {
	if len(impPatch) == 0 {
		return requestJSON, nil
	}

	var reqMap map[string]json.RawMessage
	if err := jsonutil.UnmarshalValid(requestJSON, &reqMap); err != nil {
		return nil, err
	}

	impsRaw, hasImps := reqMap["imp"]
	if !hasImps || len(impsRaw) == 0 {
		return requestJSON, nil
	}

	var imps []json.RawMessage
	if err := jsonutil.UnmarshalValid(impsRaw, &imps); err != nil {
		return nil, err
	}

	// Decode the patch once; check for deferred media.
	var patchMap map[string]json.RawMessage
	if err := json.Unmarshal(impPatch, &patchMap); err != nil {
		return nil, err
	}
	deferredRaw, hasDeferred := patchMap["_deferred_media"]

	for i, imp := range imps {
		effectivePatch := impPatch
		if hasDeferred {
			var resolveErr error
			effectivePatch, resolveErr = resolveGETImpPatch(patchMap, deferredRaw, imp)
			if resolveErr != nil {
				return nil, fmt.Errorf("resolving deferred media for imp[%d]: %w", i, resolveErr)
			}
		} else {
			// Explicit mtype: carry ext from the displaced media object to the new one.
			var transferErr error
			effectivePatch, transferErr = withTransferredMediaExt(imp, patchMap, impPatch)
			if transferErr != nil {
				return nil, fmt.Errorf("transferring media ext for imp[%d]: %w", i, transferErr)
			}
		}
		merged, err := jsonpatch.MergePatch(imp, effectivePatch)
		if err != nil {
			return nil, fmt.Errorf("applying GET imp patch to imp[%d]: %w", i, err)
		}
		imps[i] = merged
	}

	impsBytes, err := json.Marshal(imps)
	if err != nil {
		return nil, err
	}
	reqMap["imp"] = impsBytes
	return json.Marshal(reqMap)
}

// resolveGETImpPatch builds an effective patch for one imp when the original patch
// contains deferred media params (no explicit mtype). It detects the imp's existing
// media type and injects the matching sub-map from deferredRaw, discarding the rest.
func resolveGETImpPatch(patchMap map[string]json.RawMessage, deferredRaw json.RawMessage, imp json.RawMessage) (json.RawMessage, error) {
	var deferredMedia map[string]json.RawMessage
	if err := json.Unmarshal(deferredRaw, &deferredMedia); err != nil {
		return nil, err
	}

	// Build effective patch without the internal key.
	effective := make(map[string]json.RawMessage, len(patchMap))
	for k, v := range patchMap {
		if k != "_deferred_media" {
			effective[k] = v
		}
	}

	// Apply only the media params that match the stored imp's media type.
	if mediaType := detectStoredImpMediaType(imp); mediaType != "" {
		if mediaParams, ok := deferredMedia[mediaType]; ok {
			effective[mediaType] = mediaParams
		}
	}

	return json.Marshal(effective)
}

// withTransferredMediaExt returns the patch with ext from a displaced media object
// injected into the newly activated one, or the original patch unchanged when no
// transfer is needed.
//
// When an explicit mtype param switches the active media type (e.g., video → audio),
// the patch nullifies the old media object via RFC 7396. Any vendor ext carried by
// that object (e.g., video.ext.vendor) is otherwise silently lost. This function
// detects that case and injects the ext into the new media object, but only when the
// new object does not already carry one (explicit GET params take precedence).
func withTransferredMediaExt(imp json.RawMessage, patchMap map[string]json.RawMessage, patch json.RawMessage) (json.RawMessage, error) {
	// Identify the media type being activated by the patch (present, non-null object).
	var newMT string
	for _, mt := range []string{"video", "audio", "banner", "native"} {
		if v, ok := patchMap[mt]; ok && string(v) != "null" {
			newMT = mt
			break
		}
	}
	if newMT == "" {
		return patch, nil
	}

	// No transfer needed if the new media object already carries ext.
	var newMediaFields map[string]json.RawMessage
	if err := json.Unmarshal(patchMap[newMT], &newMediaFields); err != nil {
		return patch, nil
	}
	if _, alreadyHasExt := newMediaFields["ext"]; alreadyHasExt {
		return patch, nil
	}

	// Decode the stored imp fields once.
	var impFields map[string]json.RawMessage
	if err := json.Unmarshal(imp, &impFields); err != nil {
		return patch, nil
	}

	// Find ext from the highest-priority displaced media object (video > audio > banner).
	var extToTransfer json.RawMessage
	for _, mt := range []string{"video", "audio", "banner"} {
		if mt == newMT {
			continue
		}
		v, beingNulled := patchMap[mt]
		if !beingNulled || string(v) != "null" {
			continue
		}
		storedMedia, exists := impFields[mt]
		if !exists || string(storedMedia) == "null" {
			continue
		}
		var storedMediaFields map[string]json.RawMessage
		if err := json.Unmarshal(storedMedia, &storedMediaFields); err != nil {
			continue
		}
		ext, hasExt := storedMediaFields["ext"]
		if !hasExt || string(ext) == "null" {
			continue
		}
		extToTransfer = ext
		break
	}
	if len(extToTransfer) == 0 {
		return patch, nil
	}

	// Inject ext into the new media object and rebuild the patch.
	newMediaFields["ext"] = extToTransfer
	updatedMedia, err := json.Marshal(newMediaFields)
	if err != nil {
		return patch, err
	}
	updatedPatch := make(map[string]json.RawMessage, len(patchMap))
	for k, v := range patchMap {
		updatedPatch[k] = v
	}
	updatedPatch[newMT] = updatedMedia
	return json.Marshal(updatedPatch)
}

// detectStoredImpMediaType returns the primary media type present in a stored imp.
// Priority order: video > audio > banner > native.
// Returns "" when no recognised media object is found.
func detectStoredImpMediaType(imp json.RawMessage) string {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(imp, &fields); err != nil {
		return ""
	}
	for _, mt := range []string{"video", "audio", "banner", "native"} {
		if raw, ok := fields[mt]; ok && len(raw) > 0 && string(raw) != "null" {
			return mt
		}
	}
	return ""
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

// getIntDomain defines the inclusive range of acceptable integer values for a GET param.
type getIntDomain struct{ min, max int }

var (
	getDomainAny         = getIntDomain{math.MinInt, math.MaxInt}
	getDomainInt8        = getIntDomain{math.MinInt8, math.MaxInt8}
	getDomainNonNegative = getIntDomain{0, math.MaxInt}
	getDomainPositive    = getIntDomain{1, math.MaxInt}
	getDomainMinDim      = getIntDomain{minAdDim, math.MaxInt}
)

// qIntIn parses the first matching param as an int and returns (value, true) only
// when the parsed value falls within domain. Absent or malformed params return (0, false).
func qIntIn(q url.Values, domain getIntDomain, names ...string) (int, bool) {
	s := qFirst(q, names...)
	if s == "" {
		return 0, false
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < domain.min || v > domain.max {
		return 0, false
	}
	return v, true
}

// mapWriter is a thin helper that encapsulates the "read param → validate → write to map"
// pattern, making parameter definitions more uniform and reducing repetitive conditionals.
type mapWriter struct {
	q   url.Values
	dst map[string]interface{}
}

func newMapWriter(q url.Values, dst map[string]interface{}) mapWriter {
	return mapWriter{q: q, dst: dst}
}

// intN reads the first matching param, validates it within domain, and writes the integer
// to dstKey. When no names are provided dstKey is used as the param name.
func (w mapWriter) intN(dstKey string, domain getIntDomain, names ...string) {
	if len(names) == 0 {
		names = []string{dstKey}
	}
	if v, ok := qIntIn(w.q, domain, names...); ok {
		w.dst[dstKey] = v
	}
}

// text reads the first non-empty matching param and writes it to dstKey.
// When no names are provided dstKey is used as the param name.
func (w mapWriter) text(dstKey string, names ...string) {
	if len(names) == 0 {
		names = []string{dstKey}
	}
	if v := qFirst(w.q, names...); v != "" {
		w.dst[dstKey] = v
	}
}

// intsN reads the first matching param as a comma-separated integer list, drops entries outside
// domain, and writes the result to dstKey. When no names are provided dstKey is used as the param name.
func (w mapWriter) intsN(dstKey string, domain getIntDomain, names ...string) {
	if len(names) == 0 {
		names = []string{dstKey}
	}
	if v := qIntsIn(w.q, domain, names...); len(v) > 0 {
		w.dst[dstKey] = v
	}
}

// csv reads a comma-separated param and writes the resulting string slice to dstKey.
func (w mapWriter) csv(dstKey string, names ...string) {
	if len(names) == 0 {
		names = []string{dstKey}
	}
	if v := qCSV(w.q, names...); len(v) > 0 {
		w.dst[dstKey] = v
	}
}

// float reads the first matching param as a positive finite float64 and writes it to dstKey.
// When no names are provided dstKey is used as the param name.
func (w mapWriter) float(dstKey string, names ...string) {
	if len(names) == 0 {
		names = []string{dstKey}
	}
	if v, ok := qFloat(w.q, names...); ok {
		w.dst[dstKey] = v
	}
}

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

// qFloat parses the first matching param as a float64.
// Returns (value, true) only when the value is finite and > 0.
// Inf, NaN, zero, and negative values all return (0, false), as do absent or
// unparsable params — consistent with the invalid-values-are-dropped rule.
func qFloat(q url.Values, names ...string) (float64, bool) {
	s := qFirst(q, names...)
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
		return 0, false
	}
	return v, true
}

// qIntsIn parses a comma-separated string of integers from the first matching param,
// keeping only entries that fall within domain.
func qIntsIn(q url.Values, domain getIntDomain, names ...string) []int {
	s := qFirst(q, names...)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		v, err := strconv.Atoi(strings.TrimSpace(p))
		if err == nil && v >= domain.min && v <= domain.max {
			result = append(result, v)
		}
	}
	return result
}
