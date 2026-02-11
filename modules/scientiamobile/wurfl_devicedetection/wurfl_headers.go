package wurfl_devicedetection

import (
	"fmt"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/util/iterutil"
)

const (
	secCHUA                = "Sec-CH-UA"
	secCHUAPlatform        = "Sec-CH-UA-Platform"
	secCHUAPlatformVersion = "Sec-CH-UA-Platform-Version"
	secCHUAMobile          = "Sec-CH-UA-Mobile"
	secCHUAArch            = "Sec-CH-UA-Arch"
	secCHUAModel           = "Sec-CH-UA-Model"
	secCHUAFullVersionList = "Sec-CH-UA-Full-Version-List"
	userAgent              = "User-Agent"
)

// clientHintEscaper escapes special characters in Client Hint header values per RFC 9651.
// Only backslash and double-quote need escaping. Backslash must be escaped first.
var clientHintEscaper = strings.NewReplacer(
	`\`, `\\`, // Must escape backslash FIRST
	`"`, `\"`, // Then escape quotes
)

func makeHeaders(ortb2Device openrtb2.Device, rawHeaders map[string]string) map[string]string {
	sua := ortb2Device.SUA
	ua := ortb2Device.UA
	if ua == "" {
		if sua == nil {
			return rawHeaders
		}
		if sua.Browsers == nil {
			return rawHeaders
		}
	}
	headers := make(map[string]string)

	if ua != "" {
		headers[userAgent] = ua
	}

	if sua == nil {
		return headers
	}

	if sua.Browsers == nil {
		return headers
	}

	brandList := makeBrandList(sua.Browsers)
	headers[secCHUA] = brandList
	headers[secCHUAFullVersionList] = brandList

	if sua.Platform != nil {
		headers[secCHUAPlatform] = quoteAndEscapeClientHintField(sua.Platform.Brand)
		headers[secCHUAPlatformVersion] = quoteAndEscapeClientHintField(strings.Join(sua.Platform.Version, "."))
	}

	if sua.Model != "" {
		headers[secCHUAModel] = quoteAndEscapeClientHintField(sua.Model)
	}

	if sua.Architecture != "" {
		headers[secCHUAArch] = quoteAndEscapeClientHintField(sua.Architecture)
	}

	if sua.Mobile != nil {
		headers[secCHUAMobile] = fmt.Sprintf("?%d", *sua.Mobile)
	}

	return headers
}

func makeBrandList(brandVersions []openrtb2.BrandVersion) string {
	var builder strings.Builder
	first := true
	for version := range iterutil.SlicePointerValues(brandVersions) {
		if version.Brand == "" {
			continue
		}
		if !first {
			builder.WriteString(", ")
		}
		first = false

		brandName := quoteAndEscapeClientHintField(version.Brand)
		builder.WriteString(brandName)
		builder.WriteString(`;v="`)
		builder.WriteString(strings.Join(version.Version, "."))
		builder.WriteString(`"`)
	}
	return builder.String()
}

// quoteAndEscapeClientHintField escapes special characters per RFC 9651 and wraps
// the value in double quotes for use in HTTP Client Hint header values.
// Backslashes and double-quotes are escaped as required by the structured field specification.
func quoteAndEscapeClientHintField(value string) string {
	return `"` + clientHintEscaper.Replace(value) + `"`
}
