package openrtb_ext

import (
	"github.com/mxmCherry/openrtb"
)

type BidRequestVideo struct {
	// Attribute:
	//   accountid
	// Type:
	//   string; required
	// Description:
	//   Unique ID of the stored request
	AccountId string `json:"accountid"`

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
	App openrtb.App `json:"app,omitempty"`

	// Attribute:
	//   site
	// Type:
	//   object; App or Site required
	// Description:
	//   Site where the impression will be shown
	Site openrtb.Site `json:"site,omitempty"`

	// Attribute:
	//   user
	// Type:
	//   object; optional
	// Description:
	//   Container object for the user of of the actual device
	User SimplifiedUser `json:"user,omitempty"`

	// Attribute:
	//   device
	// Type:
	//   object; optional
	// Description:
	//   Device specific data
	Device openrtb.Device `json:"device,omitempty"`

	// Attribute:
	//   includebrandcategory
	// Type:
	//   object; optional
	// Description:
	//   Indicates that the response requires an adserver specific content category
	IncludeBrandCategory IncludeBrandCategory `json:"includebrandcategory"`

	// Attribute:
	//   video
	// Type:
	//   object; required
	// Description:
	//   Player container object
	Video SimplifiedVideo `json:"video,omitempty"`

	// Attribute:
	//   content
	// Type:
	//   object; optional
	// Description:
	//  Misc content meta data that can be used for targeting the adPod(s)
	Content openrtb.Content `json:"content,omitempty"`

	// Attribute:
	//   cacheconfig
	// Type:
	//   object; optional
	// Description:
	//  Container object for all Prebid Cache configs
	Cacheconfig Cacheconfig `json:"cacheconfig,omitempty"`
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
}

type Cacheconfig struct {
	// Attribute:
	//   ttl
	// Type:
	//   int; optional
	//  Time to Live for a cache entry specified in seconds
	Ttl int `json:"ttl"`
}

type Gdpr struct {
	// Attribute:
	//   consentrequired
	// Type:
	//   boolean; optional
	//  Indicates whether GDPR is in effect
	ConsentRequired bool `json:"consentrequired"`

	// Attribute:
	//   consentstring
	// Type:
	//   string; optional
	//  Contains the data structure developed by the GDPR
	ConsentString string `json:"consentstring"`
}

type SimplifiedUser struct {
	// Attribute:
	//   buyeruids
	// Type:
	//   map; optional
	//  ID of the stored config that corresponds to a single pod request
	Buyeruids map[string]string `json:"buyeruids"`

	// Attribute:
	//   gdpr
	// Type:
	//   object; optional
	//  Container object for GDPR
	Gdpr Gdpr `json:"gdpr"`

	// Attribute:
	//   yob
	// Type:
	//   int; optional
	//  Year of birth as a 4-digit integer
	Yob int `json:"yob"`

	// Attribute:
	//   gender
	// Type:
	//   string; optional
	//  Gender, where “M” = male, “F” = female, “O” = known to be other
	Gender string `json:"gender"`

	// Attribute:
	//   keywords
	// Type:
	//   string; optional
	//  Comma separated list of keywords, interests, or intent.
	Keywords string `json:"keywords"`
}

type SimplifiedVideo struct {
	// Attribute:
	//   w
	// Type:
	//   uint64; optional
	//  Width of video
	W uint64 `json:"w"`

	// Attribute:
	//   h
	// Type:
	//   uint64; optional
	//  Height of video
	H uint64 `json:"h"`

	// Attribute:
	//   mime
	// Type:
	//   array of strings; optional
	//  Video mime types
	Mime []string `json:"mime"`

	// Attribute:
	//   protocols
	// Type:
	//   array of objects; optional
	//  protocols
	Protocols []openrtb.Protocol `json:"protocols"`
}

type StoredRequestId struct {

	// Attribute:
	//   storedrequestid
	// Type:
	//   string; required
	// Description:
	//   Unique ID of the stored request
	StoredRequestId string `json:"storedrequestid"`
}
