package openrtb_ext

import "github.com/mxmCherry/openrtb/v15/openrtb2"

type BidRequestVideo struct {
	// Attribute:
	//   storedrequestid
	// Type:
	//   string; required
	// Description:
	//   Unique ID of the stored request
	StoredRequestId string `json:"storedrequestid"`

	// Attribute:
	//   podconfig
	// Type:
	//   object; required
	// Description:
	//   Container object for describing all the pod configurations
	PodConfig PodConfig `json:"podconfig"`

	// Attribute:
	//   app
	// Type:
	//   object; App or Site required
	// Description:
	//   Application where the impression will be shown
	App *openrtb2.App `json:"app"`

	// Attribute:
	//   site
	// Type:
	//   object; App or Site required
	// Description:
	//   Site where the impression will be shown
	Site *openrtb2.Site `json:"site"`

	// Attribute:
	//   user
	// Type:
	//   object; optional
	// Description:
	//   Container object for the user of of the actual device
	User *openrtb2.User `json:"user,omitempty"`

	// Attribute:
	//   device
	// Type:
	//   object; optional
	// Description:
	//   Device specific data
	Device openrtb2.Device `json:"device,omitempty"`

	// Attribute:
	//   includebrandcategory
	// Type:
	//   object; optional
	// Description:
	//   Indicates that the response requires an adserver specific content category
	IncludeBrandCategory *IncludeBrandCategory `json:"includebrandcategory,omitempty"`

	// Attribute:
	//   video
	// Type:
	//   object; required
	// Description:
	//   Player container object
	Video *openrtb2.Video `json:"video,omitempty"`

	// Attribute:
	//   content
	// Type:
	//   object; optional
	// Description:
	//  Misc content meta data that can be used for targeting the adPod(s)
	Content openrtb2.Content `json:"content,omitempty"`

	// Attribute:
	//   cacheconfig
	// Type:
	//   object; optional
	// Description:
	//  Container object for all Prebid Cache configs
	Cacheconfig Cacheconfig `json:"cacheconfig,omitempty"`

	// Attribute:
	//   test
	// Type:
	//   integer; default 0
	// Description:
	//    Indicator of test mode in which auctions are not billable,
	//    where 0 = live mode, 1 = test mode.
	Test int8 `json:"test,omitempty"`

	// Attribute:
	//   pricegranularity
	// Type:
	//   object; optional
	// Description:
	//    Object to tell ad server how much money the “bidder” demand is worth to you
	PriceGranularity PriceGranularity `json:"pricegranularity,omitempty"`

	// Attribute:
	//   tmax
	// Type:
	//   integer
	// Description:
	//    Maximum time in milliseconds the exchange allows for bids to
	//    be received including Internet latency to avoid timeout. This
	//    value supersedes any a priori guidance from the exchange.
	TMax int64 `json:"tmax,omitempty"`

	// Attribute:
	//   bcat
	// Type:
	//   string array
	// Description:
	//   Blocked advertiser categories using the IAB content
	//   categories. Refer to List 5.1.
	BCat []string `json:"bcat,omitempty"`

	// Attribute:
	//   badv
	// Type:
	//   string array
	// Description:
	//   Block list of advertisers by their domains (e.g., “ford.com”).
	BAdv []string `json:"badv,omitempty"`

	// Attribute:
	//   regs
	// Type:
	//   object; optional
	// Description:
	//   Contains the OpenRTB Regs object to be passed to OpenRTB request
	Regs *openrtb2.Regs `json:"regs,omitempty"`

	// Attribute:
	//   supportdeals
	// Type:
	//   bool; optional
	// Description:
	//   Indicates that the response should update key to include prefix and tier
	SupportDeals bool `json:"supportdeals,omitempty"`

	// Attribute:
	//   appendbiddernames
	// Type:
	//   boolean, optional
	//  Flag indicating if the bidder name will be added to the hb_pb_cat_dur. Default is false.
	AppendBidderNames bool `json:"appendbiddernames,omitempty"`
}

type PodConfig struct {
	// Attribute:
	//   durationrangesec
	// Type:
	//  int array, required
	// Description:
	//  Range of ad durations allowed in the response
	DurationRangeSec []int `json:"durationrangesec"`

	// Attribute:
	//   requireexactduration
	// Type:
	//   boolean, optional
	//  Flag indicating exact ad duration requirement. Default is false.
	RequireExactDuration bool `json:"requireexactduration,omitempty"`

	// Attribute:
	//   pods
	// Type:
	//   object; required
	//  Container object for describing the adPod(s) to be requested.
	Pods []Pod `json:"pods"`
}

type Pod struct {
	// Attribute:
	//   podid
	// Type:
	//   integer; required
	//  Unique id of the pod within a particular request.
	PodId int `json:"podid"`

	// Attribute:
	//   adpoddurationsec
	// Type:
	//   integer; required
	//  Duration of the adPod
	AdPodDurationSec int `json:"adpoddurationsec"`

	// Attribute:
	//   configid
	// Type:
	//   string; required
	//  ID of the stored config that corresponds to a single pod request
	ConfigId string `json:"configid"`
}

type IncludeBrandCategory struct {
	// Attribute:
	//   primaryadserver
	// Type:
	//   int; optional
	//  The ad server used by the publisher. Supported Values 1- Freewheel , 2- DFP
	PrimaryAdserver int `json:"primaryadserver"`

	// Attribute:
	//   publisher
	// Type:
	//   string; optional
	//  Identifier for the Publisher
	Publisher string `json:"publisher"`

	// Attribute:
	//   translatecategories
	// Type:
	//   *bool; optional
	// Description:
	//   Indicates if IAB categories should be translated to adserver category
	TranslateCategories *bool `json:"translatecategories,omitempty"`
}

type Cacheconfig struct {
	// Attribute:
	//   ttl
	// Type:
	//   int; optional
	//  Time to Live for a cache entry specified in seconds
	Ttl int `json:"ttl"`
}
