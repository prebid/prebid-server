package smaato

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/prebid-server/errortypes"
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

func extractAdmRichMedia(adMarkup string) (string, error) {
	var richMediaAd richMediaAd
	if err := json.Unmarshal([]byte(adMarkup), &richMediaAd); err != nil {
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Invalid ad markup %s.", adMarkup),
		}
	}

	var clickEvent strings.Builder
	var impressionTracker strings.Builder

	for _, clicktracker := range richMediaAd.RichMedia.Clicktrackers {
		clickEvent.WriteString("fetch(decodeURIComponent('" + url.QueryEscape(clicktracker) + "'), " +
			"{cache: 'no-cache'});")
	}
	for _, impression := range richMediaAd.RichMedia.Impressiontrackers {
		impressionTracker.WriteString(fmt.Sprintf(`<img src="%s" alt="" width="0" height="0"/>`, impression))
	}

	richmediaAdMarkup := fmt.Sprintf(`<div onclick="%s">%s%s</div>`,
		&clickEvent, richMediaAd.RichMedia.MediaData.Content, &impressionTracker)

	return richmediaAdMarkup, nil
}
