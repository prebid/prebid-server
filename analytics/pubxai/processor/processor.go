package processor

import (
	"github.com/golang/glog"
	"github.com/prebid/openrtb/v20/openrtb2"
	utils "github.com/prebid/prebid-server/v3/analytics/pubxai/utils"
)

type ProcessorService interface {
	ProcessLogData(ao *utils.LogObject) (*utils.AuctionBids, []utils.WinningBid)
	ProcessBidData(bidResponses []map[string]interface{}, impsById map[string]openrtb2.Imp, ao *utils.LogObject) (*utils.AuctionBids, []utils.WinningBid)
}

type ProcessorServiceImpl struct {
	publisherId  string
	samplingRate int
}

func NewProcessorService(publisherId string, samplingRate int) ProcessorService {
	return &ProcessorServiceImpl{
		publisherId:  publisherId,
		samplingRate: samplingRate,
	}
}

func (p *ProcessorServiceImpl) ProcessBidData(bidResponses []map[string]interface{}, impsById map[string]openrtb2.Imp, ao *utils.LogObject) (*utils.AuctionBids, []utils.WinningBid) {
	startTime := ao.StartTime.UTC().UnixMilli()

	requestExt, responseExt, err := utils.UnmarshalExtensions(ao)
	if err != nil {
		return nil, nil
	}

	auctionId := requestExt["id"].(string)

	adUnitCodes := utils.ExtractAdunitCodes(requestExt)
	floorDetail := utils.ExtractFloorDetail(requestExt)
	pageDetail := utils.ExtractPageData(requestExt)
	deviceDetail := utils.ExtractDeviceData(requestExt)
	userDetail := utils.ExtractUserIds(requestExt)
	consentDetail := utils.ExtractConsentTypes(requestExt)
	pmacDetail := map[string]interface{}{}

	auctionDetail := utils.AuctionDetail{
		AuctionId:   auctionId,
		RefreshRank: 0,
		Timestamp:   startTime,
		AdUnitCodes: adUnitCodes,
	}

	initOptions := utils.InitOptions{
		Pubxid:       p.publisherId,
		SamplingRate: p.samplingRate,
	}

	auctionBids, winningBids := utils.ProcessBidResponses(bidResponses, auctionId, startTime, requestExt, responseExt, floorDetail)

	auctionBids = utils.AppendTimeoutBids(auctionBids, impsById, ao)
	// make AuctionBids object and winningBids Object
	auctionObj := utils.AuctionBids{
		AuctionDetail: auctionDetail,
		FloorDetail:   floorDetail,
		PageDetail:    pageDetail,
		DeviceDetail:  deviceDetail,
		UserDetail:    userDetail,
		ConsentDetail: consentDetail,
		PmacDetail:    pmacDetail,
		InitOptions:   initOptions,
		Bids:          auctionBids,
		Source:        "PBS",
	}
	var winningBidsList []utils.WinningBid
	for _, winningBid := range winningBids {
		temp := utils.WinningBid{
			AuctionDetail: auctionDetail,
			FloorDetail:   floorDetail,
			PageDetail:    pageDetail,
			DeviceDetail:  deviceDetail,
			UserDetail:    userDetail,
			ConsentDetail: consentDetail,
			PmacDetail:    pmacDetail,
			InitOptions:   initOptions,
			WinningBid:    winningBid,
			Source:        "PBS",
		}
		winningBidsList = append(winningBidsList, temp)
	}
	return &auctionObj, winningBidsList

}

func (p *ProcessorServiceImpl) ProcessLogData(ao *utils.LogObject) (*utils.AuctionBids, []utils.WinningBid) {
	if ao == nil {
		return nil, nil
	}
	if ao.RequestWrapper == nil {
		glog.Errorf("[pubxai] RequestWrapper is nil")
		return nil, nil
	}
	request := ao.RequestWrapper.BidRequest

	imps := request.Imp
	var impsById = make(map[string]openrtb2.Imp)
	for _, imp := range imps {
		impsById[imp.ID] = imp
	}

	if len(imps) == 0 {
		glog.Errorf("[pubxai] No impressions in request")
		return nil, nil
	}

	response := ao.Response
	if response == nil {
		glog.Errorf("[pubxai] Response is nil")
		return nil, nil
	}
	seatBids := response.SeatBid

	var BidResponses []map[string]interface{}
	for _, seatBid := range seatBids {
		bidderName := seatBid.Seat
		bids := seatBid.Bid
		for _, bid := range bids {
			if _, ok := impsById[bid.ImpID]; ok {
				temp := map[string]interface{}{
					"bidder": bidderName,
					"bid":    bid,
					"imp":    impsById[bid.ImpID],
				}
				BidResponses = append(BidResponses, temp)
			}
		}
	}
	return p.ProcessBidData(BidResponses, impsById, ao)
}
