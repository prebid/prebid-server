package smaato

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/prebid-server/errortypes"
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

func extractAdmImage(adMarkup string) (string, error) {
	var imageAd imageAd
	if err := json.Unmarshal([]byte(adMarkup), &imageAd); err != nil {
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Invalid ad markup %s.", adMarkup),
		}
	}

	var clickEvent strings.Builder
	for _, clicktracker := range imageAd.Image.Clicktrackers {
		clickEvent.WriteString("fetch(decodeURIComponent('" + url.QueryEscape(clicktracker) + "'.replace(/\\+/g, ' ')), " +
			"{cache: 'no-cache'});")
	}

	var impressionTracker strings.Builder
	for _, impression := range imageAd.Image.Impressiontrackers {
		impressionTracker.WriteString(fmt.Sprintf(`<img src="%s" alt="" width="0" height="0"/>`, impression))
	}

	imageAdMarkup := fmt.Sprintf(`<div style="cursor:pointer" onclick="%s;window.open(decodeURIComponent('%s'.replace(/\+/g, ' ')));"><img src="%s" width="%d" height="%d"/>%s</div>`,
		&clickEvent, url.QueryEscape(imageAd.Image.Img.Ctaurl), imageAd.Image.Img.URL, imageAd.Image.Img.W, imageAd.Image.Img.H, &impressionTracker)

	return imageAdMarkup, nil
}
