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
	"net"
	"net/url"
	"regexp"
	"strconv"
)

const minChromeVersion = 53
const minSafariVersion = 10

type UtilityInterface interface {
	gdprApplies(*openrtb.BidRequest) bool
	parseUserExt(*openrtb.User) userInfo

	getAdMarkup(openrtb_ext.ExtImpSharethroughResponse, *StrAdSeverParams) (string, error)
	getPlacementSize([]openrtb.Format) (uint64, uint64)

	canAutoPlayVideo(string, UserAgentParsers) bool
	isAndroid(string) bool
	isiOS(string) bool
	isAtMinChromeVersion(string, *regexp.Regexp) bool
	isAtMinSafariVersion(string, *regexp.Regexp) bool

	parseDomain(string) string
}

type Util struct{}

type userExt struct {
	Consent string                   `json:"consent,omitempty"`
	Eids    []openrtb_ext.ExtUserEid `json:"eids,omitempty"`
}

type userInfo struct {
	Consent string
	TtdUid  string
}

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
		return u.isAtMinChromeVersion(userAgent, parsers.ChromeVersion)
	} else if u.isiOS(userAgent) {
		return u.isAtMinSafariVersion(userAgent, parsers.SafariVersion) || u.isAtMinChromeVersion(userAgent, parsers.ChromeiOSVersion)
	}
	return true
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

func (u Util) isAtMinVersion(userAgent string, versionParser *regexp.Regexp, minVersion int64) bool {
	var version int64
	var err error

	versionMatch := versionParser.FindStringSubmatch(userAgent)
	if len(versionMatch) > 1 {
		version, err = strconv.ParseInt(versionMatch[1], 10, 64)
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

func (u Util) parseUserExt(user *openrtb.User) (ui userInfo) {
	var userExt userExt
	if user != nil && user.Ext != nil {
		if err := json.Unmarshal(user.Ext, &userExt); err == nil {
			ui.Consent = userExt.Consent
			for i := 0; i < len(userExt.Eids); i++ {
				if userExt.Eids[i].Source == "adserver.org" && len(userExt.Eids[i].Uids) > 0 {
					if userExt.Eids[i].Uids[0].ID != "" {
						ui.TtdUid = userExt.Eids[i].Uids[0].ID
					}
					break
				}
			}
		}
	}

	return
}

func (u Util) parseDomain(fullUrl string) string {
	domain := ""
	uri, err := url.Parse(fullUrl)
	if err == nil {
		host, _, errSplit := net.SplitHostPort(uri.Host)
		if errSplit == nil {
			domain = host
		} else {
			domain = uri.Host
		}

		if domain != "" {
			domain = uri.Scheme + "://" + domain
		}
	}

	return domain
}
