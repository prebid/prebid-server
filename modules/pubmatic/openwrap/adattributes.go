package openwrap

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
)

// Wire IDs for ext.owsdk.adattributes (numeric; product spec).
const (
	AdAttrWireEngageToClose     = 1
	AdAttrWireTrueDoubleEndCard = 2
	AdAttrWireCTAOverlay        = 3
	AdAttrWireMRAIDAppStatus    = 4
	AdAttrWireCountdownOverlay  = 5
	AdAttrWireFrcClkBrowser     = 6
)

// AdFormat is a matrix row key: OS + SDK range + slot-specific logical format (Approach 2 request spec).
type AdFormat string

const (
	AdFormatInterstitialDisplay AdFormat = "interstitial_display" // instl=1 && banner — banner.ext
	AdFormatInterstitialVideo   AdFormat = "interstitial_video"   // instl=1 && video — video.ext
	AdFormatRewardedVideo       AdFormat = "rewarded_video"       // reward=1 && video — video.ext
	AdFormatMRECDisplay         AdFormat = "mrec_display"         // instl=0, banner, MREC size — banner.ext
	AdFormatMRECVideo           AdFormat = "mrec_video"           // instl=0, video, MREC size — video.ext
	AdFormatBannerDisplay       AdFormat = "banner_display"       // instl=0, banner, non-MREC — banner.ext
)

// OS represents the operating system
type OS string

const (
	OSAndroid OS = "android"
	OSiOS     OS = "ios"
)

const (
	MRECWidth  = 300
	MRECHeight = 250

	// Server injects format-level ext.owsdk.adattributes only for displaymanagerver in [4.1.0, 5.3.0].
	// SDK 5.3.1+ sends adattributes on the request; OpenWrap does not add them.
	OWSDKServerAdAttributesMinSDKVersion = "4.1.0"
	OWSDKServerAdAttributesMaxSDKVersion = "5.3.0"
)

const (
	extOWSDKKey     = "owsdk"
	adAttributesKey = "adattributes"
)

const (
	sdk410 = "4.1.0"
	sdk420 = "4.2.0"
	sdk430 = "4.3.0"
	sdk440 = "4.4.0"
	sdk480 = "4.8.0"
	sdk490 = "4.9.0"
	sdk500 = "5.0.0"
	sdk510 = "5.1.0"
	sdk530 = "5.3.0"
)

// FeatureConfig defines supported ext.owsdk adattribute wire IDs for OS, SDK version, and ad format.
// MinVersion and MaxVersion are inclusive; MaxVersion empty means no upper bound (min only).
// 5.1.0–5.3.0 bands use sdk510–sdk530 range rows.
type FeatureConfig struct {
	OS         OS
	MinVersion string
	MaxVersion string
	AdFormat   AdFormat
	WireIDs    []int
}

// unifiedFeatureMatrix maps OS, SDK version range, and slot-specific AdFormat to supported wire IDs.
// Order matters: GetSupportedAdAttributeWireIDs returns the first match — list newer / narrower version rows before older ones for the same OS+AdFormat.
var unifiedFeatureMatrix = []FeatureConfig{
	// --- Android — interstitial display (banner.ext) ---
	{OS: OSAndroid, MinVersion: sdk510, MaxVersion: sdk530, AdFormat: AdFormatInterstitialDisplay, WireIDs: []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus}},
	{OS: OSAndroid, MinVersion: sdk490, MaxVersion: sdk500, AdFormat: AdFormatInterstitialDisplay, WireIDs: []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay}},
	{OS: OSAndroid, MinVersion: sdk440, MaxVersion: sdk480, AdFormat: AdFormatInterstitialDisplay, WireIDs: []int{AdAttrWireEngageToClose, AdAttrWireTrueDoubleEndCard}},
	{OS: OSAndroid, MinVersion: sdk430, MaxVersion: sdk430, AdFormat: AdFormatInterstitialDisplay, WireIDs: []int{AdAttrWireEngageToClose}},
	{OS: OSAndroid, MinVersion: sdk410, MaxVersion: sdk420, AdFormat: AdFormatInterstitialDisplay, WireIDs: []int{AdAttrWireEngageToClose}},
	// --- Android — interstitial video (video.ext) ---
	{OS: OSAndroid, MinVersion: sdk510, MaxVersion: sdk530, AdFormat: AdFormatInterstitialVideo, WireIDs: []int{AdAttrWireEngageToClose, AdAttrWireTrueDoubleEndCard, AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus}},
	{OS: OSAndroid, MinVersion: sdk490, MaxVersion: sdk500, AdFormat: AdFormatInterstitialVideo, WireIDs: []int{AdAttrWireEngageToClose, AdAttrWireTrueDoubleEndCard, AdAttrWireCTAOverlay}},
	{OS: OSAndroid, MinVersion: sdk440, MaxVersion: sdk480, AdFormat: AdFormatInterstitialVideo, WireIDs: []int{AdAttrWireEngageToClose, AdAttrWireTrueDoubleEndCard}},
	{OS: OSAndroid, MinVersion: sdk430, MaxVersion: sdk430, AdFormat: AdFormatInterstitialVideo, WireIDs: []int{AdAttrWireEngageToClose}},
	// --- Android — rewarded video (video.ext) ---
	{OS: OSAndroid, MinVersion: sdk510, MaxVersion: sdk530, AdFormat: AdFormatRewardedVideo, WireIDs: []int{AdAttrWireEngageToClose, AdAttrWireTrueDoubleEndCard, AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus}},
	{OS: OSAndroid, MinVersion: sdk490, MaxVersion: sdk500, AdFormat: AdFormatRewardedVideo, WireIDs: []int{AdAttrWireEngageToClose, AdAttrWireTrueDoubleEndCard, AdAttrWireCTAOverlay}},
	{OS: OSAndroid, MinVersion: sdk440, MaxVersion: sdk480, AdFormat: AdFormatRewardedVideo, WireIDs: []int{AdAttrWireEngageToClose, AdAttrWireTrueDoubleEndCard}},
	{OS: OSAndroid, MinVersion: sdk430, MaxVersion: sdk430, AdFormat: AdFormatRewardedVideo, WireIDs: []int{AdAttrWireEngageToClose}},
	// --- Android — MREC display / MREC video ---
	{OS: OSAndroid, MinVersion: sdk510, MaxVersion: sdk530, AdFormat: AdFormatMRECDisplay, WireIDs: []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus}},
	{OS: OSAndroid, MinVersion: sdk490, MaxVersion: sdk500, AdFormat: AdFormatMRECDisplay, WireIDs: []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay}},
	// Android 4.1.0–4.8.0 MREC display: same [1] as banner display band (spec: non-MREC and MREC banners)
	{OS: OSAndroid, MinVersion: sdk410, MaxVersion: sdk480, AdFormat: AdFormatMRECDisplay, WireIDs: []int{AdAttrWireEngageToClose}},
	{OS: OSAndroid, MinVersion: sdk510, MaxVersion: sdk530, AdFormat: AdFormatMRECVideo, WireIDs: []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus}},
	{OS: OSAndroid, MinVersion: sdk490, MaxVersion: sdk500, AdFormat: AdFormatMRECVideo, WireIDs: []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay}},
	// --- Android — banner display (non-MREC, instl=0) ---
	{OS: OSAndroid, MinVersion: sdk510, MaxVersion: sdk530, AdFormat: AdFormatBannerDisplay, WireIDs: []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay}},
	// Android 4.1.0–4.8.0 banner display (non-MREC and MREC banners): banner.ext.adattributes [1]
	{OS: OSAndroid, MinVersion: sdk410, MaxVersion: sdk480, AdFormat: AdFormatBannerDisplay, WireIDs: []int{AdAttrWireEngageToClose}},

	// --- iOS — interstitial display (banner.ext) ---
	{OS: OSiOS, MinVersion: sdk510, MaxVersion: sdk530, AdFormat: AdFormatInterstitialDisplay, WireIDs: []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus}},
	{OS: OSiOS, MinVersion: sdk490, MaxVersion: sdk500, AdFormat: AdFormatInterstitialDisplay, WireIDs: []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay}},
	// iOS 4.1.0–4.8.0 interstitial display (covers 4.1–4.2 and 4.3–4.8 bands): banner.ext [1]
	{OS: OSiOS, MinVersion: sdk410, MaxVersion: sdk480, AdFormat: AdFormatInterstitialDisplay, WireIDs: []int{AdAttrWireEngageToClose}},

	// --- iOS — interstitial video ---
	{OS: OSiOS, MinVersion: sdk510, MaxVersion: sdk530, AdFormat: AdFormatInterstitialVideo, WireIDs: []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus}},
	{OS: OSiOS, MinVersion: sdk490, MaxVersion: sdk500, AdFormat: AdFormatInterstitialVideo, WireIDs: []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay}},
	{OS: OSiOS, MinVersion: sdk430, MaxVersion: sdk480, AdFormat: AdFormatInterstitialVideo, WireIDs: []int{AdAttrWireEngageToClose}},
	// --- iOS — rewarded video ---
	{OS: OSiOS, MinVersion: sdk510, MaxVersion: sdk530, AdFormat: AdFormatRewardedVideo, WireIDs: []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus}},
	{OS: OSiOS, MinVersion: sdk490, MaxVersion: sdk500, AdFormat: AdFormatRewardedVideo, WireIDs: []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay}},
	{OS: OSiOS, MinVersion: sdk430, MaxVersion: sdk480, AdFormat: AdFormatRewardedVideo, WireIDs: []int{AdAttrWireEngageToClose}},
	// --- iOS — MREC display / MREC video ---
	{OS: OSiOS, MinVersion: sdk510, MaxVersion: sdk530, AdFormat: AdFormatMRECDisplay, WireIDs: []int{AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus}},
	{OS: OSiOS, MinVersion: sdk490, MaxVersion: sdk500, AdFormat: AdFormatMRECDisplay, WireIDs: []int{AdAttrWireCTAOverlay}},
	// iOS 4.1.0–4.8.0 MREC display: banner.ext [1] (spec groups with banner display band)
	{OS: OSiOS, MinVersion: sdk410, MaxVersion: sdk480, AdFormat: AdFormatMRECDisplay, WireIDs: []int{AdAttrWireEngageToClose}},
	{OS: OSiOS, MinVersion: sdk510, MaxVersion: sdk530, AdFormat: AdFormatMRECVideo, WireIDs: []int{AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus}},
	{OS: OSiOS, MinVersion: sdk490, MaxVersion: sdk500, AdFormat: AdFormatMRECVideo, WireIDs: []int{AdAttrWireCTAOverlay}},
	// --- iOS — banner display (non-MREC) ---
	// iOS 4.1.0–4.8.0 banner display (non-MREC and MREC banners): banner.ext [1]
	{OS: OSiOS, MinVersion: sdk410, MaxVersion: sdk480, AdFormat: AdFormatBannerDisplay, WireIDs: []int{AdAttrWireEngageToClose}},
	{OS: OSiOS, MinVersion: sdk510, MaxVersion: sdk530, AdFormat: AdFormatBannerDisplay, WireIDs: []int{AdAttrWireMRAIDAppStatus}},
}

type adAttrMatrixKey struct {
	os       OS
	adFormat AdFormat
}

type adAttrLookupCacheKey struct {
	os         OS
	sdkVersion string
	adFormat   AdFormat
}

type adAttrCacheEntry struct {
	wireIDs          []int
	serverOWSDKJSON  []byte // {"adattributes":[...]} for inject / fallback
	adAttributesJSON []byte // [...] for jsonparser.Set on existing owsdk
}

var (
	// unifiedFeatureMatrixByOSFormat indexes matrix rows by OS + AdFormat
	// while preserving row order for first-match semantics.
	unifiedFeatureMatrixByOSFormat map[adAttrMatrixKey][]FeatureConfig

	// supportedAdAttributeWireIDsCache memoizes wire IDs and pre-marshaled server
	// owsdk JSON per (OS, SDK version, AdFormat). Negative lookups use empty entry.
	supportedAdAttributeWireIDsCache sync.Map
)

func init() {
	unifiedFeatureMatrixByOSFormat = make(map[adAttrMatrixKey][]FeatureConfig, 16)

	for _, config := range unifiedFeatureMatrix {
		key := adAttrMatrixKey{
			os:       config.OS,
			adFormat: config.AdFormat,
		}
		unifiedFeatureMatrixByOSFormat[key] = append(unifiedFeatureMatrixByOSFormat[key], config)
	}
}

// shouldServerInjectFormatLevelAdAttributes is true when displaymanagerver is in [4.1.0, 5.3.0] inclusive.
func shouldServerInjectFormatLevelAdAttributes(sdkVersion string) bool {
	if sdkVersion == "" {
		return false
	}
	if isVersionLessThan(sdkVersion, OWSDKServerAdAttributesMinSDKVersion) {
		return false
	}
	if isVersionGreaterThan(sdkVersion, OWSDKServerAdAttributesMaxSDKVersion) {
		return false
	}
	return true
}

// mergeOWSDKServerFieldsIntoExtJSON merges pre-marshaled server owsdk JSON into a format object's ext JSON.
// Sibling ext keys are preserved via jsonparser without a full ext map round-trip. On inject, jsonparser.Set is tried
// first; if Set fails or the result is not valid JSON, fall back to owsdk-only ext (invalid ext cannot be patched safely).
func mergeOWSDKServerFieldsIntoExtJSON(extJSON json.RawMessage, serverOWSDKJSON, adAttributesJSON []byte) (json.RawMessage, error) {
	if len(serverOWSDKJSON) == 0 {
		return extJSON, nil
	}

	if len(extJSON) == 0 {
		return marshalExtOWSDKOnly(serverOWSDKJSON)
	}

	existingOWSDK, _, _, err := jsonparser.Get(extJSON, extOWSDKKey)
	if err != nil {
		if errors.Is(err, jsonparser.KeyPathNotFoundError) {
			out, setErr := jsonparser.Set(extJSON, serverOWSDKJSON, extOWSDKKey)
			if setErr != nil || !json.Valid(out) {
				return marshalExtOWSDKOnly(serverOWSDKJSON)
			}
			return out, nil
		}
		return marshalExtOWSDKOnly(serverOWSDKJSON)
	}

	mergedOWSDKBytes, err := mergeServerFieldsIntoOWSDKJSON(existingOWSDK, adAttributesJSON)
	if err != nil {
		return extJSON, err
	}
	return jsonparser.Set(extJSON, mergedOWSDKBytes, extOWSDKKey)
}

// mergeServerFieldsIntoOWSDKJSON patches adattributes into existing owsdk JSON via jsonparser.Set.
// Falls back to map merge when Set fails or produces invalid JSON.
func mergeServerFieldsIntoOWSDKJSON(existingOWSDK []byte, adAttributesJSON []byte) ([]byte, error) {
	if len(adAttributesJSON) == 0 {
		if len(existingOWSDK) == 0 {
			return []byte("{}"), nil
		}
		return existingOWSDK, nil
	}

	owsdkJSON := existingOWSDK
	if len(owsdkJSON) == 0 {
		owsdkJSON = []byte("{}")
	}

	merged, setErr := jsonparser.Set(owsdkJSON, adAttributesJSON, adAttributesKey)
	if setErr != nil || !json.Valid(merged) {
		return mergeServerFieldsIntoOWSDKJSONMap(existingOWSDK, adAttributesJSON)
	}
	return merged, nil
}

func mergeServerFieldsIntoOWSDKJSONMap(existingOWSDK []byte, adAttributesJSON []byte) ([]byte, error) {
	owsdkOut := make(map[string]any)
	if len(existingOWSDK) > 0 {
		_ = json.Unmarshal(existingOWSDK, &owsdkOut)
	}
	if len(adAttributesJSON) > 0 {
		var wireIDs []int
		if err := json.Unmarshal(adAttributesJSON, &wireIDs); err == nil {
			owsdkOut[adAttributesKey] = wireIDs
		}
	}
	return json.Marshal(owsdkOut)
}

func marshalExtOWSDKOnly(owsdkBytes []byte) (json.RawMessage, error) {
	return json.Marshal(map[string]json.RawMessage{
		extOWSDKKey: owsdkBytes,
	})
}

// ApplyOWSDKFormatLevelAdAttributes sets imp.banner|video ext.owsdk.adattributes from unifiedFeatureMatrix
// for displaymanagerver 4.1.0–5.3.0 only. imp.native is skipped.
func ApplyOWSDKFormatLevelAdAttributes(imp *openrtb2.Imp, impCtx models.ImpCtx, deviceOS string) error {
	if imp == nil {
		return nil
	}
	// adattributes feature is not supported for native
	if imp.Native != nil && imp.Banner == nil && imp.Video == nil {
		return nil
	}

	deviceOS = strings.TrimSpace(deviceOS)
	if deviceOS == "" {
		return nil
	}
	sdkVersion := strings.TrimSpace(impCtx.DisplayManagerVer)
	if !shouldServerInjectFormatLevelAdAttributes(sdkVersion) {
		return nil
	}

	os := DetermineOS(deviceOS)
	if os == "" {
		return nil
	}

	var errs []error
	applyAdAttributes := func(ext json.RawMessage, format AdFormat) (json.RawMessage, error) {
		if format == "" {
			return ext, nil
		}
		entry := loadAdAttrCacheEntry(os, sdkVersion, format)
		if len(entry.serverOWSDKJSON) == 0 {
			return ext, nil
		}
		return mergeOWSDKServerFieldsIntoExtJSON(ext, entry.serverOWSDKJSON, entry.adAttributesJSON)
	}

	if imp.Banner != nil {
		if out, err := applyAdAttributes(imp.Banner.Ext, DetermineAdFormatForBanner(impCtx)); err != nil {
			errs = append(errs, fmt.Errorf("banner: %w", err))
		} else {
			imp.Banner.Ext = out
		}
	}
	if imp.Video != nil {
		if out, err := applyAdAttributes(imp.Video.Ext, DetermineAdFormatForVideo(impCtx)); err != nil {
			errs = append(errs, fmt.Errorf("video: %w", err))
		} else {
			imp.Video.Ext = out
		}
	}
	return errors.Join(errs...)
}

// GetSupportedAdAttributeWireIDs returns the supported ext.owsdk.adattributes
// wire IDs for the given OS, SDK version, and ad format.
//
// Callers are expected to pass a trimmed and validated SDK version
// (currently supported range: [4.1.0, 5.3.0]).
func GetSupportedAdAttributeWireIDs(os OS, sdkVersion string, adFormat AdFormat) []int {
	entry := loadAdAttrCacheEntry(os, sdkVersion, adFormat)
	if len(entry.wireIDs) == 0 {
		return nil
	}
	return slices.Clone(entry.wireIDs)
}

func loadAdAttrCacheEntry(os OS, sdkVersion string, adFormat AdFormat) adAttrCacheEntry {
	cacheKey := adAttrLookupCacheKey{
		os:         os,
		sdkVersion: sdkVersion,
		adFormat:   adFormat,
	}

	if cached, ok := supportedAdAttributeWireIDsCache.Load(cacheKey); ok {
		return cached.(adAttrCacheEntry)
	}

	entry := adAttrCacheEntry{}
	if wireIDs := lookupSupportedAdAttributeWireIDs(os, sdkVersion, adFormat); wireIDs != nil {
		serverOWSDK := CreateOWSDKExtension(wireIDs)
		if len(serverOWSDK) > 0 {
			ids, ok := serverOWSDK[adAttributesKey].([]int)
			if !ok {
				return entry
			}
			entry.wireIDs = slices.Clone(ids)
			var err error
			entry.serverOWSDKJSON, err = json.Marshal(serverOWSDK)
			if err != nil {
				return adAttrCacheEntry{}
			}
			entry.adAttributesJSON, err = json.Marshal(ids)
			if err != nil {
				return adAttrCacheEntry{}
			}
		}
	}

	actual, _ := supportedAdAttributeWireIDsCache.LoadOrStore(cacheKey, entry)
	return actual.(adAttrCacheEntry)
}

func lookupSupportedAdAttributeWireIDs(os OS, sdkVersion string, adFormat AdFormat) []int {
	key := adAttrMatrixKey{
		os:       os,
		adFormat: adFormat,
	}

	for _, config := range unifiedFeatureMatrixByOSFormat[key] {
		if isVersionInRange(sdkVersion, config.MinVersion, config.MaxVersion) {
			return config.WireIDs
		}
	}

	return nil
}

// isVideoEffectiveForAdFormat returns true when the impression still carries video for format / adattribute
// selection. Video may remain on imp until applyVideoAdUnitConfig runs, while ad unit config can already mark
// video disabled — align with the post-mutation request shape.
func isVideoEffectiveForAdFormat(impCtx models.ImpCtx) bool {
	if impCtx.Video == nil {
		return false
	}
	cfg := impCtx.VideoAdUnitCtx.AppliedSlotAdUnitConfig
	if cfg == nil || cfg.Video == nil || cfg.Video.Enabled == nil {
		return true
	}
	return *cfg.Video.Enabled
}

// isBannerEffectiveForAdFormat is the banner counterpart of isVideoEffectiveForAdFormat.
func isBannerEffectiveForAdFormat(impCtx models.ImpCtx) bool {
	if impCtx.Banner == nil {
		return false
	}
	cfg := impCtx.BannerAdUnitCtx.AppliedSlotAdUnitConfig
	if cfg == nil || cfg.Banner == nil || cfg.Banner.Enabled == nil {
		return true
	}
	return *cfg.Banner.Enabled
}

// isMRECSize implements the request-spec isMRECSize() gate: 300×250 on imp.banner.
func isMRECSize(impCtx models.ImpCtx) bool {
	if impCtx.Banner == nil || impCtx.Banner.W == nil || impCtx.Banner.H == nil {
		return false
	}
	return *impCtx.Banner.W == MRECWidth && *impCtx.Banner.H == MRECHeight
}

// isMRECVideoSize returns true when imp.video is 300×250.
func isMRECVideoSize(impCtx models.ImpCtx) bool {
	if impCtx.Video == nil || impCtx.Video.W == nil || impCtx.Video.H == nil {
		return false
	}
	return *impCtx.Video.W == MRECWidth && *impCtx.Video.H == MRECHeight
}

// isMRECVideo is true when instl is 0 or absent and imp.video is MREC size.
// Caller must ensure video is effective via isVideoEffectiveForAdFormat.
func isMRECVideo(impCtx models.ImpCtx) bool {
	return impCtx.Instl != 1 && isMRECVideoSize(impCtx)
}

func isRewardedVideoRequest(impCtx models.ImpCtx) bool {
	return impCtx.IsRewardInventory != nil && *impCtx.IsRewardInventory == 1
}

// DetermineAdFormatForVideo maps imp.video to a matrix row (Approach 2 request spec):
//   - Rewarded video:     reward == 1 && imp.video != nil (effective)
//   - Interstitial video: instl == 1 && imp.video != nil (effective)
//   - MREC video:         isMRECVideo (instl == 0 or absent && imp.video != nil && imp.video is 300×250)
func DetermineAdFormatForVideo(impCtx models.ImpCtx) AdFormat {
	if !isVideoEffectiveForAdFormat(impCtx) {
		return ""
	}
	if isRewardedVideoRequest(impCtx) {
		return AdFormatRewardedVideo
	}
	if impCtx.Instl == 1 {
		return AdFormatInterstitialVideo
	}
	if isMRECVideo(impCtx) {
		return AdFormatMRECVideo
	}
	return ""
}

// DetermineAdFormatForBanner maps imp.banner to a matrix row (Approach 2 request spec):
//   - Interstitial display: instl == 1 && imp.banner != nil (effective)
//   - MREC display:         instl == 0 or absent && imp.banner != nil (effective) && isMRECSize()
//   - Banner display:       instl == 0 or absent && imp.banner != nil (effective) && !isMRECSize()
func DetermineAdFormatForBanner(impCtx models.ImpCtx) AdFormat {
	if !isBannerEffectiveForAdFormat(impCtx) {
		return ""
	}
	if impCtx.Instl == 1 {
		return AdFormatInterstitialDisplay
	}
	if isMRECSize(impCtx) {
		return AdFormatMRECDisplay
	}
	return AdFormatBannerDisplay
}

// DetermineOS determines the OS based on device information
func DetermineOS(deviceOS string) OS {
	os := strings.ToLower(strings.TrimSpace(deviceOS))
	switch {
	case strings.Contains(os, "android"):
		return OSAndroid
	case strings.Contains(os, "ios"), strings.Contains(os, "iphone"), strings.Contains(os, "ipad"):
		return OSiOS
	default:
		return "" // Unknown OS
	}
}

// isVersionLessThan checks if version1 is less than version2
func isVersionLessThan(version1, version2 string) bool {
	return compareVersions(version1, version2) < 0
}

// isVersionGreaterThan checks if version1 is greater than version2
func isVersionGreaterThan(version1, version2 string) bool {
	return compareVersions(version1, version2) > 0
}

// isVersionInRange checks if version is within the specified range (inclusive)
func isVersionInRange(version, minVersion, maxVersion string) bool {
	// If maxVersion is empty, it means no upper bound
	if maxVersion == "" {
		return compareVersions(version, minVersion) >= 0
	}

	return compareVersions(version, minVersion) >= 0 && compareVersions(version, maxVersion) <= 0
}

// compareVersions compares two dot-separated numeric version strings (e.g. "5.1.0").
// Non-numeric segments are treated as 0; leading/trailing whitespace is ignored.
func compareVersions(v1, v2 string) int {
	p1 := strings.Split(strings.TrimSpace(v1), ".")
	p2 := strings.Split(strings.TrimSpace(v2), ".")

	for i := 0; i < max(len(p1), len(p2)); i++ {
		n1, n2 := 0, 0

		if i < len(p1) {
			n1, _ = strconv.Atoi(p1[i])
		}
		if i < len(p2) {
			n2, _ = strconv.Atoi(p2[i])
		}

		switch {
		case n1 < n2:
			return -1
		case n1 > n2:
			return 1
		}
	}

	return 0
}

// CreateOWSDKExtension builds ext.owsdk with sorted, deduplicated, positive adattribute wire IDs.
func CreateOWSDKExtension(wireIDs []int) map[string]any {
	if len(wireIDs) == 0 {
		return nil
	}

	out := make([]int, 0, len(wireIDs))
	for _, id := range wireIDs {
		if id > 0 {
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		return nil
	}

	slices.Sort(out)
	out = slices.Compact(out)

	return map[string]any{
		adAttributesKey: out,
	}
}
