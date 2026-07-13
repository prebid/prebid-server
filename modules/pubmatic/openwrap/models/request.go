package models

import (
	"encoding/json"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

type ExtRegs struct {
	// GDPR should be "1" if the caller believes the user is subject to GDPR laws, "0" if not, and undefined
	// if it's unknown. For more info on this parameter, see: https://iabtechlab.com/wp-content/uploads/2018/02/OpenRTB_Advisory_GDPR_2018-02.pdf
	Gdpr *int `json:"gdpr,omitempty"`
	// USPrivacy should be a four character string, see: https://iabtechlab.com/wp-content/uploads/2019/11/OpenRTB-Extension-U.S.-Privacy-IAB-Tech-Lab.pdf
	USPrivacy *string `json:"us_privacy,omitempty"`
}

// ExtRequestAdPod holds AdPod specific extension parameters at request level
type ExtRequestAdPod struct {
	AdPod
	CrossPodAdvertiserExclusionPercent  int `json:"crosspodexcladv,omitempty"`    //Percent Value - Across multiple impression there will be no ads from same advertiser. Note: These cross pod rule % values can not be more restrictive than per pod
	CrossPodIABCategoryExclusionPercent int `json:"crosspodexcliabcat,omitempty"` //Percent Value - Across multiple impression there will be no ads from same advertiser
	IABCategoryExclusionWindow          int `json:"excliabcatwindow,omitempty"`   //Duration in minute between pods where exclusive IAB rule needs to be applied
	AdvertiserExclusionWindow           int `json:"excladvwindow,omitempty"`      //Duration in minute between pods where exclusive advertiser rule needs to be applied
}

// AdPod holds Video AdPod specific extension parameters at impression level
type AdPod struct {
	MinAds                      int  `json:"minads,omitempty"`        //Default 1 if not specified
	MaxAds                      int  `json:"maxads,omitempty"`        //Default 1 if not specified
	MinDuration                 int  `json:"adminduration,omitempty"` // (adpod.adminduration * adpod.minads) should be greater than or equal to video.minduration
	MaxDuration                 int  `json:"admaxduration,omitempty"` // (adpod.admaxduration * adpod.maxads) should be less than or equal to video.maxduration + video.maxextended
	AdvertiserExclusionPercent  *int `json:"excladv,omitempty"`       // Percent value 0 means none of the ads can be from same advertiser 100 means can have all same advertisers
	IABCategoryExclusionPercent *int `json:"excliabcat,omitempty"`    // Percent value 0 means all ads should be of different IAB categories.
}

// ImpExtension - Impression Extension
type ImpExtension struct {
	openrtb_ext.GoogleSDKParams
	Wrapper     *ExtImpWrapper              `json:"wrapper,omitempty"`
	Reward      *int8                       `json:"reward,omitempty"`
	Bidder      map[string]*BidderExtension `json:"bidder,omitempty"`
	SKAdnetwork json.RawMessage             `json:"skadn,omitempty"`
	Data        openrtb_ext.ExtImpData      `json:"data,omitempty"`
	GpId        string                      `json:"gpid,omitempty"`
	Prebid      openrtb_ext.ExtImpPrebid    `json:"prebid,omitempty"`
	OWSDK       map[string]any              `json:"owsdk,omitempty"`
}

// BidderExtension - Bidder specific items
type BidderExtension struct {
	KeyWords []KeyVal              `json:"keywords,omitempty"`
	DealTier *openrtb_ext.DealTier `json:"dealtier,omitempty"`
}

// ExtImpWrapper - Impression wrapper Extension
type ExtImpWrapper struct {
	AdServerURL string `json:"adserverurl,omitempty"`
	Div         string `json:"div,omitempty"`
}

// ExtVideo structure to accept video specific more parameters like adpod
type ExtVideo struct {
	Offset int    `json:"offset,omitempty"` // Minutes from start where this ad is intended to show
	AdPod  *AdPod `json:"adpod,omitempty"`
}

// RequestExt Request Extension
type RequestExt struct {
	openrtb_ext.ExtRequest

	// Move this to bidder params
	Wrapper *RequestExtWrapper                `json:"wrapper,omitempty"`
	Bidder  map[string]map[string]interface{} `json:"bidder,omitempty"`
	AdPod   *ExtRequestAdPod                  `json:"adpod,omitempty"`
}

// pbopenrtb_ext alias for prebid server openrtb_ext
// type PriceFloorRules = openrtb_ext.PriceFloorRules

// TransparencyRule contains transperancy rule for a single bidder
type TransparencyRule struct {
	Include bool     `json:"include,omitempty"`
	Keys    []string `json:"keys,omitempty"`
}

// ExtTransparency holds bidder level content transparency rules
type ExtTransparency struct {
	Content map[string]TransparencyRule `json:"content,omitempty"`
}

// KeyVal structure to store bidder related custom key-values
type KeyVal struct {
	Key    string   `json:"key,omitempty"`
	Values []string `json:"value,omitempty"`
}

// RequestExtWrapper holds wrapper specific extension parameters
type RequestExtWrapper struct {
	ProfileId             int                    `json:"profileid,omitempty"`
	VersionId             int                    `json:"versionid,omitempty"`
	SSAuctionFlag         int                    `json:"ssauction,omitempty"`
	SumryDisableFlag      int                    `json:"sumry_disable,omitempty"`
	ClientConfigFlag      int                    `json:"clientconfig,omitempty"`
	SupportDeals          bool                   `json:"supportdeals,omitempty"`
	IncludeBrandCategory  *int                   `json:"includebrandcategory,omitempty"`
	ABTestConfig          int                    `json:"abtest,omitempty"`
	LoggerImpressionID    string                 `json:"wiid,omitempty"`
	SSAI                  string                 `json:"ssai,omitempty"`
	KeyValues             map[string]interface{} `json:"kv,omitempty"`
	Video                 ExtRequestWrapperVideo `json:"video,omitempty"`
	PubId                 int                    `json:"-"`
	SdkSubIntegrationPath *int                   `json:"sdksubintegration,omitempty"`
}

type ExtRequestWrapperVideo struct {
	AdruleFlag bool `json:"adrule,omitempty"`
}

type BidderWrapper struct {
	Flag        bool
	VASTagFlags map[string]bool
}
