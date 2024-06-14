package smaato

import (
	"fmt"
	"net/url"
	"strings"
)

func extractAdmBanner(adMarkup string, curls []string) string {
	if len(curls) > 0 {
		var clickEvent string
		var clicks strings.Builder
		for _, clicktracker := range curls {
			clicks.WriteString("fetch(decodeURIComponent('" + url.QueryEscape(clicktracker) + "'.replace(/\\+/g, ' ')), " +
				"{cache: 'no-cache'});")
		}
		clickEvent = fmt.Sprintf(`onclick="%s"`, clicks.String())
		return fmt.Sprintf(`<div style="cursor:pointer" %s>%s</div>`, clickEvent, adMarkup)
	}

	return adMarkup
}
