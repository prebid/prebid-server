package identity

import (
	"encoding/json"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

// IntentIQ S2S query-parameter names for the identity-resolution request (mirrors Java IiqParam).
const (
	paramAT        = "at"
	paramMI        = "mi"
	paramDPI       = "dpi"
	paramPT        = "pt"
	paramDPN       = "dpn"
	paramSrvrReq   = "srvrReq"
	paramSource    = "source"
	paramIP        = "ip"
	paramIPv6      = "ipv6"
	paramUAS       = "uas"
	paramUH        = "uh"
	paramPCID      = "pcid"
	paramIDType    = "idtype"
	paramRef       = "ref"
	paramIIQUID    = "iiquid"
	paramGDPR      = "gdpr"
	paramUSPrivacy = "us_privacy"
	paramGPP       = "gpp"
	paramGPPSID    = "gpp_sid"
)

// uaHintsHighEntropySource is the OpenRTB device.sua.source value the IntentIQ backend consumes
// UA client hints for (high-entropy client hints).
const uaHintsHighEntropySource = 2

// resolveURL builds the identity-resolution request URL from the effective config and the request.
// It faithfully mirrors the Java resolveUrl: fixed params first, then device/consent-derived params,
// each URL-encoded and appended only when non-blank.
func resolveURL(cfg Config, rw *openrtb_ext.RequestWrapper) string {
	var b strings.Builder
	b.WriteString(cfg.APIEndpoint)
	if strings.Contains(cfg.APIEndpoint, "?") {
		b.WriteByte('&')
	} else {
		b.WriteByte('?')
	}
	b.WriteString(paramAT)
	b.WriteString("=39")

	appendIfPresent(&b, paramMI, "10")
	appendIfPresent(&b, paramDPI, cfg.PartnerID)
	appendIfPresent(&b, paramPT, "17")
	appendIfPresent(&b, paramDPN, "1")
	appendIfPresent(&b, paramSrvrReq, "true")
	appendIfPresent(&b, paramSource, sourcePBSGo)

	req := rw.BidRequest
	if device := req.Device; device != nil {
		appendIfPresent(&b, paramIP, device.IP)
		appendIfPresent(&b, paramIPv6, device.IPv6)
		appendIfPresent(&b, paramUAS, device.UA)
		appendIfPresent(&b, paramUH, buildUaHints(device.SUA))
		appendDeviceID(&b, device)
	}
	appendIfPresent(&b, paramRef, resolveRef(req))
	appendIfPresent(&b, paramIIQUID, resolveIiqUID(req.User))
	appendConsent(&b, rw)

	return b.String()
}

// appendConsent appends the gdpr / us_privacy / gpp / gpp_sid query params (consent itself is a
// header, not a query param — see resolveConsent).
func appendConsent(b *strings.Builder, rw *openrtb_ext.RequestWrapper) {
	regs := rw.BidRequest.Regs
	if regs == nil {
		return
	}
	appendIfPresent(b, paramGDPR, resolveGDPR(rw, regs))
	appendIfPresent(b, paramUSPrivacy, resolveUSPrivacy(rw, regs))
	appendIfPresent(b, paramGPP, regs.GPP)
	// Forwarded ahead of backend support so the GPP section ids are available if the backend adds it.
	appendIfPresent(b, paramGPPSID, joinGPPSID(regs.GPPSID))
}

// resolveGDPR returns the GDPR flag from regs.gdpr, falling back to regs.ext.gdpr.
func resolveGDPR(rw *openrtb_ext.RequestWrapper, regs *openrtb2.Regs) string {
	if regs.GDPR != nil {
		return strconv.Itoa(int(*regs.GDPR))
	}
	if regExt, err := rw.GetRegExt(); err == nil && regExt != nil {
		if g := regExt.GetGDPR(); g != nil {
			return strconv.Itoa(int(*g))
		}
	}
	return ""
}

// resolveUSPrivacy returns us_privacy from regs.us_privacy, falling back to regs.ext.us_privacy.
func resolveUSPrivacy(rw *openrtb_ext.RequestWrapper, regs *openrtb2.Regs) string {
	if notBlank(regs.USPrivacy) {
		return regs.USPrivacy
	}
	if regExt, err := rw.GetRegExt(); err == nil && regExt != nil {
		return regExt.GetUSPrivacy()
	}
	return ""
}

// resolveConsent returns the TCF consent string from user.consent, falling back to user.ext.consent.
// It is sent as the gdpr-consent request header (not a query param).
func resolveConsent(rw *openrtb_ext.RequestWrapper) string {
	user := rw.BidRequest.User
	if user == nil {
		return ""
	}
	if notBlank(user.Consent) {
		return user.Consent
	}
	if userExt, err := rw.GetUserExt(); err == nil && userExt != nil {
		if c := userExt.GetConsent(); c != nil {
			return *c
		}
	}
	return ""
}

// joinGPPSID joins the GPP section ids into a comma-separated string (empty when absent).
func joinGPPSID(sids []int8) string {
	if len(sids) == 0 {
		return ""
	}
	parts := make([]string, len(sids))
	for i, s := range sids {
		parts[i] = strconv.Itoa(int(s))
	}
	return strings.Join(parts, ",")
}

// buildUaHints builds the `uh` UA client-hints JSON from OpenRTB device.sua, matching the IntentIQ
// backend's numeric-keyed (0-8) UA-CH format: brands sorted, major vs full version. Returns "" when
// there are no high-entropy hints to send.
func buildUaHints(sua *openrtb2.UserAgent) string {
	// The IntentIQ backend consumes hints only for high-entropy client hints (sua.source == 2).
	if sua == nil || int(sua.Source) != uaHintsHighEntropySource {
		return ""
	}
	hints := make(map[string]string)
	appendBrowserHints(hints, sua.Browsers)
	if sua.Mobile != nil {
		hints["1"] = "?" + strconv.Itoa(int(*sua.Mobile))
	}
	appendPlatformHints(hints, sua.Platform)
	putQuoted(hints, "3", sua.Architecture)
	putQuoted(hints, "4", sua.Bitness)
	putQuoted(hints, "5", sua.Model)
	if len(hints) == 0 {
		return ""
	}
	// encoding/json sorts string map keys, giving deterministic "0".."8" ordering.
	encoded, err := json.Marshal(hints)
	if err != nil {
		return ""
	}
	return string(encoded)
}

// appendBrowserHints populates "0" (major versions) and "8" (full versions) from device.sua.browsers,
// each a brand-sorted list of `"brand";v="version"` entries.
func appendBrowserHints(hints map[string]string, browsers []openrtb2.BrandVersion) {
	if browsers == nil {
		return
	}
	majorByBrand := make(map[string]string)
	fullByBrand := make(map[string]string)
	for _, browser := range browsers {
		if !notBlank(browser.Brand) || len(browser.Version) == 0 {
			continue
		}
		fullVersion := strings.Join(browser.Version, ".")
		if dot := strings.IndexByte(fullVersion, '.'); dot > 0 {
			majorByBrand[browser.Brand] = fullVersion[:dot]
		} else {
			majorByBrand[browser.Brand] = fullVersion
		}
		fullByBrand[browser.Brand] = fullVersion
	}
	putBrandList(hints, "0", majorByBrand)
	putBrandList(hints, "8", fullByBrand)
}

// putBrandList joins the brand->version map (sorted by brand) into a UA-CH brand list under key.
func putBrandList(hints map[string]string, key string, brandToVersion map[string]string) {
	if len(brandToVersion) == 0 {
		return
	}
	brands := make([]string, 0, len(brandToVersion))
	for brand := range brandToVersion {
		brands = append(brands, brand)
	}
	sort.Strings(brands)
	parts := make([]string, 0, len(brands))
	for _, brand := range brands {
		parts = append(parts, quote(brand)+";v="+quote(brandToVersion[brand]))
	}
	hints[key] = strings.Join(parts, ", ")
}

// appendPlatformHints populates "2" (platform brand) and "6" (platform version) from device.sua.platform.
func appendPlatformHints(hints map[string]string, platform *openrtb2.BrandVersion) {
	if platform == nil || !notBlank(platform.Brand) {
		return
	}
	hints["2"] = quote(platform.Brand)
	if len(platform.Version) > 0 {
		hints["6"] = quote(strings.Join(platform.Version, "."))
	}
}

// putQuoted stores a quoted value under key when value is non-blank.
func putQuoted(hints map[string]string, key, value string) {
	if notBlank(value) {
		hints[key] = quote(value)
	}
}

func quote(value string) string {
	return "\"" + value + "\""
}

// appendDeviceID appends the pcid + idtype params derived from device.ifa. Skipped when ifa is blank
// or lmt==1. CTV devices (devicetype 3/7) use an uppercased pcid and idtype 8; otherwise idtype 4.
func appendDeviceID(b *strings.Builder, device *openrtb2.Device) {
	ifa := device.IFA
	if !notBlank(ifa) || (device.Lmt != nil && *device.Lmt == 1) {
		return
	}
	ctv := device.DeviceType == 3 || device.DeviceType == 7
	pcid := ifa
	idtype := "4"
	if ctv {
		// CTV ids (idtype 8) must be uppercase; MAID/AAID (idtype 4) is case-insensitive.
		pcid = strings.ToUpper(ifa)
		idtype = "8"
	}
	b.WriteByte('&')
	b.WriteString(paramPCID)
	b.WriteByte('=')
	b.WriteString(encodeComponent(pcid))
	b.WriteByte('&')
	b.WriteString(paramIDType)
	b.WriteByte('=')
	b.WriteString(idtype)
}

// resolveRef returns the referrer: site.domain||site.page, else app.bundle||app.name.
func resolveRef(req *openrtb2.BidRequest) string {
	if site := req.Site; site != nil {
		if notBlank(site.Domain) {
			return site.Domain
		}
		return site.Page
	}
	if app := req.App; app != nil {
		if notBlank(app.Bundle) {
			return app.Bundle
		}
		return app.Name
	}
	return ""
}

// resolveIiqUID returns the first non-blank uid id of the eid whose source is intentiq.com.
func resolveIiqUID(user *openrtb2.User) string {
	if user == nil {
		return ""
	}
	for _, eid := range user.EIDs {
		if eid.Source != iiqSource {
			continue
		}
		for _, uid := range eid.UIDs {
			if notBlank(uid.ID) {
				return uid.ID
			}
		}
	}
	return ""
}

// appendIfPresent appends `&key=<encoded value>` when value is non-blank.
func appendIfPresent(b *strings.Builder, key, value string) {
	if !notBlank(value) {
		return
	}
	b.WriteByte('&')
	b.WriteString(key)
	b.WriteByte('=')
	b.WriteString(encodeComponent(value))
}

// encodeComponent URL-encodes a query-parameter value like Java's URLEncoder.encode(...).replace(
// "+", "%20") — space becomes %20 rather than +.
func encodeComponent(value string) string {
	return strings.ReplaceAll(url.QueryEscape(value), "+", "%20")
}

// notBlank reports whether s has non-whitespace content (mirrors Java StringUtils.isNotBlank).
func notBlank(s string) bool {
	return strings.TrimSpace(s) != ""
}
