package pubmatic

import (
	"encoding/json"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/openrtb/v20/openrtb3"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// WloggerRecord structure for wrapper analytics logger object
type WloggerRecord struct {
	record
}

type record struct {
	Timeout               int              `json:"to,omitempty"`
	PubID                 int              `json:"pubid,omitempty"`
	PageURL               string           `json:"purl,omitempty"`
	Timestamp             int64            `json:"tst,omitempty"`
	IID                   string           `json:"iid,omitempty"`
	ProfileID             string           `json:"pid,omitempty"`
	VersionID             string           `json:"pdvid,omitempty"`
	IP                    string           `json:"-"`
	UserAgent             string           `json:"-"`
	UID                   string           `json:"-"`
	GDPR                  int8             `json:"gdpr,omitempty"`
	ConsentString         string           `json:"cns,omitempty"`
	PubmaticConsent       int              `json:"pmc,omitempty"`
	UserID                string           `json:"uid,omitempty"` // Not logged currently
	PageValue             float64          `json:"pv,omitempty"`  //sum of all winning bids // Not logged currently
	ServerLogger          int              `json:"sl,omitempty"`
	Slots                 []SlotRecord     `json:"s,omitempty"`
	CachePutMiss          int              `json:"cm,omitempty"`
	Origin                string           `json:"orig,omitempty"`
	Device                Device           `json:"dvc,omitempty"`
	AdPodPercentage       *AdPodPercentage `json:"aps,omitempty"`
	Content               *Content         `json:"ct,omitempty"`
	TestConfigApplied     int              `json:"tgid,omitempty"`
	VastUnwrapEnabled     int              `json:"vu,omitempty"`
	FloorModelVersion     string           `json:"fmv,omitempty"`
	FloorSource           *int             `json:"fsrc,omitempty"`
	FloorType             int              `json:"ft"`
	IntegrationType       string           `json:"it,omitempty"`
	FloorFetchStatus      *int             `json:"ffs,omitempty"`
	FloorProvider         string           `json:"fp,omitempty"`
	PDC                   string           `json:"pdc,omitempty"`
	CustomDimensions      string           `json:"cds,omitempty"`
	Geo                   GeoRecord        `json:"geo,omitempty"`
	ProfileType           int              `json:"pt,omitempty"`
	ProfileTypePlatform   int              `json:"ptp,omitempty"`
	AppPlatform           int              `json:"ap,omitempty"`
	AppIntegrationPath    *int             `json:"aip,omitempty"`
	AppSubIntegrationPath *int             `json:"asip,omitempty"`
	FloorSkippedFlag      *int             `json:"fskp,omitempty"`
	EdsStatus             *int             `json:"edss,omitempty"`
}

// Device struct for storing device information
type Device struct {
	Platform models.DevicePlatform `json:"plt,omitempty"`
	IFAType  *models.DeviceIFAType `json:"ifty,omitempty"` //OTT-416, adding device.ext.ifa_type
	ATTS     *float64              `json:"atts,omitempty"` //device.ext.atts
	ID       string                `json:"id,omitempty"`
	Model    string                `json:"md,omitempty"`
}

// GeoRecord structure for storing geo information
type GeoRecord struct {
	CountryCode string `json:"cc,omitempty"`
}

// AdPodPercentage will store adpod percentage value comes in request
type AdPodPercentage struct {
	CrossPodAdvertiserExclusionPercent  *int `json:"cpexap,omitempty"` //Percent Value - Across multiple impression there will be no ads from same advertiser. Note: These cross pod rule % values can not be more restrictive than per pod
	CrossPodIABCategoryExclusionPercent *int `json:"cpexip,omitempty"` //Percent Value - Across multiple impression there will be no ads from same advertiser
	IABCategoryExclusionWindow          *int `json:"exapw,omitempty"`  //Duration in minute between pods where exclusive IAB rule needs to be applied
	AdvertiserExclusionWindow           *int `json:"exipw,omitempty"`  //Duration in minute between pods where exclusive advertiser rule needs to be applied
}

// Content of openrtb request object
type Content struct {
	ID      string   `json:"id,omitempty"`  // ID uniquely identifying the content
	Episode int      `json:"eps,omitempty"` // Episode number (typically applies to video content).
	Title   string   `json:"ttl,omitempty"` // Content title.
	Series  string   `json:"srs,omitempty"` // Content series
	Season  string   `json:"ssn,omitempty"` // Content season
	Cat     []string `json:"cat,omitempty"` // Array of IAB content categories that describe the content producer
}

// AdPodSlot of adpod object logging
type AdPodSlot struct {
	MinAds                      int `json:"mnad,omitempty"` //Default 1 if not specified
	MaxAds                      int `json:"mxad,omitempty"` //Default 1 if not specified
	MinDuration                 int `json:"amnd,omitempty"` // (adpod.adminduration * adpod.minads) should be greater than or equal to video.minduration
	MaxDuration                 int `json:"amxd,omitempty"` // (adpod.admaxduration * adpod.maxads) should be less than or equal to video.maxduration + video.maxextended
	AdvertiserExclusionPercent  int `json:"exap,omitempty"` // Percent value 0 means none of the ads can be from same advertiser 100 means can have all same advertisers
	IABCategoryExclusionPercent int `json:"exip,omitempty"` // Percent value 0 means all ads should be of different IAB categories.
}

// SlotRecord structure for storing slot level information
type SlotRecord struct {
	SlotId            string          `json:"sid"`
	SlotName          string          `json:"sn,omitempty"`
	SlotSize          []string        `json:"sz,omitempty"`
	Adunit            string          `json:"au,omitempty"`
	AdPodSlot         *AdPodSlot      `json:"aps,omitempty"`
	PartnerData       []PartnerRecord `json:"ps"`
	RewardedInventory int             `json:"rwrd,omitempty"` // Indicates if the ad slot was enabled (rwrd=1) for rewarded or disabled (rwrd=0)
	DisplayManager    string          `json:"dm,omitempty"`
	DisplayManagerVer string          `json:"dmv,omitempty"`
}

// PartnerRecord structure for storing partner information
type PartnerRecord struct {
	PartnerID            string  `json:"pn"`
	BidderCode           string  `json:"bc"`
	KGPV                 string  `json:"kgpv"`  // In case of Regex mapping, this will contain the regex string.
	KGPSV                string  `json:"kgpsv"` // In case of Regex mapping, this will contain the actual slot name that matched the regex.
	PartnerSize          string  `json:"psz"`   //wxh
	Adformat             string  `json:"af"`
	GrossECPM            float64 `json:"eg"`
	NetECPM              float64 `json:"en"`
	Latency1             int     `json:"l1"` //response time
	Latency2             int     `json:"l2"`
	PostTimeoutBidStatus int     `json:"t"`
	WinningBidStaus      int     `json:"wb"`
	BidID                string  `json:"bidid"`
	OrigBidID            string  `json:"origbidid"`
	DealID               string  `json:"di"`
	DealChannel          string  `json:"dc"`
	DealPriority         int     `json:"dp,omitempty"`
	DefaultBidStatus     int     `json:"db"`
	ServerSide           int     `json:"ss"`
	MatchedImpression    int     `json:"mi"`

	//AdPod Specific
	AdPodSequenceNumber *int     `json:"adsq,omitempty"`
	AdDuration          *int     `json:"dur,omitempty"`
	ADomain             string   `json:"adv,omitempty"`
	Cat                 []string `json:"cat,omitempty"`
	NoBidReason         *int     `json:"aprc,omitempty"`

	OriginalCPM float64   `json:"ocpm"`
	OriginalCur string    `json:"ocry"`
	MetaData    *MetaData `json:"md,omitempty"`

	FloorValue             float64               `json:"fv,omitempty"`
	FloorRuleValue         float64               `json:"frv,omitempty"`
	Nbr                    *openrtb3.NoBidReason `json:"nbr,omitempty"` // NonBR reason code
	PriceBucket            string                `json:"pb,omitempty"`
	MultiBidMultiFloorFlag int                   `json:"mbmf,omitempty"`
	Bundle                 string                `json:"bndl,omitempty"`
	InViewCountingFlag     int                   `json:"ctm,omitempty"`
}

type MetaData struct {
	NetworkID            int             `json:"nwid,omitempty"`
	AdvertiserID         int             `json:"adid,omitempty"`
	NetworkName          string          `json:"nwnm,omitempty"`
	PrimaryCategoryID    string          `json:"pcid,omitempty"`
	AdvertiserName       string          `json:"adnm,omitempty"`
	AgencyID             int             `json:"agid,omitempty"`
	AgencyName           string          `json:"agnm,omitempty"`
	BrandID              int             `json:"brid,omitempty"`
	BrandName            string          `json:"brnm,omitempty"`
	DChain               json.RawMessage `json:"dc,omitempty"`
	DemandSource         string          `json:"ds,omitempty"`
	SecondaryCategoryIDs []string        `json:"secondaryCatIds,omitempty"`
}

var FloorSourceMap = map[string]int{
	openrtb_ext.NoDataLocation:  0,
	openrtb_ext.RequestLocation: 1,
	openrtb_ext.FetchLocation:   2,
}

// FetchStatusMap maps floor fetch status with integer codes
var FetchStatusMap = map[string]int{
	openrtb_ext.FetchNone:       0,
	openrtb_ext.FetchSuccess:    1,
	openrtb_ext.FetchError:      2,
	openrtb_ext.FetchInprogress: 3,
	openrtb_ext.FetchTimeout:    4,
}

// SetIntegrationType sets the integration type in WloggerRecord
func (wlog *WloggerRecord) logIntegrationType(endpoint string) {
	switch endpoint {
	case models.EndpointAMP:
		wlog.IntegrationType = models.TypeAmp
	case models.EndpointV25, models.EndpointAppLovinMax, models.EndpointGoogleSDK, models.EndpointUnityLevelPlay, models.EndpointAPS:
		wlog.IntegrationType = models.TypeSDK
	case models.EndpointVAST:
		wlog.IntegrationType = models.TypeTag
	case models.EndpointJson, models.EndpointVideo:
		wlog.IntegrationType = models.TypeInline
	case models.EndpointORTB:
		wlog.IntegrationType = models.TypeS2S
	case models.EndpointWebS2S:
		wlog.IntegrationType = models.TypeWebS2S
	}
}

func (wlog *WloggerRecord) logDeviceObject(dvc *models.DeviceCtx) {
	if dvc == nil {
		return
	}

	wlog.Device.Platform = dvc.Platform
	wlog.Device.IFAType = dvc.IFATypeID
	wlog.Device.ID = dvc.ID
	wlog.Device.Model = dvc.Model
	if dvc.Ext != nil {
		wlog.record.Device.ATTS, _ = dvc.Ext.GetAtts()
	}
}

// logFloorType will be used to log floor type
func (wlog *WloggerRecord) logFloorType(prebid *openrtb_ext.ExtRequestPrebid) {
	wlog.record.FloorType = models.SoftFloor
	if prebid != nil && prebid.Floors != nil &&
		prebid.Floors.Enabled != nil && *prebid.Floors.Enabled &&
		prebid.Floors.Enforcement != nil && prebid.Floors.Enforcement.EnforcePBS != nil && *prebid.Floors.Enforcement.EnforcePBS {
		wlog.record.FloorType = models.HardFloor
	}
}

// logContentObject adds the content object details in logger
func (wlog *WloggerRecord) logContentObject(content *openrtb2.Content) {
	if nil == content {
		return
	}

	wlog.Content = &Content{
		ID:      content.ID,
		Episode: int(content.Episode),
		Title:   content.Title,
		Series:  content.Series,
		Season:  content.Season,
		Cat:     content.Cat,
	}
}

// setMetaDataObject sets the MetaData object for partner-record
func (partnerRecord *PartnerRecord) setMetaDataObject(meta *openrtb_ext.ExtBidPrebidMeta) {
	if meta.NetworkID != 0 || meta.AdvertiserID != 0 || len(meta.SecondaryCategoryIDs) > 0 {
		partnerRecord.MetaData = &MetaData{
			NetworkID:            meta.NetworkID,
			AdvertiserID:         meta.AdvertiserID,
			PrimaryCategoryID:    meta.PrimaryCategoryID,
			AgencyID:             meta.AgencyID,
			DemandSource:         meta.DemandSource,
			SecondaryCategoryIDs: meta.SecondaryCategoryIDs,
		}
	}
	//NOTE : We Don't get following Data points in Response, whenever got from translator,
	//they can be populated.
	//partnerRecord.MetaData.NetworkName = meta.NetworkName
	//partnerRecord.MetaData.AdvertiserName = meta.AdvertiserName
	//partnerRecord.MetaData.AgencyName = meta.AgencyName
	//partnerRecord.MetaData.BrandName = meta.BrandName
	//partnerRecord.MetaData.BrandID = meta.BrandID
	//partnerRecord.MetaData.DChain = meta.DChain (type is json.RawMessage)
}
