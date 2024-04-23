package device_detection

import (
	"fmt"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
	"net/http"
	"strings"
)

const (
	SecChUaArch            = "Sec-Ch-Ua-Arch"
	SecChUaMobile          = "Sec-Ch-Ua-Mobile"
	SecChUaModel           = "Sec-Ch-Ua-Model"
	SecChUaPlatform        = "Sec-Ch-Ua-Platform"
	SecUaFullVersionList   = "Sec-Ch-Ua-Full-Version-List"
	SecChUaPlatformVersion = "Sec-Ch-Ua-Platform-Version"
	SecChUa                = "Sec-Ch-Ua"

	UserAgentHeader = "User-Agent"
)

// EvidenceFromSUAPayloadExtractor extracts evidence from the SUA payload of device
type EvidenceFromSUAPayloadExtractor struct{}

func NewEvidenceFromSUAPayloadExtractor() *EvidenceFromSUAPayloadExtractor {
	return &EvidenceFromSUAPayloadExtractor{}
}

// Extract extracts evidence from the SUA payload
func (x EvidenceFromSUAPayloadExtractor) Extract(r *http.Request, paylod []byte) []StringEvidence {
	if paylod != nil {
		return x.extractEvidenceStrings(paylod)
	}

	return nil
}

var (
	uaPath              = "device.ua"
	archPath            = "device.sua.architecture"
	mobilePath          = "device.sua.mobile"
	modelPath           = "device.sua.model"
	platformBrandPath   = "device.sua.platform.brand"
	platformVersionPath = "device.sua.platform.version"
	browsersPath        = "device.sua.browsers"
)

// extractEvidenceStrings extracts evidence from the SUA payload
func (x EvidenceFromSUAPayloadExtractor) extractEvidenceStrings(payload []byte) []StringEvidence {
	res := make([]StringEvidence, 0, 10)

	uaResult := gjson.GetBytes(payload, uaPath)
	if uaResult.Exists() {
		res = append(
			res,
			StringEvidence{Prefix: HeaderPrefix, Key: UserAgentHeader, Value: uaResult.String()},
		)
	}

	archResult := gjson.GetBytes(payload, archPath)
	if archResult.Exists() {
		res = x.appendEvidenceIfExists(res, SecChUaArch, archResult.String())
	}

	mobileResult := gjson.GetBytes(payload, mobilePath)
	if mobileResult.Exists() {
		res = x.appendEvidenceIfExists(res, SecChUaMobile, mobileResult.String())
	}

	modelResult := gjson.GetBytes(payload, modelPath)
	if modelResult.Exists() {
		res = x.appendEvidenceIfExists(res, SecChUaModel, modelResult.String())
	}

	platformBrandResult := gjson.GetBytes(payload, platformBrandPath)
	if platformBrandResult.Exists() {
		res = x.appendEvidenceIfExists(res, SecChUaPlatform, platformBrandResult.String())
	}

	platformVersionResult := gjson.GetBytes(payload, platformVersionPath)
	if platformVersionResult.Exists() {
		res = x.appendEvidenceIfExists(
			res,
			SecChUaPlatformVersion,
			strings.Join(resultToStringArray(platformVersionResult.Array()), "."),
		)
	}

	browsersResult := gjson.GetBytes(payload, browsersPath)
	if browsersResult.Exists() {
		res = x.appendEvidenceIfExists(res, SecUaFullVersionList, x.extractBrowsers(browsersResult))

	}

	return res
}

func resultToStringArray(array []gjson.Result) []string {
	strArray := make([]string, len(array))
	for i, result := range array {
		strArray[i] = result.String()
	}

	return strArray
}

// appendEvidenceIfExists appends evidence to the destination if the value is not nil
func (x EvidenceFromSUAPayloadExtractor) appendEvidenceIfExists(destination []StringEvidence, name string, value interface{}) []StringEvidence {
	if value != nil {
		valStr := cast.ToString(value)
		if len(valStr) == 0 {
			return destination
		}

		return append(
			destination,
			StringEvidence{Prefix: HeaderPrefix, Key: name, Value: valStr},
		)
	}

	return destination
}

// extractBrowsers extracts browsers from the SUA payload
func (x EvidenceFromSUAPayloadExtractor) extractBrowsers(browsers gjson.Result) string {
	if !browsers.IsArray() {
		return ""
	}

	browsersRaw := make([]string, len(browsers.Array()))

	for i, result := range browsers.Array() {
		brand := result.Get("brand").String()
		versionsRaw := result.Get("version").Array()
		versions := resultToStringArray(versionsRaw)

		browsersRaw[i] = fmt.Sprintf(`"%s";v="%s"`, brand, strings.Join(versions, "."))
	}

	res := strings.Join(browsersRaw, ",")

	return res
}
