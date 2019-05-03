package sharethrough

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"html/template"
	"regexp"
	"strconv"
)

func getAdMarkup(strResp openrtb_ext.ExtImpSharethroughResponse, params *hbUriParams) string {
	var errs []error

	strRespId := fmt.Sprintf("str_response_%s", strResp.BidID)
	jsonPayload, err := json.Marshal(strResp)
	if err != nil {
		return ""
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
	errs = append(errs, err)

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

	errs = append(errs, err)

	return templatedBuf.String()
}

func getPlacementSize(formats []openrtb.Format) (height uint64, width uint64) {
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

func canAutoPlayVideo(userAgent string) bool {
	const minChromeVersion = 53
	const minSafariVersion = 10

	isAndroid, _ := regexp.MatchString("(?i)Android", userAgent)
	isiOS, _ := regexp.MatchString("(?i)iPhone|iPad|iPod", userAgent)

	var chromeVersion, chromeiOSVersion, safariVersion int64

	chromeVersionRegex := regexp.MustCompile(`Chrome\/(?P<ChromeVersion>\d+)`)
	chromeVersionMatch := chromeVersionRegex.FindStringSubmatch(userAgent)
	if len(chromeVersionMatch) > 1 {
		chromeVersion, _ = strconv.ParseInt(chromeVersionMatch[1], 10, 64)
	}

	chromeiOSVersionRegex := regexp.MustCompile(`CriOS\/(?P<chromeiOSVersion>\d+)`)
	chromeiOSVersionMatch := chromeiOSVersionRegex.FindStringSubmatch(userAgent)
	if len(chromeiOSVersionMatch) > 1 {
		chromeiOSVersion, _ = strconv.ParseInt(chromeiOSVersionMatch[1], 10, 64)
	}

	safariVersionRegex := regexp.MustCompile(`Version\/(?P<safariVersion>\d+)`)
	safariVersionMatch := safariVersionRegex.FindStringSubmatch(userAgent)
	if len(safariVersionMatch) > 1 {
		safariVersion, _ = strconv.ParseInt(safariVersionMatch[1], 10, 64)
	}

	return (isAndroid && chromeVersion >= minChromeVersion) || (isiOS && (safariVersion >= minSafariVersion || chromeiOSVersion >= minChromeVersion)) || !(isAndroid || isiOS)
}
