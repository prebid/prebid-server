package sharethrough

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const minChromeVersion = 53
const minSafariVersion = 10

type UtilityInterface interface {
	gdprApplies(*openrtb2.BidRequest) bool
	parseUserInfo(*openrtb2.User) userInfo

	getAdMarkup([]byte, openrtb_ext.ExtImpSharethroughResponse, *StrAdSeverParams) (string, error)
	getBestFormat([]openrtb2.Format) (int64, int64)
	getPlacementSize(openrtb2.Imp, openrtb_ext.ExtImpSharethrough) (int64, int64)

	canAutoPlayVideo(string, UserAgentParsers) bool
	isAndroid(string) bool
	isiOS(string) bool
	isAtMinChromeVersion(string, *regexp.Regexp) bool
	isAtMinSafariVersion(string, *regexp.Regexp) bool

	parseDomain(string) string
	getClock() ClockInterface
}

type ClockInterface interface {
	now() time.Time
}

type Clock struct{}

type Util struct {
	Clock ClockInterface
}

type userExt struct {
	Consent string                   `json:"consent,omitempty"`
	Eids    []openrtb_ext.ExtUserEid `json:"eids,omitempty"`
}

type userInfo struct {
	Consent string
	TtdUid  string
	StxUid  string
}

func (u Util) getAdMarkup(strRawResp []byte, strResp openrtb_ext.ExtImpSharethroughResponse, params *StrAdSeverParams) (string, error) {
	landingTime := u.Clock.now()
	strRespId := fmt.Sprintf("str_response_%s", strResp.BidID)

	tmplBody := `
		<img src="//b.sharethrough.com/butler?type=s2s-win&arid={{.Arid}}&adReceivedAt={{.LandingTime}}" />

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

	b64EncodedJson := base64.StdEncoding.EncodeToString(strRawResp)
	err = tmpl.Execute(templatedBuf, struct {
		Arid           template.JS
		Pkey           string
		StrRespId      template.JS
		B64EncodedJson string
		LandingTime    string
	}{
		template.JS(strResp.AdServerRequestID),
		params.Pkey,
		template.JS(strRespId),
		b64EncodedJson,
		landingTime.Format(time.RFC3339Nano),
	})
	if err != nil {
		return "", err
	}

	return templatedBuf.String(), nil
}

func (u Util) getPlacementSize(imp openrtb2.Imp, strImpParams openrtb_ext.ExtImpSharethrough) (height, width int64) {
	height, width = 1, 1
	if len(strImpParams.IframeSize) >= 2 {
		height, width = int64(strImpParams.IframeSize[0]), int64(strImpParams.IframeSize[1])
	} else if imp.Banner != nil {
		height, width = u.getBestFormat(imp.Banner.Format)
	}

	return
}

func (u Util) getBestFormat(formats []openrtb2.Format) (height, width int64) {
	height, width = 1, 1
	for i := 0; i < len(formats); i++ {
		format := formats[i]
		if (format.H * format.W) > (height * width) {
			height = format.H
			width = format.W
		}
	}

	return
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

func (u Util) gdprApplies(request *openrtb2.BidRequest) bool {
	var gdprApplies int64

	if request.Regs != nil {
		if jsonExtRegs, err := request.Regs.Ext.MarshalJSON(); err == nil {
			// 0 is the return value if error, so no need to handle
			gdprApplies, _ = jsonparser.GetInt(jsonExtRegs, "gdpr")
		}
	}

	return gdprApplies != 0
}

func (u Util) parseUserInfo(user *openrtb2.User) (ui userInfo) {
	if user == nil {
		return
	}

	ui.StxUid = user.BuyerUID

	var userExt userExt
	if user.Ext != nil {
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

func (u Util) getClock() ClockInterface {
	return u.Clock
}

func (c Clock) now() time.Time {
	return time.Now().UTC()
}
