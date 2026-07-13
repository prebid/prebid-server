package models

import (
	"encoding/json"
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/openrtb/v20/openrtb3"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/metrics"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models/adunitconfig"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models/nbr"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/ortb"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/wakanda"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/usersync"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

type RequestCtx struct {
	// PubID is the publisher id retrieved from request
	PubID int
	// ProfileID is the value received in profileid field in wrapper object.
	ProfileID int
	// DisplayID is the value received in versionid field in wrapper object.
	DisplayID int
	// VersionID is the unique id from DB associated with the incoming DisplayID
	VersionID int
	// DisplayVersionID is the DisplayID of the profile selected by OpenWrap incase DisplayID/versionid is 0
	DisplayVersionID int

	SSAuction          int
	SummaryDisable     int
	SSAI               string
	PartnerConfigMap   map[int]map[string]string
	SupportDeals       bool
	Platform           string
	LoggerImpressionID string
	ClientConfigFlag   int
	TMax               int64

	//NYC_TODO: use enum?
	IsTestRequest                     int8
	ABTestConfig, ABTestConfigApplied int
	IsCTVRequest                      bool

	TrackerEndpoint, VideoErrorTrackerEndpoint string

	Cookies         []string
	UidCookie       *http.Cookie
	KADUSERCookie   *http.Cookie
	ParsedUidCookie *usersync.Cookie
	OriginCookie    string

	Debug bool
	Trace bool

	//tracker
	PageURL   string
	StartTime int64
	DeviceCtx DeviceCtx

	//trackers per bid
	Trackers map[string]OWTracker

	//prebid-biddercode to seat/alias mapping
	PrebidBidderCode map[string]string

	// imp-bid ctx to avoid computing same thing for bidder params, logger and tracker
	ImpBidCtx          map[string]ImpCtx
	Aliases            map[string]string
	NewReqExt          *RequestExt
	ResponseExt        openrtb_ext.ExtBidResponse
	MarketPlaceBidders map[string]struct{}

	AdapterThrottleMap map[string]struct{}
	AdapterFilteredMap map[string]struct{}

	AdUnitConfig *adunitconfig.AdUnitConfig

	Source, Origin string

	SendAllBids bool
	WinningBids WinningBids
	DroppedBids map[string][]openrtb2.Bid
	DefaultBids map[string]map[string][]openrtb2.Bid
	SeatNonBids map[string][]openrtb_ext.NonBid // map of bidder to list of nonbids

	BidderResponseTimeMillis map[string]int

	Endpoint                        string
	PubIDStr, ProfileIDStr          string // TODO: remove this once we completely move away from header-bidding
	MetricsEngine                   metrics.MetricsEngine
	HostName                        string
	ReturnAllBidStatus              bool   // ReturnAllBidStatus stores the value of request.ext.prebid.returnallbidstatus
	Sshb                            string //Sshb query param to identify that the request executed heder-bidding or not, sshb=1(executed HB(8001)), sshb=2(reverse proxy set from HB(8001->8000)), sshb=""(direct request(8000)).
	DCName                          string
	CachePutMiss                    int // to be used in case of CTV JSON endpoint/amp/inapp-ott-video endpoint
	CurrencyConversion              func(from string, to string, value float64) (float64, error)
	MatchedImpression               map[string]int
	CustomDimensions                map[string]CustomDimension
	AmpVideoEnabled                 bool //AmpVideoEnabled indicates whether to include a Video object in an AMP request.
	IsTBFFeatureEnabled             bool
	AppLovinMax                     AppLovinMax
	LoggerDisabled                  bool
	TrackerDisabled                 bool
	ProfileType                     int
	ProfileTypePlatform             int
	AppPlatform                     int
	AppIntegrationPath              *int
	AppSubIntegrationPath           *int
	Method                          string
	Errors                          []error
	RedirectURL                     string
	ResponseFormat                  string
	WakandaDebug                    wakanda.WakandaDebug
	PriceGranularity                *openrtb_ext.PriceGranularity
	IsMaxFloorsEnabled              bool
	SendBurl                        bool
	ImpCountingMethodEnabledBidders map[string]struct{}     // Bidders who have enabled ImpCountingMethod feature
	MultiFloors                     map[string]*MultiFloors // impression level floors
	GoogleSDK                       GoogleSDK
	AppStoreUrl                     string
	UnityLevelPlay                  UnityLevelPlay
	APS                             APS
	VastUnWrap                      VastUnWrap
	PerformanceDSPs                 map[int]struct{}
	InViewEnabledPublishers         map[int]struct{}

	// Adpod
	AdruleFlag         bool
	AdpodProfileConfig *AdpodProfileConfig
	ImpAdPodConfig     map[string][]PodConfig
}

type VastUnWrap struct {
	IsPrivacyEnforced bool
	Enabled           bool
	StatsEnabled      bool
}

type GoogleSDK struct {
	StartTime           time.Time
	Reject              bool
	SDKRenderedAdID     string
	RejectedBidResponse *openrtb2.BidResponse
}

type UnityLevelPlay struct {
	Reject bool
}
type APS struct {
	Reject bool
}

type AdpodProfileConfig struct {
	AdserverCreativeDurations              []int  `json:"videoadduration,omitempty"`         //Range of ad durations allowed in the response
	AdserverCreativeDurationMatchingPolicy string `json:"videoaddurationmatching,omitempty"` //Flag indicating exact ad duration requirement. (default)empty/exact/round.
}

type OwBid struct {
	ID                   string
	NetEcpm              float64
	BidDealTierSatisfied bool
	Nbr                  *openrtb3.NoBidReason
}

func (r RequestCtx) GetVersionLevelKey(key string) string {
	if len(r.PartnerConfigMap) == 0 || len(r.PartnerConfigMap[VersionLevelConfigID]) == 0 {
		return ""
	}
	v := r.PartnerConfigMap[VersionLevelConfigID][key]
	return v
}

// DeviceCtx to cache device specific parameters
type DeviceCtx struct {
	DeviceIFA          string
	IFATypeID          *DeviceIFAType
	Platform           DevicePlatform
	Ext                *ExtDevice
	ID                 string
	Model              string
	UA                 string
	Country            string
	IP                 string
	DerivedCountryCode string
}

type ImpCtx struct {
	ImpID               string
	TagID               string
	DisplayManager      string
	DisplayManagerVer   string
	Div                 string
	SlotName            string
	AdUnitName          string
	Secure              int
	BidFloor            float64
	BidFloorCur         string
	IsRewardInventory   *int8
	Instl               int8
	IsCTAOverlayRequest bool
	IsAppOpenAd         int8
	IsBanner            bool
	Banner              *openrtb2.Banner
	Video               *openrtb2.Video
	Native              *openrtb2.Native
	IncomingSlots       []string
	Type                string // banner, video, native, etc
	Bidders             map[string]PartnerData
	NonMapped           map[string]struct{}
	NewExt              json.RawMessage
	BidCtx              map[string]BidCtx
	BannerAdUnitCtx     AdUnitCtx
	VideoAdUnitCtx      AdUnitCtx
	NativeAdUnitCtx     AdUnitCtx
	//temp
	BidderError string

	// Adpod
	IsAdPodRequest bool
	AdpodConfig    *AdPod
	ImpAdPodCfg    []*ImpAdPodConfig
	BidIDToAPRC    map[string]int64
	AdserverURL    string
	BidIDToDur     map[string]int64
}

type PartnerData struct {
	PartnerID        int
	PrebidBidderCode string
	MatchedSlot      string
	KGP              string
	KGPV             string
	IsRegex          bool
	Params           json.RawMessage
	VASTTagFlags     map[string]bool
}

type BidCtx struct {
	BidExt

	// EG gross net in USD for tracker and logger
	EG float64
	// EN gross net in USD for tracker and logger
	EN float64
	// OmitBidExpFromTracker when true, impression tracker omits bexp/bexpef; response may strip bid.exp per auction response hook.
	OmitBidExpFromTracker bool
}

type AdUnitCtx struct {
	MatchedSlot              string
	IsRegex                  bool
	MatchedRegex             string
	SelectedSlotAdUnitConfig *adunitconfig.AdConfig
	AppliedSlotAdUnitConfig  *adunitconfig.AdConfig
	UsingDefaultConfig       bool
	AllowedConnectionTypes   []int
}

type CustomDimension struct {
	Value     string `json:"value,omitempty"`
	SendToGAM *bool  `json:"sendtoGAM,omitempty"`
}

// FeatureData struct to hold feature data from cache
type FeatureData struct {
	Enabled int    // feature enabled/disabled
	Value   string // feature value if any
}

type AppLovinMax struct {
	Reject bool
}

type MultiFloorsConfig struct {
	Enabled bool
	Config  ApplovinAdUnitFloors
}

type ApplovinAdUnitFloors map[string][]float64

type WinningBids map[string][]*OwBid

type HashSet map[string]struct{}

type ProfileAdUnitMultiFloors map[int]map[string]*MultiFloors

type MultiFloors struct {
	IsActive bool    `json:"isActive,omitempty"`
	Tier1    float64 `json:"tier1,omitempty"`
	Tier2    float64 `json:"tier2,omitempty"`
	Tier3    float64 `json:"tier3,omitempty"`
	Tier4    float64 `json:"tier4,omitempty"`
	Tier5    float64 `json:"tier5,omitempty"`
}

func (w WinningBids) IsWinningBid(impId, bidId string) bool {
	var isWinningBid bool

	wbids, ok := w[impId]
	if !ok {
		return isWinningBid
	}

	for i := range wbids {
		if bidId == wbids[i].ID {
			isWinningBid = true
			break
		}
	}

	return isWinningBid
}

func (w WinningBids) AppendBid(impId string, bid *OwBid) {
	wbid, ok := w[impId]
	if !ok {
		wbid = make([]*OwBid, 0)
	}

	wbid = append(wbid, bid)
	w[impId] = wbid
}

// isNewWinningBid calculates if the new bid (nbid) will win against the current winning bid (wbid) given preferDeals.
func IsNewWinningBid(bid, wbid *OwBid, preferDeals bool) bool {
	if preferDeals {
		//only wbid has deal
		if wbid.BidDealTierSatisfied && !bid.BidDealTierSatisfied {
			bid.Nbr = nbr.LossBidLostToDealBid.Ptr()
			return false
		}
		//only bid has deal
		if !wbid.BidDealTierSatisfied && bid.BidDealTierSatisfied {
			wbid.Nbr = nbr.LossBidLostToDealBid.Ptr()
			return true
		}
	}
	//both have deal or both do not have deal
	if bid.NetEcpm > wbid.NetEcpm {
		wbid.Nbr = nbr.LossBidLostToHigherBid.Ptr()
		return true
	}
	bid.Nbr = nbr.LossBidLostToHigherBid.Ptr()
	return false
}

func (ic *ImpCtx) DeepCopy() ImpCtx {
	impCtx := *ic
	impCtx.IsRewardInventory = ptrutil.Clone(ic.IsRewardInventory)
	impCtx.Video = ortb.DeepCopyImpVideo(ic.Video)
	impCtx.Native = ortb.DeepCopyImpNative(ic.Native)
	impCtx.IncomingSlots = slices.Clone(ic.IncomingSlots)
	impCtx.Bidders = maps.Clone(ic.Bidders)
	impCtx.NonMapped = maps.Clone(ic.NonMapped)
	impCtx.NewExt = slices.Clone(ic.NewExt)
	impCtx.BidCtx = maps.Clone(ic.BidCtx)

	return impCtx
}
