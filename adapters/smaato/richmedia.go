package smaato

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type richMediaAd struct {
	RichMedia richmedia `json:"richmedia"`
}
type mediadata struct {
	Content string `json:"content"`
	W       int    `json:"w"`
	H       int    `json:"h"`
}

type richmedia struct {
	MediaData          mediadata `json:"mediadata"`
	Impressiontrackers []string  `json:"impressiontrackers"`
	Clicktrackers      []string  `json:"clicktrackers"`
}

func extractAdmRichMedia(adapterResponseAdm string) (string, error) {
	var richMediaMarkup string
	var err error

	var richMediaAd richMediaAd
	err = json.Unmarshal([]byte(adapterResponseAdm), &richMediaAd)
	var richMedia = richMediaAd.RichMedia

	if err == nil {
		var clickEvent strings.Builder
		var impressionTracker strings.Builder

		for _, clicktracker := range richMedia.Clicktrackers {
			clickEvent.WriteString("fetch(decodeURIComponent('" + url.QueryEscape(clicktracker) + "'), " +
				"{cache: 'no-cache'});")
		}
		for _, impression := range richMedia.Impressiontrackers {

			impressionTracker.WriteString(fmt.Sprintf(`<img src="%s" alt="" width="0" height="0"/>`, impression))
		}

		richMediaMarkup = fmt.Sprintf(`<div onclick="%s">%s%s</div>`,
			&clickEvent,
			richMedia.MediaData.Content,
			&impressionTracker)
	}
	return richMediaMarkup, err
}
