package flipp

import "github.com/prebid/prebid-server/v3/openrtb_ext"

type CampaignRequestBodyUser struct {
	Key *string `json:"key"`
}

type Properties struct {
	ContentCode *string `json:"contentCode,omitempty"`
}

type PrebidRequest struct {
	CreativeType            *string `json:"creativeType"`
	Height                  *int64  `json:"height"`
	PublisherNameIdentifier *string `json:"publisherNameIdentifier"`
	RequestID               *string `json:"requestId"`
	Width                   *int64  `json:"width"`
}

type Placement struct {
	AdTypes    []int64                        `json:"adTypes"`
	Count      *int64                         `json:"count"`
	DivName    string                         `json:"divName,omitempty"`
	NetworkID  int64                          `json:"networkId,omitempty"`
	Prebid     *PrebidRequest                 `json:"prebid,omitempty"`
	Properties *Properties                    `json:"properties,omitempty"`
	SiteID     *int64                         `json:"siteId"`
	ZoneIds    []int64                        `json:"zoneIds"`
	Options    openrtb_ext.ImpExtFlippOptions `json:"options,omitempty"`
}

type CampaignRequestBody struct {
	IP                string                   `json:"ip,omitempty"`
	Keywords          []string                 `json:"keywords"`
	Placements        []*Placement             `json:"placements"`
	PreferredLanguage *string                  `json:"preferred_language,omitempty"`
	URL               string                   `json:"url,omitempty"`
	User              *CampaignRequestBodyUser `json:"user"`
}

type CampaignResponseBody struct {
	CandidateRetrieval interface{} `json:"candidateRetrieval,omitempty"`
	Decisions          *Decisions  `json:"decisions"`
}

type Decisions struct {
	Inline Inline `json:"inline,omitempty"`
}

type Inline []*InlineModel

type Contents []*Content

type Content struct {
	Body           string `json:"body,omitempty"`
	CustomTemplate string `json:"customTemplate,omitempty"`
	Data           *Data2 `json:"data,omitempty"`
	Type           string `json:"type,omitempty"`
}

type Data2 struct {
	CustomData interface{} `json:"customData,omitempty"`
	Height     int64       `json:"height,omitempty"`
	Width      int64       `json:"width,omitempty"`
}

type InlineModel struct {
	AdID          int64           `json:"adId,omitempty"`
	AdvertiserID  int64           `json:"advertiserId,omitempty"`
	CampaignID    int64           `json:"campaignId,omitempty"`
	ClickURL      string          `json:"clickUrl,omitempty"`
	Contents      Contents        `json:"contents,omitempty"`
	CreativeID    int64           `json:"creativeId,omitempty"`
	FlightID      int64           `json:"flightId,omitempty"`
	Height        int64           `json:"height,omitempty"`
	ImpressionURL string          `json:"impressionUrl,omitempty"`
	Prebid        *PrebidResponse `json:"prebid,omitempty"`
	PriorityID    int64           `json:"priorityId,omitempty"`
	Width         int64           `json:"width,omitempty"`
}

type PrebidResponse struct {
	Cpm          *float64 `json:"cpm"`
	Creative     *string  `json:"creative"`
	CreativeType *string  `json:"creativeType"`
	RequestID    *string  `json:"requestId"`
}
