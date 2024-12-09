package version

import (
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

const xPrebidHeaderVersionPrefix = "pbs-go"

func BuildXPrebidHeader(version string) string {
	sb := &strings.Builder{}
	writeNameVersionRecord(sb, xPrebidHeaderVersionPrefix, version)
	return sb.String()
}

func BuildXPrebidHeaderForRequest(bidRequest *openrtb2.BidRequest, version string) string {
	req := &openrtb_ext.RequestWrapper{BidRequest: bidRequest}

	sb := &strings.Builder{}
	writeNameVersionRecord(sb, xPrebidHeaderVersionPrefix, version)

	if reqExt, err := req.GetRequestExt(); err == nil && reqExt != nil {
		if prebidExt := reqExt.GetPrebid(); prebidExt != nil {
			if channel := prebidExt.Channel; channel != nil {
				writeNameVersionRecord(sb, channel.Name, channel.Version)
			}
		}
	}
	if appExt, err := req.GetAppExt(); err == nil && appExt != nil {
		if prebidExt := appExt.GetPrebid(); prebidExt != nil {
			writeNameVersionRecord(sb, prebidExt.Source, prebidExt.Version)
		}
	}
	return sb.String()
}

func writeNameVersionRecord(sb *strings.Builder, name, version string) {
	if name == "" {
		return
	}
	if version == "" {
		version = VerUnknown
	}
	if sb.Len() != 0 {
		sb.WriteString(",")
	}
	sb.WriteString(name)
	sb.WriteString("/")
	sb.WriteString(version)
}
