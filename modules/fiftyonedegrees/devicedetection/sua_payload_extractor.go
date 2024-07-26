package devicedetection

import (
	"fmt"
	"strings"

	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
)

const (
	secChUaArch            = "Sec-Ch-Ua-Arch"
	secChUaMobile          = "Sec-Ch-Ua-Mobile"
	secChUaModel           = "Sec-Ch-Ua-Model"
	secChUaPlatform        = "Sec-Ch-Ua-Platform"
	secUaFullVersionList   = "Sec-Ch-Ua-Full-Version-List"
	secChUaPlatformVersion = "Sec-Ch-Ua-Platform-Version"
	secChUa                = "Sec-Ch-Ua"

	userAgentHeader = "User-Agent"
)

// evidenceFromSUAPayloadExtractor extracts evidence from the SUA payload of device
type evidenceFromSUAPayloadExtractor struct{}

func newEvidenceFromSUAPayloadExtractor() evidenceFromSUAPayloadExtractor {
	return evidenceFromSUAPayloadExtractor{}
}

// Extract extracts evidence from the SUA payload
func (x evidenceFromSUAPayloadExtractor) extract(payload []byte) []stringEvidence {
	if payload != nil {
		return x.extractEvidenceStrings(payload)
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
func (x evidenceFromSUAPayloadExtractor) extractEvidenceStrings(payload []byte) []stringEvidence {
	res := make([]stringEvidence, 0, 10)

	uaResult := gjson.GetBytes(payload, uaPath)
	if uaResult.Exists() {
		res = append(
			res,
			stringEvidence{Prefix: headerPrefix, Key: userAgentHeader, Value: uaResult.String()},
		)
	}

	archResult := gjson.GetBytes(payload, archPath)
	if archResult.Exists() {
		res = x.appendEvidenceIfExists(res, secChUaArch, archResult.String())
	}

	mobileResult := gjson.GetBytes(payload, mobilePath)
	if mobileResult.Exists() {
		res = x.appendEvidenceIfExists(res, secChUaMobile, mobileResult.String())
	}

	modelResult := gjson.GetBytes(payload, modelPath)
	if modelResult.Exists() {
		res = x.appendEvidenceIfExists(res, secChUaModel, modelResult.String())
	}

	platformBrandResult := gjson.GetBytes(payload, platformBrandPath)
	if platformBrandResult.Exists() {
		res = x.appendEvidenceIfExists(res, secChUaPlatform, platformBrandResult.String())
	}

	platformVersionResult := gjson.GetBytes(payload, platformVersionPath)
	if platformVersionResult.Exists() {
		res = x.appendEvidenceIfExists(
			res,
			secChUaPlatformVersion,
			strings.Join(resultToStringArray(platformVersionResult.Array()), "."),
		)
	}

	browsersResult := gjson.GetBytes(payload, browsersPath)
	if browsersResult.Exists() {
		res = x.appendEvidenceIfExists(res, secUaFullVersionList, x.extractBrowsers(browsersResult))

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
func (x evidenceFromSUAPayloadExtractor) appendEvidenceIfExists(destination []stringEvidence, name string, value interface{}) []stringEvidence {
	if value != nil {
		valStr := cast.ToString(value)
		if len(valStr) == 0 {
			return destination
		}

		return append(
			destination,
			stringEvidence{Prefix: headerPrefix, Key: name, Value: valStr},
		)
	}

	return destination
}

// extractBrowsers extracts browsers from the SUA payload
func (x evidenceFromSUAPayloadExtractor) extractBrowsers(browsers gjson.Result) string {
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
