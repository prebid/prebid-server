package openrtb2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

// parseGETRequest builds an OpenRTB BidRequest JSON from HTTP GET query parameters.
// The stored request ID (srid) is required — without it we cannot know the auction structure.
// All other parameters are optional and overlay on top of the stored request.
//
// Parameter precedence (lowest → highest):
//  1. Stored request (loaded later in the normal parseRequest / processStoredRequests flow)
//  2. Request profiles (rprof / req_profiles) — declared in ext.prebid.profiles
//  3. Individual GET query params mapped to OpenRTB fields
//  4. HTTP headers (Referer, User-Agent, X-Forwarded-For handled by existing parseRequest logic)
func parseGETRequest(r *http.Request) ([]byte, error) {
	q := r.URL.Query()

	// srid is required — without a stored request we cannot construct a valid auction request.
	srid := qFirst(q, "srid")
	if srid == "" {
		return nil, fmt.Errorf("GET /openrtb2/auction requires 'srid' (stored request ID) query parameter")
	}

	req := &openrtb2.BidRequest{}

	// Build ext.prebid skeleton
	prebid := openrtb_ext.ExtRequestPrebid{}
	prebid.StoredRequest = &openrtb_ext.ExtStoredRequest{ID: srid}

	// Request-level profiles
	if rprof := qCSV(q, "rprof", "req_profiles"); len(rprof) > 0 {
		prebid.Profiles = rprof
	}

	// Output format / module (for exit-point modules)
	if of := qFirst(q, "of"); of != "" {
		prebid.OutputFormat = of
	}
	if om := qFirst(q, "om"); om != "" {
		prebid.OutputModule = om
	}

	// debug
	if d := qFirst(q, "debug"); d == "1" || d == "true" {
		prebid.Debug = true
	}

	// Mark request method so exit-point modules can detect GET channel.
	prebid.Server = &openrtb_ext.ExtRequestPrebidServer{
		RequestMethod: http.MethodGet,
	}

	// tmax
	if tmaxStr := qFirst(q, "tmax"); tmaxStr != "" {
		if tmax, err := strconv.ParseInt(tmaxStr, 10, 64); err == nil && tmax >= 100 {
			req.TMax = tmax
		}
	}

	// Privacy params
	applyGETPrivacyParams(q, req)

	// Build imp[0]
	imp := buildImpFromGET(q)
	req.Imp = []openrtb2.Imp{imp}

	// Publisher ID
	if pubid := qFirst(q, "pubid"); pubid != "" {
		if req.Site == nil {
			req.Site = &openrtb2.Site{}
		}
		if req.Site.Publisher == nil {
			req.Site.Publisher = &openrtb2.Publisher{}
		}
		req.Site.Publisher.ID = pubid
	}

	// Content params (site.content / app.content)
	applyGETContentParams(q, req)

	// Blocking
	if bcat := qCSV(q, "bcat"); len(bcat) > 0 {
		req.BCat = bcat
	}
	if badv := qCSV(q, "badv"); len(badv) > 0 {
		req.BAdv = badv
	}

	// Attach ext
	extWrapper := openrtb_ext.ExtRequest{Prebid: prebid}
	extBytes, err := json.Marshal(extWrapper)
	if err != nil {
		return nil, fmt.Errorf("GET request: failed to marshal ext: %w", err)
	}
	req.Ext = extBytes

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("GET request: failed to marshal BidRequest: %w", err)
	}
	return reqBytes, nil
}

// buildImpFromGET creates the single impression object from GET query params.
// GET interface supports exactly one impression per request.
func buildImpFromGET(q url.Values) openrtb2.Imp {
	imp := openrtb2.Imp{}

	// slot → imp.tagid
	if slot := qFirst(q, "slot"); slot != "" {
		imp.TagID = slot
	}

	// stored auction response
	if sarid := qFirst(q, "sarid"); sarid != "" {
		imp.Ext = setGETImpExtField(imp.Ext, "prebid", "storedauctionresponse", map[string]string{"id": sarid})
	}

	// Imp-level profiles
	if iprof := qCSV(q, "iprof", "imp_profiles"); len(iprof) > 0 {
		impPrebid := openrtb_ext.ExtImpPrebid{Profiles: iprof}
		if extBytes, merr := json.Marshal(map[string]interface{}{"prebid": impPrebid}); merr == nil {
			imp.Ext = extBytes
		}
	}

	// Determine media type and populate accordingly
	mtype := qFirst(q, "mtype")
	switch mtype {
	case "2", "vid":
		v := &openrtb2.Video{}
		applyGETVideoParams(q, v)
		imp.Video = v
	case "3", "aud":
		a := &openrtb2.Audio{}
		applyGETAudioParams(q, a)
		imp.Audio = a
	default:
		// Default to banner (mtype=1 or empty)
		b := &openrtb2.Banner{}
		applyGETBannerParams(q, b)
		if b.W != nil || b.H != nil || len(b.Format) > 0 {
			imp.Banner = b
		}
	}

	return imp
}

func applyGETBannerParams(q url.Values, b *openrtb2.Banner) {
	if w := qInt(q, "w"); w > 0 {
		w64 := int64(w)
		b.W = &w64
	}
	if h := qInt(q, "h"); h > 0 {
		h64 := int64(h)
		b.H = &h64
	}
	if pos := qInt(q, "pos"); pos >= 0 {
		p := adcom1.PlacementPosition(pos)
		b.Pos = &p
	}
	if topframe := qInt(q, "topframe"); topframe >= 0 {
		tf := int8(topframe)
		b.TopFrame = tf
	}
	if battr := qInts(q, "battr"); len(battr) > 0 {
		for _, ba := range battr {
			b.BAttr = append(b.BAttr, adcom1.CreativeAttribute(ba))
		}
	}
	if btype := qInts(q, "btype"); len(btype) > 0 {
		for _, bt := range btype {
			b.BType = append(b.BType, openrtb2.BannerAdType(bt))
		}
	}
	if expdir := qInts(q, "expdir"); len(expdir) > 0 {
		for _, ed := range expdir {
			b.ExpDir = append(b.ExpDir, adcom1.ExpandableDirection(ed))
		}
	}
	if mimes := qCSV(q, "mimes"); len(mimes) > 0 {
		b.MIMEs = mimes
	}
	if api := qInts(q, "api"); len(api) > 0 {
		for _, a := range api {
			b.API = append(b.API, adcom1.APIFramework(a))
		}
	}
}

func applyGETVideoParams(q url.Values, v *openrtb2.Video) {
	if mindur := qInt(q, "mindur"); mindur > 0 {
		v.MinDuration = int64(mindur)
	}
	if maxdur := qInt(q, "maxdur"); maxdur > 0 {
		v.MaxDuration = int64(maxdur)
	}
	if w := qInt(q, "w"); w > 0 {
		w64 := int64(w)
		v.W = &w64
	}
	if h := qInt(q, "h"); h > 0 {
		h64 := int64(h)
		v.H = &h64
	}
	if skip := qInt(q, "skip"); skip >= 0 {
		s := int8(skip)
		v.Skip = &s
	}
	if skipmin := qInt(q, "skipmin"); skipmin > 0 {
		v.SkipMin = int64(skipmin)
	}
	if skipafter := qInt(q, "skipafter"); skipafter > 0 {
		v.SkipAfter = int64(skipafter)
	}
	if startdelay := qInt(q, "startdelay"); startdelay != -1 {
		sd := adcom1.StartDelay(startdelay)
		v.StartDelay = &sd
	}
	if linearity := qInt(q, "linearity"); linearity > 0 {
		v.Linearity = adcom1.LinearityMode(linearity)
	}
	if placement := qInt(q, "placement"); placement > 0 {
		v.Placement = adcom1.VideoPlacementSubtype(placement)
	}
	if plcmt := qInt(q, "plcmt"); plcmt > 0 {
		v.Plcmt = adcom1.VideoPlcmtSubtype(plcmt)
	}
	if pos := qInt(q, "pos"); pos >= 0 {
		p := adcom1.PlacementPosition(pos)
		v.Pos = &p
	}
	if poddur := qInt(q, "poddur"); poddur > 0 {
		v.PodDur = int64(poddur)
	}
	if podid := qFirst(q, "podid"); podid != "" {
		v.PodID = podid
	}
	if podseq := qInt(q, "podseq"); podseq != -1 {
		v.PodSeq = adcom1.PodSequence(podseq)
	}
	if seq := qInt(q, "seq"); seq > 0 {
		v.Sequence = int8(seq)
	}
	if slotinpod := qInt(q, "slotinpod"); slotinpod != -1 {
		v.SlotInPod = adcom1.SlotPositionInPod(slotinpod)
	}
	if minbr := qInt(q, "minbr"); minbr > 0 {
		v.MinBitRate = int64(minbr)
	}
	if maxbr := qInt(q, "maxbr"); maxbr > 0 {
		v.MaxBitRate = int64(maxbr)
	}
	if maxex := qInt(q, "maxex"); maxex != -1 {
		v.MaxExtended = int64(maxex)
	}
	if playbackend := qInt(q, "playbackend"); playbackend > 0 {
		v.PlaybackEnd = adcom1.PlaybackCessationMode(playbackend)
	}
	if boxingallowed := qInt(q, "boxingallowed"); boxingallowed >= 0 {
		ba := int8(boxingallowed)
		v.BoxingAllowed = &ba
	}
	if mimes := qCSV(q, "mimes"); len(mimes) > 0 {
		v.MIMEs = mimes
	}
	if proto := qInts(q, "proto"); len(proto) > 0 {
		for _, p := range proto {
			v.Protocols = append(v.Protocols, adcom1.MediaCreativeSubtype(p))
		}
	}
	if api := qInts(q, "api"); len(api) > 0 {
		for _, a := range api {
			v.API = append(v.API, adcom1.APIFramework(a))
		}
	}
	if delivery := qInts(q, "delivery"); len(delivery) > 0 {
		for _, d := range delivery {
			v.Delivery = append(v.Delivery, adcom1.DeliveryMethod(d))
		}
	}
	if battr := qInts(q, "battr"); len(battr) > 0 {
		for _, ba := range battr {
			v.BAttr = append(v.BAttr, adcom1.CreativeAttribute(ba))
		}
	}
	if playbackmethod := qInts(q, "playbackmethod"); len(playbackmethod) > 0 {
		for _, pm := range playbackmethod {
			v.PlaybackMethod = append(v.PlaybackMethod, adcom1.PlaybackMethod(pm))
		}
	}
	if rqddurs := qInts(q, "rqddurs"); len(rqddurs) > 0 {
		for _, rd := range rqddurs {
			v.RqdDurs = append(v.RqdDurs, int64(rd))
		}
	}
	if maxseq := qInt(q, "maxseq"); maxseq > 0 {
		v.MaxSeq = int64(maxseq)
	}
	if mincpms := qInt(q, "mincpms"); mincpms > 0 {
		v.MinCPMPerSec = float64(mincpms)
	}
}

func applyGETAudioParams(q url.Values, a *openrtb2.Audio) {
	if mindur := qInt(q, "mindur"); mindur > 0 {
		a.MinDuration = int64(mindur)
	}
	if maxdur := qInt(q, "maxdur"); maxdur > 0 {
		a.MaxDuration = int64(maxdur)
	}
	if minbr := qInt(q, "minbr"); minbr > 0 {
		a.MinBitrate = int64(minbr)
	}
	if maxbr := qInt(q, "maxbr"); maxbr > 0 {
		a.MaxBitrate = int64(maxbr)
	}
	if maxseq := qInt(q, "maxseq"); maxseq > 0 {
		a.MaxSeq = int64(maxseq)
	}
	if stitched := qInt(q, "stitched"); stitched >= 0 {
		s := int8(stitched)
		a.Stitched = &s
	}
	if feed := qInt(q, "feed"); feed > 0 {
		a.Feed = adcom1.FeedType(feed)
	}
	if nvol := qInt(q, "nvol"); nvol > 0 {
		nvolVal := adcom1.VolumeNormalizationMode(nvol)
		a.NVol = &nvolVal
	}
	if mimes := qCSV(q, "mimes"); len(mimes) > 0 {
		a.MIMEs = mimes
	}
	if api := qInts(q, "api"); len(api) > 0 {
		for _, ap := range api {
			a.API = append(a.API, adcom1.APIFramework(ap))
		}
	}
	if delivery := qInts(q, "delivery"); len(delivery) > 0 {
		for _, d := range delivery {
			a.Delivery = append(a.Delivery, adcom1.DeliveryMethod(d))
		}
	}
	if battr := qInts(q, "battr"); len(battr) > 0 {
		for _, ba := range battr {
			a.BAttr = append(a.BAttr, adcom1.CreativeAttribute(ba))
		}
	}
	if proto := qInts(q, "proto"); len(proto) > 0 {
		for _, p := range proto {
			a.Protocols = append(a.Protocols, adcom1.MediaCreativeSubtype(p))
		}
	}
	if startdelay := qInt(q, "startdelay"); startdelay != -1 {
		sd := adcom1.StartDelay(startdelay)
		a.StartDelay = &sd
	}
	if poddur := qInt(q, "poddur"); poddur > 0 {
		a.PodDur = int64(poddur)
	}
	if podid := qFirst(q, "podid"); podid != "" {
		a.PodID = podid
	}
	if podseq := qInt(q, "podseq"); podseq != -1 {
		a.PodSeq = adcom1.PodSequence(podseq)
	}
	if seq := qInt(q, "seq"); seq > 0 {
		a.Sequence = int64(seq)
	}
	if slotinpod := qInt(q, "slotinpod"); slotinpod != -1 {
		a.SlotInPod = adcom1.SlotPositionInPod(slotinpod)
	}
	if mincpms := qInt(q, "mincpms"); mincpms > 0 {
		a.MinCPMPerSec = float64(mincpms)
	}
	if rqddurs := qInts(q, "rqddurs"); len(rqddurs) > 0 {
		for _, rd := range rqddurs {
			a.RqdDurs = append(a.RqdDurs, int64(rd))
		}
	}
	// Note: openrtb2.Audio has no Linearity field — audio linearity is not an OpenRTB 2.x concept.
	// The linearity param is silently ignored for audio requests.
}

func applyGETPrivacyParams(q url.Values, req *openrtb2.BidRequest) {
	regs := &openrtb2.Regs{}
	hasRegs := false

	if gdpr := qInt(q, "gdpr", "gdpr_applies"); gdpr >= 0 {
		g := int8(gdpr)
		regs.GDPR = &g
		hasRegs = true
	}
	if gpp := qFirst(q, "gppc"); gpp != "" {
		regs.GPP = gpp
		hasRegs = true
	}
	if gpps := qCSV(q, "gpps"); len(gpps) > 0 {
		for _, s := range gpps {
			if i := qParseInt(s); i > 0 {
				regs.GPPSID = append(regs.GPPSID, int8(i))
			}
		}
		hasRegs = true
	}
	if coppa := qInt(q, "coppa"); coppa >= 0 {
		regs.COPPA = int8(coppa)
		hasRegs = true
	}
	if usp := qFirst(q, "usp"); usp != "" {
		regs.USPrivacy = usp
		hasRegs = true
	}
	if hasRegs {
		req.Regs = regs
	}

	// user.consent (GDPR TCF string)
	if consent := qFirst(q, "gdpr_consent", "consent_string", "tcfc", "cs"); consent != "" {
		if req.User == nil {
			req.User = &openrtb2.User{}
		}
		req.User.Consent = consent
	}

	// device fields
	if dnt := qInt(q, "dnt"); dnt >= 0 {
		d := int8(dnt)
		if req.Device == nil {
			req.Device = &openrtb2.Device{}
		}
		req.Device.DNT = &d
	}
	if lmt := qInt(q, "lmt"); lmt >= 0 {
		l := int8(lmt)
		if req.Device == nil {
			req.Device = &openrtb2.Device{}
		}
		req.Device.Lmt = &l
	}
	if ifa := qFirst(q, "ifa"); ifa != "" {
		if req.Device == nil {
			req.Device = &openrtb2.Device{}
		}
		req.Device.IFA = ifa
	}
	if ua := qFirst(q, "ua"); ua != "" {
		if req.Device == nil {
			req.Device = &openrtb2.Device{}
		}
		req.Device.UA = ua
	}
	if dtype := qFirst(q, "dtype"); dtype != "" {
		if req.Device == nil {
			req.Device = &openrtb2.Device{}
		}
		if dt := qParseInt(dtype); dt > 0 {
			req.Device.DeviceType = adcom1.DeviceType(dt)
		}
	}
}

func applyGETContentParams(q url.Values, req *openrtb2.BidRequest) {
	content := &openrtb2.Content{}
	hasContent := false

	if genre := qFirst(q, "cgenre"); genre != "" {
		content.Genre = genre
		hasContent = true
	}
	if lang := qFirst(q, "clang"); lang != "" {
		content.Language = lang
		hasContent = true
	}
	if rating := qFirst(q, "crating"); rating != "" {
		content.ContentRating = rating
		hasContent = true
	}
	if title := qFirst(q, "ctitle"); title != "" {
		content.Title = title
		hasContent = true
	}
	if series := qFirst(q, "cseries"); series != "" {
		content.Series = series
		hasContent = true
	}
	if curl := qFirst(q, "curl", "url_override"); curl != "" {
		content.URL = curl
		hasContent = true
	}
	if livestream := qInt(q, "clivestream"); livestream >= 0 {
		ls := int8(livestream)
		content.LiveStream = &ls
		hasContent = true
	}

	if !hasContent {
		return
	}

	if req.Site != nil {
		req.Site.Content = content
	} else if req.App != nil {
		req.App.Content = content
	} else {
		// Default to site context
		req.Site = &openrtb2.Site{Content: content}
	}
}

// setGETImpExtField merges a value into imp.ext at path ext["outerKey"]["innerKey"].
func setGETImpExtField(ext json.RawMessage, outerKey, innerKey string, value interface{}) json.RawMessage {
	m := map[string]interface{}{}
	if len(ext) > 0 {
		_ = json.Unmarshal(ext, &m)
	}
	outer, _ := m[outerKey].(map[string]interface{})
	if outer == nil {
		outer = map[string]interface{}{}
	}
	outer[innerKey] = value
	m[outerKey] = outer
	b, _ := json.Marshal(m)
	return b
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

// qInt parses the first matching param as an integer. Returns -1 if absent or invalid.
func qInt(q url.Values, names ...string) int {
	return qParseInt(qFirst(q, names...))
}

// qParseInt parses a string as int, returning -1 on error or empty string.
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
