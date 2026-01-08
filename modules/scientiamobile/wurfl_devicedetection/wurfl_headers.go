package wurfl_devicedetection

import (
	"fmt"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/util/iterutil"
)

const (
	secCHUA                 = "Sec-CH-UA"
	secCHUAPlatform         = "Sec-CH-UA-Platform"
	secCHUAPlatformVersion  = "Sec-CH-UA-Platform-Version"
	secCHUAMobile           = "Sec-CH-UA-Mobile"
	secCHUAArch             = "Sec-CH-UA-Arch"
	secCHUAModel            = "Sec-CH-UA-Model"
	secCHUAFullVersion      = "Sec-CH-UA-Full-Version"
	secCHUAFullVersionList  = "Sec-CH-UA-Full-Version-List"
	userAgent               = "User-Agent"
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
		headers[secCHUAPlatform] = quoteClientHintField(sua.Platform.Brand)
		headers[secCHUAPlatformVersion] = quoteClientHintField(strings.Join(sua.Platform.Version, "."))
	}

	if sua.Model != "" {
		headers[secCHUAModel] = quoteClientHintField(sua.Model)
	}

	if sua.Architecture != "" {
		headers[secCHUAArch] = quoteClientHintField(sua.Architecture)
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

		brandName := quoteClientHintField(version.Brand)
		builder.WriteString(brandName)
		builder.WriteString(`;v="`)
		builder.WriteString(strings.Join(version.Version, "."))
		builder.WriteString(`"`)
	}
	return builder.String()
}

func quoteClientHintField(value string) string {
	return `"` + value + `"`
}
