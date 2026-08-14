package sdkutils

import (
	"strconv"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
)

func MergeDevice(dst *openrtb2.Device, src *openrtb2.Device) *openrtb2.Device {
	if src == nil {
		return dst
	}

	if dst == nil {
		dst = &openrtb2.Device{}
	}

	if len(src.UA) > 0 {
		dst.UA = src.UA
	}

	if len(src.Make) > 0 {
		dst.Make = src.Make
	}

	if len(src.Model) > 0 {
		dst.Model = src.Model
	}

	if src.JS != nil {
		dst.JS = src.JS
	}

	if src.IP != "" {
		dst.IP = src.IP
	}

	if src.IPv6 != "" {
		dst.IPv6 = src.IPv6
	}

	if src.DeviceType > 0 {
		dst.DeviceType = src.DeviceType
	}

	if src.IFA != "" {
		dst.IFA = src.IFA
	}

	if src.Geo != nil {
		dst.Geo = mergeGeo(dst.Geo, src.Geo)
	}

	if src.HWV != "" {
		dst.HWV = src.HWV
	}

	if src.Lmt != nil {
		dst.Lmt = src.Lmt
	}

	if src.OS != "" {
		dst.OS = src.OS
	}

	if src.OSV != "" {
		dst.OSV = src.OSV
	}

	if src.W > 0 {
		dst.W = src.W
	}

	if src.H > 0 {
		dst.H = src.H
	}

	if src.PxRatio > 0 {
		dst.PxRatio = src.PxRatio
	}

	if src.Language != "" {
		dst.Language = src.Language
	}

	if src.Carrier != "" {
		dst.Carrier = src.Carrier
	}

	if src.MCCMNC != "" {
		dst.MCCMNC = src.MCCMNC
	}

	if src.ConnectionType != nil {
		dst.ConnectionType = src.ConnectionType
	}

	// UOE-13721 device.ppi: copied from SDK signal when present.
	if src.PPI > 0 {
		dst.PPI = src.PPI
	}

	return dst
}

// MergeBanner copies imp.banner fields from SDK signal into the shared request banner.
// UOE-13721 imp.banner.mimes and api are taken from signal when non-empty; outer-request values are overwritten.
func MergeBanner(dst, src *openrtb2.Banner) {
	if dst == nil || src == nil {
		return
	}

	if len(src.MIMEs) > 0 {
		dst.MIMEs = src.MIMEs
	}

	if src.API != nil {
		dst.API = src.API
	}
}

func mergeGeo(dst *openrtb2.Geo, src *openrtb2.Geo) *openrtb2.Geo {
	if dst == nil {
		dst = &openrtb2.Geo{}
	}

	hasReqLatLon := dst.Lat != nil && dst.Lon != nil
	if !hasReqLatLon {
		dst.Lat = src.Lat
		dst.Lon = src.Lon
		dst.Type = src.Type
		dst.LastFix = src.LastFix
		dst.Accuracy = src.Accuracy
	}

	if src.Country != "" {
		dst.Country = src.Country
	}
	if src.Region != "" {
		dst.Region = src.Region
	}
	if src.Metro != "" {
		dst.Metro = src.Metro
	}
	if src.City != "" {
		dst.City = src.City
	}
	if src.ZIP != "" {
		dst.ZIP = src.ZIP
	}
	if src.UTCOffset != 0 {
		dst.UTCOffset = src.UTCOffset
	}

	return dst
}

func CopyPath(source []byte, target []byte, path ...string) ([]byte, error) {
	if source == nil {
		return target, nil
	}

	value, dataType, _, err := jsonparser.Get(source, path...)
	if value == nil || err != nil {
		return target, err
	}

	// Initialize target if nil
	if target == nil {
		target = []byte(`{}`)
	}

	// Early return for null values
	if dataType == jsonparser.Null {
		return jsonparser.Set(target, nil, path...)
	}

	// Handle empty values based on data type
	switch dataType {
	case jsonparser.String:
		// Only skip if it's an empty string
		if len(value) == 0 {
			return target, nil
		}
		// Quote the string value
		quotedValue := []byte(`"` + string(value) + `"`)
		return jsonparser.Set(target, quotedValue, path...)

	case jsonparser.Number:
		// Preserve numeric value
		return jsonparser.Set(target, value, path...)

	case jsonparser.Boolean:
		// Preserve boolean value
		return jsonparser.Set(target, value, path...)

	case jsonparser.Array, jsonparser.Object:
		// Skip empty arrays/objects
		if len(value) <= 2 { // "[]" or "{}" are 2 chars
			return target, nil
		}
		// Preserve the complex value as is
		return jsonparser.Set(target, value, path...)

	default:
		// For unknown types, copy as is
		return jsonparser.Set(target, value, path...)
	}
}

// CopyIFV copies the ifv key from source to target JSON, including empty string values.
// Unlike CopyPath, this does not skip empty strings.
func CopyIFV(source, target []byte) []byte {
	value, _, _, err := jsonparser.Get(source, "ifv")
	if err != nil {
		return target
	}
	if target == nil {
		target = []byte(`{}`)
	}
	if result, err := jsonparser.Set(target, []byte(`"`+string(value)+`"`), "ifv"); err == nil {
		return result
	}
	return target
}

// SetIfKeysExists copies keys from source JSON into target when present in source.
// Unlike CopyPath, numeric zero and other falsy values are preserved.
func SetIfKeysExists(source []byte, target []byte, keys ...string) []byte {
	newTarget := target
	if len(keys) > 0 && len(newTarget) == 0 {
		newTarget = []byte(`{}`)
	}

	for _, key := range keys {
		field, dataType, _, err := jsonparser.Get(source, key)
		if err != nil {
			continue
		}

		if dataType == jsonparser.String {
			quotedStr := strconv.Quote(string(field))
			field = []byte(quotedStr)
		}

		newTarget, err = jsonparser.Set(newTarget, field, key)
		if err != nil {
			return target
		}
	}

	if len(newTarget) == 2 {
		return target
	}
	return newTarget
}

func IsSdkBiddingEndpoint(endpoint string) bool {
	return endpoint == models.EndpointAppLovinMax || endpoint == models.EndpointUnityLevelPlay || endpoint == models.EndpointGoogleSDK || endpoint == models.EndpointAPS
}

// IsSdkEndpoint returns true for SDK mediation endpoints and the v25 endpoint.
func IsSdkEndpoint(endpoint string) bool {
	return IsSdkBiddingEndpoint(endpoint) || endpoint == models.EndpointV25
}

func AddSize300x600ForInterstitialBanner(imp *openrtb2.Imp) {
	if imp.Banner == nil {
		return
	}
	var phonePortrait, tabletPortrait, tabletLandscape bool
	if imp.Banner.W != nil && imp.Banner.H != nil {
		if *imp.Banner.W == 320 && *imp.Banner.H == 480 {
			phonePortrait = true
		}
		if *imp.Banner.W == 768 && *imp.Banner.H == 1024 {
			tabletPortrait = true
		}
		if *imp.Banner.W == 1024 && *imp.Banner.H == 768 {
			tabletLandscape = true
		}
		if *imp.Banner.W == 300 && *imp.Banner.H == 600 {
			return
		}
	}

	for _, size := range imp.Banner.Format {
		if size.W == 320 && size.H == 480 {
			phonePortrait = true
		}
		if size.W == 768 && size.H == 1024 {
			tabletPortrait = true
		}
		if size.W == 1024 && size.H == 768 {
			tabletLandscape = true
		}
		if size.W == 300 && size.H == 600 {
			return
		}
	}

	if phonePortrait || tabletPortrait || tabletLandscape {
		imp.Banner.Format = append(imp.Banner.Format, openrtb2.Format{W: 300, H: 600})
	}
}

func IsGoogleSDKResponseRejected(rCtx *models.RequestCtx, ao analytics.AuctionObject) bool {
	if ao.Response == nil || rCtx == nil || rCtx.Endpoint != models.EndpointGoogleSDK {
		return false
	}

	if !rCtx.GoogleSDK.Reject && ao.Response.NBR == nil {
		return false
	}
	return true
}

func IsIOSDevice(device *openrtb2.Device) bool {
	return device != nil && strings.EqualFold(device.OS, "ios")
}
