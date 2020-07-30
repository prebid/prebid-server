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

func extractAdmImage(adapterResponseAdm string) (string, error) {
	var imgMarkup string
	var err error

	var imageAd imageAd
	err = json.Unmarshal([]byte(adapterResponseAdm), &imageAd)
	var image = imageAd.Image

	if err == nil {
		var clickEvent strings.Builder
		var impressionTracker strings.Builder

		for _, clicktracker := range image.Clicktrackers {
			clickEvent.WriteString("fetch(decodeURIComponent('" + url.QueryEscape(clicktracker) + "'.replace(/\\+/g, ' ')), " +
				"{cache: 'no-cache'});")
		}

		for _, impression := range image.Impressiontrackers {

			impressionTracker.WriteString(fmt.Sprintf(`<img src="%s" alt="" width="0" height="0"/>`, impression))
		}

		imgMarkup = fmt.Sprintf(`<div style="cursor:pointer" onclick="%s;window.open(decodeURIComponent('%s'.replace(/\+/g, ' ')));"><img src="%s" width="%d" height="%d"/>%s</div>`,
			&clickEvent, url.QueryEscape(image.Img.Ctaurl), image.
				Img.URL, image.Img.W, image.Img.
				H, &impressionTracker)
	}
	return imgMarkup, err
}
