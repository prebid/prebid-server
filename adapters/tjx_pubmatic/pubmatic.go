package pubmatic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/cache/skanidlist"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// Region ...
type Region string

const (
	USEast Region = "us_east"
)

type PubmaticAdapter struct {
	URI              string
	SupportedRegions map[Region]string
}

type pubmaticBidExtVideo struct {
	Duration *int `json:"duration,omitempty"`
}

type pubmaticBidExt struct {
	BidType           *int                 `json:"BidType,omitempty"`
	VideoCreativeInfo *pubmaticBidExtVideo `json:"video,omitempty"`
}

type ExtImpBidderPubmatic struct {
	adapters.ExtImpBidder
	Data *ExtData `json:"data,omitempty"`
}

type ExtData struct {
	AdServer *ExtAdServer `json:"adserver"`
	PBAdSlot string       `json:"pbadslot"`
}

type ExtAdServer struct {
	Name   string `json:"name"`
	AdSlot string `json:"adslot"`
}

const (
	dctrKeyName        = "key_val"
	pmZoneIDKeyName    = "pmZoneId"
	pmZoneIDKeyNameOld = "pmZoneID"
	ImpExtAdUnitKey    = "dfp_ad_unit_code"
	AdServerGAM        = "gam"
)

func (a *PubmaticAdapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))

	var err error
	wrapExt := ""
	pubID := ""

	// copy the bidder request
	pubmaticRequest := *request

	var impData pubmaticImpData
	for i := 0; i < len(pubmaticRequest.Imp); i++ {
		impData, err = parseImpressionObject(&pubmaticRequest.Imp[i], &wrapExt, &pubID)

		// If the parsing is failed, remove imp and add the error.
		if err != nil {
			errs = append(errs, err)
			pubmaticRequest.Imp = append(pubmaticRequest.Imp[:i], pubmaticRequest.Imp[i+1:]...)
			i--
		}
	}

	// If all the requests are invalid, Call to adaptor is skipped
	if len(pubmaticRequest.Imp) == 0 {
		return nil, errs
	}

	// Overwrite BidFloor if present
	if impData.pubmatic.BidFloor != nil {
		pubmaticRequest.Imp[0].BidFloor = *impData.pubmatic.BidFloor
	}

	if wrapExt != "" {
		rawExt := fmt.Sprintf("{\"wrapper\": %s}", wrapExt)
		pubmaticRequest.Ext = json.RawMessage(rawExt)
	}

	if pubmaticRequest.Site != nil {
		siteCopy := *pubmaticRequest.Site
		if siteCopy.Publisher != nil {
			publisherCopy := *siteCopy.Publisher
			publisherCopy.ID = pubID
			siteCopy.Publisher = &publisherCopy
		} else {
			siteCopy.Publisher = &openrtb2.Publisher{ID: pubID}
		}
		pubmaticRequest.Site = &siteCopy
	} else if pubmaticRequest.App != nil {
		appCopy := *pubmaticRequest.App
		if appCopy.Publisher != nil {
			publisherCopy := *appCopy.Publisher
			publisherCopy.ID = pubID
			appCopy.Publisher = &publisherCopy
		} else {
			appCopy.Publisher = &openrtb2.Publisher{ID: pubID}
		}

		if impData.pubmatic.SiteID != 0 {
			appCopy.ID = strconv.Itoa(impData.pubmatic.SiteID)
		}

		pubmaticRequest.App = &appCopy
	}

	thisURI := a.URI

	if endpoint, ok := a.SupportedRegions[Region(impData.pubmatic.Region)]; ok {
		thisURI = endpoint
	}

	if impData.pubmatic.SiteID > 0 {
		thisURI = thisURI + "&siteId=" + strconv.Itoa(impData.pubmatic.SiteID)
	}

	// If all the requests are invalid, Call to adaptor is skipped
	if len(pubmaticRequest.Imp) == 0 {
		return nil, errs
	}

	pubmaticRequest.Ext = nil

	reqJSON, err := json.Marshal(pubmaticRequest)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	// Tapjoy Record placement type
	placementType := adapters.Interstitial
	if impData.pubmatic.Reward == 1 {
		placementType = adapters.Rewarded
	}

	skanSent := false
	// only add if present
	if len(adapters.FilterPrebidSKADNExt(impData.bidder.Prebid, skanidlist.Get(openrtb_ext.BidderPubmatic)).SKADNetIDs) > 0 {
		skanSent = true
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     thisURI,
		Body:    reqJSON,
		Headers: headers,

		TapjoyData: adapters.TapjoyData{
			Bidder:        string(openrtb_ext.BidderPubmatic),
			PlacementType: placementType,
			Region:        "us_east",
			SKAN: adapters.SKAN{
				Supported: impData.pubmatic.SKADNSupported,
				Sent:      skanSent,
			},
			MRAID: adapters.MRAID{
				Supported: impData.pubmatic.MRAIDSupported,
			},
		},
	}}, errs
}

// validateAdslot validate the optional adslot string
// valid formats are 'adslot@WxH', 'adslot' and no adslot
func validateAdSlot(adslot string, imp *openrtb2.Imp) error {
	adSlotStr := strings.TrimSpace(adslot)

	if len(adSlotStr) == 0 {
		return nil
	}

	if !strings.Contains(adSlotStr, "@") {
		imp.TagID = adSlotStr
		return nil
	}

	adSlot := strings.Split(adSlotStr, "@")
	if len(adSlot) == 2 && adSlot[0] != "" && adSlot[1] != "" {
		imp.TagID = strings.TrimSpace(adSlot[0])

		adSize := strings.Split(strings.ToLower(adSlot[1]), "x")
		if len(adSize) != 2 {
			return fmt.Errorf("Invalid size provided in adSlot %v", adSlotStr)
		}

		width, err := strconv.Atoi(strings.TrimSpace(adSize[0]))
		if err != nil {
			return fmt.Errorf("Invalid width provided in adSlot %v", adSlotStr)
		}

		heightStr := strings.Split(adSize[1], ":")
		height, err := strconv.Atoi(strings.TrimSpace(heightStr[0]))
		if err != nil {
			return fmt.Errorf("Invalid height provided in adSlot %v", adSlotStr)
		}

		// In case of video, size could be derived from the player size
		if imp.Banner != nil {
			imp.Banner = assignBannerWidthAndHeight(imp.Banner, int64(width), int64(height))
		}
	} else {
		return fmt.Errorf("Invalid adSlot %v", adSlotStr)
	}

	return nil
}

func assignBannerSize(banner *openrtb2.Banner) (*openrtb2.Banner, error) {
	if banner.W != nil && banner.H != nil {
		return banner, nil
	}

	return assignBannerWidthAndHeight(banner, banner.Format[0].W, banner.Format[0].H), nil
}

func assignBannerWidthAndHeight(banner *openrtb2.Banner, w, h int64) *openrtb2.Banner {
	bannerCopy := *banner
	bannerCopy.W = openrtb2.Int64Ptr(w)
	bannerCopy.H = openrtb2.Int64Ptr(h)
	return &bannerCopy
}

// Tapjoy type for returning useful data from
type pubmaticImpData struct {
	bidder   ExtImpBidderPubmatic
	pubmatic openrtb_ext.ExtImpTJXPubmatic
}

// parseImpressionObject parse the imp to get it ready to send to pubmatic
func parseImpressionObject(imp *openrtb2.Imp, wrapExt *string, pubID *string) (pubmaticImpData, error) {
	pubImpData := pubmaticImpData{}

	// PubMatic supports banner and video impressions.
	if imp.Banner == nil && imp.Video == nil {
		return pubImpData, fmt.Errorf("Invalid MediaType. PubMatic only supports Banner and Video. Ignoring ImpID=%s", imp.ID)
	}

	if imp.Audio != nil {
		imp.Audio = nil
	}

	var bidderExt ExtImpBidderPubmatic
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return pubImpData, err
	}
	pubImpData.bidder = bidderExt

	var pubmaticExt openrtb_ext.ExtImpTJXPubmatic
	if err := json.Unmarshal(bidderExt.Bidder, &pubmaticExt); err != nil {
		return pubImpData, err
	}
	pubImpData.pubmatic = pubmaticExt

	if *pubID == "" {
		*pubID = strings.TrimSpace(pubmaticExt.PublisherId)
	}

	// Parse Wrapper Extension only once per request
	if *wrapExt == "" && len(pubmaticExt.WrapExt) != 0 {
		var wrapExtMap map[string]int
		err := json.Unmarshal(pubmaticExt.WrapExt, &wrapExtMap)
		if err != nil {
			return pubImpData, fmt.Errorf("Error in Wrapper Parameters = %v  for ImpID = %v WrapperExt = %v", err.Error(), imp.ID, string(pubmaticExt.WrapExt))
		}
		*wrapExt = string(pubmaticExt.WrapExt)
	}

	if err := validateAdSlot(strings.TrimSpace(pubmaticExt.AdSlot), imp); err != nil {
		return pubImpData, err
	}

	if pubmaticExt.MRAIDSupported && imp.Banner != nil {
		bannerCopy, err := assignBannerSize(imp.Banner)
		if err != nil {
			return pubImpData, err
		}
		imp.Banner = bannerCopy
	} else {
		imp.Banner = nil
	}

	extMap := make(map[string]interface{}, 0)
	if pubmaticExt.Keywords != nil && len(pubmaticExt.Keywords) != 0 {
		addKeywordsToExt(pubmaticExt.Keywords, extMap)
	}
	// Give preference to direct values of 'dctr' & 'pmZoneId' params in extension
	if pubmaticExt.Dctr != "" {
		extMap[dctrKeyName] = pubmaticExt.Dctr
	}
	if pubmaticExt.PmZoneID != "" {
		extMap[pmZoneIDKeyName] = pubmaticExt.PmZoneID
	}

	if bidderExt.Data != nil {
		if bidderExt.Data.AdServer != nil && bidderExt.Data.AdServer.Name == AdServerGAM && bidderExt.Data.AdServer.AdSlot != "" {
			extMap[ImpExtAdUnitKey] = bidderExt.Data.AdServer.AdSlot
		} else if bidderExt.Data.PBAdSlot != "" {
			extMap[ImpExtAdUnitKey] = bidderExt.Data.PBAdSlot
		}
	}

	if err := populateExtensionMap(extMap, bidderExt, pubmaticExt); err != nil {
		return pubImpData, err
	}

	ext, err := json.Marshal(extMap)
	if err == nil {
		imp.Ext = ext
	} else {
		imp.Ext = nil
	}

	return pubImpData, nil
}

func addKeywordsToExt(keywords []*openrtb_ext.ExtImpTJXPubmaticKeyVal, extMap map[string]interface{}) {
	for _, keyVal := range keywords {
		if len(keyVal.Values) == 0 {
			logf("No values present for key = %s", keyVal.Key)
			continue
		} else {
			key := keyVal.Key
			if keyVal.Key == pmZoneIDKeyNameOld {
				key = pmZoneIDKeyName
			}
			extMap[key] = strings.Join(keyVal.Values[:], ",")
		}
	}
}

func (a *PubmaticAdapter) MakeBids(_ *openrtb2.BidRequest, _ *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode)}
	}

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	var errs []error
	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			impVideo := &openrtb_ext.ExtBidPrebidVideo{}

			if len(bid.Cat) > 1 {
				bid.Cat = bid.Cat[0:1]
			}

			var bidExt *pubmaticBidExt
			bidType := openrtb_ext.BidTypeBanner
			if err := json.Unmarshal(bid.Ext, &bidExt); err == nil && bidExt != nil {
				if bidExt.VideoCreativeInfo != nil && bidExt.VideoCreativeInfo.Duration != nil {
					impVideo.Duration = *bidExt.VideoCreativeInfo.Duration
				}
				bidType = getBidType(bidExt)
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:      &bid,
				BidType:  bidType,
				BidVideo: impVideo,
			})

		}
	}
	return bidResponse, errs
}

// getBidType returns the bid type specified in the response bid.ext
func getBidType(bidExt *pubmaticBidExt) openrtb_ext.BidType {
	// setting "banner" as the default bid type
	bidType := openrtb_ext.BidTypeBanner
	if bidExt != nil && bidExt.BidType != nil {
		switch *bidExt.BidType {
		case 0:
			bidType = openrtb_ext.BidTypeBanner
		case 1:
			bidType = openrtb_ext.BidTypeVideo
		case 2:
			bidType = openrtb_ext.BidTypeNative
		default:
			// default value is banner
			bidType = openrtb_ext.BidTypeBanner
		}
	}
	return bidType
}

func logf(msg string, args ...interface{}) {
	if glog.V(2) {
		glog.Infof(msg, args...)
	}
}

// Builder builds a new instance of the Pubmatic adapter for the given bidder with the given config.
func Builder(_ openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &PubmaticAdapter{
		URI: config.Endpoint,
		SupportedRegions: map[Region]string{
			USEast: config.Endpoint,
		},
	}
	return bidder, nil
}

func populateExtensionMap(imp map[string]interface{}, bidderExt ExtImpBidderPubmatic, pubmaticExt openrtb_ext.ExtImpTJXPubmatic) error {
	if pubmaticExt.Reward == 1 {
		imp["reward"] = pubmaticExt.Reward
	}

	if pubmaticExt.SKADNSupported {
		skanIDList := skanidlist.Get(openrtb_ext.BidderPubmatic)

		skadn := adapters.FilterPrebidSKADNExt(bidderExt.Prebid, skanIDList)

		// only add if present
		if len(skadn.SKADNetIDs) > 0 {
			imp["skadn"] = &skadn
		}
	}

	return nil
}
