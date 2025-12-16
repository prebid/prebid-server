package helpers

type MileAnalyticsEvent struct {
	Ip string `json:"ip"`

	ClientVersion string `json:"clientVersion"`

	Ua string `json:"ua"`

	CityName string `json:"cityName"`

	StateName string `json:"stateName"`

	CountryName string `json:"countryName"`

	ArbitraryData string `json:"arbitraryData"`

	Device string `json:"device"`

	Publisher string `json:"publisher"`

	Site string `json:"site"`

	ReferrerURL string `json:"referrerURL"`

	AdvertiserName string `json:"advertiserName"`

	AuctionID string `json:"auctionID"`

	Page string `json:"page"`

	YetiSiteID string `json:"yetiSiteID"`

	YetiPublisherID string `json:"yetiPublisherID"`

	SessionID string `json:"sessionID"`

	EventType string `json:"eventType"`

	Section string `json:"section"`

	Cls float64 `json:"cls"`

	Fcp int64 `json:"fcp"`

	Fid int64 `json:"fid"`

	Ttfb int64 `json:"ttfb"`

	BidBidders []string `json:"bidBidders"`

	ConfiguredBidders []string `json:"configuredBidders"`

	RequestedBidders []string `json:"requestedBidders"`

	RemovedBidders []string `json:"removedBidders"`

	IABCategories map[string]map[string]string `json:"IABCategories"`

	SizePrice map[string]map[string]float64 `json:"sizePrice"`

	StatisticalQuantities map[string]map[string]float64 `json:"statisticalQuantities"`

	UserID string `json:"userID"`

	PageViewID string `json:"pageViewID"`

	Lcp int64 `json:"lcp"`

	Cpm float64 `json:"cpm"`

	GamAdvertiserID int64 `json:"gamAdvertiserID"`

	GptAdUnit string `json:"gptAdUnit"`

	HasAdServerWonAuction bool `json:"hasAdServerWonAuction"`

	IsInfiniteScroll bool `json:"isInfiniteScroll"`

	HasPrebidWon bool `json:"hasPrebidWon"`

	IsGAMBackFill bool `json:"isGAMBackFill"`

	NoBidBidders []string `json:"noBidBidders"`

	PseudoAdUnitCode string `json:"pseudoAdUnitCode"`

	Viewability bool `json:"viewability"`

	WinningSize string `json:"winningSize"`

	WinningBidder string `json:"winningBidder"`

	TimedOutBidder []string `json:"timedOutBidder"`

	ConfiguredTimeout int64 `json:"configuredTimeout"`

	AdUnitCode string `json:"adUnitCode"`

	IsAXT bool `json:"isAXT"`

	IsMultiSizedUnit bool `json:"isMultiSizedUnit"`

	SizesRequested []string `json:"sizesRequested"`

	Revenue float64 `json:"revenue"`

	WinningRatio float64 `json:"winningRatio"`

	Impressions int64 `json:"impressions"`

	SessionPageViewCount int64 `json:"sessionPageViewCount"`

	Utm map[string]string `json:"utm"`

	Params map[string]map[string]string `json:"params"`

	Timestamp int64 `json:"timestamp"`

	ServerTimestamp int64 `json:"serverTimestamp"`

	InsertedAt int64 `json:"insertedAt"`

	Browser string `json:"browser"`

	ResponseTimes map[string]int64 `json:"responseTimes"`

	GamRecordedCPM float64 `json:"gamRecordedCPM"`

	SspAdvertiserDomain string `json:"sspAdvertiserDomain"`

	SiteUID string `json:"siteUID"`

	FloorMeta map[string]string `json:"floorMeta"`

	RejectedSizePrice map[string]map[string]float64 `json:"rejectedSizePrice"`

	RejectedBidders []string `json:"rejectedBidders"`

	SizeFloors map[string]map[string]string `json:"sizeFloors"`

	IsNewUser bool `json:"isNewUser"`

	DerivedBrowser string `json:"derivedBrowser"`

	ExprTags map[string]string `json:"exprTags"`

	AdType string `json:"adType"`

	RefreshBucket string `json:"refreshBucket"`

	ReferrerType string `json:"referrerType"`

	HasBid bool `json:"hasBid"`

	FloorPrice float64 `json:"floorPrice"`

	Bidder string `json:"bidder"`

	PageLayout string `json:"pageLayout"`

	UnfilledCPM float64 `json:"unfilledCPM"`

	Uuid string `json:"uuid"`

	FloorMech string `json:"floorMech"`

	Brif float64 `json:"brif"`

	AfihbsVersion string `json:"afihbsVersion"`

	InitPageLayout string `json:"initPageLayout"`

	DealIDsByBidder map[string]map[string]string `json:"dealIDsByBidder"`

	UserIDVendorsByBidder map[string]string `json:"userIDVendorsByBidder"`

	DealID string `json:"dealID"`

	UserIDVendors string `json:"userIDVendors"`

	MetaData map[string][]string `json:"metaData"`

	BiddersFloorMeta map[string]map[string]string `json:"biddersFloorMeta"`

	RevenueSource string `json:"revenueSource"`

	GamMatchedLineItemID int64 `json:"gamMatchedLineItemID"`

	GamYieldGroupIDs []int64 `json:"gamYieldGroupIDs"`

	IsNewSession bool `json:"isNewSession"`

	IsPBS bool `json:"isPBS"`
}

type ImpressionsExt struct {
	GPID   string `json:"gpid"`
	Tid    string `json:"tid"`
	Prebid struct {
		Bidder map[string]interface{} `json:"bidder"`
	} `json:"prebid"`
}

type RespExt struct {
	ResponseTimeMillis map[string]int64       `json:"responsetimemillis"`
	Errors             map[string][]ErrStruct `json:"errors"`
}

type ErrStruct struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (r *RespExt) getTimeoutBidders(timeout int64) []string {
	timeouts := []string{}
	if r.Errors != nil {
		for bidder := range r.Errors {
			timeouts = append(timeouts, bidder)
		}

	}
	return timeouts
}
