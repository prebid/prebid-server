package sharethrough

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"html/template"
	"regexp"
	"strconv"
)

const minChromeVersion = 53
const minSafariVersion = 10

type UtilityInterface interface {
	gdprApplies(*openrtb.BidRequest) bool
	gdprConsentString(*openrtb.BidRequest) string

	getAdMarkup(openrtb_ext.ExtImpSharethroughResponse, *StrAdSeverParams) (string, error)
	getPlacementSize([]openrtb.Format) (uint64, uint64)

	canAutoPlayVideo(string, UserAgentParsers) bool
	isAndroid(string) bool
	isiOS(string) bool
	isAtMinChromeVersion(string, *regexp.Regexp) bool
	isAtMinSafariVersion(string, *regexp.Regexp) bool
}

type Util struct{}

func (u Util) getAdMarkup(strResp openrtb_ext.ExtImpSharethroughResponse, params *StrAdSeverParams) (string, error) {
	strRespId := fmt.Sprintf("str_response_%s", strResp.BidID)
	jsonPayload, err := json.Marshal(strResp)
	if err != nil {
		return "", err
	}

	tmplBody := `
		<img src="//b.sharethrough.com/butler?type=s2s-win&arid={{.Arid}}" />

		<div data-str-native-key="{{.Pkey}}" data-stx-response-name="{{.StrRespId}}"></div>
	 	<script>var {{.StrRespId}} = "{{.B64EncodedJson}}"</script>
	`

	if params.Iframe {
		tmplBody = tmplBody + `
			<script src="//native.sharethrough.com/assets/sfp.js"></script>
		`
	} else {
		tmplBody = tmplBody + `
			<script src="//native.sharethrough.com/assets/sfp-set-targeting.js"></script>
	    	<script>
		     (function() {
		       if (!(window.STR && window.STR.Tag) && !(window.top.STR && window.top.STR.Tag)) {
		         var sfp_js = document.createElement('script');
		         sfp_js.src = "//native.sharethrough.com/assets/sfp.js";
		         sfp_js.type = 'text/javascript';
		         sfp_js.charset = 'utf-8';
		         try {
		             window.top.document.getElementsByTagName('body')[0].appendChild(sfp_js);
		         } catch (e) {
		           console.log(e);
		         }
		       }
		     })()
		   </script>
		`
	}

	tmpl, err := template.New("sfpjs").Parse(tmplBody)
	if err != nil {
		return "", err
	}

	var buf []byte
	templatedBuf := bytes.NewBuffer(buf)

	b64EncodedJson := base64.StdEncoding.EncodeToString(jsonPayload)
	err = tmpl.Execute(templatedBuf, struct {
		Arid           template.JS
		Pkey           string
		StrRespId      template.JS
		B64EncodedJson string
	}{
		template.JS(strResp.AdServerRequestID),
		params.Pkey,
		template.JS(strRespId),
		b64EncodedJson,
	})
	if err != nil {
		return "", err
	}

	return templatedBuf.String(), nil
}

func (u Util) getPlacementSize(formats []openrtb.Format) (height uint64, width uint64) {
	biggest := struct {
		Height uint64
		Width  uint64
	}{
		Height: 1,
		Width:  1,
	}

	for i := 0; i < len(formats); i++ {
		format := formats[i]
		if (format.H * format.W) > (biggest.Height * biggest.Width) {
			biggest.Height = format.H
			biggest.Width = format.W
		}
	}

	return biggest.Height, biggest.Width
}

func (u Util) canAutoPlayVideo(userAgent string, parsers UserAgentParsers) bool {
	if u.isAndroid(userAgent) {
		if u.isAtMinChromeVersion(userAgent, parsers.ChromeVersion) {
			return true
		} else {
			return false
		}
	} else if u.isiOS(userAgent) {
		if u.isAtMinSafariVersion(userAgent, parsers.SafariVersion) || u.isAtMinChromeVersion(userAgent, parsers.ChromeiOSVersion) {
			return true
		} else {
			return false
		}
	} else {
		return true
	}
}

func (u Util) isAndroid(userAgent string) bool {
	isAndroid, err := regexp.MatchString("(?i)Android", userAgent)
	if err != nil {
		return false
	}
	return isAndroid
}

func (u Util) isiOS(userAgent string) bool {
	isiOS, err := regexp.MatchString("(?i)iPhone|iPad|iPod", userAgent)
	if err != nil {
		return false
	}
	return isiOS
}

func (u Util) isAtMinVersion(userAgent string, parser *regexp.Regexp, minVersion int64) bool {
	var version int64
	var err error

	chromeVersionMatch := parser.FindStringSubmatch(userAgent)
	if len(chromeVersionMatch) > 1 {
		version, err = strconv.ParseInt(chromeVersionMatch[1], 10, 64)
	}
	if err != nil {
		return false
	}

	return version >= minVersion
}

func (u Util) isAtMinChromeVersion(userAgent string, parser *regexp.Regexp) bool {
	return u.isAtMinVersion(userAgent, parser, minChromeVersion)
}

func (u Util) isAtMinSafariVersion(userAgent string, parser *regexp.Regexp) bool {
	return u.isAtMinVersion(userAgent, parser, minSafariVersion)
}

func (u Util) gdprApplies(request *openrtb.BidRequest) bool {
	var gdprApplies int64

	if request.Regs != nil {
		if jsonExtRegs, err := request.Regs.Ext.MarshalJSON(); err == nil {
			// 0 is the return value if error, so no need to handle
			gdprApplies, _ = jsonparser.GetInt(jsonExtRegs, "gdpr")
		}
	}

	return gdprApplies != 0
}

func (u Util) gdprConsentString(request *openrtb.BidRequest) string {
	var consentString string

	if request.User != nil {
		if jsonExtUser, err := request.User.Ext.MarshalJSON(); err == nil {
			// empty string is the return value if error, so no need to handle
			consentString, _ = jsonparser.GetString(jsonExtUser, "consent")
		}
	}

	return consentString
}
