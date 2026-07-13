package tracker

import (
	"bytes"
	"fmt"
	"net/url"
	"strconv"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/exchange"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/customdimensions"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/utils"
)

// pubmatic's KGP details per impression
type pubmaticMarketplaceMeta struct {
	PubmaticKGP, PubmaticKGPV, PubmaticKGPSV string
}

func CreateTrackers(rctx models.RequestCtx, bidResponse *openrtb2.BidResponse, globalAccountConfig *config.Account) map[string]models.OWTracker {
	trackers := make(map[string]models.OWTracker)

	pmMkt := make(map[string]pubmaticMarketplaceMeta)

	trackers = createTrackers(rctx, trackers, bidResponse, pmMkt, globalAccountConfig)

	// overwrite marketplace bid details with that of parent bidder
	for bidID, tracker := range trackers {
		if _, ok := rctx.MarketPlaceBidders[tracker.Tracker.PartnerInfo.BidderCode]; ok {
			if v, ok := pmMkt[tracker.Tracker.ImpID]; ok {
				tracker.Tracker.PartnerInfo.PartnerID = "pubmatic"
				tracker.Tracker.PartnerInfo.KGPV = v.PubmaticKGPV
				tracker.Tracker.LoggerData.KGPSV = v.PubmaticKGPSV
			}
		}

		var finalTrackerURL string
		trackerURL := constructTrackerURL(rctx, tracker.Tracker)
		trackURL, err := url.Parse(trackerURL)
		if err == nil {
			trackURL.Scheme = models.HTTPSProtocol
			finalTrackerURL = trackURL.String()
		}
		tracker.TrackerURL = finalTrackerURL

		trackers[bidID] = tracker
	}

	return trackers
}

func createTrackers(rctx models.RequestCtx, trackers map[string]models.OWTracker, bidResponse *openrtb2.BidResponse, pmMkt map[string]pubmaticMarketplaceMeta, globalAccountConfig *config.Account) map[string]models.OWTracker {
	floorsDetails := models.GetFloorsDetails(rctx.ResponseExt)
	customDimensions := customdimensions.ConvertCustomDimensionsToString(rctx.CustomDimensions)
	for _, seatBid := range bidResponse.SeatBid {
		for _, bid := range seatBid.Bid {
			impId := bid.ImpID
			if rctx.IsCTVRequest {
				impId, _ = models.GetImpressionID(bid.ImpID)
			}

			tracker := models.Tracker{
				PubID:             rctx.PubID,
				ProfileID:         fmt.Sprintf("%d", rctx.ProfileID),
				VersionID:         fmt.Sprintf("%d", rctx.DisplayVersionID),
				PageURL:           rctx.PageURL,
				Timestamp:         rctx.StartTime,
				IID:               rctx.LoggerImpressionID,
				Platform:          int(rctx.DeviceCtx.Platform),
				SSAI:              rctx.SSAI,
				ImpID:             impId,
				Origin:            rctx.Origin,
				AdPodSlot:         0, //TODO: Need to changes based on AdPodSlot Obj for CTV Req
				TestGroup:         rctx.ABTestConfigApplied,
				FloorModelVersion: floorsDetails.FloorModelVersion,
				FloorType:         floorsDetails.FloorType,
				FloorSkippedFlag:  floorsDetails.Skipfloors,
				FloorSource:       floorsDetails.FloorSource,
				CustomDimensions:  customDimensions,
				LoggerData: models.LoggerData{
					FloorFetchStatus: floorsDetails.FloorFetchStatus,
					FloorProvider:    floorsDetails.FloorProvider,
				},
				VastUnwrapEnabled: utils.ConvertBoolToInt(rctx.VastUnWrap.Enabled),
			}
			var (
				kgp, kgpv, kgpsv, matchedSlot, adformat, bidId = "", "", "", "", "banner", ""
				floorValue, floorRuleValue                     = float64(0), float64(0)
				partnerID                                      = seatBid.Seat
				isRewardInventory, adduration                  = 0, 0
				dspId, mbmfFlag                                int
				eg, en                                         float64
				networkId                                      int
			)

			if rctx.DeviceCtx.Ext != nil {
				tracker.ATTS, _ = rctx.DeviceCtx.Ext.GetAtts()
			}

			if impCtx, ok := rctx.ImpBidCtx[impId]; ok {
				if bidderMeta, ok := impCtx.Bidders[seatBid.Seat]; ok {
					matchedSlot = bidderMeta.MatchedSlot
					partnerID = bidderMeta.PrebidBidderCode
				}

				bidCtx, ok := impCtx.BidCtx[bid.ID]
				if ok {
					// TODO do most calculation in wt
					// marketplace/alternatebiddercodes feature
					bidExt := bidCtx.BidExt
					if bidExt.MultiBidMultiFloorValue > 0 {
						mbmfFlag = 1
					}

					if bidExt.Prebid != nil {
						if bidExt.Prebid.Video != nil && bidExt.Prebid.Video.Duration > 0 {
							adduration = bidExt.Prebid.Video.Duration
						}
						if len(bidExt.Prebid.BidId) > 0 {
							bidId = bidExt.Prebid.BidId
						}
						if bidExt.Prebid.Meta != nil {
							if len(bidExt.Prebid.Meta.AdapterCode) != 0 && seatBid.Seat != bidExt.Prebid.Meta.AdapterCode {

								if aliasSeat, ok := rctx.PrebidBidderCode[partnerID]; ok {
									if bidderMeta, ok := impCtx.Bidders[aliasSeat]; ok {
										matchedSlot = bidderMeta.MatchedSlot
									}
								}
							}
							// Extract NWID from bid.meta.networkId if present
							if bidExt.Prebid.Meta.NetworkID != 0 {
								networkId = bidExt.Prebid.Meta.NetworkID
							}
						}
					}
					dspId = bidCtx.DspId
					eg = bidCtx.EG
					en = bidCtx.EN
					adformat = models.GetAdFormat(&bid, &bidExt, &impCtx)
					floorValue, floorRuleValue = models.GetBidLevelFloorsDetails(bidExt, impCtx, rctx.CurrencyConversion)
				}

				_ = matchedSlot
				// --------------------------------------------------------------------------------------------------
				// Move this code to a function. Confirm the kgp, kgpv, kgpsv relation in wt and wl.
				// --------------------------------------------------------------------------------------------------
				// var kgp, kgpv, kgpsv string

				if bidderMeta, ok := impCtx.Bidders[seatBid.Seat]; ok {
					partnerID = bidderMeta.PrebidBidderCode
					kgp = bidderMeta.KGP
					kgpv, kgpsv = models.GetKGPSV(bid, &bidCtx.BidExt, bidderMeta, adformat, impCtx.TagID, impCtx.Div, rctx.Source)
				}
				// --------------------------------------------------------------------------------------------------

				tracker.SlotID = impCtx.SlotName
				tracker.LoggerData.KGPSV = kgpsv
				tracker.Secure = impCtx.Secure
				tracker.Adunit = impCtx.AdUnitName
				isRewardInventory = 0
				if impCtx.IsRewardInventory != nil {
					isRewardInventory = int(*impCtx.IsRewardInventory)
				}

				if impCtx.AdpodConfig != nil {
					tracker.AdPodSlot = models.AdPodEnabled
				}
				tracker.DisplayManager = impCtx.DisplayManager
				tracker.DisplayManagerVer = impCtx.DisplayManagerVer
			}

			if seatBid.Seat == "pubmatic" {
				pmMkt[impId] = pubmaticMarketplaceMeta{
					PubmaticKGP:   kgp,
					PubmaticKGPV:  kgpv,
					PubmaticKGPSV: kgpsv,
				}
			}

			tracker.RewardedInventory = isRewardInventory
			inViewCountingFlag := trackerWithOM(rctx, partnerID, dspId, bid.Ext, adformat)
			tracker.PartnerInfo = models.Partner{
				PartnerID:              partnerID,
				BidderCode:             seatBid.Seat,
				BidID:                  utils.GetOriginalBidId(bid.ID),
				OrigBidID:              utils.GetOriginalBidId(bid.ID),
				KGPV:                   kgpv,
				NetECPM:                en,
				GrossECPM:              eg,
				AdSize:                 models.GetSizeForPlatform(bid.W, bid.H, rctx.Platform),
				AdDuration:             adduration,
				Adformat:               adformat,
				ServerSide:             1,
				FloorValue:             floorValue,
				FloorRuleValue:         floorRuleValue,
				DealID:                 "-1",
				MultiBidMultiFloorFlag: mbmfFlag,
				NetworkID:              networkId,
				InViewCountingFlag:     utils.ConvertBoolToInt(inViewCountingFlag),
			}
			if rctx.PriceGranularity != nil {
				tracker.PartnerInfo.PriceBucket = exchange.GetPriceBucketOW(bid.Price, *rctx.PriceGranularity, *globalAccountConfig)
			}
			if len(bidId) > 0 {
				tracker.PartnerInfo.BidID = bidId
			}
			if len(bid.ADomain) != 0 {
				if domain, err := models.ExtractDomain(bid.ADomain[0]); err == nil {
					tracker.PartnerInfo.Advertiser = domain
				}
			}
			if len(bid.DealID) > 0 {
				tracker.PartnerInfo.DealID = bid.DealID
			}

			if impCtx, okImp := rctx.ImpBidCtx[impId]; okImp {
				if bc, okBid := impCtx.BidCtx[bid.ID]; okBid {
					if !bc.OmitBidExpFromTracker {
						if bid.Exp > 0 {
							tracker.BidExp = bid.Exp
						}
						if bc.BidExpEnf == 1 {
							tracker.BidExpEnf = 1
						}
					}
				}
			}

			var finalTrackerURL string
			trackerURL := constructTrackerURL(rctx, tracker)
			trackURL, err := url.Parse(trackerURL)
			if err == nil {
				trackURL.Scheme = models.HTTPSProtocol
				finalTrackerURL = trackURL.String()
			}

			trackers[bid.ID] = models.OWTracker{
				Tracker:       tracker,
				TrackerURL:    finalTrackerURL,
				Price:         bid.Price,
				PriceModel:    models.VideoPricingModelCPM,
				PriceCurrency: bidResponse.Cur,
				ErrorURL:      constructVideoErrorURL(rctx, rctx.VideoErrorTrackerEndpoint, bid, tracker),
				BidType:       adformat,
				IsOMEnabled:   inViewCountingFlag,
			}
		}
	}
	return trackers
}

// ConstructTrackerURL constructing tracker url for impression
func constructTrackerURL(rctx models.RequestCtx, tracker models.Tracker) string {
	trackerURL, err := url.Parse(rctx.TrackerEndpoint)
	if err != nil {
		return ""
	}

	v := url.Values{}
	v.Set(models.TRKPubID, strconv.Itoa(tracker.PubID))
	v.Set(models.TRKPageURL, tracker.PageURL)
	v.Set(models.TRKTimestamp, strconv.FormatInt(tracker.Timestamp, 10))
	v.Set(models.TRKIID, tracker.IID)
	v.Set(models.TRKProfileID, tracker.ProfileID)
	v.Set(models.TRKVersionID, tracker.VersionID)
	v.Set(models.TRKSlotID, tracker.SlotID)
	v.Set(models.TRKAdunit, tracker.Adunit)
	if tracker.RewardedInventory == 1 {
		v.Set(models.TRKRewardedInventory, strconv.Itoa(tracker.RewardedInventory))
	}
	v.Set(models.TRKPlatform, strconv.Itoa(tracker.Platform))
	if tracker.TestGroup != 0 {
		v.Set(models.TRKTestGroup, strconv.Itoa(tracker.TestGroup))
	}
	v.Set(models.TRKPubDomain, tracker.Origin)
	v.Set(models.TRKAdPodExist, strconv.Itoa(tracker.AdPodSlot))
	partner := tracker.PartnerInfo
	v.Set(models.TRKPartnerID, partner.PartnerID)
	v.Set(models.TRKBidderCode, partner.BidderCode)
	v.Set(models.TRKKGPV, partner.KGPV)
	v.Set(models.TRKGrossECPM, fmt.Sprint(partner.GrossECPM))
	v.Set(models.TRKNetECPM, fmt.Sprint(partner.NetECPM))
	v.Set(models.TRKBidID, partner.BidID)
	if tracker.SSAI != "" {
		v.Set(models.TRKSSAI, tracker.SSAI)
	}
	v.Set(models.TRKOrigBidID, partner.OrigBidID)
	v.Set(models.TRKAdSize, partner.AdSize)
	if partner.AdDuration > 0 {
		v.Set(models.TRKAdDuration, strconv.Itoa(partner.AdDuration))
	}
	v.Set(models.TRKAdformat, partner.Adformat)
	v.Set(models.TRKServerSide, strconv.Itoa(partner.ServerSide))
	v.Set(models.TRKAdvertiser, partner.Advertiser)
	if partner.NetworkID != 0 {
		v.Set(models.TRKNetworkID, strconv.Itoa(partner.NetworkID))
	}

	v.Set(models.TRKFloorType, strconv.Itoa(tracker.FloorType))
	if tracker.FloorSkippedFlag != nil {
		v.Set(models.TRKFloorSkippedFlag, strconv.Itoa(*tracker.FloorSkippedFlag))
	}
	if len(tracker.FloorModelVersion) > 0 {
		v.Set(models.TRKFloorModelVersion, tracker.FloorModelVersion)
	}
	if len(tracker.LoggerData.FloorProvider) > 0 {
		v.Set(models.TRKFloorProvider, tracker.LoggerData.FloorProvider)
	}

	if tracker.FloorSource != nil {
		v.Set(models.TRKFloorSource, strconv.Itoa(*tracker.FloorSource))
	}
	if partner.FloorValue > 0 {
		v.Set(models.TRKFloorValue, fmt.Sprint(partner.FloorValue))
	}
	if partner.FloorRuleValue > 0 {
		v.Set(models.TRKFloorRuleValue, fmt.Sprint(partner.FloorRuleValue))
	}
	v.Set(models.TRKServerLogger, "1")
	v.Set(models.TRKDealID, partner.DealID)
	if tracker.CustomDimensions != "" {
		v.Set(models.TRKCustomDimensions, tracker.CustomDimensions)
	}
	if tracker.ATTS != nil {
		v.Set(models.TRKATTS, strconv.Itoa(int(*tracker.ATTS)))
	}
	if tracker.PartnerInfo.PriceBucket != "" {
		v.Set(models.TRKPriceBucket, tracker.PartnerInfo.PriceBucket)
	}
	if tracker.PartnerInfo.MultiBidMultiFloorFlag == 1 {
		v.Set(models.TRKMultiBidMultiFloorFlag, strconv.Itoa(tracker.PartnerInfo.MultiBidMultiFloorFlag))
	}
	if tracker.PartnerInfo.InViewCountingFlag == 1 {
		v.Set(models.TRKInViewCountingFlag, strconv.Itoa(tracker.PartnerInfo.InViewCountingFlag))
	}

	//ProfileMetadata parameters
	if rctx.ProfileType > 0 {
		v.Set(models.TRKProfileType, strconv.Itoa(rctx.ProfileType))
	}
	if rctx.ProfileTypePlatform > 0 {
		v.Set(models.TRKProfileTypePlatform, strconv.Itoa(rctx.ProfileTypePlatform))
	}
	if rctx.AppPlatform > 0 {
		v.Set(models.TRKAppPlatform, strconv.Itoa(rctx.AppPlatform))
	}
	if rctx.AppIntegrationPath != nil && *rctx.AppIntegrationPath >= 0 {
		v.Set(models.TRKAppIntegrationPath, strconv.Itoa(*rctx.AppIntegrationPath))
	}
	if rctx.AppSubIntegrationPath != nil && *rctx.AppSubIntegrationPath >= 0 {
		v.Set(models.TRKAppSubIntegrationPath, strconv.Itoa(*rctx.AppSubIntegrationPath))
	}

	if len(tracker.DisplayManager) > 0 {
		v.Set(models.TRKDisplayManager, tracker.DisplayManager)
	}
	if len(tracker.DisplayManagerVer) > 0 {
		v.Set(models.TRKDisplayManagerVer, tracker.DisplayManagerVer)
	}
	if len(rctx.DeviceCtx.DerivedCountryCode) > 0 {
		v.Set(models.TRKCountryCode, rctx.DeviceCtx.DerivedCountryCode)
	}
	if rctx.VastUnWrap.Enabled {
		v.Set(models.TRKVastUnwrapEnabled, strconv.Itoa(tracker.VastUnwrapEnabled))
	}
	if tracker.BidExp > 0 {
		v.Set(models.TRKBidExp, strconv.FormatInt(tracker.BidExp, 10))
	}
	if tracker.BidExpEnf == 1 {
		v.Set(models.TRKBidExpEnf, strconv.Itoa(tracker.BidExpEnf))
	}

	queryString := v.Encode()

	//Code for making tracker call http/https based on secure flag for in-app platform
	//TODO change platform to models.PLATFORM_APP once in-app platform starts populating from wrapper UI
	if rctx.Platform == models.PLATFORM_DISPLAY {
		if tracker.Secure == 1 {
			trackerURL.Scheme = "https"
		} else {
			trackerURL.Scheme = "http"
		}

	}
	trackerQueryStr := trackerURL.String() + models.TRKQMARK + queryString
	return trackerQueryStr
}

// ConstructVideoErrorURL constructing video error url for video impressions
func constructVideoErrorURL(rctx models.RequestCtx, errorURLString string, bid openrtb2.Bid, tracker models.Tracker) string {
	if len(errorURLString) == 0 {
		return ""
	}

	errorURL, err := url.Parse(errorURLString)
	if err != nil {
		return ""
	}

	errorURL.Scheme = models.HTTPSProtocol
	tracker.SURL = url.QueryEscape(rctx.Origin)

	//operId Note: It should be first parameter in url otherwise it will get failed at analytics side.
	if len(errorURL.RawQuery) > 0 {
		errorURL.RawQuery = models.ERROperIDParam + models.TRKAmpersand + errorURL.RawQuery
	} else {
		errorURL.RawQuery = models.ERROperIDParam
	}

	v := url.Values{}
	v.Set(models.ERRPubID, strconv.Itoa(tracker.PubID))                  //pubId
	v.Set(models.ERRProfileID, tracker.ProfileID)                        //profileId
	v.Set(models.ERRVersionID, tracker.VersionID)                        //versionId
	v.Set(models.ERRTimestamp, strconv.FormatInt(tracker.Timestamp, 10)) //ts
	v.Set(models.ERRPartnerID, tracker.PartnerInfo.PartnerID)            //pid
	v.Set(models.ERRBidderCode, tracker.PartnerInfo.BidderCode)          //bc
	v.Set(models.ERRAdunit, tracker.Adunit)                              //au
	v.Set(models.ERRSUrl, tracker.SURL)                                  // sURL
	v.Set(models.ERRPlatform, strconv.Itoa(tracker.Platform))            // pfi
	v.Set(models.ERRAdvertiser, tracker.PartnerInfo.Advertiser)          // adv
	if tracker.TestGroup != 0 {
		v.Set(models.ERRTestGroup, fmt.Sprintf("%d", tracker.TestGroup)) // tgid
	}
	if tracker.VastUnwrapEnabled == 1 {
		v.Set(models.ERRVastUnwrap, fmt.Sprintf("%d", tracker.VastUnwrapEnabled)) // vu
	}

	if tracker.SSAI != "" {
		v.Set(models.ERRSSAI, tracker.SSAI) // ssai for video/json endpoint
	}

	if bid.CrID == "" {
		v.Set(models.ERRCreativeID, "-1")
	} else {
		v.Set(models.ERRCreativeID, bid.CrID) //creativeId
	}

	var out bytes.Buffer
	out.WriteString(errorURL.String())
	out.WriteString(models.TRKAmpersand)
	out.WriteString(v.Encode())
	out.WriteString(models.TRKAmpersand)
	out.WriteString(models.ERRErrorCodeParam) //ier

	//queryString +=
	errorURLQueryStr := out.String()

	return errorURLQueryStr
}
