package smaato

import (
	"encoding/json"
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

func extractAdmImage(adType string, adapterResponseAdm string) (string, error) {
	var imgMarkup string
	var err error
	if strings.EqualFold(adType, "img") {
		var imageAd imageAd
		err := json.Unmarshal([]byte(adapterResponseAdm), &imageAd)
		var image = imageAd.Image

		if err == nil {
			var clickEvent string
			for _, clicktracker := range image.Clicktrackers {
				clickEvent += "fetch(decodeURIComponent('" + url.QueryEscape(clicktracker) + "'), " +
					"{cache: 'no-cache'});"
			}
			imgMarkup = fmt.Sprintf(`<div onclick="%s"><a href="%s"><img src="%s" width="%d" height="%d"/></a></div>`, clickEvent, image.Img.Ctaurl, image.
				Img.URL, image.Img.W, image.Img.
				H)
		}
	}
	return imgMarkup, err
}
