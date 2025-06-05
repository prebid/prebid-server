package wurfl_devicedetection

import (
	"fmt"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
)

const (
	SEC_CH_UA                   = "Sec-CH-UA"
	SEC_CH_UA_PLATFORM          = "Sec-CH-UA-Platform"
	SEC_CH_UA_PLATFORM_VERSION  = "Sec-CH-UA-Platform-Version"
	SEC_CH_UA_MOBILE            = "Sec-CH-UA-Mobile"
	SEC_CH_UA_ARCH              = "Sec-CH-UA-Arch"
	SEC_CH_UA_MODEL             = "Sec-CH-UA-Model"
	SEC_CH_UA_FULL_VERSION      = "Sec-CH-UA-Full-Version"
	SEC_CH_UA_FULL_VERSION_LIST = "Sec-CH-UA-Full-Version-List"
	USER_AGENT                  = "User-Agent"
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
		headers[USER_AGENT] = ua
	}

	if sua == nil {
		return headers
	}

	if sua.Browsers == nil {
		return headers
	}

	brandList := makeBrandList(sua.Browsers)
	headers[SEC_CH_UA] = brandList
	headers[SEC_CH_UA_FULL_VERSION_LIST] = brandList

	if sua.Platform != nil {
		headers[SEC_CH_UA_PLATFORM] = escapeClientHintField(sua.Platform.Brand)
		headers[SEC_CH_UA_PLATFORM_VERSION] = escapeClientHintField(strings.Join(sua.Platform.Version, "."))
	}

	if sua.Model != "" {
		headers[SEC_CH_UA_MODEL] = escapeClientHintField(sua.Model)
	}

	if sua.Architecture != "" {
		headers[SEC_CH_UA_ARCH] = escapeClientHintField(sua.Architecture)
	}

	if sua.Mobile != nil {
		headers[SEC_CH_UA_MOBILE] = fmt.Sprintf("?%d", *sua.Mobile)
	}

	return headers
}

func makeBrandList(brandVersions []openrtb2.BrandVersion) string {
	var builder strings.Builder
	first := true
	for _, version := range brandVersions {
		if version.Brand == "" {
			continue
		}
		if !first {
			builder.WriteString(", ")
		}
		first = false

		brandName := escapeClientHintField(version.Brand)
		builder.WriteString(brandName)
		builder.WriteString(`;v="`)
		builder.WriteString(strings.Join(version.Version, "."))
		builder.WriteString(`"`)
	}
	return builder.String()
}

func escapeClientHintField(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}
