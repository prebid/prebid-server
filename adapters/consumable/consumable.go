package consumable

import (
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"net/http"
	"net/url"
)

type ConsumableAdapter struct{}

type bidRequest struct {
	Placements         []placement `json:"placements"`
	Time               int64       `json:"time"`
	IncludePricingData bool        `json:"includePricingData"`
	User               user        `json:"user,omitempty"`
	Referrer           string      `json:"referrer,omitempty"`
	Ip                 string      `json:"ip,omitempty"`
	Url                string      `json:"url,omitempty"`
	EnableBotFiltering bool        `json:"enableBotFiltering,omitempty"`
}

type placement struct {
	DivName   string `json:"divName"`
	NetworkId int    `json:"networkId"`
	SiteId    int    `json:"siteId"`
	AdTypes   []int  `json:"adTypes"`
	ZoneIds   []int  `json:"zoneIds,omitempty"`
}

type user struct {
	Key string `json:"key,omitempty"`
}

func (a *ConsumableAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	headers := http.Header{
		"Content-Type": {"application/json"},
		"Accept":       {"application/json"},
	}

	if request.Device != nil {
		if request.Device.UA != "" {
			headers.Set("User-Agent", request.Device.UA)
		}

		if request.Device.IP != "" {
			headers.Set("Forwarded", "for="+request.Device.IP)
			headers.Set("X-Forwarded-For", request.Device.IP)
		}
	}

	if request.Site != nil && request.Site.Page != "" {
		headers.Set("Referer", request.Site.Page)

		pageUrl, err := url.Parse(request.Site.Page)
		if err == nil {
			origin := url.URL{
				Scheme: pageUrl.Scheme,
				Opaque: pageUrl.Opaque,
				Host:   pageUrl.Host,
			}

			headers.Set("Origin", origin.String())
		}
	}

	requests := []*adapters.RequestData{
		{
			Method:  "POST",
			Uri:     "https://e.serverbid.com/api/v2",
			Body:    nil, // TODO
			Headers: headers,
		},
	}

	return requests, nil
}

func (a *ConsumableAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	return nil, nil
}

func NewConsumableBidder() *ConsumableAdapter {
	return &ConsumableAdapter{}
}
