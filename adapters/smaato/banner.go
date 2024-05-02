package smaato

import (
	"fmt"
	"net/url"
	"strings"
)

type imageAd struct {
	Image image `json:"image"`
}
type image struct {
	Img                img      `json:"img"`
	Impressiontrackers []string `json:"impressiontrackers"`
	Clicktrackers      []string `json:"clicktrackers"`
}
type img struct {
	URL    string `json:"url"`
	W      int    `json:"w"`
	H      int    `json:"h"`
	Ctaurl string `json:"ctaurl"`
}

func extractAdmBanner(adMarkup string, curls []string) string {
	var clickEvent strings.Builder
	for _, clicktracker := range curls {
		clickEvent.WriteString("fetch(decodeURIComponent('" + url.QueryEscape(clicktracker) + "'.replace(/\\+/g, ' ')), " +
			"{cache: 'no-cache'});")
	}

	return fmt.Sprintf(`<div style="cursor:pointer" %s>%s</div>`, clickEvent.String(), adMarkup)
}
