package pubmatic

import (
	"encoding/json"
	"fmt"
	"strings"

	"net/http"
	"strconv"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/openrtb/v20/openrtb3"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/exchange"
	"github.com/prebid/prebid-server/v3/hooks/hookexecution"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/customdimensions"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models/nbr"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/utils"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	uuid "github.com/satori/go.uuid"
)

type bidWrapper struct {
	*openrtb2.Bid
	Nbr *openrtb3.NoBidReason
}

// GetUUID is a function variable which will return uuid
var GetUUID = func() string {
	return uuid.NewV4().String()
}

var blockListedNBR = map[openrtb3.NoBidReason]struct{}{
	nbr.RequestBlockedPartnerThrottle: {},
	nbr.RequestBlockedPartnerFiltered: {},
}

// GetLogAuctionObjectAsURL will form the owlogger-url and http-headers
func GetLogAuctionObjectAsURL(ao analytics.AuctionObject, rCtx *models.RequestCtx, logInfo, forRespExt bool) (string, http.Header) {
	if ao.RequestWrapper == nil || ao.RequestWrapper.BidRequest == nil || rCtx == nil || rCtx.PubID == 0 || rCtx.LoggerDisabled {
		return "", nil
	}
	// Get Updated Floor values using floor rules from updated request
	getFloorValueFromUpdatedRequest(ao.RequestWrapper, rCtx)

	wlog := WloggerRecord{
		record: record{
			PubID:             rCtx.PubID,
			ProfileID:         fmt.Sprintf("%d", rCtx.ProfileID),
			VersionID:         fmt.Sprintf("%d", rCtx.DisplayVersionID),
			Origin:            rCtx.Origin,
			PageURL:           rCtx.PageURL,
			IID:               rCtx.LoggerImpressionID,
			Timestamp:         rCtx.StartTime,
			ServerLogger:      1,
			TestConfigApplied: rCtx.ABTestConfigApplied,
			Timeout:           int(rCtx.TMax),
			PDC:               rCtx.DCName,
			CachePutMiss:      rCtx.CachePutMiss,
		},
	}

	if len(rCtx.DeviceCtx.DerivedCountryCode) > 0 {
		wlog.Geo.CountryCode = rCtx.DeviceCtx.DerivedCountryCode
	}

	if rCtx.VastUnWrap.Enabled {
		wlog.VastUnwrapEnabled = 1
	}

	wlog.logProfileMetaData(rCtx)
	wlog.logEdsStatus(rCtx)

	wlog.logIntegrationType(rCtx.Endpoint)

	wlog.logDeviceObject(&rCtx.DeviceCtx)

	if ao.RequestWrapper.User != nil {
		wlog.ConsentString = ao.RequestWrapper.User.Consent
	}

	if ao.RequestWrapper.Regs != nil && ao.RequestWrapper.Regs.GDPR != nil {
		wlog.GDPR = *ao.RequestWrapper.Regs.GDPR
	}

	if ao.RequestWrapper.Site != nil {
		wlog.logContentObject(ao.RequestWrapper.Site.Content)
	} else if ao.RequestWrapper.App != nil {
		wlog.logContentObject(ao.RequestWrapper.App.Content)
	}

	var ipr map[string][]PartnerRecord
	if logInfo {
		ipr = getDefaultPartnerRecordsByImp(rCtx)
	} else {
		ipr = getPartnerRecordsByImp(ao, rCtx)
	}

	// parent bidder could in one of the above and we need them by prebid's bidderCode and not seat(could be alias)
	slots := make([]SlotRecord, 0)
	for impId, impCtx := range rCtx.ImpBidCtx {
		reward := 0
		if impCtx.IsRewardInventory != nil {
			reward = int(*impCtx.IsRewardInventory)
		}

		// to keep existing response intact
		partnerData := make([]PartnerRecord, 0)
		if ipr[impId] != nil {
			partnerData = ipr[impId]
		}

		slots = append(slots, SlotRecord{
			SlotId:            GetUUID(),
			SlotName:          impCtx.SlotName,
			SlotSize:          impCtx.IncomingSlots,
			Adunit:            impCtx.AdUnitName,
			PartnerData:       partnerData,
			RewardedInventory: int(reward),
			AdPodSlot:         getAdPodSlot(impCtx.AdpodConfig),
			DisplayManager:    impCtx.DisplayManager,
			DisplayManagerVer: impCtx.DisplayManagerVer,
		})
	}

	wlog.Slots = slots

	headers := http.Header{
		models.USER_AGENT_HEADER: []string{rCtx.DeviceCtx.UA},
		models.IP_HEADER:         []string{rCtx.DeviceCtx.IP},
	}

	// first set the floor type from bidrequest.ext
	if rCtx.NewReqExt != nil {
		wlog.logFloorType(&rCtx.NewReqExt.Prebid)
	}

	// set the floor details and cds from tracker
	cdsAndfloorDetailsSet := false
	for _, tracker := range rCtx.Trackers {
		wlog.CustomDimensions = tracker.Tracker.CustomDimensions
		wlog.FloorType = tracker.Tracker.FloorType
		wlog.FloorModelVersion = tracker.Tracker.FloorModelVersion
		wlog.FloorSource = tracker.Tracker.FloorSource
		wlog.FloorFetchStatus = tracker.Tracker.LoggerData.FloorFetchStatus
		wlog.FloorProvider = tracker.Tracker.LoggerData.FloorProvider
		wlog.FloorSkippedFlag = tracker.Tracker.FloorSkippedFlag

		cdsAndfloorDetailsSet = true
		break // For all trackers, floor-details and cds are common so break the loop
	}

	// if floor details not present in tracker then use response.ext
	// this wil happen only if no valid bid is present in response.seatbid
	if !cdsAndfloorDetailsSet {
		wlog.CustomDimensions = customdimensions.ConvertCustomDimensionsToString(rCtx.CustomDimensions)
		if rCtx.ResponseExt.Prebid != nil {
			// wlog.SetFloorDetails(rCtx.ResponseExt.Prebid.Floors)
			floorDetails := models.GetFloorsDetails(rCtx.ResponseExt)
			wlog.FloorSource = floorDetails.FloorSource
			wlog.FloorModelVersion = floorDetails.FloorModelVersion
			wlog.FloorFetchStatus = floorDetails.FloorFetchStatus
			wlog.FloorProvider = floorDetails.FloorProvider
			wlog.FloorType = floorDetails.FloorType
			wlog.FloorSkippedFlag = floorDetails.Skipfloors
		}
	}

	url := ow.cfg.Endpoint
	if logInfo {
		url = ow.cfg.PublicEndpoint
	}

	return PrepareLoggerURL(&wlog, url, getGdprEnabledFlag(rCtx.PartnerConfigMap)), headers
}

// TODO filter by name. (*stageOutcomes[8].Groups[0].InvocationResults[0].AnalyticsTags.Activities[0].Results[0].Values["request-ctx"].(data))
func GetRequestCtx(hookExecutionOutcome []hookexecution.StageOutcome) *models.RequestCtx {
	for _, stageOutcome := range hookExecutionOutcome {
		for _, groups := range stageOutcome.Groups {
			for _, invocationResult := range groups.InvocationResults {
				for _, activity := range invocationResult.AnalyticsTags.Activities {
					for _, result := range activity.Results {
						if result.Values != nil {
							if irctx, ok := result.Values["request-ctx"]; ok {
								rctx, ok := irctx.(*models.RequestCtx)
								if !ok {
									return nil
								}
								return rctx
							}
						}
					}
				}
			}
		}
	}
	return nil
}

// getFloorValueFromUpdatedRequest gets updated floor values by floor module
func getFloorValueFromUpdatedRequest(reqWrapper *openrtb_ext.RequestWrapper, rCtx *models.RequestCtx) {
	for _, imp := range reqWrapper.BidRequest.Imp {
		if impCtx, ok := rCtx.ImpBidCtx[imp.ID]; ok {
			if imp.BidFloor > 0 && impCtx.BidFloor != imp.BidFloor {
				impCtx.BidFloor = imp.BidFloor
				impCtx.BidFloorCur = imp.BidFloorCur
				rCtx.ImpBidCtx[imp.ID] = impCtx
			}
		}
	}
}

func convertNonBidToBidWrapper(nonBid *openrtb_ext.NonBid) (bid bidWrapper) {
	bid.Bid = &openrtb2.Bid{
		Price:   nonBid.Ext.Prebid.Bid.Price,
		ADomain: nonBid.Ext.Prebid.Bid.ADomain,
		CatTax:  nonBid.Ext.Prebid.Bid.CatTax,
		Cat:     nonBid.Ext.Prebid.Bid.Cat,
		DealID:  nonBid.Ext.Prebid.Bid.DealID,
		W:       nonBid.Ext.Prebid.Bid.W,
		H:       nonBid.Ext.Prebid.Bid.H,
		Dur:     nonBid.Ext.Prebid.Bid.Dur,
		MType:   nonBid.Ext.Prebid.Bid.MType,
		ID:      nonBid.Ext.Prebid.Bid.ID,
		ImpID:   nonBid.ImpId,
		Bundle:  nonBid.Ext.Prebid.Bid.Bundle,
	}
	bidExt := models.BidExt{
		OriginalBidCPM:    nonBid.Ext.Prebid.Bid.OriginalBidCPM,
		OriginalBidCPMUSD: nonBid.Ext.Prebid.Bid.OriginalBidCPMUSD,
		OriginalBidCur:    nonBid.Ext.Prebid.Bid.OriginalBidCur,
		ExtBid: openrtb_ext.ExtBid{
			Prebid: &openrtb_ext.ExtBidPrebid{
				DealPriority:      nonBid.Ext.Prebid.Bid.DealPriority,
				DealTierSatisfied: nonBid.Ext.Prebid.Bid.DealTierSatisfied,
				Meta:              nonBid.Ext.Prebid.Bid.Meta,
				Targeting:         nonBid.Ext.Prebid.Bid.Targeting,
				Type:              nonBid.Ext.Prebid.Bid.Type,
				Video:             nonBid.Ext.Prebid.Bid.Video,
				BidId:             nonBid.Ext.Prebid.Bid.BidId,
				Floors:            nonBid.Ext.Prebid.Bid.Floors,
			},
		},
	}
	bidExtBytes, err := json.Marshal(bidExt)
	if err == nil {
		bid.Ext = bidExtBytes
	}
	// the 'nbr' field will be lost due to json.Marshal hence do not set it inside bid.Ext
	// set the 'nbr' code at bid level, while forming partner-records we give high priority to bid.nbr over bid.ext.nbr
	bid.Nbr = openrtb3.NoBidReason(nonBid.StatusCode).Ptr()
	return bid
}

// getPartnerRecordsByImp creates partnerRecords of valid-bids + dropped-bids + non-bids+ default-bids for owlogger
func getPartnerRecordsByImp(ao analytics.AuctionObject, rCtx *models.RequestCtx) map[string][]PartnerRecord {
	// impID-partnerRecords: partner records per impression
	ipr := make(map[string][]PartnerRecord)

	rejectedBids := map[string]map[string]struct{}{}
	defaultBidsInSeatBid := make(map[string]map[string]struct{})
	loggerSeat := make(map[string][]bidWrapper)

	// currently, ao.SeatNonBid will contain the bids that got rejected in the prebid core
	// it does not contain the bids that got rejected inside pubmatic ow module
	// As of now, pubmatic ow module logs slot-not-map and partner-throttle related nonbids
	// for which we don't create partner-record in the owlogger
	for _, seatNonBid := range ao.SeatNonBid {
		if _, ok := rejectedBids[seatNonBid.Seat]; !ok {
			rejectedBids[seatNonBid.Seat] = map[string]struct{}{}
		}
		for _, nonBid := range seatNonBid.NonBid {
			rejectedBids[seatNonBid.Seat][nonBid.ImpId] = struct{}{}
			loggerSeat[seatNonBid.Seat] = append(loggerSeat[seatNonBid.Seat], convertNonBidToBidWrapper(&nonBid))
		}
	}

	// SeatBid contains valid-bids + default/proxy bids
	// loggerSeat should not contain duplicate entry for same imp-seat combination
	for seatIndex, seatBid := range ao.Response.SeatBid {
		for bidIndex, bid := range seatBid.Bid {
			// Check if this is a default as well as nonbid.
			// Ex. if only one bid is returned by pubmatic and it got rejected due to floors.
			// then OW-Module will add one default/proxy bid in seatbid.
			// and prebid core will add one nonbid in seat-non-bid.
			// So, we want to skip this default/proxy bid to avoid duplicate entry in logger
			if models.IsDefaultBid(&bid) {
				if _, ok := rejectedBids[seatBid.Seat]; ok {
					if _, ok := rejectedBids[seatBid.Seat][bid.ImpID]; ok {
						continue
					}
				}

				if _, ok := defaultBidsInSeatBid[seatBid.Seat]; !ok {
					defaultBidsInSeatBid[seatBid.Seat] = make(map[string]struct{})
				}
				defaultBidsInSeatBid[seatBid.Seat][bid.ImpID] = struct{}{}
			}
			loggerSeat[seatBid.Seat] = append(loggerSeat[seatBid.Seat], bidWrapper{Bid: &ao.Response.SeatBid[seatIndex].Bid[bidIndex]})
		}
	}

	// When sendAllBids is false, default bids are not merged into SeatBid; still log them unless the same
	// imp+seat is already represented as a seat-non-bid (rejectedBids), to avoid duplicate partner rows.
	if !rCtx.SendAllBids && rCtx.DefaultBids != nil {
		for impID := range rCtx.DefaultBids {
			for seat := range rCtx.DefaultBids[impID] {
				bids := rCtx.DefaultBids[impID][seat]
				for i := range bids {
					bid := &rCtx.DefaultBids[impID][seat][i]

					// Skip if this default bid was already logged from SeatBid
					if seatDefaults, ok := defaultBidsInSeatBid[seat]; ok {
						if _, exists := seatDefaults[bid.ImpID]; exists {
							continue
						}
					}

					// Existing check
					if _, ok := rejectedBids[seat]; ok {
						if _, ok := rejectedBids[seat][bid.ImpID]; ok {
							continue
						}
					}
					loggerSeat[seat] = append(loggerSeat[seat], bidWrapper{Bid: bid})
				}
			}
		}
	}

	// include bids that got dropped from ao.SeatBid by pubmatic ow module. Ex. sendAllBids=false
	// in future, this dropped bids should be part of ao.SeatNonBid object
	for seat, Bids := range rCtx.DroppedBids {
		for bid := range Bids {
			loggerSeat[seat] = append(loggerSeat[seat], bidWrapper{Bid: &rCtx.DroppedBids[seat][bid]})
		}
	}

	// pubmatic's KGP details per impression
	// This is only required for groupm bids (groupm bids log pubmatic's data in ow logger)
	type pubmaticMarketplaceMeta struct {
		PubmaticKGP, PubmaticKGPV, PubmaticKGPSV string
	}
	pmMkt := make(map[string]pubmaticMarketplaceMeta)

	// loggerSeat contains valid-bids + non-bids + dropped-bids + default-bids (ex-partnerTimeout)
	for seat, bids := range loggerSeat {
		if seat == string(openrtb_ext.BidderOWPrebidCTV) {
			continue
		}

		// owlogger would not contain throttled bidders, do we need this
		if _, ok := rCtx.AdapterThrottleMap[seat]; ok {
			continue
		}

		for _, bid := range bids {
			var sequence int
			impId := bid.ImpID
			if rCtx.IsCTVRequest {
				impId, sequence = models.GetImpressionID(impId)
			}
			impCtx, ok := rCtx.ImpBidCtx[impId]
			if !ok {
				continue
			}

			// owlogger would not contain non-mapped bidders, do we need this
			if _, ok := impCtx.NonMapped[seat]; ok {
				continue
			}

			var kgp, kgpv, kgpsv, adFormat string

			revShare := 0.0
			partnerID := seat
			bidderMeta, ok := impCtx.Bidders[seat]
			if ok {
				partnerID = bidderMeta.PrebidBidderCode
				kgp = bidderMeta.KGP
				revShare, _ = strconv.ParseFloat(rCtx.PartnerConfigMap[bidderMeta.PartnerID][models.REVSHARE], 64)
			}

			// impBidCtx contains info about valid-bids + dropped-bids + default-bids
			// impBidCtx not contains info about seat-non-bids.
			// impBidCtx contains bid-id in form of 'bid-id::uuid' for valid-bids + dropped-bids
			// impBidCtx contains bid-id in form of 'uuid' for default-bids
			var bidExt models.BidExt
			json.Unmarshal(bid.Ext, &bidExt)

			bidIDForLookup := bid.ID
			if bidExt.Prebid != nil && !strings.Contains(bid.ID, models.BidIdSeparator) {
				// this block will not be executed for default-bids.
				bidIDForLookup = utils.SetUniqueBidID(bid.ID, bidExt.Prebid.BidId)
			}

			bidCtx, ok := impCtx.BidCtx[bidIDForLookup]
			if ok {
				// override bidExt for valid-bids + default-bids + dropped-bids
				// since we have already prepared it under auction-response-hook
				bidExt = bidCtx.BidExt
			}

			// get the tracker details from rctx, to avoid repetitive computation.
			// tracker will be available only for valid-bids and will be absent for dropped-bids + default-bids + seat-non-bids
			// tracker contains bid-id in form of 'bid-id::uuid'
			tracker, trackerPresent := rCtx.Trackers[bidIDForLookup]

			adFormat = tracker.Tracker.PartnerInfo.Adformat
			if adFormat == "" {
				adFormat = models.GetAdFormat(bid.Bid, &bidExt, &impCtx)
			}

			kgpv = tracker.Tracker.PartnerInfo.KGPV
			kgpsv = tracker.Tracker.LoggerData.KGPSV
			if kgpv == "" || kgpsv == "" {
				kgpv, kgpsv = models.GetKGPSV(*bid.Bid, &bidExt, bidderMeta, adFormat, impCtx.TagID, impCtx.Div, rCtx.Source)
			}

			price := bid.Price
			// If bids are rejected before setting bidExt.OriginalBidCPM, calculate the price and ocpm values based on the currency and revshare.
			price = computeBidPriceForBidsRejectedBeforeSettingOCPM(rCtx, &bidExt, price, revShare, ao)
			bid.Price = price
			if ao.Response.Cur != models.USD {
				if bidCtx.EN != 0 { // valid-bids + dropped-bids+ default-bids
					price = bidCtx.EN
				} else if bidExt.OriginalBidCPMUSD != 0 { // valid non-bids
					price = bidExt.OriginalBidCPMUSD
				}
			}

			if seat == models.BidderPubMatic {
				pmMkt[impId] = pubmaticMarketplaceMeta{
					PubmaticKGP:   kgp,
					PubmaticKGPV:  kgpv,
					PubmaticKGPSV: kgpsv,
				}
			}

			nbr := bid.Nbr // only for seat-non-bids this will present at bid level
			if nbr == nil {
				nbr = bidExt.Nbr // valid-bids + default-bids + dropped-bids
			}

			if nbr != nil {
				if _, ok := blockListedNBR[*nbr]; ok {
					continue
				}
			}

			pr := PartnerRecord{
				PartnerID:              partnerID,                           // prebid biddercode
				BidderCode:             seat,                                // pubmatic biddercode: pubmatic2
				Latency1:               rCtx.BidderResponseTimeMillis[seat], // it is set inside auctionresponsehook for all bidders
				KGPV:                   kgpv,
				KGPSV:                  kgpsv,
				BidID:                  utils.GetOriginalBidId(bid.ID),
				OrigBidID:              utils.GetOriginalBidId(bid.ID),
				DefaultBidStatus:       0, // this will be always 0 , decide whether to drop this field in future
				ServerSide:             1,
				MatchedImpression:      rCtx.MatchedImpression[seat],
				OriginalCPM:            models.GetGrossEcpm(bidExt.OriginalBidCPM),
				OriginalCur:            bidExt.OriginalBidCur,
				DealID:                 bid.DealID,
				Nbr:                    nbr,
				Adformat:               adFormat,
				NetECPM:                tracker.Tracker.PartnerInfo.NetECPM,
				GrossECPM:              tracker.Tracker.PartnerInfo.GrossECPM,
				PartnerSize:            tracker.Tracker.PartnerInfo.AdSize,
				ADomain:                tracker.Tracker.PartnerInfo.Advertiser,
				MultiBidMultiFloorFlag: tracker.Tracker.PartnerInfo.MultiBidMultiFloorFlag,
				Bundle:                 bid.Bid.Bundle,
				InViewCountingFlag:     utils.ConvertBoolToInt(tracker.IsOMEnabled),
			}

			if models.IsDefaultBid(bid.Bid) {
				pr.DefaultBidStatus = 1
			}

			if pr.MultiBidMultiFloorFlag == 0 && bidExt.MultiBidMultiFloorValue > 0 {
				pr.MultiBidMultiFloorFlag = 1
			}

			if pr.NetECPM == 0 {
				pr.NetECPM = models.ToFixed(price, models.BID_PRECISION)
			}

			if pr.GrossECPM == 0 {
				pr.GrossECPM = models.GetGrossEcpmFromNetEcpm(price, revShare)
			}

			if pr.PartnerSize == "" {
				pr.PartnerSize = models.GetSizeForPlatform(bid.W, bid.H, rCtx.Platform)
			}

			if trackerPresent {
				pr.FloorRuleValue = tracker.Tracker.PartnerInfo.FloorRuleValue
				pr.FloorValue = tracker.Tracker.PartnerInfo.FloorValue
			} else {
				pr.FloorValue, pr.FloorRuleValue = models.GetBidLevelFloorsDetails(bidExt, impCtx, rCtx.CurrencyConversion)
			}

			if nbr != nil && *nbr == exchange.ErrorTimeout {
				pr.PostTimeoutBidStatus = 1
				pr.Latency1 = 0
			}

			// WinningBids contains map of imp.id against bid.id+::+uuid
			if rCtx.WinningBids.IsWinningBid(impId, bidIDForLookup) {
				pr.WinningBidStaus = 1
			}

			if len(pr.OriginalCur) == 0 {
				pr.OriginalCPM = float64(0)
				pr.OriginalCur = models.USD
			}

			if len(pr.DealID) != 0 {
				pr.DealChannel = models.DEFAULT_DEALCHANNEL
			} else {
				pr.DealID = models.DealIDAbsent
			}

			if bidExt.Prebid != nil {
				if bidExt.Prebid.DealTierSatisfied && bidExt.Prebid.DealPriority > 0 {
					pr.DealPriority = bidExt.Prebid.DealPriority
				}

				if bidExt.Prebid.Video != nil && bidExt.Prebid.Video.Duration > 0 {
					pr.AdDuration = &bidExt.Prebid.Video.Duration
				}

				if bidExt.Prebid.Meta != nil {
					pr.setMetaDataObject(bidExt.Prebid.Meta)
				}

				if len(bidExt.Prebid.BidId) > 0 {
					pr.BidID = bidExt.Prebid.BidId
				}
			}

			if pr.ADomain == "" && len(bid.ADomain) != 0 {
				if domain, err := models.ExtractDomain(bid.ADomain[0]); err == nil {
					pr.ADomain = domain
				}
			}

			if len(bid.Cat) > 0 {
				pr.Cat = append(pr.Cat, bid.Cat...)
			}

			// Adpod parameters
			if impCtx.AdpodConfig != nil {
				pr.AdPodSequenceNumber = &sequence
				aprc := int(impCtx.BidIDToAPRC[bidIDForLookup])
				pr.NoBidReason = &aprc
			}

			pr.PriceBucket = tracker.Tracker.PartnerInfo.PriceBucket
			if ao.Account == nil {
				ao.Account = &config.Account{
					BidRounding: config.DefaultBidRoundingMode,
				}
			} else if ao.Account.BidRounding == "" {
				ao.Account.BidRounding = config.DefaultBidRoundingMode
			}

			if !models.IsDefaultBid(bid.Bid) && pr.PriceBucket == "" && rCtx.PriceGranularity != nil {
				pr.PriceBucket = exchange.GetPriceBucketOW(bid.Price, *rCtx.PriceGranularity, *ao.Account)
			}

			ipr[impId] = append(ipr[impId], pr)
		}
	}

	// overwrite marketplace bid details with that of partner adatper
	if rCtx.MarketPlaceBidders != nil {
		for impID, partnerRecords := range ipr {
			for i := 0; i < len(partnerRecords); i++ {
				if _, ok := rCtx.MarketPlaceBidders[partnerRecords[i].BidderCode]; ok {
					partnerRecords[i].PartnerID = "pubmatic"
					partnerRecords[i].KGPV = pmMkt[impID].PubmaticKGPV
					partnerRecords[i].KGPSV = pmMkt[impID].PubmaticKGPSV
				}
			}
			ipr[impID] = partnerRecords
		}
	}

	return ipr
}

// getDefaultPartnerRecordsByImp creates partnerRecord with default placeholders if req.ext.wrapper.loginfo=true.
// in future, check if this can be deprecated
func getDefaultPartnerRecordsByImp(rCtx *models.RequestCtx) map[string][]PartnerRecord {
	ipr := make(map[string][]PartnerRecord)
	for impID := range rCtx.ImpBidCtx {
		ipr[impID] = []PartnerRecord{{
			ServerSide:       1,
			DefaultBidStatus: 1,
			PartnerSize:      "0x0",
			DealID:           "-1",
		}}
	}
	return ipr
}

func getAdPodSlot(adPodConfig *models.AdPod) *AdPodSlot {
	if adPodConfig == nil {
		return nil
	}

	adPodSlot := AdPodSlot{
		MinAds:                      adPodConfig.MinAds,
		MaxAds:                      adPodConfig.MaxAds,
		MinDuration:                 adPodConfig.MinDuration,
		MaxDuration:                 adPodConfig.MaxDuration,
		AdvertiserExclusionPercent:  *adPodConfig.AdvertiserExclusionPercent,
		IABCategoryExclusionPercent: *adPodConfig.IABCategoryExclusionPercent,
	}

	return &adPodSlot
}

func GetBidPriceAfterCurrencyConversion(price float64, requestCurrencies []string, responseCurrency string,
	currencyConverter func(fromCurrency string, toCurrency string, value float64) (float64, error)) float64 {
	if len(requestCurrencies) == 0 {
		requestCurrencies = []string{models.USD}
	}
	for _, requestCurrency := range requestCurrencies {
		if value, err := currencyConverter(responseCurrency, requestCurrency, price); err == nil {
			return value
		}
	}
	return 0 // in case of error, send 0 value to make it consistent with prebid
}

func computeBidPriceForBidsRejectedBeforeSettingOCPM(rCtx *models.RequestCtx, bidExt *models.BidExt,
	price, revshare float64, ao analytics.AuctionObject) float64 {
	if price != 0 && bidExt.OriginalBidCPM == 0 {
		if len(bidExt.OriginalBidCur) == 0 {
			bidExt.OriginalBidCur = models.USD
		}
		bidExt.OriginalBidCPM = price
		price = price * models.GetBidAdjustmentValue(revshare)
		if cpmUSD, err := rCtx.CurrencyConversion(bidExt.OriginalBidCur, models.USD, price); err == nil {
			bidExt.OriginalBidCPMUSD = cpmUSD
		}
		price = GetBidPriceAfterCurrencyConversion(price, ao.RequestWrapper.Cur, bidExt.OriginalBidCur, rCtx.CurrencyConversion)
	}
	return price
}
